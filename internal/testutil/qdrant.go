package testutil

import (
	"context"
	"fmt"
	"time"

	internalqdrant "github.com/gomantics/semantix/internal/qdrant"
	"github.com/testcontainers/testcontainers-go"
	tcqdrant "github.com/testcontainers/testcontainers-go/modules/qdrant"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// WithQdrant returns an Option that starts a Qdrant v1.17.0 container, ensures
// the collection exists, and injects the gRPC connection into the qdrant package.
// The container is terminated on teardown.
func WithQdrant() Option {
	return func() Teardown {
		return withQdrant()
	}
}

func withQdrant() Teardown {
	ctx := context.Background()

	ctr, err := tcqdrant.Run(ctx, "qdrant/qdrant:v1.17.0")
	if err != nil {
		panic(fmt.Sprintf("testutil: start qdrant: %v", err))
	}

	// Wait briefly for Qdrant to be fully ready after port becomes available.
	time.Sleep(500 * time.Millisecond)

	grpcEndpoint, err := ctr.GRPCEndpoint(ctx)
	if err != nil {
		panic(fmt.Sprintf("testutil: get qdrant gRPC endpoint: %v", err))
	}

	conn, err := grpc.NewClient(grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Sprintf("testutil: connect to qdrant: %v", err))
	}

	internalqdrant.SetClients(conn)

	l := zap.NewNop()
	if err := internalqdrant.EnsureCollection(ctx, l); err != nil {
		panic(fmt.Sprintf("testutil: ensure qdrant collection: %v", err))
	}

	return func() {
		conn.Close()
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			fmt.Printf("testutil: terminate qdrant: %v\n", err)
		}
	}
}
