package process

import (
	"context"
	"fmt"
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
	processRepo := NewRepository(dao)
	process1UUID := uuid.NewString()
	process2UUID := uuid.NewString()
	executor1UUID := uuid.NewString()
	process1 := &entity.StoreProcess{
		UUID:         uuid.NewString(),
		ExecutorID:   executor1UUID,
		ProcessID:    process1UUID,
		ProcessState: "open",
		Data:         "1234567871",
		State:        "execute",
	}
	require.NoError(t, processRepo.Create(ctx, process1))
	process2 := &entity.StoreProcess{
		UUID:         uuid.NewString(),
		ExecutorID:   executor1UUID,
		ProcessID:    process2UUID,
		ProcessState: "open",
		Data:         "1234567872",
		State:        "execute",
	}
	require.NoError(t, processRepo.Create(ctx, process2))
	require.ErrorContains(t, processRepo.Create(ctx, process2), ErrProcessExists)

	processDB, err := processRepo.GetByUUID(ctx, uuid.NewString())
	require.ErrorContains(t, err, ErrProcessNotExists)
	require.Nil(t, processDB)

	processDB, err = processRepo.GetByUUID(ctx, process1.UUID)
	require.NoError(t, err)
	require.NotNil(t, processDB)
	require.Equal(t, process1.UUID, processDB.UUID)
	require.Equal(t, process1.ProcessID, processDB.ProcessID)
	require.Equal(t, process1.ProcessState, processDB.ProcessState)

	processDB, err = processRepo.GetByUUID(ctx, process2.UUID)
	require.NoError(t, err)
	require.NotNil(t, processDB)
	require.Equal(t, process2.UUID, processDB.UUID)
	require.Equal(t, process2.ProcessID, processDB.ProcessID)
	require.Equal(t, process2.ProcessState, processDB.ProcessState)

	process2.ProcessState = "close"
	require.NoError(t, processRepo.Update(ctx, process2))

	processDB, err = processRepo.GetByUUID(ctx, process2.UUID)
	require.NoError(t, err)
	require.NotNil(t, processDB)
	require.Equal(t, process2.ProcessState, processDB.ProcessState)

	processesDB, err := processRepo.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, processesDB)
	require.Len(t, processesDB, 2)

	processesDB, err = processRepo.GetByProcessID(ctx, process1.ProcessID)
	require.NoError(t, err)
	require.NotNil(t, processDB)
	require.Len(t, processesDB, 1)
	require.Equal(t, process1.ProcessID, processesDB[0].ProcessID)

	processesDB, err = processRepo.GetByProcessID(ctx, uuid.NewString())
	require.NoError(t, err)
	require.NotNil(t, processDB)
	require.Len(t, processesDB, 0)
	//require.Equal(t, process1.ProcessID, processesDB[0].ProcessID)

	process3 := &entity.StoreProcess{
		UUID:         uuid.NewString(),
		ExecutorID:   executor1UUID,
		ProcessID:    process1UUID,
		ProcessState: "",
		Data:         "1234567873",
		State:        "finished",
	}
	require.NoError(t, processRepo.Create(ctx, process3))

	processesDB, err = processRepo.GetNotFinishedByExecutorID(ctx, executor1UUID)
	require.NoError(t, err)
	require.NotNil(t, processDB)
	require.Len(t, processesDB, 1)
	for i := range processesDB {
		fmt.Printf("%#v\r\n", processesDB[i])
	}

	require.True(t, true)
}
