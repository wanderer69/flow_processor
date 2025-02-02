package clientconnector

import (
	"context"

	"github.com/wanderer69/flow_processor/pkg/entity"
	"github.com/wanderer69/flow_processor/pkg/process"
)

type ExternalTopic interface {
	Init(ctx context.Context, processName string, topicName string) error
	Send(ctx context.Context, processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error
	SetTopicHandler(ctx context.Context, processName string, topicName string, fn func(processName string, processId string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error
	SetTopicResponse(ctx context.Context, processName, topicName string, fn func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error
	CompleteTopic(ctx context.Context, processName string, processID string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error
}

type ExternalActivation interface {
	Init(ctx context.Context, processName string, taskName string) error
	//SetActivationResponse(ctx context.Context, processName, taskName string, fn func(processName, processId, taskName string, msgs []*entity.Message, vars []*entity.Variable) error) error
	CompleteActivation(ctx context.Context, processName, processID, taskName string, msgs []*entity.Message, vars []*entity.Variable) error
}

type ProcessExecutor interface {
	StartProcess(ctx context.Context, processName string, vars []*entity.Variable) (*process.Process, error)
	AddProcess(ctx context.Context, process *entity.Process) error
	ExternalSendToMailBox(processName, processID, topicName string, msgs []*entity.Message) error
	GetStopped() chan *process.FinishedProcessData
	GetProcess(uuid string) *process.Process
}
