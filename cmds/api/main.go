package main

import (
	"github.com/gomantics/semantix/api"
	"github.com/gomantics/semantix/db"
	"github.com/gomantics/semantix/domains/indexing"
	"github.com/gomantics/semantix/libs/logger"
	"github.com/gomantics/semantix/libs/milvus"
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
			milvus.Init,
			indexing.StartWorker,
			api.Run,
		),
		fx.WithLogger(func(l *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{
				Logger: l,
			}
		}),
	).Run()
}
