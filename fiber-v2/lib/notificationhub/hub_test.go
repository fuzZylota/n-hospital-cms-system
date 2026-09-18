package notificationhub_test

import (
	"context"
	"errors"
	"fmt"
	"lib/notificationhub"
	"models/notify"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func newHub(t *testing.T, capacity int) notify.Hub {
	t.Helper()
	h, err := notificationhub.New(capacity)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.Shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	return h
}

func register(t *testing.T, h notify.Hub, room notify.RoomID, id notify.ConnectionID, send notify.Send) notify.Registration {
	t.Helper()
	r, err := h.Register(context.Background(), room, notify.Client{ID: id}, send)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func broadcast(t *testing.T, h notify.Hub, room notify.RoomID, message string, predicate notify.Predicate) notify.BroadcastResult {
	t.Helper()
	result, err := h.Broadcast(room, []byte(message), predicate)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second): // Deadlock guard, never an ordering assertion.
		t.Fatal("barrier did not complete")
		var zero T
		return zero
	}
}

func TestRegisterBroadcastAndRoomIsolation(t *testing.T) {
	h := newHub(t, 4)
	got := make(chan string, 4)
	for _, tc := range []struct{ room, id string }{{"a", "one"}, {"a", "two"}, {"b", "three"}} {
		register(t, h, notify.RoomID(tc.room), notify.ConnectionID(tc.id), func(_ context.Context, p []byte) error {
			got <- tc.id + ":" + string(p)
			return nil
		})
	}
	if got := broadcast(t, h, "a", "first", nil); got.Enqueued != 2 {
		t.Fatalf("admission: %+v", got)
	}
	seen := map[string]bool{receive(t, got): true, receive(t, got): true}
	if !seen["one:first"] || !seen["two:first"] {
		t.Fatal("wrong room recipients")
	}
	if got := broadcast(t, h, "b", "second", nil); got.Enqueued != 1 {
		t.Fatalf("admission: %+v", got)
	}
	if receive(t, got) != "three:second" {
		t.Fatal("single client delivery failed")
	}
}

func TestTypedMetadataAndIdentityIsolation(t *testing.T) {
	h := newHub(t, 4)
	got := make(chan notify.ConnectionID, 4)
	for _, id := range []notify.ConnectionID{"user42_tab1", "opaque-session", "user42"} {
		metadata := notify.Metadata{UserID: "user42", BranchID: "branch7", Role: "santral", Protocol: "kullanici"}
		if id == "user42" {
			metadata.UserID = "different-user"
		}
		info := notify.Client{ID: id, Metadata: metadata}
		_, err := h.Register(context.Background(), "a", info, func(context.Context, []byte) error {
			got <- id
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		info.Metadata.UserID = "changed-after-register"
	}
	predicate := func(c notify.Client) bool {
		return c.Metadata.UserID == notify.UserID("user42") && c.Metadata.BranchID == notify.BranchID("branch7") &&
			c.Metadata.Role == notify.Role("santral") && c.Metadata.Protocol == notify.Protocol("kullanici")
	}
	if r := broadcast(t, h, "a", "event", predicate); r.Enqueued != 2 {
		t.Fatalf("identity selection: %+v", r)
	}
	seen := map[notify.ConnectionID]bool{receive(t, got): true, receive(t, got): true}
	if !seen["user42_tab1"] || !seen["opaque-session"] {
		t.Fatal("connection ID was treated as user ID")
	}
}

func TestFIFOAndSerialTransport(t *testing.T) {
	h := newHub(t, 64)
	var active, peak atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	got := make(chan byte, 32)
	register(t, h, "a", "one", func(ctx context.Context, p []byte) error {
		n := active.Add(1)
		defer active.Add(-1)
		if n > 1 {
			peak.Store(n)
		}
		if p[0] == 0 {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		got <- p[0]
		return nil
	})
	if _, err := h.Broadcast("a", []byte{0}, nil); err != nil {
		t.Fatal(err)
	}
	receive(t, entered)
	for i := byte(1); i < 32; i++ {
		if r, err := h.Broadcast("a", []byte{i}, nil); err != nil || r.Enqueued != 1 {
			t.Fatalf("admission: %+v %v", r, err)
		}
	}
	close(release)
	for i := byte(0); i < 32; i++ {
		if receive(t, got) != i {
			t.Fatal("FIFO broken")
		}
	}
	if peak.Load() > 1 {
		t.Fatal("concurrent transport calls")
	}
}

func TestBoundedQueueSlowClientAndIndependentDelivery(t *testing.T) {
	h := newHub(t, 1)
	entered := make(chan struct{})
	slow := register(t, h, "a", "slow", func(ctx context.Context, _ []byte) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	})
	fastMessages := make(chan string, 3)
	fast := register(t, h, "a", "fast", func(_ context.Context, p []byte) error {
		fastMessages <- string(p)
		return nil
	})
	broadcast(t, h, "a", "in-flight", nil)
	receive(t, entered)
	receive(t, fastMessages)
	if r := broadcast(t, h, "a", "queued", nil); r != (notify.BroadcastResult{Enqueued: 2}) {
		t.Fatalf("one waiting slot: %+v", r)
	}
	receive(t, fastMessages)
	if r := broadcast(t, h, "a", "overflow", nil); r != (notify.BroadcastResult{Enqueued: 1, Disconnected: 1}) {
		t.Fatalf("overflow policy: %+v", r)
	}
	if receive(t, fastMessages) != "overflow" {
		t.Fatal("slow client blocked fast client")
	}
	receive(t, slow.Done())
	if !errors.Is(slow.Err(), notify.ErrSlowClient) || fast.Err() != nil {
		t.Fatal("wrong terminal reasons")
	}
	if r := broadcast(t, h, "a", "after", nil); r.Enqueued != 1 {
		t.Fatal("slow client remained routed")
	}
	receive(t, fastMessages)
}

func TestSendFailuresAreSafeAndRemoveClient(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(fmt.Sprint(panics), func(t *testing.T) {
			h := newHub(t, 1)
			r := register(t, h, "a", "sensitive-metadata", func(context.Context, []byte) error {
				if panics {
					panic("sensitive-payload")
				}
				return errors.New("sensitive-payload")
			})
			broadcast(t, h, "a", "sensitive-payload", nil)
			receive(t, r.Done())
			want := notify.ErrSendFailed
			if panics {
				want = notify.ErrSendPanic
			}
			if !errors.Is(r.Err(), want) {
				t.Fatalf("reason: %v", r.Err())
			}
			if n := broadcast(t, h, "a", "after", nil); n != (notify.BroadcastResult{}) {
				t.Fatal("failed client remained routed")
			}
			for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
				for _, value := range []any{h, r, r.Err()} {
					if strings.Contains(fmt.Sprintf(format, value), "sensitive") {
						t.Fatal("diagnostic leaked caller data")
					}
				}
			}
		})
	}
}

func TestUnregisterDuplicateAndGenerationIsolation(t *testing.T) {
	h := newHub(t, 1)
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	r := register(t, h, "a", "same", func(ctx context.Context, _ []byte) error {
		close(entered)
		<-ctx.Done()
		close(canceled)
		<-release
		return nil
	})
	noop := func(context.Context, []byte) error { return nil }
	for _, room := range []notify.RoomID{"a", "b"} {
		if _, err := h.Register(context.Background(), room, notify.Client{ID: "same"}, noop); !errors.Is(err, notify.ErrDuplicateClient) {
			t.Fatalf("duplicate accepted: %v", err)
		}
	}
	broadcast(t, h, "a", "active", nil)
	receive(t, entered)
	r.Unregister()
	r.Unregister()
	receive(t, canceled)
	if _, err := h.Register(context.Background(), "b", notify.Client{ID: "same"}, noop); !errors.Is(err, notify.ErrDuplicateClient) {
		t.Fatal("retiring ID reused before writer finished")
	}
	if n := broadcast(t, h, "a", "removed", nil); n != (notify.BroadcastResult{}) {
		t.Fatal("empty room retained a recipient")
	}
	unblock()
	receive(t, r.Done())
	if r.Err() != nil {
		t.Fatal("explicit unregister should have nil reason")
	}
	next := register(t, h, "a", "same", noop)
	r.Unregister() // Old generation must not remove the new one.
	if n := broadcast(t, h, "a", "new", nil); n.Enqueued != 1 {
		t.Fatal("stale handle removed replacement")
	}
	next.Unregister()
	receive(t, next.Done())
}

func TestMissingRoomAndPredicatePanicAreNoOps(t *testing.T) {
	h := newHub(t, 8)
	if r := broadcast(t, h, "missing", "event", func(notify.Client) bool { panic("must not run") }); r != (notify.BroadcastResult{}) {
		t.Fatal("missing room is not a no-op")
	}
	var sends atomic.Int32
	for _, id := range []notify.ConnectionID{"a", "b", "c"} {
		register(t, h, "a", id, func(context.Context, []byte) error { sends.Add(1); return nil })
	}
	calls := 0
	r, err := h.Broadcast("a", []byte("private"), func(notify.Client) bool {
		calls++
		if calls == 2 {
			panic("private")
		}
		return true
	})
	if !errors.Is(err, notify.ErrPredicatePanic) || r != (notify.BroadcastResult{}) {
		t.Fatal("panic should abort admission atomically")
	}
	if err := h.Shutdown(context.Background()); err != nil || sends.Load() != 0 {
		t.Fatal("predicate panic delivered partial broadcast")
	}
}

func TestPredicateOutsideLockAndSnapshotGeneration(t *testing.T) {
	h := newHub(t, 2)
	r := register(t, h, "a", "one", func(context.Context, []byte) error { return nil })
	entered, release := make(chan struct{}), make(chan struct{})
	result := make(chan notify.BroadcastResult, 1)
	go func() {
		n, _ := h.Broadcast("a", []byte("old"), func(c notify.Client) bool {
			close(entered)
			<-release
			c.Metadata.UserID = "predicate-local-copy"
			return true
		})
		result <- n
	}()
	receive(t, entered)
	r.Unregister()
	receive(t, r.Done())
	register(t, h, "a", "one", func(context.Context, []byte) error { return nil })
	if n := broadcast(t, h, "a", "independent", nil); n.Enqueued != 1 {
		t.Fatal("other broadcast blocked")
	}
	close(release)
	if receive(t, result) != (notify.BroadcastResult{}) {
		t.Fatal("old snapshot delivered to a replacement generation")
	}
}

func TestShutdownActiveSendAndRepeatedShutdown(t *testing.T) {
	h := newHub(t, 2)
	entered := make(chan struct{})
	r := register(t, h, "a", "one", func(ctx context.Context, _ []byte) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	})
	broadcast(t, h, "a", "active", nil)
	receive(t, entered)
	broadcast(t, h, "a", "discard", nil)
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	receive(t, r.Done())
	if !errors.Is(r.Err(), notify.ErrClosed) {
		t.Fatal("shutdown reason lost")
	}
	if _, err := h.Register(context.Background(), "a", notify.Client{ID: "new"}, func(context.Context, []byte) error { return nil }); !errors.Is(err, notify.ErrClosed) {
		t.Fatal("register after shutdown accepted")
	}
	if _, err := h.Broadcast("missing", nil, nil); !errors.Is(err, notify.ErrClosed) {
		t.Fatal("broadcast after shutdown accepted")
	}
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestShutdownTimeoutCanBeRetried(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHub(t, 1)
		release := make(chan struct{})
		r := register(t, h, "a", "one", func(context.Context, []byte) error {
			<-release // Simulates an adapter violating cancellation until released.
			return nil
		})
		broadcast(t, h, "a", "active", nil)
		synctest.Wait()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := h.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout: %v", err)
		}
		select {
		case <-r.Done():
			t.Fatal("reported completion before send returned")
		default:
		}
		close(release)
		if err := h.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
		receive(t, r.Done())
	})
}

func TestClientContextCancellation(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(fmt.Sprint(active), func(t *testing.T) {
			h := newHub(t, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			entered := make(chan struct{})
			r, err := h.Register(ctx, "a", notify.Client{ID: "one"}, func(ctx context.Context, _ []byte) error {
				close(entered)
				<-ctx.Done()
				return ctx.Err()
			})
			if err != nil {
				t.Fatal(err)
			}
			if active {
				broadcast(t, h, "a", "active", nil)
				receive(t, entered)
			}
			cancel()
			receive(t, r.Done())
			if !errors.Is(r.Err(), context.Canceled) {
				t.Fatalf("reason: %v", r.Err())
			}
			if n := broadcast(t, h, "a", "after", nil); n.Enqueued != 0 {
				t.Fatal("canceled client routed")
			}
		})
	}
}

func TestPayloadMutationIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHub(t, 4)
		release := make(chan struct{})
		mutated := make(chan struct{})
		got := make(chan string, 2)
		for _, id := range []notify.ConnectionID{"a", "b"} {
			register(t, h, "room", id, func(_ context.Context, p []byte) error {
				if string(p) == "barrier" {
					<-release
					return nil
				}
				if id == "b" {
					<-mutated // Read only after client a changes its own copy.
				}
				got <- string(p)
				if id == "a" {
					p[0] = 'X'
					close(mutated)
				}
				return nil
			})
		}
		broadcast(t, h, "room", "barrier", nil)
		synctest.Wait()
		payload := []byte("original")
		if _, err := h.Broadcast("room", payload, nil); err != nil {
			t.Fatal(err)
		}
		copy(payload, "changed!")
		close(release)
		synctest.Wait()
		if receive(t, got) != "original" || receive(t, got) != "original" {
			t.Fatal("payload aliases input or another recipient")
		}
	})
}

func TestInvalidConfigurationAndArguments(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		if h, err := notificationhub.New(capacity); h != nil || !errors.Is(err, notify.ErrInvalidConfig) {
			t.Fatal("invalid capacity accepted")
		}
	}
	h := newHub(t, 1)
	noop := func(context.Context, []byte) error { return nil }
	for _, tc := range []struct {
		ctx  context.Context
		room notify.RoomID
		id   notify.ConnectionID
		send notify.Send
	}{{nil, "a", "one", noop}, {context.Background(), "", "one", noop}, {context.Background(), "a", "", noop}, {context.Background(), "a", "one", nil}} {
		if _, err := h.Register(tc.ctx, tc.room, notify.Client{ID: tc.id}, tc.send); !errors.Is(err, notify.ErrInvalidArgument) {
			t.Fatal("invalid registration accepted")
		}
	}
	if _, err := h.Broadcast("", nil, nil); !errors.Is(err, notify.ErrInvalidArgument) {
		t.Fatal("empty room accepted")
	}
	if err := h.Shutdown(nil); !errors.Is(err, notify.ErrInvalidArgument) {
		t.Fatal("nil shutdown context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.Register(ctx, "a", notify.Client{ID: "one"}, noop); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled registration accepted")
	}
	r := register(t, h, "a", "one", noop) // Invalid shutdown did not close hub.
	r.Unregister()
	receive(t, r.Done())
}

func TestConcurrentLifecycleAndChannelSafety(t *testing.T) {
	for shard := 0; shard < 8; shard++ {
		t.Run(fmt.Sprint(shard), func(t *testing.T) {
			t.Parallel()
			h := newHub(t, 4)
			var wg sync.WaitGroup
			start := make(chan struct{})
			for worker := 0; worker < 8; worker++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					for i := 0; i < 40; i++ {
						var active atomic.Int32
						r, err := h.Register(context.Background(), "a", notify.Client{ID: notify.ConnectionID(fmt.Sprint(worker))}, func(context.Context, []byte) error {
							if active.Add(1) != 1 {
								t.Error("overlapping send")
							}
							active.Add(-1)
							return nil
						})
						if err != nil {
							t.Error(err)
							return
						}
						_, err = h.Broadcast("a", []byte("event"), func(notify.Client) bool { return true })
						if err != nil {
							t.Error(err)
							return
						}
						r.Unregister()
						r.Unregister()
						<-r.Done()
					}
				}()
			}
			close(start)
			wg.Wait()
			if n := broadcast(t, h, "a", "empty", nil); n.Enqueued != 0 {
				t.Fatal("writers still registered")
			}
		})
	}
}

func TestConcurrentBroadcastOrderIsConsistentAcrossClients(t *testing.T) {
	h := newHub(t, 128)
	outputs := []chan string{make(chan string, 64), make(chan string, 64)}
	for i, output := range outputs {
		register(t, h, "a", notify.ConnectionID(fmt.Sprint(i)), func(_ context.Context, p []byte) error { output <- string(p); return nil })
	}
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := h.Broadcast("a", []byte(fmt.Sprint(i)), nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var sequences [2][]string
	for i, output := range outputs {
		for j := 0; j < 64; j++ {
			sequences[i] = append(sequences[i], receive(t, output))
		}
	}
	if !reflect.DeepEqual(sequences[0], sequences[1]) {
		t.Fatal("inconsistent admission ordering")
	}
}

func TestConcurrentShutdownAndAdmission(t *testing.T) {
	for shard := 0; shard < 8; shard++ {
		t.Run(fmt.Sprint(shard), func(t *testing.T) {
			t.Parallel()
			h := newHub(t, 2)
			initial := register(t, h, "a", "initial", func(ctx context.Context, _ []byte) error { <-ctx.Done(); return ctx.Err() })
			broadcast(t, h, "a", "first", nil)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for worker := 0; worker < 8; worker++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					if worker < 2 {
						if err := h.Shutdown(context.Background()); err != nil {
							t.Error(err)
						}
						return
					}
					for i := 0; i < 30; i++ {
						r, err := h.Register(context.Background(), "a", notify.Client{ID: notify.ConnectionID(fmt.Sprint(worker))}, func(context.Context, []byte) error { return nil })
						if err != nil && !errors.Is(err, notify.ErrClosed) {
							t.Error(err)
							return
						}
						if _, err := h.Broadcast("a", []byte("event"), nil); err != nil && !errors.Is(err, notify.ErrClosed) {
							t.Error(err)
						}
						if r != nil {
							r.Unregister()
							<-r.Done()
						}
					}
				}()
			}
			close(start)
			wg.Wait()
			receive(t, initial.Done())
		})
	}
}

func TestShutdownDoesNotWaitForPredicateOrHoldItsLock(t *testing.T) {
	h := newHub(t, 1)
	register(t, h, "a", "one", func(context.Context, []byte) error { return nil })
	result := make(chan error, 1)
	go func() {
		_, err := h.Broadcast("a", []byte("event"), func(notify.Client) bool {
			if err := h.Shutdown(context.Background()); err != nil {
				t.Error(err)
			}
			return true
		})
		result <- err
	}()
	if !errors.Is(receive(t, result), notify.ErrClosed) {
		t.Fatal("shutdown did not reject in-flight admission")
	}
}

func TestTransportCanUnregisterWithoutHubLock(t *testing.T) {
	h := newHub(t, 1)
	var r notify.Registration
	r = register(t, h, "a", "one", func(context.Context, []byte) error { r.Unregister(); return nil })
	broadcast(t, h, "a", "event", nil)
	receive(t, r.Done())
	if r.Err() != nil {
		t.Fatal("self-unregister failed")
	}
}

func TestContextDeadlineAndCanceledShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHub(t, 1)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := h.Register(ctx, "a", notify.Client{ID: "one"}, func(context.Context, []byte) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		<-r.Done() // Virtual clock advances when all goroutines are blocked.
		if !errors.Is(r.Err(), context.DeadlineExceeded) {
			t.Fatal("deadline reason lost")
		}
		if _, err := h.Register(ctx, "a", notify.Client{ID: "two"}, func(context.Context, []byte) error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("expired registration accepted")
		}
		// A canceled wait context still permanently closes an empty hub.
		if err := h.Shutdown(ctx); err != nil {
			t.Fatal("completed shutdown should win")
		}
		if _, err := h.Broadcast("a", nil, nil); !errors.Is(err, notify.ErrClosed) {
			t.Fatal("hub reopened")
		}
	})
}

func TestShutdownAllIdleWritersComplete(t *testing.T) {
	h := newHub(t, 1)
	var registrations []notify.Registration
	for i := 0; i < 64; i++ {
		registrations = append(registrations, register(t, h, notify.RoomID(fmt.Sprint(i%3)), notify.ConnectionID(fmt.Sprint(i)), func(context.Context, []byte) error { return nil }))
	}
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, r := range registrations {
		select {
		case <-r.Done():
		default:
			t.Fatal("successful shutdown left an unfinished writer")
		}
		if !errors.Is(r.Err(), notify.ErrClosed) {
			t.Fatal("wrong shutdown reason")
		}
	}
}
