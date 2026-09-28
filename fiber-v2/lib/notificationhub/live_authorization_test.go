package notificationhub_test

import (
	"context"
	"models/notify"
	"sync/atomic"
	"testing"
)

func TestSEC003CQueuedDeliveryRechecksPredicate(t *testing.T) {
	h := newHub(t, 2)
	entered := make(chan struct{})
	release := make(chan struct{})
	sent := make(chan string, 2)
	r := register(t, h, "a", "user", func(_ context.Context, payload []byte) error {
		if string(payload) == "first" {
			close(entered)
			<-release
		}
		sent <- string(payload)
		return nil
	})
	broadcast(t, h, "a", "first", nil)
	receive(t, entered)
	var active atomic.Bool
	active.Store(true)
	predicate := func(notify.Client) bool { return active.Load() }
	broadcast(t, h, "a", "second", predicate)
	active.Store(false) // Account changed while the second payload was queued.
	close(release)
	if receive(t, sent) != "first" {
		t.Fatal("first send missing")
	}
	// A later barrier send proves the queued payload was either rejected or sent.
	broadcast(t, h, "a", "barrier", nil)
	if receive(t, sent) != "barrier" {
		t.Fatal("queued payload passed stale authorization")
	}
	r.Unregister()
	receive(t, r.Done())
}
