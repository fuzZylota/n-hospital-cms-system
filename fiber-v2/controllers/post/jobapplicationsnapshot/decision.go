// Package jobapplicationsnapshot provides the narrow read boundary used by the
// public job-application handler.
package jobapplicationsnapshot

import (
	"context"
	"models/data"
	"reflect"
)

type snapshotUnavailableError struct {
	contextErr error
}

func (*snapshotUnavailableError) Error() string {
	return "job application snapshot unavailable"
}

func (e *snapshotUnavailableError) Is(target error) bool {
	return e != nil && e.contextErr != nil && target == e.contextErr
}

var errSnapshotUnavailable = &snapshotUnavailableError{}

// Read obtains one active job-application workflow snapshot. Missing data and
// dependency failures return no snapshot and do not expose backend details.
func Read(ctx context.Context, reader data.JobApplicationWorkflowSnapshotReader) (data.JobApplicationWorkflowSnapshot, error) {
	if ctx == nil || isNilReader(reader) {
		return data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable
	}

	snapshot, found, err := reader.ReadJobApplicationWorkflowSnapshot(ctx)
	if err != nil {
		switch ctx.Err() {
		case context.Canceled:
			return data.JobApplicationWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.Canceled}
		case context.DeadlineExceeded:
			return data.JobApplicationWorkflowSnapshot{}, &snapshotUnavailableError{contextErr: context.DeadlineExceeded}
		default:
			return data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable
		}
	}
	if !found {
		return data.JobApplicationWorkflowSnapshot{}, errSnapshotUnavailable
	}

	return snapshot, nil
}

func isNilReader(reader data.JobApplicationWorkflowSnapshotReader) bool {
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
