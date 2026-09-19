package passwordpolicy

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
	policy          data.PasswordPolicy
	found           bool
	err             error
	useContextError bool
	calls           int
	ctx             context.Context
}

func (r *fakeReader) ReadPasswordPolicy(ctx context.Context) (data.PasswordPolicy, bool, error) {
	r.calls++
	r.ctx = ctx
	if r.useContextError {
		return data.PasswordPolicy{}, false, ctx.Err()
	}
	return r.policy, r.found, r.err
}

func TestReadDecisionMatrix(t *testing.T) {
	backendErr := errors.New("private-backend uid=41 dto=secret password=credential")
	tests := []struct {
		name         string
		makeContext  func() (context.Context, context.CancelFunc)
		reader       data.PasswordPolicyReader
		wantStrong   bool
		wantDecision bool
		wantCalls    int
		wantContext  error
	}{
		{
			name: "found strong true", reader: &fakeReader{
				policy: data.PasswordPolicy{RequireStrong: true}, found: true,
			}, wantStrong: true, wantDecision: true, wantCalls: 1,
		},
		{
			name: "found strong false", reader: &fakeReader{
				policy: data.PasswordPolicy{RequireStrong: false}, found: true,
			}, wantDecision: true, wantCalls: 1,
		},
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
			mutationDecisionStarted := err == nil
			if mutationDecisionStarted != test.wantDecision {
				t.Fatalf("mutation decision started = %v, want %v", mutationDecisionStarted, test.wantDecision)
			}
			if test.wantDecision {
				if decision.RequireStrong != test.wantStrong {
					t.Fatalf("RequireStrong = %v, want %v", decision.RequireStrong, test.wantStrong)
				}
			} else if decision != (Decision{}) {
				t.Fatalf("failed read returned decision: %+v", decision)
			}
			if trackedReader != nil {
				if trackedReader.calls != test.wantCalls {
					t.Fatalf("reader calls = %d, want %d", trackedReader.calls, test.wantCalls)
				}
				if test.wantCalls == 1 && trackedReader.ctx != ctx {
					t.Fatal("request context identity changed")
				}
			}
			if test.wantContext != nil {
				if !errors.Is(err, test.wantContext) {
					t.Fatalf("error = %v, want context identity %v", err, test.wantContext)
				}
			} else if test.wantDecision && err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if !test.wantDecision && err == nil {
				t.Fatal("missing safe error")
			}
			if err != nil {
				message := strings.ToLower(err.Error())
				for _, forbidden := range []string{"private-backend", "uid=", "dto=", "secret", "credential"} {
					if strings.Contains(message, forbidden) {
						t.Fatalf("diagnostic leaked %q: %q", forbidden, message)
					}
				}
				if errors.Is(err, backendErr) {
					t.Fatal("backend error identity escaped")
				}
			}
		})
	}
}
