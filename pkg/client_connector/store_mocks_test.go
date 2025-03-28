// Code generated by MockGen. DO NOT EDIT.
// Source: types.go

// Package clientconnector is a generated GoMock package.
package clientconnector

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	entity "github.com/wanderer69/flow_processor/pkg/entity"
)

// Mockloader is a mock of loader interface.
type Mockloader struct {
	ctrl     *gomock.Controller
	recorder *MockloaderMockRecorder
}

// MockloaderMockRecorder is the mock recorder for Mockloader.
type MockloaderMockRecorder struct {
	mock *Mockloader
}

// NewMockloader creates a new mock instance.
func NewMockloader(ctrl *gomock.Controller) *Mockloader {
	mock := &Mockloader{ctrl: ctrl}
	mock.recorder = &MockloaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockloader) EXPECT() *MockloaderMockRecorder {
	return m.recorder
}

// Load mocks base method.
func (m *Mockloader) Load(processRaw string) (*entity.Process, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Load", processRaw)
	ret0, _ := ret[0].(*entity.Process)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Load indicates an expected call of Load.
func (mr *MockloaderMockRecorder) Load(processRaw interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Load", reflect.TypeOf((*Mockloader)(nil).Load), processRaw)
}

// Save mocks base method.
func (m *Mockloader) Save(process *entity.Process) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save", process)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Save indicates an expected call of Save.
func (mr *MockloaderMockRecorder) Save(process interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*Mockloader)(nil).Save), process)
}

// MockprocessRepository is a mock of processRepository interface.
type MockprocessRepository struct {
	ctrl     *gomock.Controller
	recorder *MockprocessRepositoryMockRecorder
}

// MockprocessRepositoryMockRecorder is the mock recorder for MockprocessRepository.
type MockprocessRepositoryMockRecorder struct {
	mock *MockprocessRepository
}

// NewMockprocessRepository creates a new mock instance.
func NewMockprocessRepository(ctrl *gomock.Controller) *MockprocessRepository {
	mock := &MockprocessRepository{ctrl: ctrl}
	mock.recorder = &MockprocessRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockprocessRepository) EXPECT() *MockprocessRepositoryMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockprocessRepository) Create(ctx context.Context, c *entity.StoreProcess) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, c)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockprocessRepositoryMockRecorder) Create(ctx, c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockprocessRepository)(nil).Create), ctx, c)
}

// DeleteByProcessID mocks base method.
func (m *MockprocessRepository) DeleteByProcessID(ctx context.Context, processID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteByProcessID", ctx, processID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteByProcessID indicates an expected call of DeleteByProcessID.
func (mr *MockprocessRepositoryMockRecorder) DeleteByProcessID(ctx, processID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteByProcessID", reflect.TypeOf((*MockprocessRepository)(nil).DeleteByProcessID), ctx, processID)
}

// GetByProcessID mocks base method.
func (m *MockprocessRepository) GetByProcessID(ctx context.Context, processID string) ([]*entity.StoreProcess, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByProcessID", ctx, processID)
	ret0, _ := ret[0].([]*entity.StoreProcess)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByProcessID indicates an expected call of GetByProcessID.
func (mr *MockprocessRepositoryMockRecorder) GetByProcessID(ctx, processID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByProcessID", reflect.TypeOf((*MockprocessRepository)(nil).GetByProcessID), ctx, processID)
}

// GetNotFinishedByExecutorID mocks base method.
func (m *MockprocessRepository) GetNotFinishedByExecutorID(ctx context.Context, executorID string) ([]*entity.StoreProcess, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNotFinishedByExecutorID", ctx, executorID)
	ret0, _ := ret[0].([]*entity.StoreProcess)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNotFinishedByExecutorID indicates an expected call of GetNotFinishedByExecutorID.
func (mr *MockprocessRepositoryMockRecorder) GetNotFinishedByExecutorID(ctx, executorID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNotFinishedByExecutorID", reflect.TypeOf((*MockprocessRepository)(nil).GetNotFinishedByExecutorID), ctx, executorID)
}

// Update mocks base method.
func (m *MockprocessRepository) Update(ctx context.Context, c *entity.StoreProcess) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, c)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockprocessRepositoryMockRecorder) Update(ctx, c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockprocessRepository)(nil).Update), ctx, c)
}

// MockdiagrammRepository is a mock of diagrammRepository interface.
type MockdiagrammRepository struct {
	ctrl     *gomock.Controller
	recorder *MockdiagrammRepositoryMockRecorder
}

// MockdiagrammRepositoryMockRecorder is the mock recorder for MockdiagrammRepository.
type MockdiagrammRepositoryMockRecorder struct {
	mock *MockdiagrammRepository
}

// NewMockdiagrammRepository creates a new mock instance.
func NewMockdiagrammRepository(ctrl *gomock.Controller) *MockdiagrammRepository {
	mock := &MockdiagrammRepository{ctrl: ctrl}
	mock.recorder = &MockdiagrammRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockdiagrammRepository) EXPECT() *MockdiagrammRepositoryMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockdiagrammRepository) Create(ctx context.Context, c *entity.Diagramm) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, c)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockdiagrammRepositoryMockRecorder) Create(ctx, c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockdiagrammRepository)(nil).Create), ctx, c)
}

// GetByName mocks base method.
func (m *MockdiagrammRepository) GetByName(ctx context.Context, name string) ([]*entity.Diagramm, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByName", ctx, name)
	ret0, _ := ret[0].([]*entity.Diagramm)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByName indicates an expected call of GetByName.
func (mr *MockdiagrammRepositoryMockRecorder) GetByName(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByName", reflect.TypeOf((*MockdiagrammRepository)(nil).GetByName), ctx, name)
}

// Update mocks base method.
func (m *MockdiagrammRepository) Update(ctx context.Context, c *entity.Diagramm) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, c)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockdiagrammRepositoryMockRecorder) Update(ctx, c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockdiagrammRepository)(nil).Update), ctx, c)
}
