// Package passwordpolicy maps the active password-policy read into the
// decision consumed by user password mutations.
package passwordpolicy

import (
	"context"
	"errors"
	"models/data"
	"reflect"
)

var errPolicyUnavailable = errors.New("policy unavailable")

// Decision is present only after exactly one successful active-policy read.
// RequireStrong retains the repository value without applying a fallback.
type Decision struct {
	RequireStrong bool
}

// Read obtains the password policy for one mutation request. Missing data and
// dependency failures return no decision; repository details are not exposed.
// Context cancellation identity is retained for callers and tests.
func Read(ctx context.Context, reader data.PasswordPolicyReader) (Decision, error) {
	if isNilReader(reader) {
		return Decision{}, errPolicyUnavailable
	}

	policy, found, err := reader.ReadPasswordPolicy(ctx)
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

	return Decision{RequireStrong: policy.RequireStrong}, nil
}

func isNilReader(reader data.PasswordPolicyReader) bool {
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
