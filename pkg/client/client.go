package client

import (
	"context"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type FlowProcessClient struct {
	topicClient ExternalTopic
}

func NewFlowProcessClient(topicClient ExternalTopic) *FlowProcessClient {
	return &FlowProcessClient{
		topicClient: topicClient,
	}
}

func (tc *FlowProcessClient) SetHandler(ctx context.Context, processName, processId string, topicName string,
	fn func(processName, processID string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error {
	return tc.topicClient.SetTopicResponse(ctx, processName, processId, topicName, fn)
}
