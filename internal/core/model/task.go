package model

import (
	"encoding/json"

	"github.com/rs/xid"
)

type TaskID string

func NewTaskID() TaskID {
	return TaskID(xid.New().String())
}

type Task interface {
	WithOwner

	json.Marshaler
	json.Unmarshaler

	ID() TaskID
	Type() TaskType
}

type TaskType string
