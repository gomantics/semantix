package indexing

import (
	"context"
	"sync"
	"time"

	"github.com/gomantics/semantix/config"
	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/domains/repos"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const pollInterval = 60 * time.Second

// Orchestrator polls for pending repos and dispatches indexing workers.
type Orchestrator struct {
	l       *zap.Logger
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	sem     chan struct{}
	trigger chan struct{}
}

var globalOrchestrator *Orchestrator

// Trigger nudges the orchestrator to check for pending repos immediately
// instead of waiting for the next poll tick. Safe to call concurrently;
// a no-op if the orchestrator hasn't started or is already processing.
func Trigger() {
	o := globalOrchestrator
	if o == nil {
		return
	}
	select {
	case o.trigger <- struct{}{}:
	default:
	}
}

// Run starts the orchestrator as part of the fx lifecycle.
func Run(lc fx.Lifecycle, l *zap.Logger) {
	o := &Orchestrator{
		l:       l.Named("indexing"),
		sem:     make(chan struct{}, config.Indexing.MaxConcurrentJobs()),
		trigger: make(chan struct{}, 1),
	}
	globalOrchestrator = o

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

	o.l.Info("orchestrator started",
		zap.Int64("max_workers", config.Indexing.MaxConcurrentJobs()),
		zap.Duration("poll_interval", pollInterval),
	)

	o.wg.Add(1)
	go o.poll(ctx)
}

// recover resets repos and index_runs that were left in transient states by a
// previous server crash. It runs once synchronously before the first poll so
// that the recovered repos are immediately visible to the poll loop.
func (o *Orchestrator) recover(ctx context.Context) {
	now := time.Now().UnixNano()

	if err := db.Tx(ctx, func(q *db.Queries) error {
		return q.FailOrphanedIndexRuns(ctx, pgtype.Int8{Int64: now, Valid: true})
	}); err != nil {
		o.l.Error("failed to fail orphaned index runs", zap.Error(err))
	} else {
		o.l.Info("marked orphaned index_runs as failed (status=running -> failed)")
	}

	if err := db.Tx(ctx, func(q *db.Queries) error {
		return q.ResetStaleRepos(ctx, now)
	}); err != nil {
		o.l.Error("failed to reset stale repos", zap.Error(err))
	} else {
		o.l.Info("reset stale repos (cloning/indexing -> pending)")
	}
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

	// Recover any repos/runs left in transient states by a previous crash,
	// then immediately process whatever is now pending.
	o.recover(ctx)
	o.processPending(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.processPending(ctx)
		case <-o.trigger:
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
