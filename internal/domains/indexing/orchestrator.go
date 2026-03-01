package indexing

import (
	"context"
	"sync"
	"time"

	"github.com/gomantics/semantix/config"
	"github.com/gomantics/semantix/internal/domains/repos"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const pollInterval = 60 * time.Second

// Orchestrator polls for pending repos and dispatches indexing workers.
type Orchestrator struct {
	l      *zap.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
	sem    chan struct{}
}

// Run starts the orchestrator as part of the fx lifecycle.
func Run(lc fx.Lifecycle, l *zap.Logger) {
	o := &Orchestrator{
		l:   l.Named("indexing"),
		sem: make(chan struct{}, config.Indexing.MaxConcurrentJobs()),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			o.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			o.stop()
			return nil
		},
	})
}

func (o *Orchestrator) start() {
	ctx, cancel := context.WithCancel(context.Background())
	o.cancel = cancel

	o.wg.Add(1)
	go o.poll(ctx)

	o.l.Info("orchestrator started",
		zap.Int64("max_workers", config.Indexing.MaxConcurrentJobs()),
		zap.Duration("poll_interval", pollInterval),
	)
}

func (o *Orchestrator) stop() {
	o.l.Info("orchestrator shutting down, waiting for active workers")
	o.cancel()
	o.wg.Wait()
	o.l.Info("orchestrator stopped")
}

func (o *Orchestrator) poll(ctx context.Context) {
	defer o.wg.Done()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Run once immediately on startup.
	o.processPending(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.processPending(ctx)
		}
	}
}

func (o *Orchestrator) processPending(ctx context.Context) {
	pending, err := repos.ListPending(ctx, int(config.Indexing.MaxConcurrentJobs()))
	if err != nil {
		o.l.Error("failed to list pending repos", zap.Error(err))
		return
	}

	if len(pending) == 0 {
		return
	}

	o.l.Info("found pending repos", zap.Int("count", len(pending)))

	for _, repo := range pending {
		select {
		case <-ctx.Done():
			return
		case o.sem <- struct{}{}:
		}

		o.wg.Add(1)
		go func(r repos.Repo) {
			defer o.wg.Done()
			defer func() { <-o.sem }()

			w := NewWorker(o.l.With(zap.Int64("repo_id", r.ID), zap.String("url", r.URL)))
			w.Process(ctx, r)
		}(repo)
	}
}
