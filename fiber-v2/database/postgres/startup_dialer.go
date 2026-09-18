package postgres

import (
	"context"
	"net"
	"sync"
	"time"
)

type dialerState uint8

const (
	starting dialerState = iota
	active
	failed
)

// Startup sockets belong to this guard until successful Ping activates it.
// Failed startup is terminal, including for pq's delayed cancellation dials.
type startupDialer struct {
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	state       dialerState
	failure     error
	connections []*startupConn
	stop        func() bool
	dial        func(context.Context, string, string) (net.Conn, error)
}

func newStartupDialer(ctx context.Context) *startupDialer {
	ctx, cancel := context.WithCancel(ctx)
	d := &startupDialer{ctx: ctx, cancel: cancel, state: starting, dial: (&net.Dialer{}).DialContext}
	d.stop = context.AfterFunc(ctx, func() {
		d.fail(ctx.Err())
	})
	return d
}

func (d *startupDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *startupDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return d.DialContext(ctx, network, address)
}

func (d *startupDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.mu.Lock()
	if d.state == failed {
		err := d.failure
		d.mu.Unlock()
		return nil, err
	}
	startupCtx := d.ctx
	d.mu.Unlock()
	// Bound TCP dialing in both states; socket I/O deadlines are separate.
	ctx, cancel := context.WithTimeout(ctx, StartupPingTimeout)
	defer cancel()
	if startupCtx != nil {
		if err := startupCtx.Err(); err != nil {
			return nil, poolError("dial", err)
		}
		stop := context.AfterFunc(startupCtx, cancel)
		defer stop()
	}
	conn, err := d.dial(ctx, network, address)
	if err != nil {
		return nil, poolError("dial", err)
	}
	owned := &startupConn{Conn: conn}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state == failed {
		_ = owned.Close()
		return nil, d.failure
	}
	if d.state == starting {
		if err := d.ctx.Err(); err != nil {
			_ = owned.Close()
			return nil, poolError("dial", err)
		}
		if deadline, ok := d.ctx.Deadline(); ok {
			if err := owned.SetDeadline(deadline); err != nil {
				_ = owned.Close()
				return nil, poolError("dial", err)
			}
		}
		d.connections = append(d.connections, owned)
	}
	return owned, nil
}

// activate is called only after Ping succeeds. A concurrent failure wins.
func (d *startupDialer) activate() error {
	d.stop()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state == active {
		return nil
	}
	if d.state == failed {
		return d.failure
	}
	err := startupContextError(d.ctx)
	for _, conn := range d.connections {
		if err != nil {
			break
		}
		// pq may have discarded a failed connection during Ping's retries.
		conn.mu.Lock()
		if !conn.closed {
			err = conn.Conn.SetDeadline(time.Time{})
		}
		conn.mu.Unlock()
	}
	if err == nil {
		err = startupContextError(d.ctx)
	}
	if err != nil {
		d.failLocked(err)
		return d.failure
	}
	d.state = active
	d.connections = nil
	d.ctx = nil
	d.cancel()
	return nil
}

func (d *startupDialer) fail(cause error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failLocked(cause)
}

// No reverse transitions: fail after activation cannot close runtime sockets.
// Conn.Close never takes d.mu, so holding it here cannot invert lock order.
func (d *startupDialer) failLocked(cause error) {
	if d.state != starting {
		return
	}
	d.state = failed
	d.failure = poolError("dial", cause)
	d.cancel()
	for _, conn := range d.connections {
		_ = conn.Close()
	}
	d.connections = nil
	d.ctx = nil
}

type startupConn struct {
	net.Conn
	mu     sync.Mutex
	closed bool
	err    error
}

func (c *startupConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		c.err = c.Conn.Close()
	}
	return c.err
}
