package common

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
	"github.com/pkg/errors"
)

type viewModelFillerFunc[T any] func(ctx context.Context, vmodel *T, r *http.Request) error

func FillViewModel[T any](ctx context.Context, vmodel *T, r *http.Request, funcs ...viewModelFillerFunc[T]) error {
	for _, fn := range funcs {
		if err := fn(ctx, vmodel, r); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func DetectMimeType(input io.Reader) (mimeType string, recycled io.Reader, err error) {
	header := bytes.NewBuffer(nil)

	mtype, err := mimetype.DetectReader(io.TeeReader(input, header))
	if err != nil {
		return
	}

	recycled = io.MultiReader(header, input)

	return mtype.String(), recycled, err
}
