package appointmentworkflowsnapshot

import (
	"context"
	"errors"
	"models/data"
	"strings"
	"testing"
	"time"
)

type fakeReader struct {
	snapshot data.AppointmentWorkflowSnapshot
	found    bool
	err      error
	calls    int
	ctx      context.Context
}

func (r *fakeReader) ReadAppointmentWorkflowSnapshot(ctx context.Context) (data.AppointmentWorkflowSnapshot, bool, error) {
	r.calls++
	r.ctx = ctx
	return r.snapshot, r.found, r.err
}

type privateError struct{ cause error }

func (*privateError) Error() string   { return "private backend SQL and SMTP credential" }
func (e *privateError) Unwrap() error { return e.cause }

func TestReadPreservesSnapshotAndCallerContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "request")
	want := data.AppointmentWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "7", IsActive: true}, SMTPPassword: "test-only"}
	reader := &fakeReader{snapshot: want, found: true}
	got, err := Read(ctx, reader)
	if err != nil || got != want || reader.calls != 1 || reader.ctx != ctx {
		t.Fatal("snapshot or caller context changed")
	}
}

func TestReadFailsClosedWithoutBackendDisclosure(t *testing.T) {
	backend := &privateError{cause: context.Canceled}
	for _, tc := range []struct {
		name   string
		ctx    context.Context
		reader data.AppointmentWorkflowSnapshotReader
		calls  int
	}{
		{"missing", context.Background(), &fakeReader{snapshot: data.AppointmentWorkflowSnapshot{SMTPPassword: "test-only"}}, 1},
		{"backend", context.Background(), &fakeReader{snapshot: data.AppointmentWorkflowSnapshot{SMTPPassword: "test-only"}, found: true, err: backend}, 1},
		{"backend sentinel", context.Background(), &fakeReader{err: context.Canceled}, 1},
		{"nil reader", context.Background(), nil, 0},
		{"typed nil reader", context.Background(), (*fakeReader)(nil), 0},
		{"nil context", nil, &fakeReader{found: true}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Read(tc.ctx, tc.reader)
			if got != (data.AppointmentWorkflowSnapshot{}) || err == nil || err.Error() != "appointment options unavailable" || strings.Contains(err.Error(), "private") || errors.Is(err, backend) || errors.Is(err, context.Canceled) {
				t.Fatal("backend detail or partial snapshot escaped")
			}
			if r, ok := tc.reader.(*fakeReader); ok && r != nil && r.calls != tc.calls {
				t.Fatal("reader invocation count changed")
			}
		})
	}
}

func TestReadContextIdentityComesOnlyFromCaller(t *testing.T) {
	for _, tc := range []struct {
		name    string
		makeCtx func() (context.Context, context.CancelFunc)
		backend error
		want    error
	}{
		{"caller canceled backend deadline", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}, &privateError{cause: context.DeadlineExceeded}, context.Canceled},
		{"caller deadline backend canceled", func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}, &privateError{cause: context.Canceled}, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.makeCtx()
			defer cancel()
			reader := &fakeReader{err: tc.backend}
			got, err := Read(ctx, reader)
			if got != (data.AppointmentWorkflowSnapshot{}) || err == nil || err.Error() != "appointment options unavailable" || !errors.Is(err, tc.want) || errors.Is(err, tc.backend) || reader.ctx != ctx || reader.calls != 1 {
				t.Fatal("caller context classification changed")
			}
		})
	}
}
