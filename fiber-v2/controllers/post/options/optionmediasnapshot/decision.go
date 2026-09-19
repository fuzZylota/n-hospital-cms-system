// Package optionmediasnapshot provides the narrow read boundary used by the
// option-media mutation handler.
package optionmediasnapshot

import (
	"context"
	"models/data"
	"reflect"
)

type snapshotUnavailableError struct {
	contextErr error
}

func (*snapshotUnavailableError) Error() string {
	return "option media snapshot unavailable"
}

func (e *snapshotUnavailableError) Is(target error) bool {
	return e != nil && e.contextErr != nil && target == e.contextErr
}

var errSnapshotUnavailable = &snapshotUnavailableError{}

// Read obtains one active option-media mutation snapshot. Missing data and
// dependency failures return no snapshot and do not expose repository details.
// Caller cancellation identity is retained.
func Read(ctx context.Context, reader data.OptionMediaMutationSnapshotReader) (data.OptionMediaMutationSnapshot, error) {
	if isNilReader(reader) {
		return data.OptionMediaMutationSnapshot{}, errSnapshotUnavailable
	}

	snapshot, found, err := reader.ReadOptionMediaMutationSnapshot(ctx)
	if err != nil {
		if ctx != nil {
			switch ctx.Err() {
			case context.Canceled:
				return data.OptionMediaMutationSnapshot{}, &snapshotUnavailableError{contextErr: context.Canceled}
			case context.DeadlineExceeded:
				return data.OptionMediaMutationSnapshot{}, &snapshotUnavailableError{contextErr: context.DeadlineExceeded}
			}
		}
		return data.OptionMediaMutationSnapshot{}, errSnapshotUnavailable
	}
	if !found {
		return data.OptionMediaMutationSnapshot{}, errSnapshotUnavailable
	}

	return snapshot, nil
}

func isNilReader(reader data.OptionMediaMutationSnapshotReader) bool {
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
