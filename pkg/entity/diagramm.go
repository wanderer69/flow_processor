package entity

import "time"

type Diagramm struct {
	UUID string
	Data string
	Name string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
