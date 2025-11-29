package milvus

import (
	"context"
	"fmt"

	"github.com/gomantics/semantix/config"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

const (
	// EmbeddingDim is the dimension of the embedding vectors (text-embedding-3-small)
	EmbeddingDim = 1536

	// FieldID is the primary key field
	FieldID = "id"
	// FieldWorkspaceID is the workspace ID for multi-tenant isolation
	FieldWorkspaceID = "workspace_id"
	// FieldRepoID is the repository ID
	FieldRepoID = "repo_id"
	// FieldFileID is the file ID
	FieldFileID = "file_id"
	// FieldChunkIndex is the chunk position in the file
	FieldChunkIndex = "chunk_index"
	// FieldFilePath is the file path for display
	FieldFilePath = "file_path"
	// FieldContent is the actual chunk text
	FieldContent = "content"
	// FieldStartLine is the start line number
	FieldStartLine = "start_line"
	// FieldEndLine is the end line number
	FieldEndLine = "end_line"
	// FieldLanguage is the programming language
	FieldLanguage = "language"
	// FieldEmbedding is the vector embedding
	FieldEmbedding = "embedding"
)

// Chunk represents a code chunk stored in Milvus
type Chunk struct {
	ID          int64
	WorkspaceID int64
	RepoID      int64
	FileID      int64
	ChunkIndex  int64
	FilePath    string
	Content     string
	StartLine   int64
	EndLine     int64
	Language    string
	Embedding   []float32
}

// SearchResult represents a search result from Milvus
type SearchResult struct {
	Chunk
	Score float32
}

// ensureCollection creates the collection if it doesn't exist
func ensureCollection(ctx context.Context, l *zap.Logger) error {
	collectionName := config.Milvus.CollectionName()

	// Check if collection exists
	exists, err := defaultClient.HasCollection(ctx, milvusclient.NewHasCollectionOption(collectionName))
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		l.Info("collection already exists", zap.String("collection", collectionName))

		// Load collection into memory for searching
		_, err := defaultClient.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(collectionName))
		if err != nil {
			l.Warn("failed to load collection", zap.Error(err))
		}
		return nil
	}

	l.Info("creating collection", zap.String("collection", collectionName))

	// Define schema with workspace_id as partition key for multi-tenant scalability
	schema := entity.NewSchema().
		WithName(collectionName).
		WithDescription("Code chunks with embeddings for semantic search").
		WithAutoID(true).
		WithField(entity.NewField().WithName(FieldID).WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName(FieldWorkspaceID).WithDataType(entity.FieldTypeInt64).WithIsPartitionKey(true)).
		WithField(entity.NewField().WithName(FieldRepoID).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(FieldFileID).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(FieldChunkIndex).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(FieldFilePath).WithDataType(entity.FieldTypeVarChar).WithMaxLength(1024)).
		WithField(entity.NewField().WithName(FieldContent).WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535)).
		WithField(entity.NewField().WithName(FieldStartLine).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(FieldEndLine).WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName(FieldLanguage).WithDataType(entity.FieldTypeVarChar).WithMaxLength(64)).
		WithField(entity.NewField().WithName(FieldEmbedding).WithDataType(entity.FieldTypeFloatVector).WithDim(EmbeddingDim))

	idx := index.NewIvfFlatIndex(entity.L2, 128)

	err = defaultClient.CreateCollection(ctx,
		milvusclient.NewCreateCollectionOption(collectionName, schema).
			WithIndexOptions(milvusclient.NewCreateIndexOption(collectionName, FieldEmbedding, idx)),
	)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	l.Info("collection created with index", zap.String("collection", collectionName))

	_, err = defaultClient.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(collectionName))
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	l.Info("collection loaded into memory")

	return nil
}

// InsertChunks inserts chunks into Milvus
func InsertChunks(ctx context.Context, chunks []Chunk) ([]int64, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	collectionName := config.Milvus.CollectionName()

	// Prepare column data
	workspaceIDs := make([]int64, len(chunks))
	repoIDs := make([]int64, len(chunks))
	fileIDs := make([]int64, len(chunks))
	chunkIndexes := make([]int64, len(chunks))
	filePaths := make([]string, len(chunks))
	contents := make([]string, len(chunks))
	startLines := make([]int64, len(chunks))
	endLines := make([]int64, len(chunks))
	languages := make([]string, len(chunks))
	embeddings := make([][]float32, len(chunks))

	for i, chunk := range chunks {
		workspaceIDs[i] = chunk.WorkspaceID
		repoIDs[i] = chunk.RepoID
		fileIDs[i] = chunk.FileID
		chunkIndexes[i] = chunk.ChunkIndex
		filePaths[i] = chunk.FilePath
		contents[i] = truncateString(chunk.Content, 65535)
		startLines[i] = chunk.StartLine
		endLines[i] = chunk.EndLine
		languages[i] = chunk.Language
		embeddings[i] = chunk.Embedding
	}

	columns := []column.Column{
		column.NewColumnInt64(FieldWorkspaceID, workspaceIDs),
		column.NewColumnInt64(FieldRepoID, repoIDs),
		column.NewColumnInt64(FieldFileID, fileIDs),
		column.NewColumnInt64(FieldChunkIndex, chunkIndexes),
		column.NewColumnVarChar(FieldFilePath, filePaths),
		column.NewColumnVarChar(FieldContent, contents),
		column.NewColumnInt64(FieldStartLine, startLines),
		column.NewColumnInt64(FieldEndLine, endLines),
		column.NewColumnVarChar(FieldLanguage, languages),
		column.NewColumnFloatVector(FieldEmbedding, EmbeddingDim, embeddings),
	}

	result, err := defaultClient.Insert(ctx, milvusclient.NewColumnBasedInsertOption(collectionName, columns...))
	if err != nil {
		return nil, fmt.Errorf("failed to insert chunks: %w", err)
	}

	ids := result.IDs.(*column.ColumnInt64).Data()

	return ids, nil
}

// SearchSimilar searches for similar chunks using vector similarity
func SearchSimilar(ctx context.Context, embedding []float32, workspaceID int64, repoID int64, limit int) ([]SearchResult, error) {
	collectionName := config.Milvus.CollectionName()

	expr := fmt.Sprintf("%s == %d", FieldWorkspaceID, workspaceID)
	if repoID > 0 {
		expr = fmt.Sprintf("%s && %s == %d", expr, FieldRepoID, repoID)
	}

	results, err := defaultClient.Search(ctx,
		milvusclient.NewSearchOption(collectionName, limit, []entity.Vector{entity.FloatVector(embedding)}).
			WithANNSField(FieldEmbedding).
			WithFilter(expr).
			WithOutputFields(FieldWorkspaceID, FieldRepoID, FieldFileID, FieldChunkIndex, FieldFilePath, FieldContent, FieldStartLine, FieldEndLine, FieldLanguage),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	var searchResults []SearchResult
	for _, result := range results {
		for i := 0; i < result.ResultCount; i++ {
			sr := SearchResult{
				Score: result.Scores[i],
			}

			if result.IDs != nil {
				if idCol, ok := result.IDs.(*column.ColumnInt64); ok {
					sr.ID = idCol.Data()[i]
				}
			}

			for _, field := range result.Fields {
				switch field.Name() {
				case FieldWorkspaceID:
					if col, ok := field.(*column.ColumnInt64); ok {
						sr.WorkspaceID = col.Data()[i]
					}
				case FieldRepoID:
					if col, ok := field.(*column.ColumnInt64); ok {
						sr.RepoID = col.Data()[i]
					}
				case FieldFileID:
					if col, ok := field.(*column.ColumnInt64); ok {
						sr.FileID = col.Data()[i]
					}
				case FieldChunkIndex:
					if col, ok := field.(*column.ColumnInt64); ok {
						sr.ChunkIndex = col.Data()[i]
					}
				case FieldFilePath:
					if col, ok := field.(*column.ColumnVarChar); ok {
						sr.FilePath = col.Data()[i]
					}
				case FieldContent:
					if col, ok := field.(*column.ColumnVarChar); ok {
						sr.Content = col.Data()[i]
					}
				case FieldStartLine:
					if col, ok := field.(*column.ColumnInt64); ok {
						sr.StartLine = col.Data()[i]
					}
				case FieldEndLine:
					if col, ok := field.(*column.ColumnInt64); ok {
						sr.EndLine = col.Data()[i]
					}
				case FieldLanguage:
					if col, ok := field.(*column.ColumnVarChar); ok {
						sr.Language = col.Data()[i]
					}
				}
			}

			searchResults = append(searchResults, sr)
		}
	}

	return searchResults, nil
}

func DeleteByFileID(ctx context.Context, fileID int64) error {
	collectionName := config.Milvus.CollectionName()
	expr := fmt.Sprintf("%s == %d", FieldFileID, fileID)

	_, err := defaultClient.Delete(ctx, milvusclient.NewDeleteOption(collectionName).WithExpr(expr))
	if err != nil {
		return fmt.Errorf("failed to delete chunks by file ID: %w", err)
	}

	return nil
}

func DeleteByRepoID(ctx context.Context, repoID int64) error {
	collectionName := config.Milvus.CollectionName()
	expr := fmt.Sprintf("%s == %d", FieldRepoID, repoID)

	_, err := defaultClient.Delete(ctx, milvusclient.NewDeleteOption(collectionName).WithExpr(expr))
	if err != nil {
		return fmt.Errorf("failed to delete chunks by repo ID: %w", err)
	}

	return nil
}

func DeleteByWorkspaceID(ctx context.Context, workspaceID int64) error {
	collectionName := config.Milvus.CollectionName()
	expr := fmt.Sprintf("%s == %d", FieldWorkspaceID, workspaceID)

	_, err := defaultClient.Delete(ctx, milvusclient.NewDeleteOption(collectionName).WithExpr(expr))
	if err != nil {
		return fmt.Errorf("failed to delete chunks by workspace ID: %w", err)
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
