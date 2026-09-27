// Package uploadpolicy maps the active upload-policy read into the byte limit
// consumed by upload handlers.
package uploadpolicy

import (
	"context"
	"database/sql"
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
	return decision(policy, found, err)
}

type transactionReader interface {
	ReadUploadPolicyTx(context.Context, *sql.Tx) (data.UploadPolicy, bool, error)
}

// ReadInTx requires the existing reader to query the supplied transaction. It
// never falls back to a pool read when the transaction capability is absent.
func ReadInTx(ctx context.Context, reader data.UploadPolicyReader, tx *sql.Tx) (Decision, error) {
	if isNilReader(reader) || tx == nil {
		return Decision{}, errPolicyUnavailable
	}
	txReader, ok := reader.(transactionReader)
	if !ok {
		return Decision{}, errPolicyUnavailable
	}
	policy, found, err := txReader.ReadUploadPolicyTx(ctx, tx)
	return decision(policy, found, err)
}

func decision(policy data.UploadPolicy, found bool, err error) (Decision, error) {
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
