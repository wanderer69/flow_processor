package clientconnector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
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
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
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

	processByProcessName := make(map[string]*entity.Diagramm)

	diagrammRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *entity.Diagramm) error {
		processByProcessName[c.Name] = c
		return nil
	}).AnyTimes()

	diagrammRepo.EXPECT().GetByName(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) ([]*entity.Diagramm, error) {
		processDiagramm, ok := processByProcessName[name]
		if !ok {
			return nil, fmt.Errorf("diagramm %v not found", name)
		}
		return []*entity.Diagramm{processDiagramm}, nil
	}).AnyTimes()

	processExecutorStatesByProcessName := make(map[string][]*entity.StoreProcess)
	processExecutorStates := []*entity.StoreProcess{}

	processRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *entity.StoreProcess) error {
		processExecutorStates = append(processExecutorStates, c)
		p, ok := processExecutorStatesByProcessName[c.ProcessID]
		if !ok {
			p = []*entity.StoreProcess{}
		}
		p = append(p, c)
		processExecutorStatesByProcessName[c.ProcessID] = p
		return nil
	}).AnyTimes()

	processRepo.EXPECT().GetByProcessID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, processID string) ([]*entity.StoreProcess, error) {
		p, ok := processExecutorStatesByProcessName[processID]
		if !ok {
			return nil, fmt.Errorf("not found process by id %v", processID)
		}
		return p, nil
	}).AnyTimes()

	processRepo.EXPECT().DeleteByProcessID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, processID string) error {
		if len(processExecutorStates) == 0 {
			posForDelete := []int{}
			for i := range processExecutorStates {
				if len(processExecutorStates[i].ProcessID) > 0 {
					if processExecutorStates[i].ProcessID == processID {
						posForDelete = append(posForDelete, i)
					}
				}
			}
			for i := len(posForDelete) - 1; i >= 0; i-- {
				processExecutorStates = append(processExecutorStates[:posForDelete[i]], processExecutorStates[posForDelete[i]+1:]...)
			}
		}

		delete(processExecutorStatesByProcessName, processID)
		return nil
	}).AnyTimes()

	processRepo.EXPECT().GetNotFinishedByExecutorID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, executorID string) ([]*entity.StoreProcess, error) {
		processesStates := []*entity.StoreProcess{}
		for _, processStates := range processExecutorStatesByProcessName {
			isFinished := false
			for i := range processStates {
				if len(processStates[i].ProcessState) == 0 {
					isFinished = true
				}
			}
			if !isFinished {
				processesStates = append(processesStates, processStates...)
			}
		}
		return processesStates, nil
	}).AnyTimes()

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
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
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

	processByProcessName := make(map[string]*entity.Diagramm)

	diagrammRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *entity.Diagramm) error {
		processByProcessName[c.Name] = c
		return nil
	}).AnyTimes()

	diagrammRepo.EXPECT().GetByName(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) ([]*entity.Diagramm, error) {
		processDiagramm, ok := processByProcessName[name]
		if !ok {
			return nil, fmt.Errorf("diagramm %v not found", name)
		}
		return []*entity.Diagramm{processDiagramm}, nil
	}).AnyTimes()

	processExecutorStatesByProcessName := make(map[string][]*entity.StoreProcess)
	processExecutorStates := []*entity.StoreProcess{}

	processRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *entity.StoreProcess) error {
		processExecutorStates = append(processExecutorStates, c)
		p, ok := processExecutorStatesByProcessName[c.ProcessID]
		if !ok {
			p = []*entity.StoreProcess{}
		}
		p = append(p, c)
		processExecutorStatesByProcessName[c.ProcessID] = p
		return nil
	}).AnyTimes()

	processRepo.EXPECT().GetByProcessID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, processID string) ([]*entity.StoreProcess, error) {
		p, ok := processExecutorStatesByProcessName[processID]
		if !ok {
			return nil, fmt.Errorf("not found process by id %v", processID)
		}
		return p, nil
	}).AnyTimes()

	processRepo.EXPECT().DeleteByProcessID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, processID string) error {
		if len(processExecutorStates) == 0 {
			posForDelete := []int{}
			for i := range processExecutorStates {
				if len(processExecutorStates[i].ProcessID) > 0 {
					if processExecutorStates[i].ProcessID == processID {
						posForDelete = append(posForDelete, i)
					}
				}
			}
			for i := len(posForDelete) - 1; i >= 0; i-- {
				processExecutorStates = append(processExecutorStates[:posForDelete[i]], processExecutorStates[posForDelete[i]+1:]...)
			}
		}

		delete(processExecutorStatesByProcessName, processID)
		return nil
	}).AnyTimes()

	processRepo.EXPECT().GetNotFinishedByExecutorID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, executorID string) ([]*entity.StoreProcess, error) {
		processesStates := []*entity.StoreProcess{}
		for _, processStates := range processExecutorStatesByProcessName {
			isFinished := false
			for i := range processStates {
				if len(processStates[i].ProcessState) == 0 {
					isFinished = true
				}
			}
			if !isFinished {
				processesStates = append(processesStates, processStates...)
			}
		}
		return processesStates, nil
	}).AnyTimes()

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

func TestProcessSimpleIII(t *testing.T) {
	ctx := context.Background()
	logger.SetLogger()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
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

	processByProcessName := make(map[string]*entity.Diagramm)

	diagrammRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *entity.Diagramm) error {
		processByProcessName[c.Name] = c
		return nil
	}).AnyTimes()

	diagrammRepo.EXPECT().GetByName(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) ([]*entity.Diagramm, error) {
		processDiagramm, ok := processByProcessName[name]
		if !ok {
			return nil, fmt.Errorf("diagramm %v not found", name)
		}
		return []*entity.Diagramm{processDiagramm}, nil
	}).AnyTimes()

	processExecutorStatesByProcessName := make(map[string][]*entity.StoreProcess)
	processExecutorStates := []*entity.StoreProcess{}

	processRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, c *entity.StoreProcess) error {
		processExecutorStates = append(processExecutorStates, c)
		p, ok := processExecutorStatesByProcessName[c.ProcessID]
		if !ok {
			p = []*entity.StoreProcess{}
		}
		p = append(p, c)
		processExecutorStatesByProcessName[c.ProcessID] = p
		return nil
	}).AnyTimes()

	processRepo.EXPECT().GetByProcessID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, processID string) ([]*entity.StoreProcess, error) {
		p, ok := processExecutorStatesByProcessName[processID]
		if !ok {
			return nil, fmt.Errorf("not found process by id %v", processID)
		}
		return p, nil
	}).AnyTimes()

	processRepo.EXPECT().DeleteByProcessID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, processID string) error {
		if len(processExecutorStates) == 0 {
			posForDelete := []int{}
			for i := range processExecutorStates {
				if len(processExecutorStates[i].ProcessID) > 0 {
					if processExecutorStates[i].ProcessID == processID {
						posForDelete = append(posForDelete, i)
					}
				}
			}
			for i := len(posForDelete) - 1; i >= 0; i-- {
				processExecutorStates = append(processExecutorStates[:posForDelete[i]], processExecutorStates[posForDelete[i]+1:]...)
			}
		}

		delete(processExecutorStatesByProcessName, processID)
		return nil
	}).AnyTimes()

	processRepo.EXPECT().GetNotFinishedByExecutorID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, executorID string) ([]*entity.StoreProcess, error) {
		processesStates := []*entity.StoreProcess{}
		for _, processStates := range processExecutorStatesByProcessName {
			isFinished := false
			for i := range processStates {
				if len(processStates[i].ProcessState) == 0 {
					isFinished = true
				}
			}
			if !isFinished {
				processesStates = append(processesStates, processStates...)
			}
		}
		return processesStates, nil
	}).AnyTimes()

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
