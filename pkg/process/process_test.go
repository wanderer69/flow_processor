package process

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	camunda7convertor "github.com/wanderer69/flow_processor/pkg/camunda_7_convertor"
	"github.com/wanderer69/flow_processor/pkg/entity"
	externalactivation "github.com/wanderer69/flow_processor/pkg/external_activation"
	externaltopic "github.com/wanderer69/flow_processor/pkg/external_topic"
	internalformat "github.com/wanderer69/flow_processor/pkg/internal_format"
	"github.com/wanderer69/flow_processor/pkg/loader"
	"github.com/wanderer69/flow_processor/pkg/store"
	"github.com/wanderer69/flow_processor/pkg/timer"
)

type TestClient struct {
	topicClient *externaltopic.ExternalTopic
}

func NewTestClient(topicClient *externaltopic.ExternalTopic) *TestClient {
	return &TestClient{
		topicClient: topicClient,
	}
}

func (tc *TestClient) SetHandler(ctx context.Context, processName string, topicName string,
	fn func(processName, processID string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error {
	return tc.topicClient.SetTopicHandler(ctx, processName, topicName, fn)
}

func TestProcessSimpleI(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()
	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

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

	require.NoError(t, pe.AddProcess(ctx, p))
	var currentProcessId *string

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		require.NoError(t, testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1}))
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID

	<-pe.Stopped
}

func TestProcessSimpleII(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	/*
	   тестовая последовательность
	   1. старт -> подаем переменные a b
	   2. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   3. user task -> передает полученные переменные
	   4. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   5. стоп
	*/
	e1 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeStartEvent,
		CamundaModelerID:   "start_1",
		CamundaModelerName: "element_start_1",
	}

	f1 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e2 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic1,
		CamundaModelerID:   "service_task_1",
		CamundaModelerName: "element_service_task_1",
	}

	f2 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e3 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeExternal,
		ElementType:        entity.ElementTypeUserTask,
		IsExternal:         true,
		CamundaModelerID:   "user_task_1",
		CamundaModelerName: "element_user_task_1",
	}

	f3 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e4 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic2,
		CamundaModelerID:   "service_task_2",
		CamundaModelerName: "element_service_task_2",
	}

	f4 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e5 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeEndEvent,
		CamundaModelerID:   "end_1",
		CamundaModelerName: "element_end_1",
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

	userProcess := make(chan bool)
	require.NoError(t, pe.AddProcess(ctx, p))
	var currentProcessId *string

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1})
		userProcess <- true
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID

	<-userProcess
	time.Sleep(time.Millisecond * 20)
	require.NoError(t, externalActivationClient.CompleteActivation(ctx, currentProcessName, *currentProcessId, e3.CamundaModelerName, nil, nil))

	<-pe.Stopped
}

func TestProcessSimpleIII(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	topic3 := "topic3"
	topic4 := "topic4"
	topic5 := "topic5"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	/*
	   тестовая последовательность
	   1. старт -> подаем переменные a b
	   2. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   3. user task -> передает полученные переменные
	   4. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   5. стоп
	*/
	e1 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeStartEvent,
		CamundaModelerID:   "start_1",
		CamundaModelerName: "element_start_1",
	}

	f1 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_1",
	}

	e2 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic1,
		CamundaModelerID:   "service_task_1",
		CamundaModelerName: "element_service_task_1",
	}

	f2 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_2",
	}

	e3 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeExternal,
		ElementType:        entity.ElementTypeUserTask,
		IsExternal:         true,
		CamundaModelerID:   "user_task_1",
		CamundaModelerName: "element_user_task_1",
	}

	f3 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_3",
	}

	e4 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic2,
		CamundaModelerID:   "service_task_2",
		CamundaModelerName: "element_service_task_2",
	}

	f31 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_31",
	}

	e41 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeExclusiveGateway,
		CamundaModelerID:   "gateway_1",
		CamundaModelerName: "element_gateway_1",
	}

	f32 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		IsDefault:          true,
		CamundaModelerName: "element_flow_32",
	}

	e42 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic3,
		CamundaModelerID:   "service_task_3",
		CamundaModelerName: "element_service_task_3",
	}

	f33 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		Script:             "${isLeft}",
		CamundaModelerName: "element_flow_33",
	}

	e43 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic4,
		CamundaModelerID:   "service_task_4",
		CamundaModelerName: "element_service_task_4",
	}

	f34 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_34",
	}

	f35 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_35",
	}

	e44 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic5,
		CamundaModelerID:   "service_task_5",
		CamundaModelerName: "element_service_task_5",
	}

	f4 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_4",
	}

	e5 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeEndEvent,
		CamundaModelerID:   "end_1",
		CamundaModelerName: "element_end_1",
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
	e4.OutputsElementID = append(e4.OutputsElementID, f31.UUID)

	f31.InputsElementID = append(f31.InputsElementID, e4.UUID)
	f31.OutputsElementID = append(f31.OutputsElementID, e41.UUID)

	e41.InputsElementID = append(e41.InputsElementID, f31.UUID)
	e41.OutputsElementID = append(e41.OutputsElementID, f32.UUID)
	e41.OutputsElementID = append(e41.OutputsElementID, f33.UUID)

	f32.InputsElementID = append(f32.InputsElementID, e41.UUID)
	f32.OutputsElementID = append(f32.OutputsElementID, e42.UUID)

	e42.InputsElementID = append(e42.InputsElementID, f32.UUID)
	e42.OutputsElementID = append(e42.OutputsElementID, f35.UUID)

	f33.InputsElementID = append(f33.InputsElementID, e41.UUID)
	f33.OutputsElementID = append(f33.OutputsElementID, e43.UUID)

	e43.InputsElementID = append(e43.InputsElementID, f33.UUID)
	e43.OutputsElementID = append(e43.OutputsElementID, f34.UUID)

	f34.InputsElementID = append(f34.InputsElementID, e43.UUID)
	f34.OutputsElementID = append(f34.OutputsElementID, e44.UUID)

	f35.InputsElementID = append(f35.InputsElementID, e42.UUID)
	f35.OutputsElementID = append(f35.OutputsElementID, e44.UUID)

	e44.InputsElementID = append(e44.InputsElementID, f33.UUID)
	e44.OutputsElementID = append(e44.OutputsElementID, f4.UUID)

	f4.InputsElementID = append(f4.InputsElementID, e44.UUID)
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

			f31,
			e41,

			f32,
			e42,

			f33,
			e43,

			f34,
			f35,
			e44,

			f4,
			e5,
		},
	}

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

	userProcess := make(chan bool)
	require.NoError(t, pe.AddProcess(ctx, p))
	var currentProcessId *string

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1})
		userProcess <- true
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic3, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic3, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic4, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic4, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic5, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic5, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID

	<-userProcess
	time.Sleep(time.Millisecond * 20)
	require.NoError(t, externalActivationClient.CompleteActivation(ctx, currentProcessName, *currentProcessId, e3.CamundaModelerName, nil, nil))

	<-pe.Stopped

	var2.Name = "isLeft"
	var2.Type = "boolean"
	var2.Value = "true"

	process, err = pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID

	<-userProcess
	time.Sleep(time.Millisecond * 20)
	require.NoError(t, externalActivationClient.CompleteActivation(ctx, currentProcessName, *currentProcessId, e3.CamundaModelerName, nil, nil))

	<-pe.Stopped
}

func TestProcessSimpleIV(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	topic3 := "topic3"
	topic4 := "topic4"
	topic5 := "topic5"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	/*
	   тестовая последовательность
	   1. старт -> подаем переменные a b
	   2. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   3. user task -> передает полученные переменные
	   4. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   5. стоп
	*/
	e1 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeStartEvent,
		CamundaModelerID:   "start_1",
		CamundaModelerName: "element_start_1",
	}

	f1 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e2 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic1,
		CamundaModelerID:   "service_task_1",
		CamundaModelerName: "element_service_task_1",
	}

	f2 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e3 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeExternal,
		ElementType:        entity.ElementTypeUserTask,
		IsExternal:         true,
		CamundaModelerID:   "user_task_1",
		CamundaModelerName: "element_user_task_1",
	}

	f3 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e4 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic2,
		CamundaModelerID:   "service_task_2",
		CamundaModelerName: "element_service_task_2",
	}

	f31 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e41 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeParallelGateway,
		CamundaModelerID:   "parallel_gateway_1",
		CamundaModelerName: "element_parallel_gateway_1",
	}

	f32 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
		//		IsDefault:          true,
		CamundaModelerName: "element_flow_32",
	}

	e42 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic3,
		CamundaModelerID:   "service_task_3",
		CamundaModelerName: "element_service_task_3",
	}

	f33 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "element_flow_33",
	}

	e43 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic4,
		CamundaModelerID:   "service_task_4",
		CamundaModelerName: "element_service_task_4",
	}

	f34 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	f35 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e6 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeParallelGateway,
		CamundaModelerID:   "parallel_gateway_2",
		CamundaModelerName: "element_parallel_gateway_2",
	}

	f36 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e44 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic5,
		CamundaModelerID:   "service_task_5",
		CamundaModelerName: "element_service_task_5",
	}

	f4 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e5 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeEndEvent,
		CamundaModelerID:   "end_1",
		CamundaModelerName: "element_end_1",
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
	e4.OutputsElementID = append(e4.OutputsElementID, f31.UUID)

	f31.InputsElementID = append(f31.InputsElementID, e4.UUID)
	f31.OutputsElementID = append(f31.OutputsElementID, e41.UUID)

	e41.InputsElementID = append(e41.InputsElementID, f31.UUID)
	e41.OutputsElementID = append(e41.OutputsElementID, f32.UUID)
	e41.OutputsElementID = append(e41.OutputsElementID, f33.UUID)

	f32.InputsElementID = append(f32.InputsElementID, e41.UUID)
	f32.OutputsElementID = append(f32.OutputsElementID, e42.UUID)

	e42.InputsElementID = append(e42.InputsElementID, f32.UUID)
	e42.OutputsElementID = append(e42.OutputsElementID, f35.UUID)

	f33.InputsElementID = append(f33.InputsElementID, e41.UUID)
	f33.OutputsElementID = append(f33.OutputsElementID, e43.UUID)

	e43.InputsElementID = append(e43.InputsElementID, f33.UUID)
	e43.OutputsElementID = append(e43.OutputsElementID, f34.UUID)

	f34.InputsElementID = append(f34.InputsElementID, e43.UUID)
	f34.OutputsElementID = append(f34.OutputsElementID, e6.UUID)

	f35.InputsElementID = append(f35.InputsElementID, e42.UUID)
	f35.OutputsElementID = append(f35.OutputsElementID, e6.UUID)

	e6.InputsElementID = append(e6.InputsElementID, f34.UUID)
	e6.InputsElementID = append(e6.InputsElementID, f35.UUID)
	e6.OutputsElementID = append(e6.OutputsElementID, f36.UUID)

	f36.InputsElementID = append(f36.InputsElementID, e6.UUID)
	f36.OutputsElementID = append(f36.OutputsElementID, e44.UUID)

	e44.InputsElementID = append(e44.InputsElementID, f33.UUID)
	e44.OutputsElementID = append(e44.OutputsElementID, f4.UUID)

	f4.InputsElementID = append(f4.InputsElementID, e44.UUID)
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

			f31,
			e41,

			f32,
			e42,

			f33,
			e43,

			f34,
			f35,
			e6,

			f36,
			e44,

			f4,
			e5,
		},
	}

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

	userProcess := make(chan bool)
	require.NoError(t, pe.AddProcess(ctx, p))
	var currentProcessId *string

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1})
		userProcess <- true
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic3, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic3, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic4, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic4, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic5, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic5, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID

	<-userProcess
	time.Sleep(time.Millisecond * 20)
	require.NoError(t, externalActivationClient.CompleteActivation(ctx, currentProcessName, *currentProcessId, e3.CamundaModelerName, nil, nil))

	<-pe.Stopped
}

func TestProcessSimpleV(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	msgSend1 := &entity.Message{
		Name: "msgSend1",
		Fields: []*entity.Field{
			{
				Name:  "field3",
				Type:  "string",
				Value: "test3",
			},
		},
	}

	msgSend2 := &entity.Message{
		Name: "msgSend2",
		Fields: []*entity.Field{
			{
				Name:  "field4",
				Type:  "string",
				Value: "test4",
			},
		},
	}

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
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f1",
	}

	e2 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic1,
		CamundaModelerName: "e2",
	}

	f2 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f2",
	}

	e3 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeExternal,
		ElementType:        entity.ElementTypeReceiveTask,
		IsRecieveMail:      true,
		CamundaModelerName: "e3",
		InputMessages:      []*entity.Message{msgSend2},
	}

	f3 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f3",
	}

	e4 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic2,
		CamundaModelerName: "e4",
	}

	f4 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f4",
	}

	e5 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeEndEvent,
		CamundaModelerName: "e5",
	}

	e6 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeIntermediateCatchEvent,
		IsTimer:            true,
		TimerID:            uuid.NewString(),
		TimerDuration:      time.Duration(5000) * time.Millisecond,
		CamundaModelerName: "e6",
	}

	f6 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f6",
	}

	e7 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeIntermediateThrowEvent,
		IsSendMail:         true,
		OutputMessages:     []*entity.Message{msgSend1},
		CamundaModelerName: "e7",
	}

	f7 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f7",
	}

	e8 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeExternal,
		ElementType:        entity.ElementTypeIntermediateCatchEvent,
		IsRecieveMail:      true,
		InputMessages:      []*entity.Message{msgSend1},
		CamundaModelerName: "e8",
	}

	f8 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f8",
	}

	e9 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeIntermediateThrowEvent,
		IsSendMail:         true,
		OutputMessages:     []*entity.Message{msgSend2},
		CamundaModelerName: "e9",
	}

	f9 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f9",
	}

	e10 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeParallelGateway,
		IsRecieveMail:      true,
		CamundaModelerName: "e10",
	}

	f10 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeFlow,
		CamundaModelerName: "f10",
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
	f4.OutputsElementID = append(f4.OutputsElementID, e10.UUID)

	e5.InputsElementID = append(e5.InputsElementID, f4.UUID)

	e6.OutputsElementID = append(e6.OutputsElementID, f6.UUID)

	f6.InputsElementID = append(f6.InputsElementID, e6.UUID)
	f6.OutputsElementID = append(f6.OutputsElementID, e7.UUID)

	e7.InputsElementID = append(e7.InputsElementID, f6.UUID)
	e7.OutputsElementID = append(e7.OutputsElementID, f7.UUID)

	f7.InputsElementID = append(f7.InputsElementID, e7.UUID)
	f7.OutputsElementID = append(f7.OutputsElementID, e8.UUID)

	e8.InputsElementID = append(e8.InputsElementID, f7.UUID)
	e8.OutputsElementID = append(e8.OutputsElementID, f8.UUID)

	f8.InputsElementID = append(f8.InputsElementID, e8.UUID)
	f8.OutputsElementID = append(f8.OutputsElementID, e9.UUID)

	e9.InputsElementID = append(e9.InputsElementID, f8.UUID)
	e9.OutputsElementID = append(e9.OutputsElementID, f9.UUID)

	f9.InputsElementID = append(f9.InputsElementID, e8.UUID)
	f9.OutputsElementID = append(f9.OutputsElementID, e10.UUID)

	e10.InputsElementID = append(e10.InputsElementID, f4.UUID)
	e10.InputsElementID = append(e10.InputsElementID, f9.UUID)
	e10.OutputsElementID = append(e10.OutputsElementID, f10.UUID)

	f10.InputsElementID = append(f10.InputsElementID, e10.UUID)
	f10.OutputsElementID = append(f10.OutputsElementID, e5.UUID)

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
			e6,
			f6,
			e7,
			f7,
			e8,
			f8,
			e9,
			f9,
			e10,
			f10,
		},
	}

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

	require.NoError(t, pe.AddProcess(ctx, p))
	var currentProcessId *string

	userProcess := make(chan bool)

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		require.NoError(t, testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1}))
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		userProcess <- true
		return nil
	})

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID
	<-userProcess
	<-pe.Stopped
}

func TestProcessSimpleVI(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "Тест1"
	topic1 := "ExecProcess1"
	topic2 := "ExecProcess2"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	var ps []*entity.Process

	require.NoError(t, json.Unmarshal([]byte(processes1), &ps))

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

	require.NoError(t, pe.AddProcess(ctx, ps[0]))
	var currentProcessId *string

	userProcess := make(chan bool)

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		require.NoError(t, testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{}, []*entity.Variable{}))
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		userProcess <- true
		return nil
	})

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID
	<-userProcess
	<-pe.Stopped
}

func TestProcessSimpleIProcessRestartedI(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

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

	require.NoError(t, pe.AddProcess(ctx, p))
	var currentProcessId *string

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		require.NoError(t, testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1}))
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID

	time.Sleep(time.Millisecond * 50)
	stop <- struct{}{}

	<-pe.Stopped
}

func TestProcessSimpleIProcessRestartedII(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	testClient := NewTestClient(topicClient)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()

	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	processRepo := NewMockprocessRepository(ctrl)
	diagrammRepo := NewMockdiagrammRepository(ctrl)
	storeClient := store.NewStore(loader, processRepo, diagrammRepo)
	stop := make(chan struct{})

	currentProcessName := "test_process"
	topic1 := "topic1"
	topic2 := "topic2"
	pe := NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

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

	require.NoError(t, pe.AddProcess(ctx, p))
	var currentProcessId *string

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic1, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic1, topicName)
		require.NoError(t, testClient.topicClient.CompleteTopic(ctx, processName, processId, topic1, []*entity.Message{msg1}, []*entity.Variable{var1}))
		return nil
	})

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

	testClient.topicClient.SetTopicHandler(ctx, currentProcessName, topic2, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		require.Equal(t, *currentProcessId, processId)
		require.Equal(t, currentProcessName, currentProcessName)
		require.Equal(t, topic2, topicName)
		testClient.topicClient.CompleteTopic(ctx, processName, processId, topicName, []*entity.Message{msg2}, []*entity.Variable{var2})
		return nil
	})

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

	/*
		Create (ctx context.Context, c *entity.StoreProcess) error
		Update(ctx context.Context, c *entity.StoreProcess) error
		GetByProcessID(ctx context.Context, processID string) ([]*entity.StoreProcess, error)
		DeleteByProcessID(ctx context.Context, processID string) error
		LoadNotFinishedProcessStates(processExecutor string) ([]*entity.StoreProcess, error)
	*/

	/*
		// processExecutorByProcessName := make(map[string]string)
		type processExecutorStateRecord struct {
			processExecutor string
			processID       string
			//		processExecutorState string
			processState string
			data         string
			state        string
		}

		processExecutorStates := []*processExecutorStateRecord{}

		storeStartProcessStateCallback := func(processExecutor, processID, processExecutorState string) error {
			pes := &processExecutorStateRecord{
				processExecutor: processExecutor,
				processID:       processID,
				processState:    processExecutorState,
				state:           "start",
			}
			processExecutorStates = append(processExecutorStates, pes)
			return nil
		}

		storeChangeProcessStateCallback := func(processExecutor, processID, processState string, data string) error {
			pes := &processExecutorStateRecord{
				processExecutor: processExecutor,
				processID:       processID,
				processState:    processState,
				state:           "change",
				data:            data,
			}
			processExecutorStates = append(processExecutorStates, pes)
			return nil
		}

		storeFinishProcessStateCallback := func(processExecutor, processID, processExecutorState string) error {
			pes := &processExecutorStateRecord{
				processExecutor: processExecutor,
				processID:       processID,
				processState:    processExecutorState,
				state:           "finish",
			}
			if len(processExecutorState) == 0 {
				posForDelete := []int{}
				for i := range processExecutorStates {
					if len(processExecutorStates[i].processID) > 0 {
						if processExecutorStates[i].processID == processID {
							posForDelete = append(posForDelete, i)
						}
					}
				}
				for i := len(posForDelete) - 1; i >= 0; i-- {
					processExecutorStates = append(processExecutorStates[:posForDelete[i]], processExecutorStates[posForDelete[i]+1:]...)
				}
			}
			processExecutorStates = append(processExecutorStates, pes)
			return nil
		}
	*/

	process, err := pe.StartProcess(ctx, currentProcessName, nil)
	require.NoError(t, err)
	currentProcessId = &process.UUID

	time.Sleep(time.Millisecond * 30)
	stop <- struct{}{}
	/*
		for i := range processExecutorStates {
			//fmt.Printf("%v %v %v\r\n", processExecutorStates[i].processExecutor, processExecutorStates[i].processID, processExecutorStates[i].processState)
			if len(processExecutorStates[i].Data) > 0 {
				var dt map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(processExecutorStates[i].Data), &dt))
				//dataRaw, err := json.MarshalIndent(&dt, " ", "  ")
				//require.NoError(t, err)
				//fmt.Printf("%v\r\n", string(dataRaw))
			}
		}
	*/
	<-pe.Stopped

	processExecutorUUID := pe.UUID
	/*
		processExecutorStateItems := []*ProcessExecutorStateItem{}
		processStateByProcessID := make(map[string]*ProcessExecutorStateItem)

		for i := range processExecutorStates {
			processExecutorStateItem, ok := processStateByProcessID[processExecutorStates[i].processID]
			if !ok {
				processName := ""
				var spc StoreProcessContext
				if len(processExecutorStates[i].processState) > 0 {
					require.NoError(t, json.Unmarshal([]byte(processExecutorStates[i].processState), &spc))
					if len(spc.ProcessName) > 0 {
						processName = spc.ProcessName
					}
				}
				if len(processExecutorStates[i].processExecutor) > 0 {
					processExecutorUUID = processExecutorStates[i].processExecutor
				}
				processExecutorStateItem = &ProcessExecutorStateItem{
					ProcessID:   processExecutorStates[i].processID, //string
					ProcessName: processName,                        //string
				}
			}
			processExecutorStateItem.State = processExecutorStates[i].processState
			if len(processExecutorStates[i].processState) > 0 {
				processExecutorStateItem.Execute = processExecutorStates[i].processState
				processExecutorStateItem.State = "start_process"
			}
			processExecutorStateItem.ProcessStates = append(processExecutorStateItem.ProcessStates, processExecutorStates[i].data)
			processStateByProcessID[processExecutorStates[i].processID] = processExecutorStateItem
		}
		for i := range processStateByProcessID {
			processExecutorStateItems = append(processExecutorStateItems, processStateByProcessID[i])
		}
		//processExecutorState := processExecutorStates[len(processExecutorStates)-1]
	*/

	require.NoError(t, pe.ContinueProcessExecutor(ctx, processExecutorUUID /*, processExecutorStateItems*/))

	<-pe.Stopped

	for i := range processExecutorStates {
		fmt.Printf("'%v' '%v' '%v' '%v'\r\n", processExecutorStates[i].ExecutorID, processExecutorStates[i].ProcessID, processExecutorStates[i].ProcessState,
			processExecutorStates[i].State)
		if len(processExecutorStates[i].Data) > 0 {
			var dt map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(processExecutorStates[i].Data), &dt))
			dataRaw, err := json.MarshalIndent(&dt, " ", "  ")
			require.NoError(t, err)
			fmt.Printf("%v\r\n", string(dataRaw))
		}
	}

}
