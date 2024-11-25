package store

import (
	"encoding/json"
	"fmt"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type InternalProcess struct {
	ProcessName string
	ProcessID   string
}

type StoreProcessDiagrammCallback func(processName string, processDiagramm string) error
type LoadProcessDiagrammCallback func(processName string) (string, error)

type StoreProcessStateCallback func(processName string, processID string, processContext string) error
type LoadProcessStateCallback func(processName string, processID string) (string, error)
type LoadStoredProcessesListCallback func() ([]*InternalProcess, error)
type StoreProcessExecutorStateCallback func(processExecutor string, processExecutorContext string) error
type LoadProcessExecutorStateCallback func(processExecutor string) (string, error)
type StoreStartProcessStateCallback func(processExecutor, processID, processExecutorState string) error
type StoreChangeProcessStateCallback func(processExecutor, processID, processState string, data string) error
type StoreFinishProcessStateCallback func(processExecutor, processID, processExecutorState string) error

type Store struct {
	storeProcessDiagrammCallback      StoreProcessDiagrammCallback
	loadProcessDiagrammCallback       LoadProcessDiagrammCallback
	storeProcessStateCallback         StoreProcessStateCallback
	loadProcessStateCallback          LoadProcessStateCallback
	loadStoredProcessesListCallback   LoadStoredProcessesListCallback
	storeProcessExecutorStateCallback StoreProcessExecutorStateCallback
	loadProcessExecutorStateCallback  LoadProcessExecutorStateCallback
	loader                            Loader
	storeFinishProcessStateCallback   StoreFinishProcessStateCallback
	storeChangeProcessStateCallback   StoreChangeProcessStateCallback
	storeStartProcessStateCallback    StoreStartProcessStateCallback
}

func NewStore(loader Loader) *Store {
	return &Store{
		loader: loader,
	}
}

func (s *Store) SetStoreProcessDiagramm(fn StoreProcessDiagrammCallback) {
	s.storeProcessDiagrammCallback = fn
}

func (s *Store) SetLoadProcessDiagramm(fn LoadProcessDiagrammCallback) {
	s.loadProcessDiagrammCallback = fn
}

func (s *Store) SetStoreProcessState(fn StoreProcessStateCallback) {
	s.storeProcessStateCallback = fn
}

func (s *Store) SetLoadProcessState(fn LoadProcessStateCallback) {
	s.loadProcessStateCallback = fn
}

func (s *Store) SetStoreProcessExecutorState(fn StoreProcessExecutorStateCallback) {
	s.storeProcessExecutorStateCallback = fn
}

func (s *Store) SetLoadProcessExecutorState(fn LoadProcessExecutorStateCallback) {
	s.loadProcessExecutorStateCallback = fn
}

func (s *Store) SetLoadStoredProcessesListCallback(fn LoadStoredProcessesListCallback) {
	s.loadStoredProcessesListCallback = fn
}

func (s *Store) SetStoreStartProcessStateCallback(fn StoreStartProcessStateCallback) {
	s.storeStartProcessStateCallback = fn
}

func (s *Store) SetStoreChangeProcessStateCallback(fn StoreChangeProcessStateCallback) {
	s.storeChangeProcessStateCallback = fn
}

func (s *Store) SetStoreFinishProcessStateCallback(fn StoreFinishProcessStateCallback) {
	s.storeFinishProcessStateCallback = fn
}

func (s *Store) StoreProcessDiagramm(processName string, process *entity.Process) error {
	processDiagramm, err := s.loader.Save(process)
	if err != nil {
		return err
	}

	if s.storeProcessDiagrammCallback != nil {
		return s.storeProcessDiagrammCallback(processName, processDiagramm)
	}
	return fmt.Errorf("store diagramm callback not found")
}

func (s *Store) LoadProcessDiagramm(processName string) (*entity.Process, error) {
	if s.loadProcessDiagrammCallback != nil {
		processDiagramm, err := s.loadProcessDiagrammCallback(processName)
		if err != nil {
			return nil, err
		}
		return s.loader.Load(processDiagramm)
	}
	return nil, fmt.Errorf("load diagramm callback not found")
}

type taskState struct {
	ProcessName string
	ProcessID   string
	ElementUUID string
	State       string
	Ctx         *entity.Context
}

func (s *Store) StoreProcessState(processName, processID, elementUUID string, state string, ctx *entity.Context) error {
	ts := &taskState{
		ProcessName: processName,
		ProcessID:   processID,
		ElementUUID: elementUUID,
		State:       state,
		Ctx:         ctx,
	}
	dataRaw, err := json.Marshal(ts)
	if err != nil {
		return err
	}
	if s.storeProcessStateCallback != nil {
		return s.storeProcessStateCallback(processName, processID, string(dataRaw))
	}
	return fmt.Errorf("store process state callback not found")
}

func (s *Store) LoadProcessState(processName, processID string) (string, string, string, string, *entity.Context, error) {
	if s.loadProcessStateCallback == nil {
		return processName, processID, "", "", nil, fmt.Errorf("load process state callback not found")
	}
	dataRaw, err := s.loadProcessStateCallback(processName, processID)
	if err != nil {
		return processName, processID, "", "", nil, err
	}

	var ts taskState
	err = json.Unmarshal([]byte(dataRaw), &ts)
	if err != nil {
		return processName, processID, "", "", nil, err
	}

	return ts.ProcessName, ts.ProcessID, ts.ElementUUID, ts.State, ts.Ctx, nil
}

func (s *Store) LoadStoredProcessesList() ([]*InternalProcess, error) {
	if s.loadStoredProcessesListCallback != nil {
		return s.loadStoredProcessesListCallback()
	}
	return nil, fmt.Errorf("load processes list callback not found")
}

func (s *Store) StoreProcessExecutorState(processExecutor, processExecutorState string) error {
	if s.storeProcessExecutorStateCallback != nil {
		return s.storeProcessExecutorStateCallback(processExecutor, processExecutorState)
	}
	return fmt.Errorf("store process executor state callback not found")
}

func (s *Store) LoadProcessExecutorState(processExecutor string) (string, error) {
	if s.loadProcessExecutorStateCallback == nil {
		return "", fmt.Errorf("load process executor state callback not found")
	}
	dataRaw, err := s.loadProcessExecutorStateCallback(processExecutor)
	if err != nil {
		return "", err
	}

	return dataRaw, nil
}

func (s *Store) StoreStartProcessState(processExecutor, processID string, processExecutorState string) error {
	if s.storeStartProcessStateCallback != nil {
		return s.storeStartProcessStateCallback(processExecutor, processID, processExecutorState)
	}
	return fmt.Errorf("store start process state callback not found")
}

func (s *Store) StoreChangeProcessState(processExecutor, processID string, processState string, data string) error {
	if s.storeChangeProcessStateCallback != nil {
		return s.storeChangeProcessStateCallback(processExecutor, processID, processState, data)
	}
	return fmt.Errorf("store change process state callback not found")
}

func (s *Store) StoreFinishProcessState(processExecutor, processID string, processExecutorState string) error {
	if s.storeFinishProcessStateCallback != nil {
		return s.storeFinishProcessStateCallback(processExecutor, processID, processExecutorState)
	}
	return fmt.Errorf("store finish process state callback not found")
}
