package app

import (
	"context"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type processRepository interface {
	Create(ctx context.Context, c *entity.StoreProcess) error
	Update(ctx context.Context, c *entity.StoreProcess) error
	GetByProcessID(ctx context.Context, processID string) ([]*entity.StoreProcess, error)
	DeleteByProcessID(ctx context.Context, processID string) error
	GetNotFinishedByExecutorID(ctx context.Context, executorID string) ([]*entity.StoreProcess, error)
}

type diagrammRepository interface {
	Create(ctx context.Context, c *entity.Diagramm) error
	Update(ctx context.Context, c *entity.Diagramm) error
	GetByName(ctx context.Context, name string) ([]*entity.Diagramm, error)
}
