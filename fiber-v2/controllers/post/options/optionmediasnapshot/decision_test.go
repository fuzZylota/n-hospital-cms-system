package optionmediasnapshot

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
	snapshot data.OptionMediaMutationSnapshot
	found    bool
	err      error
	calls    int
	ctx      context.Context
}

func (r *fakeReader) ReadOptionMediaMutationSnapshot(ctx context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	r.calls++
	r.ctx = ctx
	return r.snapshot, r.found, r.err
}

type snapshotReaderFunc func(context.Context) (data.OptionMediaMutationSnapshot, bool, error)

func (f snapshotReaderFunc) ReadOptionMediaMutationSnapshot(ctx context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	return f(ctx)
}

type snapshotReaderMap map[string]struct{}

func (snapshotReaderMap) ReadOptionMediaMutationSnapshot(context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	panic("nil map reader was invoked")
}

type snapshotReaderSlice []string

func (snapshotReaderSlice) ReadOptionMediaMutationSnapshot(context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	panic("nil slice reader was invoked")
}

type snapshotReaderChan chan struct{}

func (snapshotReaderChan) ReadOptionMediaMutationSnapshot(context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	panic("nil channel reader was invoked")
}

type valueReader struct {
	snapshot data.OptionMediaMutationSnapshot
}

func (r valueReader) ReadOptionMediaMutationSnapshot(context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	return r.snapshot, true, nil
}

type privateBackendError struct {
	cause error
}

func (*privateBackendError) Error() string {
	return "private backend uid dto path media-id maxbytes credential"
}

func (e *privateBackendError) Unwrap() error {
	return e.cause
}

func TestReadSnapshotMatrix(t *testing.T) {
	siteLogoID := "11"
	siteLightLogoID := "12"
	faviconID := "13"
	defaultPageMediaID := "14"
	negativeSnapshot := data.OptionMediaMutationSnapshot{
		Set:                data.OptionSetIdentity{ID: "7", IsActive: true, IsTesting: true},
		MaxBytes:           -9,
		SiteLogoID:         &siteLogoID,
		SiteLightLogoID:    &siteLightLogoID,
		FaviconID:          &faviconID,
		DefaultPageMediaID: &defaultPageMediaID,
	}
	positiveSnapshot := negativeSnapshot
	positiveSnapshot.MaxBytes = 5242880
	zeroSnapshot := data.OptionMediaMutationSnapshot{Set: data.OptionSetIdentity{ID: "8", IsActive: true}}
	backendErr := &privateBackendError{cause: errors.New("private backend sentinel")}

	tests := []struct {
		name         string
		reader       data.OptionMediaMutationSnapshotReader
		wantSnapshot data.OptionMediaMutationSnapshot
		wantSuccess  bool
		wantCalls    int
	}{
		{name: "found positive", reader: &fakeReader{snapshot: positiveSnapshot, found: true}, wantSnapshot: positiveSnapshot, wantSuccess: true, wantCalls: 1},
		{name: "found zero with null ids", reader: &fakeReader{snapshot: zeroSnapshot, found: true}, wantSnapshot: zeroSnapshot, wantSuccess: true, wantCalls: 1},
		{name: "found negative", reader: &fakeReader{snapshot: negativeSnapshot, found: true}, wantSnapshot: negativeSnapshot, wantSuccess: true, wantCalls: 1},
		{name: "missing", reader: &fakeReader{}, wantCalls: 1},
		{name: "backend error", reader: &fakeReader{err: backendErr}, wantCalls: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), contextKey("request"), test.name)
			trackedReader := test.reader.(*fakeReader)
			got, err := Read(ctx, test.reader)
			if test.wantSuccess {
				if err != nil || got != test.wantSnapshot {
					t.Fatal("successful snapshot was not preserved")
				}
			} else {
				assertOpaqueFailure(t, got, err, false, false, backendErr)
			}
			if trackedReader.calls != test.wantCalls || trackedReader.ctx != ctx {
				t.Fatal("reader call or context identity changed")
			}
		})
	}
}

func TestReadBackendContextIsolationMatrix(t *testing.T) {
	backendCanceled := &privateBackendError{cause: context.Canceled}
	backendDeadline := &privateBackendError{cause: context.DeadlineExceeded}
	backendNormal := &privateBackendError{cause: errors.New("private backend sentinel")}
	sameText := errors.New("context canceled")

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
		{name: "live same text", makeContext: context.Background, backend: sameText},
		{name: "caller canceled backend deadline", makeContext: canceledContext, backend: backendDeadline, wantCanceled: true},
		{name: "caller deadline backend canceled", makeContext: deadlineContext, backend: backendCanceled, wantDeadline: true},
		{name: "caller canceled backend normal", makeContext: canceledContext, backend: backendNormal, wantCanceled: true},
		{name: "caller deadline backend normal", makeContext: deadlineContext, backend: backendNormal, wantDeadline: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := test.makeContext()
			reader := &fakeReader{err: test.backend}
			got, err := Read(ctx, reader)
			assertOpaqueFailure(t, got, err, test.wantCanceled, test.wantDeadline, test.backend)
			if reader.calls != 1 || reader.ctx != ctx {
				t.Fatal("reader call or context identity changed")
			}
		})
	}
}

func TestReadTypedNilReaderMatrix(t *testing.T) {
	successSnapshot := data.OptionMediaMutationSnapshot{Set: data.OptionSetIdentity{ID: "9", IsActive: true}, MaxBytes: 31}
	functionCalls := 0
	functionReader := snapshotReaderFunc(func(context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
		functionCalls++
		return successSnapshot, true, nil
	})

	nilReaders := []data.OptionMediaMutationSnapshotReader{
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

	for _, reader := range []data.OptionMediaMutationSnapshotReader{
		functionReader,
		valueReader{snapshot: successSnapshot},
		&fakeReader{snapshot: successSnapshot, found: true},
	} {
		got, err := Read(context.Background(), reader)
		if err != nil || got != successSnapshot {
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

func assertOpaqueFailure(t *testing.T, snapshot data.OptionMediaMutationSnapshot, err error, wantCanceled, wantDeadline bool, backend error) {
	t.Helper()
	if snapshot != (data.OptionMediaMutationSnapshot{}) || err == nil {
		t.Fatal("failed read result changed")
	}
	if err.Error() != "option media snapshot unavailable" {
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
	for _, forbidden := range []string{"private backend", "uid", "dto", "path", "media-id", "maxbytes", "credential"} {
		if strings.Contains(message, forbidden) {
			t.Fatal("private diagnostic escaped")
		}
	}
}
