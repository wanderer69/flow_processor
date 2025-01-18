package entity

import "time"

type StoreProcess struct {
	UUID         string
	ExecutorID   string
	ProcessID    string
	ProcessState string
	Data         string
	State        string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type ProcessExecutorStateItem struct {
	ProcessID     string
	ProcessName   string
	State         string
	Execute       string
	ProcessStates []string
}

type ProcessElementData struct {
	NextElements []*Element
	WaitFlowCnt  int
}

type StoreProcessContext struct {
	ProcessID          string
	Ctx                *Context
	Msg                *ChannelMessage
	ProcessElementData *ProcessElementData
	IsFinish           bool
	IsWait             bool
	ProcessName        string
}
