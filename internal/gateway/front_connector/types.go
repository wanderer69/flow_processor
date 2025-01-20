package frontconnector

import (
	"context"

	"github.com/wanderer69/flow_processor/pkg/entity"
	"github.com/wanderer69/flow_processor/pkg/process"
)

type StoreClient interface {
	LoadProcessDiagramm(ctx context.Context, processName string) (*entity.Process, error)
	//StoreProcessDiagramm(processName string, process *entity.Process) error

	// LoadProcessExecutorState(processExecutor string) (string, error)
	StoreStartProcessState(ctx context.Context, processExecutor, processID string, data string) error
	StoreChangeProcessState(ctx context.Context, processExecutor, processID string, processExecutorState string, data string) error
	StoreFinishProcessState(ctx context.Context, processExecutor, processID string, processExecutorState string) error

	LoadProcessStates(ctx context.Context, processExecutor string) ([]*entity.ProcessExecutorStateItem, error)
}

type ProcessExecutor interface {
	StartProcess(ctx context.Context, processName string, vars []*entity.Variable) (*process.Process, error)
	AddProcess(ctx context.Context, process *entity.Process) error
	ExternalSendToMailBox(processName, processID, topicName string, msgs []*entity.Message) error
	GetStopped() chan *process.FinishedProcessData
	GetProcessExecutor() *process.ProcessExecutor
}
