package main

import (
	"github.com/gomantics/semantix/internal/api"
	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/domains/indexing"
	"github.com/gomantics/semantix/internal/libs/openai"
	"github.com/gomantics/semantix/internal/qdrant"
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
			return l.With(zap.String("service", "dev"))
		}),
		fx.Invoke(
			db.Init,
			qdrant.Init,
			openai.Init,
			api.Run,
			indexing.Run,
		),
		fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{
				Logger: l,
			}
		}),
	).Run()
}
