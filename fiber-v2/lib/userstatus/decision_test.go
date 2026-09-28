package userstatus

import (
	"bytes"
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"models/data"
)

type readerStub struct {
	status data.UserStatus
	err    error
	calls  int
	ctx    context.Context
	userID string
}

func (r *readerStub) LookupUserStatus(ctx context.Context, userID string) (data.UserStatus, error) {
	r.calls++
	r.ctx = ctx
	r.userID = userID
	return r.status, r.err
}

func TestShouldExpireAuthCookiesOutcomes(t *testing.T) {
	backend := errors.New("private backend credential uid=41 role=admin")
	for _, test := range []struct {
		name    string
		status  data.UserStatus
		err     error
		want    bool
		role    string
		wantErr bool
	}{
		{name: "active", status: data.UserStatus{Found: true, Active: true, Role: "admin"}, role: "admin"},
		{name: "inactive", status: data.UserStatus{Found: true, Active: false, Role: "moderator"}, want: true},
		{name: "missing", status: data.UserStatus{Found: false}, want: true},
		{name: "reader error", err: backend, wantErr: true},
		{name: "missing role", status: data.UserStatus{Found: true, Active: true}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), struct{}{}, "request-bound")
			reader := &readerStub{status: test.status, err: test.err}
			got, role, err := ShouldExpireAuthCookies(ctx, reader, "41")
			if got != test.want || role != test.role || (err != nil) != test.wantErr {
				t.Fatal("unexpected expiration decision")
			}
			if reader.calls != 1 || reader.ctx != ctx || reader.userID != "41" {
				t.Fatal("reader call contract changed")
			}
		})
	}
}

func TestShouldExpireAuthCookiesUnavailableInputFailsClosed(t *testing.T) {
	if expire, role, err := ShouldExpireAuthCookies(context.Background(), nil, "41"); expire || role != "" || err == nil {
		t.Fatal("nil reader was treated as a valid status")
	}
	reader := &readerStub{status: data.UserStatus{Found: false}}
	if expire, role, err := ShouldExpireAuthCookies(context.Background(), reader, ""); expire || role != "" || err == nil || reader.calls != 0 {
		t.Fatal("empty user ID was treated as a valid status")
	}
}

func TestShouldExpireAuthCookiesDoesNotLogReaderData(t *testing.T) {
	var output bytes.Buffer
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})

	reader := &readerStub{err: errors.New("private backend credential uid=41 role=admin")}
	if expire, role, err := ShouldExpireAuthCookies(context.Background(), reader, "41"); expire || role != "" || err == nil {
		t.Fatal("reader error expired cookies")
	}
	if output.Len() != 0 {
		t.Fatal("reader data was logged")
	}
}

func TestShouldExpireAuthCookiesContextFailuresDoNotExpire(t *testing.T) {
	for _, test := range []struct {
		name    string
		makeCtx func() (context.Context, context.CancelFunc)
		wantErr error
	}{
		{
			name: "canceled",
			makeCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline",
			makeCtx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
			wantErr: context.DeadlineExceeded,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := test.makeCtx()
			defer cancel()
			reader := &contextReaderStub{}
			if expire, role, err := ShouldExpireAuthCookies(ctx, reader, "41"); expire || role != "" || !errors.Is(err, test.wantErr) {
				t.Fatal("context failure expired cookies")
			}
			if reader.calls != 1 || reader.ctx != ctx || reader.userID != "41" || !errors.Is(reader.err, test.wantErr) {
				t.Fatal("context was not passed through to the reader")
			}
		})
	}
}

type contextReaderStub struct {
	calls  int
	ctx    context.Context
	userID string
	err    error
}

func (r *contextReaderStub) LookupUserStatus(ctx context.Context, userID string) (data.UserStatus, error) {
	r.calls++
	r.ctx = ctx
	r.userID = userID
	r.err = ctx.Err()
	return data.UserStatus{}, r.err
}
