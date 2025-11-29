package milvus

import (
	"context"
	"fmt"

	"github.com/gomantics/semantix/config"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var defaultClient *milvusclient.Client

func Init(lc fx.Lifecycle, l *zap.Logger) error {
	ctx := context.Background()

	c, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: config.Milvus.Address(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to milvus: %w", err)
	}

	defaultClient = c

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			l.Info("closing milvus connection")
			if defaultClient != nil {
				return defaultClient.Close(ctx)
			}
			return nil
		},
	})

	l.Info("milvus client initialized", zap.String("address", config.Milvus.Address()))

	if err := ensureCollection(ctx, l); err != nil {
		return fmt.Errorf("failed to ensure collection: %w", err)
	}

	return nil
}

func GetClient() *milvusclient.Client {
	return defaultClient
}
