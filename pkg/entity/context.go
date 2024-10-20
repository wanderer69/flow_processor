package entity

type Context struct {
	VariablesByName map[string]*Variable
	MessagesByName  map[string]*Message
}
