package qdrant

import (
	"context"
	"fmt"

	"github.com/gomantics/semantix/config"
	pb "github.com/qdrant/go-client/qdrant"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	defaultConn       *grpc.ClientConn
	defaultClient     pb.QdrantClient
	collectionsClient pb.CollectionsClient
	pointsClient      pb.PointsClient
)

// Init initializes the Qdrant client and ensures collection exists
func Init(lc fx.Lifecycle, l *zap.Logger) error {
	ctx := context.Background()

	// Connect to Qdrant using config
	addr := config.Qdrant.Address()
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to qdrant at %s: %w", addr, err)
	}
	defaultConn = conn

	defaultClient = pb.NewQdrantClient(defaultConn)
	collectionsClient = pb.NewCollectionsClient(defaultConn)
	pointsClient = pb.NewPointsClient(defaultConn)

	healthResult, err := defaultClient.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to ping qdrant: %w", err)
	}

	l.Info("qdrant client initialized",
		zap.String("address", addr),
		zap.String("version", healthResult.GetVersion()),
	)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			l.Info("closing qdrant client")
			if defaultConn != nil {
				return defaultConn.Close()
			}
			return nil
		},
	})

	if err := EnsureCollection(ctx, l); err != nil {
		return fmt.Errorf("failed to ensure collection: %w", err)
	}

	l.Info("qdrant collection ready", zap.String("collection", config.Qdrant.CollectionName()))
	return nil
}

// GetConn returns the default gRPC connection
func GetConn() *grpc.ClientConn {
	return defaultConn
}

// GetClient returns the default Qdrant client
func GetClient() pb.QdrantClient {
	return defaultClient
}

// GetCollectionsClient returns the collections client
func GetCollectionsClient() pb.CollectionsClient {
	return collectionsClient
}

// GetPointsClient returns the points client
func GetPointsClient() pb.PointsClient {
	return pointsClient
}

// HealthCheck pings Qdrant for health endpoint
func HealthCheck(ctx context.Context) error {
	if defaultClient == nil {
		return fmt.Errorf("qdrant client not initialized")
	}
	_, err := defaultClient.HealthCheck(ctx, &pb.HealthCheckRequest{})
	return err
}
