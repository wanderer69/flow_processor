package clientconnector

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/wanderer69/flow_processor/pkg/entity"
	externalactivation "github.com/wanderer69/flow_processor/pkg/external_activation"
	externaltopic "github.com/wanderer69/flow_processor/pkg/external_topic"
	"github.com/wanderer69/flow_processor/pkg/integration"
	internalformat "github.com/wanderer69/flow_processor/pkg/internal_format"
	"github.com/wanderer69/flow_processor/pkg/logger"
	"github.com/wanderer69/flow_processor/pkg/process"
	"github.com/wanderer69/flow_processor/pkg/timer"
)

func TestServerSimpleI(t *testing.T) {
	ctx := context.Background()
	logger.SetLogger()

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()

	pe := process.NewProcessExecutor(topicClient, timerClient, externalActivationClient)

	port := 50005
	go func() {
		require.NoError(t, ServerConnect(port, topicClient, externalActivationClient, pe))
	}()

	time.Sleep(time.Second)

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	client := integration.NewProcessorClient(port)

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"

	/*
	   тестовая последовательность
	   1. старт -> подаем переменные a b
	   2. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   3. user task -> передает полученные переменные
	   4. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   5. стоп
	*/
	e1 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeStartEvent,
	}

	f1 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e2 := &entity.Element{
		UUID:              uuid.NewString(),
		ActivationType:    entity.ActivationTypeInternal,
		ElementType:       entity.ElementTypeServiceTask,
		IsExternalByTopic: true,
		TopicName:         topic1,
	}

	f2 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e3 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeExternal,
		ElementType:    entity.ElementTypeUserTask,
	}

	f3 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e4 := &entity.Element{
		UUID:              uuid.NewString(),
		ActivationType:    entity.ActivationTypeInternal,
		ElementType:       entity.ElementTypeServiceTask,
		IsExternalByTopic: true,
		TopicName:         topic2,
	}

	f4 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e5 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeEndEvent,
	}

	e1.OutputsElementID = append(e1.OutputsElementID, f1.UUID)

	f1.InputsElementID = append(f1.InputsElementID, e1.UUID)
	f1.OutputsElementID = append(f1.OutputsElementID, e2.UUID)

	e2.InputsElementID = append(e2.InputsElementID, f1.UUID)
	e2.OutputsElementID = append(e2.OutputsElementID, f2.UUID)

	f2.InputsElementID = append(f2.InputsElementID, e2.UUID)
	f2.OutputsElementID = append(f2.OutputsElementID, e3.UUID)

	e3.InputsElementID = append(e3.InputsElementID, f2.UUID)
	e3.OutputsElementID = append(e3.OutputsElementID, f3.UUID)

	f3.InputsElementID = append(f3.InputsElementID, e3.UUID)
	f3.OutputsElementID = append(f3.OutputsElementID, e4.UUID)

	e4.InputsElementID = append(e4.InputsElementID, f3.UUID)
	e4.OutputsElementID = append(e4.OutputsElementID, f4.UUID)

	f4.InputsElementID = append(f4.InputsElementID, e4.UUID)
	f4.OutputsElementID = append(f4.OutputsElementID, e5.UUID)

	e5.InputsElementID = append(e5.InputsElementID, f4.UUID)

	p := &entity.Process{
		Name: currentProcessName,
		Elements: []*entity.Element{
			e1,
			f1,
			e2,
			f2,
			e3,
			f3,
			e4,
			f4,
			e5,
		},
	}

	internalFormatClient := internalformat.NewInternalFormat()
	processRaw, err := json.Marshal(p)
	require.NoError(t, err)
	processRawFull, err := internalFormatClient.Store(string(processRaw))
	require.NoError(t, err)

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

	var currentProcessId *string

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

	require.NoError(t, client.AddProcess(ctx, processRawFull))

	done := make(chan bool)
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
		done <- true
		return nil
	})

	connected := make(chan bool)
	go func() {
		require.NoError(t, client.Connect(currentProcessName, connected))
	}()
	<-connected
	processID, err := client.StartProcess(ctx, currentProcessName)
	require.NoError(t, err)
	currentProcessId = &processID
	<-done
	time.Sleep(time.Duration(2) * time.Second)
}
