package gorm

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

// JSONColumn is a generic GORM column type that serializes T as JSON text.
// A nil Val is stored as SQL NULL and reads back as nil.
type JSONColumn[T any] struct {
	Val *T
}

// Value implements driver.Valuer.
func (j JSONColumn[T]) Value() (driver.Value, error) {
	if j.Val == nil {
		return nil, nil
	}
	b, err := json.Marshal(j.Val)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return string(b), nil
}

// Scan implements sql.Scanner.
func (j *JSONColumn[T]) Scan(value any) error {
	if value == nil {
		j.Val = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	default:
		return errors.Errorf("JSONColumn: unsupported type %T", value)
	}
	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return errors.WithStack(err)
	}
	j.Val = &t
	return nil
}
