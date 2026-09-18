// Package notificationhub implements a bounded, process-local notification hub.
// It depends only on the standard library and the owned notify contracts.
package notificationhub

import (
	"context"
	"models/notify"
	"sync"
)

// New creates a hub with queueCapacity waiting messages per client, in addition
// to at most one dispatched send. There is no default or unbounded mode.
// The caller must eventually Shutdown the hub or unregister all clients.
func New(queueCapacity int) (notify.Hub, error) {
	if queueCapacity <= 0 {
		return nil, notify.ErrInvalidConfig
	}
	return &hub{
		capacity: queueCapacity,
		clients:  make(map[notify.ConnectionID]*client),
		rooms:    make(map[notify.RoomID]map[*client]struct{}),
		done:     make(chan struct{}),
	}, nil
}

type hub struct {
	// mu owns registry, room membership, admission, stopped and terminal error.
	// There is no client mutex and no reverse lock ordering. Neither transport
	// nor predicates run under mu. Data channels are NEVER closed.
	mu       sync.Mutex
	capacity int
	closed   bool
	clients  map[notify.ConnectionID]*client // Includes retiring writers.
	rooms    map[notify.RoomID]map[*client]struct{}
	done     chan struct{}
}

type client struct {
	hub     *hub
	room    notify.RoomID
	info    notify.Client // Immutable copy.
	ctx     context.Context
	cancel  context.CancelFunc
	send    notify.Send
	queue   chan []byte
	done    chan struct{}
	stopped bool
	err     error
}

var _ notify.Hub = (*hub)(nil)
var _ notify.Registration = (*client)(nil)

// Keep formatting lifecycle handles from exposing recipient/payload internals.
func (*hub) String() string      { return "notificationhub.Hub" }
func (*hub) GoString() string    { return "notificationhub.Hub" }
func (*client) String() string   { return "notificationhub.Registration" }
func (*client) GoString() string { return "notificationhub.Registration" }

func (h *hub) Register(ctx context.Context, room notify.RoomID, info notify.Client, send notify.Send) (notify.Registration, error) {
	if ctx == nil || room == "" || info.ID == "" || send == nil {
		return nil, notify.ErrInvalidArgument
	}
	// Deriving/canceling a context is also kept outside the registry lock.
	clientCtx, cancel := context.WithCancel(ctx)
	h.mu.Lock()
	var err error
	switch {
	case h.closed:
		err = notify.ErrClosed
	case clientCtx.Err() != nil:
		err = clientCtx.Err()
	case h.clients[info.ID] != nil:
		err = notify.ErrDuplicateClient
	}
	if err != nil {
		h.mu.Unlock()
		cancel()
		return nil, err
	}
	c := &client{
		hub: h, room: room, info: info, ctx: clientCtx, cancel: cancel,
		send: send, queue: make(chan []byte, h.capacity), done: make(chan struct{}),
	}
	h.clients[info.ID] = c
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*client]struct{})
	}
	h.rooms[room][c] = struct{}{}
	h.mu.Unlock()
	go c.write()
	return c, nil
}

func (h *hub) Broadcast(room notify.RoomID, payload []byte, predicate notify.Predicate) (notify.BroadcastResult, error) {
	result := notify.BroadcastResult{}
	if room == "" {
		return result, notify.ErrInvalidArgument
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return result, notify.ErrClosed
	}
	snapshot := make([]*client, 0, len(h.rooms[room]))
	for c := range h.rooms[room] {
		snapshot = append(snapshot, c)
	}
	h.mu.Unlock()
	// Freeze input before any caller callback; each send also gets its own copy.
	message := append([]byte(nil), payload...)
	selected := make([]*client, 0, len(snapshot))
	messages := make([][]byte, 0, len(snapshot))
	for _, c := range snapshot {
		ok, err := matches(predicate, c.info)
		if err != nil {
			return result, err // No partial broadcast on predicate panic.
		}
		if ok {
			selected = append(selected, c)
			messages = append(messages, append([]byte(nil), message...))
		}
	}
	var stopped []*client
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return result, notify.ErrClosed
	}
	for i, c := range selected {
		if c.stopped {
			continue
		}
		if err := c.ctx.Err(); err != nil {
			h.stopLocked(c, err)
			stopped = append(stopped, c)
			continue
		}
		select {
		case c.queue <- messages[i]:
			result.Enqueued++
		default:
			h.stopLocked(c, notify.ErrSlowClient)
			stopped = append(stopped, c)
			result.Disconnected++
		}
	}
	h.mu.Unlock()
	for _, c := range stopped {
		c.cancel()
	}
	return result, nil
}

func matches(predicate notify.Predicate, info notify.Client) (ok bool, err error) {
	defer func() {
		if recover() != nil {
			ok, err = false, notify.ErrPredicatePanic
		}
	}()
	return predicate == nil || predicate(info), nil
}

func deliver(send notify.Send, ctx context.Context, payload []byte) (err error) {
	defer func() {
		if recover() != nil {
			err = notify.ErrSendPanic
		}
	}()
	if send(ctx, payload) != nil {
		return notify.ErrSendFailed
	}
	return nil
}

// stopLocked detaches routing immediately; first terminal reason wins.
// ID ownership lasts until finish, preventing overlapping writers on ID reuse.
func (h *hub) stopLocked(c *client, err error) {
	if c.stopped {
		return
	}
	c.stopped, c.err = true, err
	delete(h.rooms[c.room], c)
	if len(h.rooms[c.room]) == 0 {
		delete(h.rooms, c.room)
	}
}

func (c *client) stop(err error) {
	c.hub.mu.Lock()
	c.hub.stopLocked(c, err)
	c.hub.mu.Unlock()
	c.cancel()
}

func (c *client) Unregister()           { c.stop(nil) }
func (c *client) Done() <-chan struct{} { return c.done }
func (c *client) Err() error {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	return c.err
}

func (c *client) write() {
	defer c.finish()
	for {
		select {
		case <-c.ctx.Done():
			c.stop(c.ctx.Err())
			return
		case payload := <-c.queue:
			c.hub.mu.Lock()
			if err := c.ctx.Err(); err != nil {
				c.hub.stopLocked(c, err)
			}
			stopped := c.stopped
			c.hub.mu.Unlock()
			if stopped {
				return
			}
			// Dispatch occurs above; unregister may race with this in-flight call.
			if err := deliver(c.send, c.ctx, payload); err != nil {
				if contextErr := c.ctx.Err(); contextErr != nil {
					err = contextErr
				}
				c.stop(err)
				return
			}
		}
	}
}

func (c *client) finish() {
	c.cancel()
	h := c.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopLocked(c, notify.ErrSendFailed) // Also cleans up a callback's Goexit.
	delete(h.clients, c.info.ID)
	for len(c.queue) > 0 {
		<-c.queue // No send can enter once stopped; release pending payloads.
	}
	c.queue = nil
	c.send = nil
	close(c.done)
	if h.closed && len(h.clients) == 0 {
		close(h.done)
	}
}

func (h *hub) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return notify.ErrInvalidArgument
	}
	h.mu.Lock()
	if !h.closed {
		h.closed = true
		if len(h.clients) == 0 {
			close(h.done)
		}
	}
	stopped := make([]*client, 0, len(h.clients))
	for _, c := range h.clients {
		h.stopLocked(c, notify.ErrClosed)
		stopped = append(stopped, c)
	}
	h.mu.Unlock()
	for _, c := range stopped {
		c.cancel()
	}
	select {
	case <-h.done:
		return nil
	default:
	}
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
