// Package contactrequestresponsesnapshot provides the narrow read boundary
// used by the contact-request response handler.
package contactrequestresponsesnapshot

import (
	"context"
	"models/data"
	"reflect"
)

type snapshotUnavailableError struct {
	contextErr error
}

func (*snapshotUnavailableError) Error() string {
	return "contact request response options unavailable"
}

func (e *snapshotUnavailableError) Is(target error) bool {
	return e != nil && e.contextErr != nil && target == e.contextErr
}

// Read obtains one active contact-request response workflow snapshot. Missing
// data and dependency failures return no snapshot and do not expose repository
// details. Caller cancellation identity is retained without exposing backend
// identity.
func Read(ctx context.Context, reader data.ContactRequestResponseWorkflowSnapshotReader) (data.ContactRequestResponseWorkflowSnapshot, error) {
	if ctx == nil || isNilReader(reader) {
		return data.ContactRequestResponseWorkflowSnapshot{}, &snapshotUnavailableError{}
	}

	snapshot, found, err := reader.ReadContactRequestResponseWorkflowSnapshot(ctx)
	if err != nil {
		switch ctx.Err() {
		case context.Canceled:
			return data.ContactRequestResponseWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.Canceled}
		case context.DeadlineExceeded:
			return data.ContactRequestResponseWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.DeadlineExceeded}
		default:
			return data.ContactRequestResponseWorkflowSnapshot{}, &snapshotUnavailableError{}
		}
	}
	if !found {
		return data.ContactRequestResponseWorkflowSnapshot{}, &snapshotUnavailableError{}
	}

	return snapshot, nil
}

func isNilReader(reader data.ContactRequestResponseWorkflowSnapshotReader) bool {
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
