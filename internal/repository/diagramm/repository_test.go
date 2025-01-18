package diagramm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/wanderer69/flow_processor/pkg/entity"
	"github.com/wanderer69/flow_processor/pkg/tests"
)

func TestRepository(t *testing.T) {
	ctx := context.Background()
	dao, err := tests.InitDAO("../../../migrations")
	require.NoError(t, err)
	diagrammRepo := NewRepository(dao)
	diagramm1 := &entity.Diagramm{
		UUID: uuid.NewString(),
		Data: "123456787",
	}
	require.NoError(t, diagrammRepo.Create(ctx, diagramm1))
	diagramm2 := &entity.Diagramm{
		UUID: uuid.NewString(),
		Data: "123456787",
	}
	require.NoError(t, diagrammRepo.Create(ctx, diagramm2))
	require.ErrorContains(t, diagrammRepo.Create(ctx, diagramm2), ErrDiagrammExists)

	diagrammDB, err := diagrammRepo.GetByUUID(ctx, uuid.NewString())
	require.ErrorContains(t, err, ErrDiagrammNotExists)
	require.Nil(t, diagrammDB)

	diagrammDB, err = diagrammRepo.GetByUUID(ctx, diagramm1.UUID)
	require.NoError(t, err)
	require.NotNil(t, diagrammDB)
	require.Equal(t, diagramm1.UUID, diagrammDB.UUID)
	require.Equal(t, diagramm1.Data, diagrammDB.Data)

	diagrammDB, err = diagrammRepo.GetByUUID(ctx, diagramm2.UUID)
	require.NoError(t, err)
	require.NotNil(t, diagrammDB)
	require.Equal(t, diagramm2.UUID, diagrammDB.UUID)
	require.Equal(t, diagramm2.Data, diagrammDB.Data)

	diagramm2.Data = "close"
	require.NoError(t, diagrammRepo.Update(ctx, diagramm2))

	diagrammDB, err = diagrammRepo.GetByUUID(ctx, diagramm2.UUID)
	require.NoError(t, err)
	require.NotNil(t, diagrammDB)
	require.Equal(t, diagramm2.Data, diagrammDB.Data)

	diagrammesDB, err := diagrammRepo.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, diagrammesDB)
	require.Len(t, diagrammesDB, 2)

	diagrammsDB, err := diagrammRepo.GetByName(ctx, diagramm1.Name)
	require.NoError(t, err)
	require.NotNil(t, diagrammDB)
	require.Len(t, diagrammsDB, 1)
	require.Equal(t, diagramm1.Name, diagrammsDB[0].Name)

	diagrammsDB, err = diagrammRepo.GetByName(ctx, uuid.NewString())
	require.NoError(t, err)
	require.NotNil(t, diagrammDB)
	require.Len(t, diagrammsDB, 0)

	require.True(t, true)
}
