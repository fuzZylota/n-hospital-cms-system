// Package contactrequestsnapshot provides the narrow read boundary used by the
// public contact-request handler.
package contactrequestsnapshot

import (
	"context"
	"models/data"
	"reflect"
)

type snapshotUnavailableError struct {
	contextErr error
}

func (*snapshotUnavailableError) Error() string {
	return "contact request options unavailable"
}

func (e *snapshotUnavailableError) Is(target error) bool {
	return e != nil && e.contextErr != nil && target == e.contextErr
}

var errSnapshotUnavailable = &snapshotUnavailableError{}

// Read obtains one active contact-request workflow snapshot. Missing data and
// dependency failures return no snapshot and do not expose repository details.
// Caller cancellation identity is retained without exposing backend identity.
func Read(ctx context.Context, reader data.ContactRequestWorkflowSnapshotReader) (data.ContactRequestWorkflowSnapshot, error) {
	if ctx == nil || isNilReader(reader) {
		return data.ContactRequestWorkflowSnapshot{}, errSnapshotUnavailable
	}

	snapshot, found, err := reader.ReadContactRequestWorkflowSnapshot(ctx)
	if err != nil {
		switch ctx.Err() {
		case context.Canceled:
			return data.ContactRequestWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.Canceled}
		case context.DeadlineExceeded:
			return data.ContactRequestWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.DeadlineExceeded}
		default:
			return data.ContactRequestWorkflowSnapshot{}, errSnapshotUnavailable
		}
	}
	if !found {
		return data.ContactRequestWorkflowSnapshot{}, errSnapshotUnavailable
	}

	return snapshot, nil
}

func isNilReader(reader data.ContactRequestWorkflowSnapshotReader) bool {
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
