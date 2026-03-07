package setup

import (
	"net/url"

	"github.com/pkg/errors"
)

var ErrNotRegistered = errors.New("not registered")

type Factory[T any] func(u *url.URL) (T, error)

type Registry[T any] struct {
	mappings map[string]Factory[T]
}

func (r *Registry[T]) Register(scheme string, factory Factory[T]) {
	r.mappings[scheme] = factory
}

func (r *Registry[T]) From(rawURL string) (T, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return *new(T), errors.WithStack(err)
	}

	factory, exists := r.mappings[u.Scheme]
	if !exists {
		return *new(T), errors.Wrapf(ErrNotRegistered, "scheme '%s' not found", u.Scheme)
	}

	value, err := factory(u)
	if err != nil {
		return *new(T), errors.WithStack(err)
	}

	return value, nil
}

func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		mappings: make(map[string]Factory[T]),
	}
}
