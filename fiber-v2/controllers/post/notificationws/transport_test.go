package notificationws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"lib/notificationhub"
	"models/notify"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type frame struct {
	kind int
	data []byte
}
type fakeSocket struct {
	closed       chan struct{}
	reads        chan frame
	writes       chan frame
	entered      chan struct{}
	block        bool
	fail         bool
	deadlineFail bool
	panics       bool
	once         sync.Once
	closeCalls   atomic.Int32
	active       atomic.Int32
	overlap      atomic.Bool
	deadline     time.Time // Single writer, observed after frame/channel synchronization.
}

func socket() *fakeSocket {
	return &fakeSocket{closed: make(chan struct{}), reads: make(chan frame, 2), writes: make(chan frame, 64), entered: make(chan struct{}, 64)}
}
func (s *fakeSocket) Close() error {
	s.closeCalls.Add(1)
	s.once.Do(func() { close(s.closed) })
	return errors.New("private socket detail")
}
func (s *fakeSocket) SetWriteDeadline(d time.Time) error {
	s.deadline = d
	if s.deadlineFail {
		return errors.New("private deadline")
	}
	return nil
}
func (s *fakeSocket) ReadMessage() (int, []byte, error) {
	select {
	case f := <-s.reads:
		return f.kind, f.data, nil
	case <-s.closed:
		return 0, nil, io.EOF
	}
}
func (s *fakeSocket) WriteMessage(kind int, data []byte) error {
	if s.active.Add(1) != 1 {
		s.overlap.Store(true)
	}
	defer s.active.Add(-1)
	s.entered <- struct{}{}
	if s.panics {
		panic("private payload socket detail")
	}
	if s.block {
		<-s.closed
		return errors.New("private blocked write")
	}
	if s.fail {
		return errors.New("private write detail")
	}
	s.writes <- frame{kind, append([]byte(nil), data...)}
	return nil
}

func wait(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not finish")
	}
}
func hub(t *testing.T) notify.Hub {
	t.Helper()
	h, err := notificationhub.New(64)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if h.Shutdown(ctx) != nil {
			t.Error("hub shutdown failed")
		}
	})
	return h
}
func attach(t *testing.T, h notify.Hub, s *fakeSocket) (*transport, notify.Registration) {
	t.Helper()
	tr := newTransport(s)
	r, err := h.Register(context.Background(), Notifications, notify.Client{ID: "connection", Metadata: notify.Metadata{UserID: "server", Protocol: recipientProtocol}}, tr.send)
	if err != nil {
		t.Fatal(err)
	}
	go tr.watch(r.Done())
	t.Cleanup(func() { r.Unregister(); tr.close(); wait(t, r.Done()); wait(t, tr.watchDone) })
	return tr, r
}

func TestTransportRawTextDeadlineSerialAndMutation(t *testing.T) {
	h := hub(t)
	s := socket()
	_, r := attach(t, h, s)
	for i := 0; i < 32; i++ {
		if _, err := h.Broadcast(Notifications, []byte(`{"message":"<safe>"}`), nil); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 32; i++ {
		select {
		case f := <-s.writes:
			if f.kind != 1 || string(f.data) != `{"message":"<safe>"}` {
				t.Fatal("text payload changed or encoded twice")
			}
		case <-time.After(time.Second):
			t.Fatal("missing frame")
		}
	}
	r.Unregister()
	wait(t, r.Done())
	if s.overlap.Load() {
		t.Fatal("writer overlap")
	}
	if s.deadline.IsZero() || time.Until(s.deadline) > writeTimeout {
		t.Fatal("unbounded write")
	}
}

func TestTransportCancellationAndWatcherExit(t *testing.T) {
	for _, mode := range []string{"context", "shutdown", "unregister", "error", "deadline", "panic", "idle shutdown"} {
		t.Run(mode, func(t *testing.T) {
			h := hub(t)
			s := socket()
			s.block = mode == "context" || mode == "shutdown" || mode == "unregister"
			s.fail = mode == "error"
			s.deadlineFail = mode == "deadline"
			s.panics = mode == "panic"
			tr := newTransport(s)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, err := h.Register(ctx, Notifications, notify.Client{ID: "id"}, tr.send)
			if err != nil {
				t.Fatal(err)
			}
			go tr.watch(r.Done())
			if mode != "idle shutdown" {
				h.Broadcast(Notifications, []byte("private payload"), nil)
				if mode != "deadline" {
					wait(t, s.entered)
				}
			}
			switch mode {
			case "context":
				cancel()
			case "unregister":
				r.Unregister()
			case "shutdown", "idle shutdown":
				end, stop := context.WithTimeout(context.Background(), time.Second)
				defer stop()
				if h.Shutdown(end) != nil {
					t.Fatal("shutdown blocked")
				}
			}
			wait(t, r.Done())
			wait(t, tr.watchDone)
			tr.close()
			tr.close()
			if s.closeCalls.Load() != 1 {
				t.Fatal("physical close repeated")
			}
			if mode == "error" || mode == "deadline" {
				if r.Err() != notify.ErrSendFailed {
					t.Fatal("wrong terminal error")
				}
			}
			for _, value := range []any{tr, r, r.Err(), errTransport} {
				for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
					if strings.Contains(fmt.Sprintf(format, value), "private") {
						t.Fatal("unsafe formatting")
					}
				}
			}
		})
	}
}

func TestTransportContextDeadlineAndConcurrentClose(t *testing.T) {
	s := socket()
	s.block = true
	tr := newTransport(s)
	regDone := make(chan struct{})
	go tr.watch(regDone)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if tr.send(ctx, []byte("private")) != errTransport {
			t.Error("unsafe error")
		}
	}()
	wait(t, s.entered)
	deadline, _ := ctx.Deadline()
	if s.deadline != deadline {
		t.Fatal("context deadline not applied")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); tr.close() }()
	}
	wg.Wait()
	cancel()
	wait(t, done)
	wait(t, tr.watchDone)
	if s.closeCalls.Load() != 1 {
		t.Fatal("close repeated")
	}
}

type observedHub struct {
	notify.Hub
	registered   chan notify.Client
	registration chan notify.Registration
	room         chan notify.RoomID
	calls        atomic.Int32
}

func (h *observedHub) Register(ctx context.Context, room notify.RoomID, c notify.Client, send notify.Send) (notify.Registration, error) {
	h.calls.Add(1)
	r, err := h.Hub.Register(ctx, room, c, send)
	if err == nil {
		h.registered <- c
		h.registration <- r
		h.room <- room
	}
	return r, err
}
func observe(h notify.Hub) *observedHub {
	return &observedHub{Hub: h, registered: make(chan notify.Client, 4), registration: make(chan notify.Registration, 4), room: make(chan notify.RoomID, 4)}
}

type badRandom struct{}

func (badRandom) Read([]byte) (int, error) { return 0, errors.New("private entropy detail") }

func TestSessionIdentityFailureAndRandomFailure(t *testing.T) {
	for _, value := range []any{nil, "client string", notify.UserID(""), 1, notify.Metadata{UserID: "wrong type"}} {
		h := observe(hub(t))
		s := socket()
		err := serve(h, s, value, Subscriber, nil, badRandom{})
		if err != errIdentity || h.calls.Load() != 0 || s.closeCalls.Load() != 1 {
			t.Fatal("invalid identity registered")
		}
	}
	h := observe(hub(t))
	s := socket()
	if serve(h, s, notify.UserID("42"), Subscriber, nil, badRandom{}) != errRandom || h.calls.Load() != 0 || s.closeCalls.Load() != 1 {
		t.Fatal("random failure cleanup")
	}
}

func TestSessionIdentityMismatchDisconnectAndProducerIsolation(t *testing.T) {
	for _, event := range []Event{Subscriber, Appointment, Application, Contact, Unknown} {
		t.Run(fmt.Sprint(event), func(t *testing.T) {
			h := observe(hub(t))
			s := socket()
			done := make(chan struct{})
			var callback atomic.Bool
			go func() {
				defer close(done)
				if serve(h, s, notify.UserID("42"), event, func(e Event, p []byte) { callback.Store(true) }, bytes.NewReader(make([]byte, 32))) != nil {
					t.Error("session failed")
				}
			}()
			var c notify.Client
			select {
			case c = <-h.registered:
			case <-time.After(time.Second):
				t.Fatal("registration missing")
			}
			r := <-h.registration
			room := <-h.room
			if c.Metadata.UserID != "42" || string(c.ID) == string(c.Metadata.UserID) || len(c.ID) != 64 || c.Metadata.Protocol != recipientProtocol || c.Metadata.Role != "" || c.Metadata.BranchID != "" {
				t.Fatal("identity conflation")
			}
			if (room == Notifications) != (event == Subscriber) {
				t.Fatal("producer routed as recipient")
			}
			// Hostile UID/role/protocol are never inputs to registration/routing.
			s.reads <- frame{1, []byte(`{"uid":"another_user","role":"admin","protocol":"anything"}`)}
			wait(t, done)
			wait(t, r.Done())
			if !callback.Load() || s.closeCalls.Load() != 1 {
				t.Fatal("handler cleanup")
			}
			result, err := h.Broadcast(room, []byte("x"), nil)
			if err != nil || result.Enqueued != 0 {
				t.Fatal("stale registration")
			}
		})
	}
}

func TestSessionDuplicateReadErrorWriterFailureAndPanic(t *testing.T) {
	for _, mode := range []string{"duplicate", "read", "writer", "panic", "shutdown"} {
		t.Run(mode, func(t *testing.T) {
			h := observe(hub(t))
			s := socket()
			s.fail = mode == "writer"
			done := make(chan struct{})
			var result error
			go func() {
				defer close(done)
				result = serve(h, s, notify.UserID("42"), Subscriber, func(Event, []byte) { panic("private panic") }, bytes.NewReader(make([]byte, 32)))
			}()
			select {
			case <-h.registered:
			case <-time.After(time.Second):
				t.Fatal("registration missing")
			}
			r := <-h.registration
			switch mode {
			case "duplicate":
				second := socket()
				if serve(h, second, notify.UserID("43"), Subscriber, nil, bytes.NewReader(make([]byte, 32))) != errRegistration || second.closeCalls.Load() != 1 {
					t.Fatal("duplicate cleanup")
				}
				r.Unregister()
			case "read":
				s.once.Do(func() { close(s.closed) }) // Peer disconnect, not adapter Close.
			case "writer":
				h.Broadcast(Notifications, []byte("private"), nil)
			case "panic":
				s.reads <- frame{1, []byte("private")}
			case "shutdown":
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if h.Shutdown(ctx) != nil {
					t.Fatal("shutdown failed")
				}
			}
			wait(t, done)
			wait(t, r.Done())
			if s.closeCalls.Load() != 1 {
				t.Fatal("close repeated")
			}
			if mode == "panic" && result != errHandler {
				t.Fatal("panic leaked")
			}
		})
	}
}

func TestConnectionIDsAndProtocolAllowlist(t *testing.T) {
	ids := map[notify.ConnectionID]bool{}
	for i := 0; i < 32; i++ {
		id, err := connectionID(strings.NewReader(strings.Repeat(string(rune(i+1)), 32)))
		if err != nil || ids[id] || strings.Contains(string(id), "_") {
			t.Fatal("identity encoding")
		}
		ids[id] = true
	}
	for _, tc := range []struct {
		input string
		want  Event
	}{{"kullanici", Subscriber}, {"randevu", Appointment}, {"is-basvurusu", Application}, {"iletisim", Contact}, {"admin", Unknown}, {"kullanici, randevu", Unknown}} {
		if eventFor(tc.input) != tc.want {
			t.Fatal("event allowlist")
		}
	}
}
