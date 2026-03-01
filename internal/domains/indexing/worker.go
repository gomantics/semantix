package indexing

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomantics/semantix/config"
	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/domains/gittokens"
	"github.com/gomantics/semantix/internal/domains/repos"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"github.com/gomantics/semantix/internal/libs/chunking"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
	"github.com/gomantics/semantix/internal/libs/openai"
	"github.com/gomantics/semantix/internal/qdrant"
	"github.com/gomantics/semantix/pkg/pgconv"
	pb "github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

// Worker processes a single repo indexing job.
type Worker struct {
	l        *zap.Logger
	cloner   gitrepo.Cloner
	embedder openai.Embedder
}

// WorkerOption configures a Worker.
type WorkerOption func(*Worker)

// WithCloner sets the Cloner implementation (for testing).
func WithCloner(c gitrepo.Cloner) WorkerOption {
	return func(w *Worker) { w.cloner = c }
}

// WithEmbedder sets the Embedder implementation (for testing).
func WithEmbedder(e openai.Embedder) WorkerOption {
	return func(w *Worker) { w.embedder = e }
}

// NewWorker creates a Worker with optional overrides. Production code uses
// the real cloner and the default OpenAI embedder.
func NewWorker(l *zap.Logger, opts ...WorkerOption) *Worker {
	w := &Worker{
		l:        l,
		cloner:   &gitrepo.DefaultCloner{},
		embedder: openai.GetDefaultEmbedder(),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// runStats tracks progress during an indexing run.
type runStats struct {
	filesProcessed      int32
	chunksCreated       int32
	embeddingsGenerated int32
}

func (w *Worker) Process(ctx context.Context, repo repos.Repo) {
	startTime := time.Now()
	l := w.l

	run, err := w.createRun(ctx, repo.ID)
	if err != nil {
		l.Error("failed to create index run", zap.Error(err))
		return
	}

	stats := &runStats{}
	if err := w.processRepo(ctx, repo, stats); err != nil {
		w.failRun(ctx, run.ID, repo.ID, startTime, err)
		l.Error("indexing failed", zap.Error(err))
		return
	}

	w.completeRun(ctx, run.ID, repo.ID, startTime, stats)
	l.Info("indexing complete",
		zap.Int32("files", stats.filesProcessed),
		zap.Int32("chunks", stats.chunksCreated),
		zap.Int32("embeddings", stats.embeddingsGenerated),
		zap.Duration("duration", time.Since(startTime)),
	)
}

func (w *Worker) processRepo(ctx context.Context, repo repos.Repo, stats *runStats) error {
	ws, err := workspaces.GetByID(ctx, repo.WorkspaceID)
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	// Step 1: Clone
	if _, err := repos.UpdateStatus(ctx, repo.ID, repos.StatusCloning, nil); err != nil {
		return fmt.Errorf("set cloning status: %w", err)
	}

	cloneDir := gitrepo.RepoDir(config.Indexing.CloneDir(), repo.WorkspaceID, repo.ID)
	if err := w.cloneRepo(ctx, repo, cloneDir); err != nil {
		return fmt.Errorf("clone: %w", err)
	}

	// Step 2: Set indexing status
	if _, err := repos.UpdateStatus(ctx, repo.ID, repos.StatusIndexing, nil); err != nil {
		return fmt.Errorf("set indexing status: %w", err)
	}

	// Step 3: Walk, chunk, embed, upsert
	if err := w.indexFiles(ctx, repo, cloneDir, ws.Settings, stats); err != nil {
		return err
	}

	return nil
}

func (w *Worker) cloneRepo(ctx context.Context, repo repos.Repo, cloneDir string) error {
	provider := gitrepo.DetectProvider(repo.URL)

	var token string
	if repo.GitTokenID != nil {
		gt, err := gittokens.GetByID(ctx, *repo.GitTokenID)
		if err != nil {
			return fmt.Errorf("load git token %d: %w", *repo.GitTokenID, err)
		}
		token = gt.Token
	}

	return w.cloner.Clone(ctx, gitrepo.CloneOptions{
		URL:      repo.URL,
		Branch:   repo.Branch,
		Token:    token,
		Provider: provider,
		DestDir:  cloneDir,
	})
}

func (w *Worker) indexFiles(ctx context.Context, repo repos.Repo, rootDir string, settings workspaces.WorkspaceSettings, stats *runStats) error {
	maxFileSize := config.Indexing.MaxFileSizeBytes()

	type fileEntry struct {
		relPath string
		absPath string
		size    int64
	}

	var files []fileEntry
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)

		// Always apply hard infrastructure excludes (.git, node_modules, etc.).
		if gitrepo.ShouldExclude(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply workspace-level exclude patterns.
		if matchesAnyPattern(relPath, settings.ExcludePatterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Apply workspace-level include patterns (when set, only matching files are indexed).
		if len(settings.IncludePatterns) > 0 && !matchesAnyPattern(relPath, settings.IncludePatterns) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if info.Size() > maxFileSize || info.Size() == 0 {
			return nil
		}

		files = append(files, fileEntry{relPath: relPath, absPath: path, size: info.Size()})
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}

	w.l.Info("found files to index", zap.Int("count", len(files)))

	var allChunks []chunking.Chunk
	var allFileIDs []int64

	for _, f := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		content, err := os.ReadFile(f.absPath)
		if err != nil {
			w.l.Warn("failed to read file, skipping", zap.String("path", f.relPath), zap.Error(err))
			continue
		}

		contentHash := fmt.Sprintf("%x", sha256.Sum256(content))

		// Check if file content has changed
		existing, _ := db.Query1(ctx, func(q *db.Queries) (db.File, error) {
			return q.GetFileByRepoAndPath(ctx, db.GetFileByRepoAndPathParams{
				RepoID: repo.ID,
				Path:   f.relPath,
			})
		})
		if existing.ID != 0 && existing.ContentHash == contentHash {
			stats.filesProcessed++
			continue
		}

		chunks, err := chunking.ChunkFile(f.relPath, content)
		if err != nil {
			w.l.Warn("failed to chunk file, skipping", zap.String("path", f.relPath), zap.Error(err))
			continue
		}

		now := time.Now().UnixNano()
		dbFile, err := db.Tx1(ctx, func(q *db.Queries) (db.File, error) {
			return q.UpsertFile(ctx, db.UpsertFileParams{
				RepoID:      repo.ID,
				Path:        f.relPath,
				ContentHash: contentHash,
				SizeBytes:   f.size,
				Language:    pgconv.ToText(languageFromChunks(chunks)),
				IndexedAt:   now,
			})
		})
		if err != nil {
			return fmt.Errorf("upsert file %s: %w", f.relPath, err)
		}

		for i := range chunks {
			allChunks = append(allChunks, chunks[i])
			allFileIDs = append(allFileIDs, dbFile.ID)
		}

		stats.filesProcessed++
	}

	if len(allChunks) == 0 {
		w.l.Info("no chunks to embed")
		return nil
	}

	stats.chunksCreated = int32(len(allChunks))

	// Delete old points for this repo before upserting new ones
	if err := qdrant.DeletePointsByFilter(ctx, &pb.Filter{
		Must: []*pb.Condition{
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "repo_id",
						Match: &pb.Match{
							MatchValue: &pb.Match_Integer{Integer: repo.ID},
						},
					},
				},
			},
		},
	}); err != nil {
		w.l.Warn("failed to delete old points, continuing", zap.Error(err))
	}

	// Build embedding inputs
	texts := make([]string, len(allChunks))
	for i, c := range allChunks {
		texts[i] = fmt.Sprintf("File: %s\n\n%s", c.FilePath, c.Content)
	}

	embResult, err := w.embedder.GenerateEmbeddings(ctx, w.l, texts)
	if err != nil {
		return fmt.Errorf("generate embeddings: %w", err)
	}

	stats.embeddingsGenerated = int32(len(embResult.Embeddings))

	// Build Qdrant points
	points := make([]*pb.PointStruct, len(allChunks))
	for i, c := range allChunks {
		pointID := pb.NewIDNum(uint64(allFileIDs[i])*10000 + uint64(i))

		points[i] = &pb.PointStruct{
			Id:      pointID,
			Vectors: pb.NewVectors(embResult.Embeddings[i]...),
			Payload: map[string]*pb.Value{
				"workspace_id":  pb.NewValueInt(repo.WorkspaceID),
				"repo_id":       pb.NewValueInt(repo.ID),
				"file_id":       pb.NewValueInt(allFileIDs[i]),
				"file_path":     pb.NewValueString(c.FilePath),
				"language":      pb.NewValueString(c.Language),
				"start_line":    pb.NewValueInt(int64(c.StartLine)),
				"end_line":      pb.NewValueInt(int64(c.EndLine)),
				"chunk_content": pb.NewValueString(c.Content),
				"chunk_type":    pb.NewValueString(c.ChunkType),
				"symbol_name":   pb.NewValueString(c.SymbolName),
			},
		}
	}

	if err := qdrant.UpsertPoints(ctx, points); err != nil {
		return fmt.Errorf("upsert points: %w", err)
	}

	return nil
}

func (w *Worker) createRun(ctx context.Context, repoID int64) (*db.IndexRun, error) {
	now := time.Now().UnixNano()
	run, err := db.Tx1(ctx, func(q *db.Queries) (db.IndexRun, error) {
		return q.CreateIndexRun(ctx, db.CreateIndexRunParams{
			RepoID:    repoID,
			Status:    "running",
			StartedAt: now,
		})
	})
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (w *Worker) failRun(ctx context.Context, runID, repoID int64, startTime time.Time, runErr error) {
	duration := time.Since(startTime).Milliseconds()
	errMsg := runErr.Error()

	db.Tx1(ctx, func(q *db.Queries) (db.IndexRun, error) {
		return q.UpdateIndexRunStatus(ctx, db.UpdateIndexRunStatusParams{
			ID:           runID,
			Status:       "failed",
			CompletedAt:  pgconv.ToInt8(&[]int64{time.Now().UnixNano()}[0]),
			ErrorMessage: pgconv.ToText(&errMsg),
			DurationMs:   pgconv.ToInt8(&duration),
		})
	})

	repos.UpdateStatus(ctx, repoID, repos.StatusError, &errMsg)
}

func (w *Worker) completeRun(ctx context.Context, runID, repoID int64, startTime time.Time, stats *runStats) {
	duration := time.Since(startTime).Milliseconds()
	now := time.Now().UnixNano()

	db.Tx1(ctx, func(q *db.Queries) (db.IndexRun, error) {
		return q.UpdateIndexRunStatus(ctx, db.UpdateIndexRunStatusParams{
			ID:          runID,
			Status:      "completed",
			CompletedAt: pgconv.ToInt8(&now),
			DurationMs:  pgconv.ToInt8(&duration),
		})
	})

	db.Tx1(ctx, func(q *db.Queries) (db.IndexRun, error) {
		return q.UpdateIndexRunStats(ctx, db.UpdateIndexRunStatsParams{
			ID:                  runID,
			FilesProcessed:      stats.filesProcessed,
			ChunksCreated:       stats.chunksCreated,
			EmbeddingsGenerated: stats.embeddingsGenerated,
		})
	})

	repos.UpdateStatus(ctx, repoID, repos.StatusReady, nil)
}

func languageFromChunks(chunks []chunking.Chunk) *string {
	if len(chunks) == 0 {
		return nil
	}
	lang := chunks[0].Language
	if lang == "" || lang == "generic" {
		return nil
	}
	return &lang
}

// matchesAnyPattern reports whether relPath matches any of the given patterns.
// Patterns follow the same conventions as WorkspaceSettings:
//   - "dir/" matches the directory and everything under it
//   - "*.ext" matches files by extension
//   - any other value is treated as an exact path match
func matchesAnyPattern(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/") {
			dir := strings.TrimSuffix(pattern, "/")
			if relPath == dir || strings.HasPrefix(relPath, dir+"/") || strings.Contains(relPath, "/"+dir+"/") {
				return true
			}
		} else if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(relPath, suffix) {
				return true
			}
		} else if relPath == pattern {
			return true
		}
	}
	return false
}
