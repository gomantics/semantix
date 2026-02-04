package main

import (
	"github.com/gomantics/semantix/internal/api"
	"github.com/gomantics/semantix/db"
	"github.com/gomantics/semantix/pkg/logger"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(
			logger.New,
		),
		fx.Decorate(func(l *zap.Logger) *zap.Logger {
			return l.With(zap.String("service", "semantix"))
		}),
		fx.Invoke(
			db.Init,
			// TODO: Add Qdrant initialization (Phase 1)
			// TODO: Re-enable indexing worker (Phase 2)
			api.Run,
		),
		fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{
				Logger: l,
			}
		}),
	).Run()
}

