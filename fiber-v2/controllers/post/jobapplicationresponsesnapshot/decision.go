// Package jobapplicationresponsesnapshot provides the narrow read boundary for
// the job-application response handler.
package jobapplicationresponsesnapshot

import (
	"context"
	"models/data"
	"reflect"
)

type snapshotUnavailableError struct {
	contextErr error
}

func (*snapshotUnavailableError) Error() string {
	return "job application response snapshot unavailable"
}

func (e *snapshotUnavailableError) Is(target error) bool {
	return e != nil && e.contextErr != nil && target == e.contextErr
}

// Read obtains one active response snapshot without exposing backend failures.
func Read(ctx context.Context, reader data.JobApplicationResponseWorkflowSnapshotReader) (data.JobApplicationResponseWorkflowSnapshot, error) {
	if ctx == nil || isNilReader(reader) {
		return data.JobApplicationResponseWorkflowSnapshot{}, unavailable(ctx)
	}

	snapshot, found, err := reader.ReadJobApplicationResponseWorkflowSnapshot(ctx)
	if err != nil {
		return data.JobApplicationResponseWorkflowSnapshot{}, unavailable(ctx)
	}
	if !found {
		return data.JobApplicationResponseWorkflowSnapshot{}, unavailable(ctx)
	}

	return snapshot, nil
}

func unavailable(ctx context.Context) error {
	result := &snapshotUnavailableError{}
	if ctx == nil {
		return result
	}
	switch ctx.Err() {
	case context.Canceled:
		result.contextErr = context.Canceled
	case context.DeadlineExceeded:
		result.contextErr = context.DeadlineExceeded
	}
	return result
}

func isNilReader(reader data.JobApplicationResponseWorkflowSnapshotReader) bool {
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
