package entity

type Context struct {
	VariablesByName map[string]*Variable
	MessagesByName  map[string]*Message
}

type ProcessorContext struct {
	ProcessName string
	ProcessID   string
	CurrentTask string
}
