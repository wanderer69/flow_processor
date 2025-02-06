package clientconnector

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/wanderer69/flow_processor/pkg/entity"
	internalformat "github.com/wanderer69/flow_processor/pkg/internal_format"
)

func makeDiagrammI(t *testing.T, processName, topic1, topic2 string) string {
	/*
	   тестовая последовательность
	   1. старт -> подаем переменные a b
	   2. service task -> получает переменные, вызывает внешнюю таску, передает переменные
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
		Name: processName,
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
	return processRawFull
}

func makeDiagrammII(t *testing.T, processName, topic1, topic2, taskName3 string) string {
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
		CamundaModelerName: taskName3,
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
		Name: processName,
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
	return processRawFull
}

/*
func makeDiagrammIII(t *testing.T, processName, topic1, topic2, taskName3 string) string {
}
*/
