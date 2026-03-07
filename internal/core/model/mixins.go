package model

import (
	"time"
)

type WithID[T ~string] interface {
	ID() T
}

type WithOwner interface {
	Owner() User
}

type WithLifecycle interface {
	CreatedAt() time.Time
	UpdatedAt() time.Time
}
