package client

import (
	"context"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type ExternalTopic interface {
	Init(ctx context.Context, processName string, topicName string) error
	Send(ctx context.Context, processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error
	SetTopicResponse(ctx context.Context, processName, processId, topicName string, fn func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error
}
