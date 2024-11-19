package clientconnector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	camunda7convertor "github.com/wanderer69/flow_processor/pkg/camunda_7_convertor"
	"github.com/wanderer69/flow_processor/pkg/entity"
	externalactivation "github.com/wanderer69/flow_processor/pkg/external_activation"
	externaltopic "github.com/wanderer69/flow_processor/pkg/external_topic"
	"github.com/wanderer69/flow_processor/pkg/integration"
	internalformat "github.com/wanderer69/flow_processor/pkg/internal_format"
	"github.com/wanderer69/flow_processor/pkg/loader"
	"github.com/wanderer69/flow_processor/pkg/logger"
	"github.com/wanderer69/flow_processor/pkg/process"
	"github.com/wanderer69/flow_processor/pkg/store"
	"github.com/wanderer69/flow_processor/pkg/timer"
)

func TestServerSimpleI(t *testing.T) {
	ctx := context.Background()
	logger.SetLogger()

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()
	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	storeClient := store.NewStore(loader)
	stop := make(chan struct{})

	pe := process.NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	port := 50005
	go func() {
		require.NoError(t, ServerConnect(port, topicClient, externalActivationClient, pe))
	}()

	time.Sleep(time.Second)

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	client := integration.NewProcessorClient("", port)

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"

	msg1 := &entity.Message{
		Name: "msg1",
		Fields: []*entity.Field{
			{
				Name:  "field1",
				Type:  "string",
				Value: "test1",
			},
		},
	}
	var1 := &entity.Variable{
		Name:  "var1",
		Type:  "string",
		Value: "var_value1",
	}

	msg2 := &entity.Message{
		Name: "msg2",
		Fields: []*entity.Field{
			{
				Name:  "field2",
				Type:  "string",
				Value: "test2",
			},
		},
	}
	var2 := &entity.Variable{
		Name:  "var2",
		Type:  "string",
		Value: "var_value2",
	}

	var currentProcessId *string
	processRawFull := makeDiagrammI(t, currentProcessName, topic1, topic2)
	require.NoError(t, client.AddProcess(ctx, processRawFull))

	done := make(chan bool)
	client.SetProcessFinished(ctx, currentProcessName, func(ctx context.Context, processName string, processId string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		done <- true
		return nil
	})

	client.SetCallback(ctx, currentProcessName, topic1, func(ctx context.Context, processName string, processId string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		client.TopicComplete(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1})
		return nil
	})

	client.SetCallback(ctx, currentProcessName, topic2, func(ctx context.Context, processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		client.TopicComplete(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	connected := make(chan bool)
	go func() {
		require.NoError(t, client.Connect(currentProcessName, connected))
	}()
	<-connected
	processID, err := client.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &processID
	<-done
}

func TestProcessSimpleII(t *testing.T) {
	ctx := context.Background()
	logger.SetLogger()

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()
	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	storeClient := store.NewStore(loader)
	stop := make(chan struct{})

	pe := process.NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	port := 50005
	go func() {
		require.NoError(t, ServerConnect(port, topicClient, externalActivationClient, pe))
	}()

	time.Sleep(time.Second)

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	client := integration.NewProcessorClient("", port)

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	taskName3 := "element_user_task_1"

	msg1 := &entity.Message{
		Name: "msg1",
		Fields: []*entity.Field{
			{
				Name:  "field1",
				Type:  "string",
				Value: "test1",
			},
		},
	}
	var1 := &entity.Variable{
		Name:  "var1",
		Type:  "string",
		Value: "var_value1",
	}

	msg2 := &entity.Message{
		Name: "msg2",
		Fields: []*entity.Field{
			{
				Name:  "field2",
				Type:  "string",
				Value: "test2",
			},
		},
	}
	var2 := &entity.Variable{
		Name:  "var2",
		Type:  "string",
		Value: "var_value2",
	}

	var currentProcessId *string
	processRawFull := makeDiagrammII(t, currentProcessName, topic1, topic2, taskName3)
	require.NoError(t, client.AddProcess(ctx, processRawFull))

	userProcess := make(chan bool)
	done := make(chan bool)

	client.SetProcessFinished(ctx, currentProcessName, func(ctx context.Context, processName string, processId string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		done <- true
		return nil
	})

	client.SetCallback(ctx, currentProcessName, topic1, func(ctx context.Context, processName string, processId string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		time.Sleep(time.Millisecond * 20000)

		client.TopicComplete(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1})
		userProcess <- true
		return nil
	})

	client.SetCallback(ctx, currentProcessName, topic2, func(ctx context.Context, processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		time.Sleep(time.Millisecond * 20000)

		client.TopicComplete(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	connected := make(chan bool)
	go func() {
		require.NoError(t, client.Connect(currentProcessName, connected))
	}()
	<-connected
	processID, err := client.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &processID

	<-userProcess
	time.Sleep(time.Millisecond * 20000)
	client.ExternalActivation(ctx, currentProcessName, *currentProcessId, taskName3, nil, nil)

	<-done
}
