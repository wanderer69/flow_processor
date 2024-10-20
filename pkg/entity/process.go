package entity

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
