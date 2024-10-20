package process

import (
	"context"
	"time"

	"github.com/wanderer69/flow_processor/pkg/entity"
	"github.com/wanderer69/flow_processor/pkg/timer"
)

type ExternalTopic interface {
	Init(ctx context.Context, processName string, topicName string) error
	Send(ctx context.Context, processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error
	SetTopicResponse(ctx context.Context, processName, topicName string, fn func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error
}

type Timer interface {
	Set(ctx context.Context, processName string, processId string, timerID string, timerValue *timer.TimerValue, msgs []*entity.Message, vars []*entity.Variable) error
	SetTimerResponse(ctx context.Context, processName, timerID string, fn func(processName, processId, timerID string, t time.Time, msgs []*entity.Message, vars []*entity.Variable) error) error
}

type MailBox interface {
	Set(ctx context.Context, processName string, processId string, mailBoxID string, msgTemplates []*entity.Message) error
	TimerResponse(ctx context.Context, fn func(processId, mailBoxID, msgs *entity.Message) error) error
}

type ExternalActivation interface {
	Init(ctx context.Context, processName string, taskName string) error
	SetActivationResponse(ctx context.Context, processName, taskName string, fn func(processName, processId, taskName string, msgs []*entity.Message, vars []*entity.Variable) error) error
}
