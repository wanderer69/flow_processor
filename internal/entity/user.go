package entity

import "time"

type User struct {
	UUID    string
	Login   string
	Email   string
	Hash    string
	RegCode string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
