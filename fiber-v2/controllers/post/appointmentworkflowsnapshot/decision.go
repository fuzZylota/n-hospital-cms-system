// Package appointmentworkflowsnapshot keeps the appointment option read behind
// a request-local, opaque error boundary.
package appointmentworkflowsnapshot

import (
	"context"
	"models/data"
	"reflect"
)

type unavailableError struct{ contextErr error }

func (*unavailableError) Error() string { return "appointment options unavailable" }

func (e *unavailableError) Is(target error) bool {
	return e != nil && e.contextErr != nil && target == e.contextErr
}

// Read returns one active snapshot. Backend details and partial snapshots stay
// private; cancellation identity comes only from the caller context.
func Read(ctx context.Context, reader data.AppointmentWorkflowSnapshotReader) (data.AppointmentWorkflowSnapshot, error) {
	if ctx == nil || nilReader(reader) {
		return data.AppointmentWorkflowSnapshot{}, unavailable(ctx)
	}
	snapshot, found, err := reader.ReadAppointmentWorkflowSnapshot(ctx)
	if err != nil || !found {
		return data.AppointmentWorkflowSnapshot{}, unavailable(ctx)
	}
	return snapshot, nil
}

func unavailable(ctx context.Context) error {
	result := &unavailableError{}
	if ctx != nil {
		switch ctx.Err() {
		case context.Canceled:
			result.contextErr = context.Canceled
		case context.DeadlineExceeded:
			result.contextErr = context.DeadlineExceeded
		}
	}
	return result
}

func nilReader(reader data.AppointmentWorkflowSnapshotReader) bool {
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
