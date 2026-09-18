// Package notificationws owns one physical WebSocket per hub registration.
package notificationws

import (
	"context"
	"models/notify"
	"sync"
	"time"
)

const writeTimeout = 5 * time.Second
const textMessage = 1

// Socket is implemented by the existing Fiber WebSocket connection. Close must
// interrupt I/O and be safe concurrently with the single reader and writer.
type Socket interface {
	SetWriteDeadline(time.Time) error
	WriteMessage(int, []byte) error
	ReadMessage() (int, []byte, error)
	Close() error
}

type failure uint8

const (
	errTransport failure = iota + 1
	errIdentity
	errRandom
	errRegistration
	errHandler
)

func (e failure) Error() string {
	switch e {
	case errIdentity:
		return "notification websocket: identity unavailable"
	case errRandom:
		return "notification websocket: connection identity failed"
	case errRegistration:
		return "notification websocket: registration failed"
	case errHandler:
		return "notification websocket: handler failed"
	default:
		return "notification websocket: transport failed"
	}
}

type transport struct {
	socket    Socket
	once      sync.Once
	closed    chan struct{}
	active    chan context.Context
	watchDone chan struct{}
}

func (*transport) String() string   { return "notificationws.Transport" }
func (*transport) GoString() string { return "notificationws.Transport" }

func newTransport(socket Socket) *transport {
	return &transport{socket: socket, closed: make(chan struct{}), active: make(chan context.Context), watchDone: make(chan struct{})}
}

func (t *transport) close() {
	t.once.Do(func() {
		close(t.closed)
		_ = t.socket.Close()
	})
}

// Exactly one watcher is started after successful registration. It also closes
// an idle socket when the hub writer exits. No goroutine is created per Send.
func (t *transport) watch(registrationDone <-chan struct{}) {
	defer close(t.watchDone)
	var sendDone <-chan struct{}
	for {
		select {
		case ctx := <-t.active:
			sendDone = ctx.Done()
		case <-sendDone:
			t.close()
			return
		case <-registrationDone:
			t.close()
			return
		case <-t.closed:
			return
		}
	}
}

// send is invoked serially by the hub. Only this method writes application
// frames or changes the write deadline; cancellation uses concurrent Close.
func (t *transport) send(ctx context.Context, payload []byte) error {
	if ctx == nil || ctx.Err() != nil {
		t.close()
		return errTransport
	}
	select {
	case t.active <- ctx:
	case <-ctx.Done():
		t.close()
		return errTransport
	case <-t.closed:
		return errTransport
	}
	deadline := time.Now().Add(writeTimeout)
	if end, ok := ctx.Deadline(); ok && end.Before(deadline) {
		deadline = end
	}
	if t.socket.SetWriteDeadline(deadline) != nil {
		t.close()
		return errTransport
	}
	// Already serialized JSON goes straight into a text frame, never WriteJSON.
	if t.socket.WriteMessage(textMessage, payload) != nil {
		t.close()
		return errTransport
	}
	return nil
}

var _ notify.Send = (*transport)(nil).send
