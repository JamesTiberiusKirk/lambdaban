package models

import (
	"time"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusDone       Status = "done"
)

func (s Status) String() string {
	return string(s)
}

type Ticket struct {
	Id            string
	Title         string
	Description   string
	CreatedAt     time.Time
	LastUpdatedAt time.Time
	Status        Status
}
