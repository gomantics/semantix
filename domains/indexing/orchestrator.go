package indexing

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gomantics/semantix/db"
	"github.com/gomantics/semantix/domains/gittokens"
	"github.com/gomantics/semantix/domains/repos"
	"github.com/gomantics/semantix/libs/chunking"
	"github.com/gomantics/semantix/libs/gitrepo"
	"github.com/gomantics/semantix/libs/milvus"
	"github.com/gomantics/semantix/libs/openai"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// Orchestrator handles the full indexing workflow for a repository
type Orchestrator struct {
	l       *zap.Logger
	chunker *chunking.Chunker
}

// NewOrchestrator creates a new indexing orchestrator
func NewOrchestrator(l *zap.Logger) *Orchestrator {
	return &Orchestrator{
		l:       l,
		chunker: chunking.NewChunker(),
	}
}

// IndexRepo indexes a repository by ID
func (o *Orchestrator) IndexRepo(ctx context.Context, repoID int64) error {
	// Get repo details
	repo, err := repos.GetByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to get repo: %w", err)
	}

	o.l.Info("starting indexing",
		zap.Int64("repo_id", repoID),
		zap.Int64("workspace_id", repo.WorkspaceID),
		zap.String("url", repo.URL),
	)

	// Update status to cloning
	if err := repos.UpdateStatus(ctx, repoID, repos.StatusCloning); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Get the git token if one is configured
	var token string
	if repo.GitTokenID != nil {
		token, err = gittokens.GetDecryptedToken(ctx, *repo.GitTokenID)
		if err != nil {
			o.l.Warn("failed to get git token, attempting without auth",
				zap.Int64("token_id", *repo.GitTokenID),
				zap.Error(err),
			)
		}
	}

	// Get provider with token
	provider := gitrepo.GetProviderForURL(repo.URL, token)
	if provider == nil {
		repos.SetError(ctx, repoID, "unsupported git provider")
		return fmt.Errorf("unsupported git provider for URL: %s", repo.URL)
	}

	// Clone the repository
	repoPath := gitrepo.GetRepoPath(repoID)
	gitRepo, err := gitrepo.Clone(ctx, o.l, provider, repo.URL, repoPath)
	if err != nil {
		repos.SetError(ctx, repoID, "clone failed: "+err.Error())
		return fmt.Errorf("failed to clone repo: %w", err)
	}

	// Get metadata
	metadata, err := gitrepo.GetMetadata(gitRepo, provider, repo.URL)
	if err != nil {
		repos.SetError(ctx, repoID, "metadata extraction failed: "+err.Error())
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Update repo with commit SHA and status
	if err := repos.UpdateAfterClone(ctx, repoID, metadata.HeadCommitSHA); err != nil {
		return fmt.Errorf("failed to update repo after clone: %w", err)
	}

	o.l.Info("repository cloned",
		zap.String("commit", metadata.HeadCommitSHA),
		zap.String("branch", metadata.DefaultBranch),
	)

	// List all files
	files, err := gitrepo.ListFiles(repoPath)
	if err != nil {
		repos.SetError(ctx, repoID, "file listing failed: "+err.Error())
		return fmt.Errorf("failed to list files: %w", err)
	}

	o.l.Info("found files to index", zap.Int("count", len(files)))

	// Get existing files from database (still using db directly for files)
	existingFiles, err := db.Query1(ctx, func(q *db.Queries) ([]db.File, error) {
		return q.ListFilesByRepoID(ctx, repoID)
	})
	if err != nil {
		return fmt.Errorf("failed to list existing files: %w", err)
	}

	existingFileMap := make(map[string]db.File)
	for _, f := range existingFiles {
		existingFileMap[f.Path] = f
	}

	// Track statistics
	var filesProcessed, filesSkipped, chunksCreated int

	// Process each file
	for _, filePath := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get file info
		fileInfo, err := gitrepo.GetFileInfo(repoPath, filePath)
		if err != nil {
			o.l.Warn("failed to get file info", zap.String("path", filePath), zap.Error(err))
			continue
		}

		// Check if file needs indexing (incremental)
		if existing, ok := existingFileMap[filePath]; ok {
			if existing.Shasum == fileInfo.Shasum {
				filesSkipped++
				continue
			}
			// File changed, delete old chunks from Milvus
			if err := milvus.DeleteByFileID(ctx, existing.ID); err != nil {
				o.l.Warn("failed to delete old chunks", zap.Int64("file_id", existing.ID), zap.Error(err))
			}
		}

		// Read file content
		content, err := gitrepo.ReadFile(repoPath, filePath)
		if err != nil {
			o.l.Warn("failed to read file", zap.String("path", filePath), zap.Error(err))
			continue
		}

		// Chunk the file
		fullPath := filepath.Join(repoPath, filePath)
		chunks, err := o.chunker.ChunkFile(fullPath)
		if err != nil {
			o.l.Warn("failed to chunk file", zap.String("path", filePath), zap.Error(err))
			// Fallback: treat entire file as one chunk
			chunks = []chunking.Chunk{{
				Content:   content,
				StartLine: 1,
				EndLine:   countLines(content),
				Index:     0,
				Language:  fileInfo.Language,
			}}
		}

		if len(chunks) == 0 {
			continue
		}

		// Generate embeddings for chunks
		chunkTexts := make([]string, len(chunks))
		for i, chunk := range chunks {
			chunkTexts[i] = chunk.Content
		}

		embeddings, err := openai.GenerateBatchEmbeddings(ctx, chunkTexts)
		if err != nil {
			o.l.Warn("failed to generate embeddings", zap.String("path", filePath), zap.Error(err))
			continue
		}

		// Upsert file in database
		now := time.Now().Unix()
		dbFile, err := db.Query1(ctx, func(q *db.Queries) (db.File, error) {
			return q.UpsertFile(ctx, db.UpsertFileParams{
				RepoID:     repoID,
				Path:       filePath,
				Shasum:     fileInfo.Shasum,
				Language:   pgtype.Text{String: fileInfo.Language, Valid: true},
				SizeBytes:  fileInfo.SizeBytes,
				ChunkCount: int32(len(chunks)),
				IndexedAt:  pgtype.Int8{Int64: now, Valid: true},
				Created:    now,
				Updated:    now,
			})
		})
		if err != nil {
			o.l.Warn("failed to upsert file", zap.String("path", filePath), zap.Error(err))
			continue
		}

		// Prepare chunks for Milvus
		milvusChunks := make([]milvus.Chunk, len(chunks))
		for i, chunk := range chunks {
			milvusChunks[i] = milvus.Chunk{
				WorkspaceID: repo.WorkspaceID,
				RepoID:      repoID,
				FileID:      dbFile.ID,
				ChunkIndex:  int64(chunk.Index),
				FilePath:    filePath,
				Content:     chunk.Content,
				StartLine:   int64(chunk.StartLine),
				EndLine:     int64(chunk.EndLine),
				Language:    fileInfo.Language,
				Embedding:   embeddings[i],
			}
		}

		// Insert chunks into Milvus
		_, err = milvus.InsertChunks(ctx, milvusChunks)
		if err != nil {
			o.l.Warn("failed to insert chunks", zap.String("path", filePath), zap.Error(err))
			continue
		}

		filesProcessed++
		chunksCreated += len(chunks)

		o.l.Debug("indexed file",
			zap.String("path", filePath),
			zap.Int("chunks", len(chunks)),
		)
	}

	// Update repo status to completed
	if err := repos.UpdateStatus(ctx, repoID, repos.StatusCompleted); err != nil {
		return fmt.Errorf("failed to update repo status: %w", err)
	}

	o.l.Info("indexing completed",
		zap.Int64("repo_id", repoID),
		zap.Int("files_processed", filesProcessed),
		zap.Int("files_skipped", filesSkipped),
		zap.Int("chunks_created", chunksCreated),
	)

	return nil
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	lines := 1
	for _, c := range s {
		if c == '\n' {
			lines++
		}
	}
	return lines
}
