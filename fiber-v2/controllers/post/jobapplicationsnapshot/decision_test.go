package jobapplicationsnapshot

import (
	"context"
	"errors"
	"models/data"
	"strings"
	"testing"
	"time"
)

type contextKey string

type fakeReader struct {
	snapshot data.JobApplicationWorkflowSnapshot
	found    bool
	err      error
	calls    int
	ctx      context.Context
}

func (r *fakeReader) ReadJobApplicationWorkflowSnapshot(ctx context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
	r.calls++
	r.ctx = ctx
	return r.snapshot, r.found, r.err
}

type snapshotReaderFunc func(context.Context) (data.JobApplicationWorkflowSnapshot, bool, error)

func (f snapshotReaderFunc) ReadJobApplicationWorkflowSnapshot(ctx context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
	return f(ctx)
}

type snapshotReaderMap map[string]struct{}

func (snapshotReaderMap) ReadJobApplicationWorkflowSnapshot(context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
	panic("nil map reader was invoked")
}

type snapshotReaderSlice []string

func (snapshotReaderSlice) ReadJobApplicationWorkflowSnapshot(context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
	panic("nil slice reader was invoked")
}

type snapshotReaderChan chan struct{}

func (snapshotReaderChan) ReadJobApplicationWorkflowSnapshot(context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
	panic("nil channel reader was invoked")
}

type valueReader struct {
	snapshot data.JobApplicationWorkflowSnapshot
}

func (r valueReader) ReadJobApplicationWorkflowSnapshot(context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
	return r.snapshot, true, nil
}

type privateBackendError struct {
	cause error
}

func (*privateBackendError) Error() string {
	return "private backend smtp password captcha secret sql row"
}

func (e *privateBackendError) Unwrap() error {
	return e.cause
}

func TestReadSnapshotMatrix(t *testing.T) {
	t.Parallel()
	snapshot := data.JobApplicationWorkflowSnapshot{
		Set:                data.OptionSetIdentity{ID: "7", IsActive: true},
		SMTPHost:           "smtp.invalid",
		SMTPPort:           587,
		SMTPUsername:       "internal-user",
		SMTPPassword:       "internal-password",
		SiteName:           "site-name",
		SiteDescription:    "site-description",
		ContactEmail:       "contact@example.invalid",
		ContactPhone:       "phone-value",
		FacebookURL:        "facebook-value",
		TwitterURL:         "twitter-value",
		InstagramURL:       "instagram-value",
		LinkedInURL:        "linkedin-value",
		PrimaryColor:       "primary-value",
		RecaptchaSiteKey:   "captcha-site-value",
		RecaptchaSecretKey: "internal-captcha-secret",
		SiteLogoPath:       "files/logo.png",
		AccentColor:        "",
		MaxBytes:           4096,
	}
	backendErr := &privateBackendError{cause: errors.New("private backend sentinel")}

	tests := []struct {
		name        string
		reader      *fakeReader
		wantSuccess bool
	}{
		{name: "found", reader: &fakeReader{snapshot: snapshot, found: true}, wantSuccess: true},
		{name: "missing", reader: &fakeReader{}},
		{name: "backend error", reader: &fakeReader{err: backendErr}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(context.Background(), contextKey("request"), test.name)
			got, err := Read(ctx, test.reader)
			if test.wantSuccess {
				if err != nil || got != snapshot {
					t.Fatal("successful snapshot was not preserved")
				}
			} else {
				assertOpaqueFailure(t, got, err, false, false, backendErr)
			}
			if test.reader.calls != 1 || test.reader.ctx != ctx {
				t.Fatal("reader call count or context identity changed")
			}
		})
	}
}

func TestReadPreservesMaxBytesBoundaries(t *testing.T) {
	for _, size := range []int64{4096, 0, -1} {
		reader := &fakeReader{snapshot: data.JobApplicationWorkflowSnapshot{MaxBytes: size}, found: true}
		got, err := Read(context.Background(), reader)
		if err != nil || got.MaxBytes != size || reader.calls != 1 {
			t.Fatal("snapshot byte limit changed")
		}
	}
}

func TestReadBackendContextIsolationMatrix(t *testing.T) {
	t.Parallel()
	backendCanceled := &privateBackendError{cause: context.Canceled}
	backendDeadline := &privateBackendError{cause: context.DeadlineExceeded}
	backendNormal := &privateBackendError{cause: errors.New("private backend sentinel")}

	tests := []struct {
		name         string
		makeContext  func() context.Context
		backend      error
		wantCanceled bool
		wantDeadline bool
	}{
		{name: "live bare canceled", makeContext: context.Background, backend: context.Canceled},
		{name: "live wrapped canceled", makeContext: context.Background, backend: backendCanceled},
		{name: "live bare deadline", makeContext: context.Background, backend: context.DeadlineExceeded},
		{name: "live wrapped deadline", makeContext: context.Background, backend: backendDeadline},
		{name: "caller canceled backend deadline", makeContext: canceledContext, backend: backendDeadline, wantCanceled: true},
		{name: "caller deadline backend canceled", makeContext: deadlineContext, backend: backendCanceled, wantDeadline: true},
		{name: "caller canceled backend normal", makeContext: canceledContext, backend: backendNormal, wantCanceled: true},
		{name: "caller deadline backend normal", makeContext: deadlineContext, backend: backendNormal, wantDeadline: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := test.makeContext()
			reader := &fakeReader{err: test.backend}
			got, err := Read(ctx, reader)
			assertOpaqueFailure(t, got, err, test.wantCanceled, test.wantDeadline, test.backend)
			if reader.calls != 1 || reader.ctx != ctx {
				t.Fatal("reader call count or context identity changed")
			}
		})
	}
}

func TestReadNilContextAndTypedNilReaderMatrix(t *testing.T) {
	t.Parallel()
	snapshot := data.JobApplicationWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "9", IsActive: true}}
	nilContextReader := &fakeReader{snapshot: snapshot, found: true}
	got, err := Read(nil, nilContextReader)
	assertOpaqueFailure(t, got, err, false, false, nil)
	if nilContextReader.calls != 0 {
		t.Fatal("nil context reached the reader")
	}

	functionCalls := 0
	functionReader := snapshotReaderFunc(func(context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
		functionCalls++
		return snapshot, true, nil
	})
	nilReaders := []data.JobApplicationWorkflowSnapshotReader{
		nil,
		(*fakeReader)(nil),
		snapshotReaderFunc(nil),
		snapshotReaderMap(nil),
		snapshotReaderSlice(nil),
		snapshotReaderChan(nil),
	}
	for _, reader := range nilReaders {
		got, err := Read(context.Background(), reader)
		assertOpaqueFailure(t, got, err, false, false, nil)
	}

	for _, reader := range []data.JobApplicationWorkflowSnapshotReader{
		functionReader,
		valueReader{snapshot: snapshot},
		&fakeReader{snapshot: snapshot, found: true},
	} {
		got, err := Read(context.Background(), reader)
		if err != nil || got != snapshot {
			t.Fatal("non-nil reader was classified as nil")
		}
	}
	if functionCalls != 1 {
		t.Fatal("non-nil function reader call count changed")
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func deadlineContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
	cancel()
	return ctx
}

func assertOpaqueFailure(t *testing.T, snapshot data.JobApplicationWorkflowSnapshot, err error, wantCanceled, wantDeadline bool, backend error) {
	t.Helper()
	if snapshot != (data.JobApplicationWorkflowSnapshot{}) || err == nil {
		t.Fatal("failed read result changed")
	}
	if err.Error() != "job application snapshot unavailable" {
		t.Fatal("opaque diagnostic changed")
	}
	if errors.Is(err, context.Canceled) != wantCanceled || errors.Is(err, context.DeadlineExceeded) != wantDeadline {
		t.Fatal("caller context identity changed")
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("opaque error unexpectedly unwraps")
	}
	if backend != nil && backend != context.Canceled && backend != context.DeadlineExceeded && errors.Is(err, backend) {
		t.Fatal("backend error identity escaped")
	}
	var leaked *privateBackendError
	if errors.As(err, &leaked) {
		t.Fatal("backend error type escaped")
	}
	message := strings.ToLower(err.Error())
	for _, forbidden := range []string{"private backend", "smtp", "password", "captcha", "secret", "sql", "row"} {
		if strings.Contains(message, forbidden) {
			t.Fatal("private diagnostic escaped")
		}
	}
}
