// Package appointmentrequestsnapshot keeps the appointment-request option read
// behind a request-local, opaque error boundary.
package appointmentrequestsnapshot

import (
	"context"
	"models/data"
	"reflect"
)

type unavailableError struct{ contextErr error }

func (*unavailableError) Error() string { return "appointment request options unavailable" }

func (e *unavailableError) Is(target error) bool {
	return e != nil && e.contextErr != nil && target == e.contextErr
}

var errUnavailable = &unavailableError{}

// Read returns only one active snapshot. Neither backend errors nor missing
// data expose backend details; context identity comes solely from the caller.
func Read(ctx context.Context, reader data.AppointmentRequestWorkflowSnapshotReader) (data.AppointmentRequestWorkflowSnapshot, error) {
	if ctx == nil || nilReader(reader) {
		return data.AppointmentRequestWorkflowSnapshot{}, errUnavailable
	}
	snapshot, found, err := reader.ReadAppointmentRequestWorkflowSnapshot(ctx)
	if err != nil {
		switch ctx.Err() {
		case context.Canceled:
			return data.AppointmentRequestWorkflowSnapshot{}, &unavailableError{contextErr: context.Canceled}
		case context.DeadlineExceeded:
			return data.AppointmentRequestWorkflowSnapshot{}, &unavailableError{contextErr: context.DeadlineExceeded}
		default:
			return data.AppointmentRequestWorkflowSnapshot{}, errUnavailable
		}
	}
	if !found {
		return data.AppointmentRequestWorkflowSnapshot{}, errUnavailable
	}
	return snapshot, nil
}

func nilReader(reader data.AppointmentRequestWorkflowSnapshotReader) bool {
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
