package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/wanderer69/flow_processor/pkg/entity"
)

type InternalProcess struct {
	ProcessName string
	ProcessID   string
}

type ProcessStateItem struct {
	ProcessExecutor      string
	ProcessID            string
	ProcessExecutorState string
	State                string
	Data                 string
}

type Store struct {
	loader loader

	processRepository  processRepository
	diagrammRepository diagrammRepository
}

func NewStore(
	loader loader,
	processRepository processRepository,
	diagrammRepository diagrammRepository,
) *Store {
	return &Store{
		loader:             loader,
		processRepository:  processRepository,
		diagrammRepository: diagrammRepository,
	}
}

func (s *Store) StoreProcessDiagramm(ctx context.Context, processName string, process *entity.Process) error {
	processDiagramm, err := s.loader.Save(process)
	if err != nil {
		return err
	}

	diagramms, err := s.diagrammRepository.GetByName(ctx, processName)
	if err != nil {
		return err
	}
	if len(diagramms) > 0 {
		diagramms[0].Data = processDiagramm
		return s.diagrammRepository.Update(ctx, diagramms[0])
	}
	return s.diagrammRepository.Create(ctx, &entity.Diagramm{
		UUID: uuid.NewString(),
		Name: processName,
		Data: processDiagramm,
	})
}

func (s *Store) LoadProcessDiagramm(ctx context.Context, processName string) (*entity.Process, error) {
	diagramms, err := s.diagrammRepository.GetByName(ctx, processName)
	if err != nil {
		return nil, err
	}
	if len(diagramms) > 0 {
		return s.loader.Load(diagramms[0].Data)
	}

	return nil, fmt.Errorf("%v not found", processName)
}

func (s *Store) LoadProcessExecutorState(processExecutor string) (string, error) {
	/*
	   }

	   processDiagramm, err := s.loader.Save(process)
	   if err != nil {
	   	return err
	   }

	   if s.storeProcessDiagrammCallback != nil {
	   	return s.storeProcessDiagrammCallback(processName, processDiagramm)
	   }
	   return fmt.Errorf("store diagramm callback not found")

	   	if s.loadProcessDiagrammCallback != nil {
	   		processDiagramm, err := s.loadProcessDiagrammCallback(processName)
	   		if err != nil {
	   			return nil, err
	   		}
	   		return s.loader.Load(processDiagramm)
	   	}
	   	return nil, fmt.Errorf("load diagramm callback not found")




	   	if s.loadProcessExecutorStateCallback == nil {
	   		return "", fmt.Errorf("load process executor state callback not found")
	   	}
	   	dataRaw, err := s.loadProcessExecutorStateCallback(processExecutor)
	   	if err != nil {
	   		return "", err
	   	}
	*/
	return "", nil
}

func (s *Store) LoadProcessStates(ctx context.Context, processExecutor string) ([]*entity.ProcessExecutorStateItem, error) {
	processExecutorStateItems := []*entity.ProcessExecutorStateItem{}
	//	processExecutorUUID := ""
	processStateByProcessID := make(map[string]*entity.ProcessExecutorStateItem)

	processExecutorStates, err := s.processRepository.GetNotFinishedByExecutorID(ctx, processExecutor)
	if err != nil {
		return nil, err
	}

	for i := range processExecutorStates {
		processExecutorStateItem, ok := processStateByProcessID[processExecutorStates[i].ProcessID]
		if !ok {
			processName := ""
			var spc entity.StoreProcessContext
			if len(processExecutorStates[i].Data) > 0 {
				err = json.Unmarshal([]byte(processExecutorStates[i].Data), &spc)
				if err != nil {
					return nil, err
				}
				if len(spc.ProcessName) > 0 {
					processName = spc.ProcessName
				}
			}
			if len(processExecutorStates[i].UUID) > 0 {
				//				processExecutorUUID = processExecutorStates[i].ProcessExecutor
			}
			processExecutorStateItem = &entity.ProcessExecutorStateItem{
				ProcessID:   processExecutorStates[i].ProcessID, //string
				ProcessName: processName,                        //string
			}
		}
		processExecutorStateItem.State = processExecutorStates[i].State
		if len(processExecutorStates[i].State) > 0 {
			processExecutorStateItem.Execute = processExecutorStates[i].State
			processExecutorStateItem.State = "start_process"
		}
		processExecutorStateItem.ProcessStates = append(processExecutorStateItem.ProcessStates, processExecutorStates[i].Data)
		processStateByProcessID[processExecutorStates[i].ProcessID] = processExecutorStateItem
	}
	for i := range processStateByProcessID {
		processExecutorStateItems = append(processExecutorStateItems, processStateByProcessID[i])
	}

	/*
		if s.loadProcessStatesCallback == nil {
			return nil, fmt.Errorf("load process states callback not found")
		}
		dataRaw, err := s.loadProcessStatesCallback(processExecutor, processID)
		if err != nil {
			return nil, err
		}
	*/
	return processExecutorStateItems, nil
}

/*
	func (s *Store) LoadNotFinishedProcessStates(processExecutor string) ([]*ProcessStateItem, error) {
		if s.loadProcessStatesCallback == nil {
			return nil, fmt.Errorf("load process states callback not found")
		}
		dataRaw, err := s.loadNotFinishedProcessStatesCallback(processExecutor)
		if err != nil {
			return nil, err
		}

		return dataRaw, nil
	}
*/
func (s *Store) StoreStartProcessState(ctx context.Context, processExecutor, processID string, processExecutorState string) error {
	pes := &entity.StoreProcess{
		UUID:         uuid.NewString(),
		ExecutorID:   processExecutor,
		ProcessID:    processID,
		ProcessState: processExecutorState,
		Data:         "{}",
		State:        "start",
		//processExecutor: processExecutor,
	}
	return s.processRepository.Create(ctx, pes)
}

func (s *Store) StoreChangeProcessState(ctx context.Context, processExecutor, processID string, processState string, data string) error {
	pes := &entity.StoreProcess{
		UUID:         uuid.NewString(),
		ExecutorID:   processExecutor,
		ProcessID:    processID,
		ProcessState: "data",
		Data:         data,
		State:        "start",
		//processExecutor: processExecutor,
	}
	return s.processRepository.Create(ctx, pes)
}

func (s *Store) StoreFinishProcessState(ctx context.Context, processExecutor, processID string, processExecutorState string) error {
	pes := &entity.StoreProcess{
		UUID:         uuid.NewString(),
		ExecutorID:   processExecutor,
		ProcessID:    processID,
		ProcessState: processExecutorState,
		Data:         "{}",
		State:        "finish",
		//processExecutor: processExecutor,
	}
	if len(processExecutorState) == 0 {
		err := s.processRepository.DeleteByProcessID(ctx, processID)
		if err != nil {
			return err
		}
	}
	return s.processRepository.Create(ctx, pes)
}
