package entity

import "time"

type Definition struct {
	Name             string
	Version          string
	CamundaModelerID string
}

type Process struct {
	Definition       *Definition
	Name             string
	Version          string
	CamundaModelerID string
	Elements         []*Element
}

type ChannelMessage struct {
	CurrentElement *Element

	Variables         []*Variable
	Messages          []*Message
	ActivationTime    time.Time
	ProcessID         string
	NextElementsNames []string
}
