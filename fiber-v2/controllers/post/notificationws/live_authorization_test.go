package notificationws

import (
	"bytes"
	"context"
	"errors"
	"models/data"
	"models/notify"
	"post/notificationevent"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type liveReader struct {
	mu     sync.Mutex
	status data.UserStatus
	err    error
}

func (r *liveReader) LookupUserStatus(context.Context, string) (data.UserStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, r.err
}

func (r *liveReader) change(status data.UserStatus, err error) {
	r.mu.Lock()
	r.status, r.err = status, err
	r.mu.Unlock()
}

// The authorization callback models a fresh DB read after the upgrade.
func TestSEC003CInboundAfterAccountChange(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed bool
	}{
		{"active authorized", true},
		{"inactive", false},
		{"deleted", false},
		{"role downgraded", false},
		{"branch changed", false},
		{"lookup failed", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := observe(hub(t))
			s := socket()
			var handled atomic.Bool
			done := make(chan struct{})
			result := make(chan error, 1)
			go func() {
				defer close(done)
				result <- serveWithAuthorization(h, s, notify.UserID("42"), Appointment,
					func(uid notify.UserID, event Event, payload []byte) bool {
						return uid == "42" && event == Appointment && tc.allowed
					}, func(Event, []byte) { handled.Store(true) }, bytes.NewReader(make([]byte, 32)))
			}()
			select {
			case <-h.registered:
			case <-time.After(time.Second):
				t.Fatal("registration missing")
			}
			s.reads <- frame{kind: textMessage, data: []byte(`{"message":"event"}`)}
			wait(t, done)
			if handled.Load() != tc.allowed {
				t.Fatal("stale session processed or valid session rejected")
			}
			if err := <-result; (err == nil) != tc.allowed || (!tc.allowed && err != errAuthorization) {
				t.Fatal("authorization result was not a safe fixed error")
			}
		})
	}
}

func TestSEC003COutboundSocketAfterAccountChange(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status data.UserStatus
		err    error
		branch notify.BranchID
		rule   string
	}{
		{"inactive", data.UserStatus{Found: true, Active: false, Role: "santral"}, nil, "a", "appointment"},
		{"deleted", data.UserStatus{}, nil, "a", "contact"},
		{"role downgraded", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, "a", "application"},
		{"branch changed", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, "b", "appointment"},
		{"lookup failed", data.UserStatus{}, errors.New("private db error"), "a", "contact"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := hub(t)
			s := socket()
			_, _ = attach(t, h, s)
			reader := &liveReader{status: data.UserStatus{Found: true, Active: true, Role: "admin"}}
			lookup := func(notify.UserID) (notificationevent.User, bool) {
				reader.mu.Lock()
				defer reader.mu.Unlock()
				return notificationevent.User{Role: notify.Role(reader.status.Role), Branch: tc.branch}, true
			}
			var eventRule notify.Predicate = notificationevent.Recipient
			if tc.rule == "appointment" {
				eventRule = notificationevent.AppointmentRecipients("a", lookup)
			} else if tc.rule == "application" {
				eventRule = notificationevent.ApplicationRecipients(lookup)
			}
			predicate := notificationevent.CurrentRecipients(reader, eventRule)
			if result, err := h.Broadcast(Notifications, []byte("allowed"), predicate); err != nil || result.Enqueued != 1 {
				t.Fatal("valid subscriber not selected")
			}
			select {
			case <-s.writes:
			case <-time.After(time.Second):
				t.Fatal("valid subscriber received no message")
			}
			reader.change(tc.status, tc.err)
			if result, err := h.Broadcast(Notifications, []byte("denied"), predicate); err != nil || result.Enqueued != 0 {
				t.Fatal("changed account selected")
			}
			select {
			case <-s.writes:
				t.Fatal("changed account received notification")
			default:
			}
		})
	}
}
