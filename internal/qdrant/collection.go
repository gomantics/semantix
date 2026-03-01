package qdrant

import (
	"context"
	"errors"
	"fmt"

	"github.com/gomantics/semantix/config"
	pb "github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// VectorSize is the dimension of OpenAI text-embedding-3-small embeddings
	VectorSize uint64 = 1536
)

// EnsureCollection creates the collection if it doesn't exist
func EnsureCollection(ctx context.Context, l *zap.Logger) error {
	collectionName := config.Qdrant.CollectionName()

	exists, err := collectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		l.Info("collection already exists", zap.String("collection", collectionName))
		return nil
	}

	l.Info("creating collection", zap.String("collection", collectionName))

	_, err = collectionsClient.Create(ctx, &pb.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     VectorSize,
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	if err := createPayloadIndexes(ctx, collectionName, l); err != nil {
		return fmt.Errorf("failed to create payload indexes: %w", err)
	}

	return nil
}

// collectionExists checks if a collection exists
func collectionExists(ctx context.Context, name string) (bool, error) {
	_, err := collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{
		CollectionName: name,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return false, nil
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, err
		}

		if st != nil && st.Code() == codes.Unknown {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// createPayloadIndexes creates indexes for efficient payload filtering
func createPayloadIndexes(ctx context.Context, collectionName string, l *zap.Logger) error {
	indexes := []struct {
		fieldName string
		fieldType pb.FieldType
	}{
		{"workspace_id", pb.FieldType_FieldTypeInteger},
		{"repo_id", pb.FieldType_FieldTypeInteger},
		{"file_id", pb.FieldType_FieldTypeInteger},
	}

	for _, idx := range indexes {
		l.Info("creating payload index",
			zap.String("collection", collectionName),
			zap.String("field", idx.fieldName),
		)

		fieldType := idx.fieldType
		_, err := pointsClient.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
			CollectionName: collectionName,
			FieldName:      idx.fieldName,
			FieldType:      &fieldType,
			Wait:           boolPtr(true),
		})
		if err != nil {
			return fmt.Errorf("failed to create index for %s: %w", idx.fieldName, err)
		}
	}

	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
