package engine

import (
	"context"

	"github.com/johandrevandeventer/devices-api-server/internal/config"
	"github.com/johandrevandeventer/persist"
	"go.uber.org/zap"
)

type Engine struct {
	cfg            *config.Config
	logger         *zap.Logger
	statePersister *persist.FilePersister
	stopFileChan   chan struct{}
	ctx            context.Context
}
