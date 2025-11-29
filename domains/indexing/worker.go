package indexing

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gomantics/semantix/config"
	"github.com/gomantics/semantix/domains/repos"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Worker handles background indexing jobs
type Worker struct {
	l            *zap.Logger
	orchestrator *Orchestrator
	wg           sync.WaitGroup
	cancel       context.CancelFunc
}

// StartWorker starts the background worker
func StartWorker(lc fx.Lifecycle, l *zap.Logger) {
	worker := &Worker{
		l:            l,
		orchestrator: NewOrchestrator(l),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			workerCtx, cancel := context.WithCancel(context.Background())
			worker.cancel = cancel
			worker.start(workerCtx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			worker.stop()
			return nil
		},
	})
}

// start begins the worker goroutines
func (w *Worker) start(ctx context.Context) {
	maxJobs := max(config.Indexing.MaxConcurrentJobs(), 1)

	w.l.Info("starting indexing workers", zap.Int64("workers", maxJobs))

	for i := range maxJobs {
		w.wg.Add(1)
		go w.run(ctx, i)
	}
}

// stop gracefully stops all workers
func (w *Worker) stop() {
	w.l.Info("stopping indexing workers")
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	w.l.Info("all workers stopped")
}

// run is the main worker loop
func (w *Worker) run(ctx context.Context, workerID int64) {
	defer w.wg.Done()

	l := w.l.With(zap.Int64("worker_id", workerID))
	l.Info("worker started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.Info("worker stopping")
			return
		case <-ticker.C:
			w.processJob(ctx, l)
		}
	}
}

// processJob attempts to claim and process a pending job
func (w *Worker) processJob(ctx context.Context, l *zap.Logger) {
	repo, err := repos.ClaimPending(ctx)
	if errors.Is(err, repos.ErrNotFound) {
		return // No pending repos
	}
	if err != nil {
		l.Error("failed to claim pending repo", zap.Error(err))
		return
	}

	l.Info("claimed pending repo",
		zap.Int64("repo_id", repo.ID),
		zap.String("url", repo.URL),
	)

	// Process the repo
	if err := w.orchestrator.IndexRepo(ctx, repo.ID); err != nil {
		l.Error("indexing failed",
			zap.Int64("repo_id", repo.ID),
			zap.Error(err),
		)
	}
}
