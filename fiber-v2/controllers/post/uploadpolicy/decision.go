// Package uploadpolicy maps the active upload-policy read into the byte limit
// consumed by upload handlers.
package uploadpolicy

import (
	"context"
	"errors"
	"models/data"
	"reflect"
)

var errPolicyUnavailable = errors.New("policy unavailable")

// Decision is present only after exactly one successful active-policy read.
// MaxBytes retains the repository value without applying a fallback or unit
// conversion.
type Decision struct {
	MaxBytes int64
}

// Read obtains the upload policy for one request. Missing data and dependency
// failures return no decision; repository details are not exposed. Context
// cancellation identity is retained for callers and tests.
func Read(ctx context.Context, reader data.UploadPolicyReader) (Decision, error) {
	if isNilReader(reader) {
		return Decision{}, errPolicyUnavailable
	}

	policy, found, err := reader.ReadUploadPolicy(ctx)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return Decision{}, context.Canceled
		case errors.Is(err, context.DeadlineExceeded):
			return Decision{}, context.DeadlineExceeded
		default:
			return Decision{}, errPolicyUnavailable
		}
	}
	if !found {
		return Decision{}, errPolicyUnavailable
	}

	return Decision{MaxBytes: policy.MaxBytes}, nil
}

func isNilReader(reader data.UploadPolicyReader) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
