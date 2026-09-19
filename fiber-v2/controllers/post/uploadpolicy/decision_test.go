package uploadpolicy

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
	policy          data.UploadPolicy
	found           bool
	err             error
	useContextError bool
	calls           int
	ctx             context.Context
}

func (r *fakeReader) ReadUploadPolicy(ctx context.Context) (data.UploadPolicy, bool, error) {
	r.calls++
	r.ctx = ctx
	if r.useContextError {
		return data.UploadPolicy{}, false, ctx.Err()
	}
	return r.policy, r.found, r.err
}

func TestReadDecisionMatrix(t *testing.T) {
	backendErr := errors.New("private-backend uid=41 dto=secret path=private maxbytes=credential")
	tests := []struct {
		name         string
		makeContext  func() (context.Context, context.CancelFunc)
		reader       data.UploadPolicyReader
		wantMaxBytes int64
		wantDecision bool
		wantCalls    int
		wantContext  error
	}{
		{name: "positive", reader: &fakeReader{policy: data.UploadPolicy{MaxBytes: 5242880}, found: true}, wantMaxBytes: 5242880, wantDecision: true, wantCalls: 1},
		{name: "zero", reader: &fakeReader{policy: data.UploadPolicy{MaxBytes: 0}, found: true}, wantDecision: true, wantCalls: 1},
		{name: "negative", reader: &fakeReader{policy: data.UploadPolicy{MaxBytes: -1}, found: true}, wantMaxBytes: -1, wantDecision: true, wantCalls: 1},
		{name: "missing", reader: &fakeReader{}, wantCalls: 1},
		{name: "reader error", reader: &fakeReader{err: backendErr}, wantCalls: 1},
		{
			name: "context canceled",
			makeContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			reader: &fakeReader{useContextError: true}, wantCalls: 1, wantContext: context.Canceled,
		},
		{
			name: "context deadline",
			makeContext: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
			reader: &fakeReader{useContextError: true}, wantCalls: 1, wantContext: context.DeadlineExceeded,
		},
		{name: "nil reader"},
		{name: "typed nil reader", reader: (*fakeReader)(nil)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), contextKey("request"), test.name)
			cancel := func() {}
			if test.makeContext != nil {
				ctx, cancel = test.makeContext()
			}
			defer cancel()

			trackedReader, _ := test.reader.(*fakeReader)
			decision, err := Read(ctx, test.reader)
			if (err == nil) != test.wantDecision {
				t.Fatal("unexpected decision availability")
			}
			if test.wantDecision && decision.MaxBytes != test.wantMaxBytes {
				t.Fatal("upload limit was not preserved")
			}
			if !test.wantDecision && decision != (Decision{}) {
				t.Fatal("failed read returned a decision")
			}
			if trackedReader != nil {
				if trackedReader.calls != test.wantCalls {
					t.Fatal("unexpected reader call count")
				}
				if test.wantCalls == 1 && trackedReader.ctx != ctx {
					t.Fatal("request context identity changed")
				}
			}
			if test.wantContext != nil {
				if !errors.Is(err, test.wantContext) {
					t.Fatal("context error identity changed")
				}
			} else if test.wantDecision && err != nil {
				t.Fatal("successful read returned an error")
			} else if !test.wantDecision && err == nil {
				t.Fatal("failed read returned no safe error")
			}
			if err != nil {
				message := strings.ToLower(err.Error())
				for _, forbidden := range []string{"private-backend", "uid=", "dto=", "path=", "maxbytes=", "secret", "credential"} {
					if strings.Contains(message, forbidden) {
						t.Fatal("private diagnostic escaped")
					}
				}
				if errors.Is(err, backendErr) {
					t.Fatal("backend error identity escaped")
				}
			}
		})
	}
}
