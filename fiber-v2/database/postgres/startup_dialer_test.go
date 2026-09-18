package postgres

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lib/pq"
)

func TestRealPQStartupStallIsBoundedWithoutNetwork(t *testing.T) {
	for _, sslmode := range []string{"disable", "require"} {
		t.Run(sslmode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				db, err := openPostgresPool(context.Background(), "user=synthetic password=synthetic dbname=synthetic sslmode="+sslmode, func(context.Context, string, string) (net.Conn, error) {
					client, peer := net.Pipe()
					go func() {
						defer peer.Close()
						// Consume startup bytes but never answer TLS/authentication.
						_, _ = io.Copy(io.Discard, peer)
					}()
					return client, nil
				})
				if db != nil {
					t.Fatal("stalled startup returned pool")
				}
				assertPoolError(t, err, "ping", context.DeadlineExceeded)
				if time.Since(start) != StartupPingTimeout {
					t.Fatal("pq startup exceeded its budget")
				}
				synctest.Wait()
			})
		})
	}
}

type memoryConn struct {
	mu       sync.Mutex
	closes   int
	deadline time.Time
}

func (*memoryConn) Read([]byte) (int, error)  { return 0, errors.New("unused") }
func (*memoryConn) Write([]byte) (int, error) { return 0, errors.New("unused") }
func (c *memoryConn) Close() error            { c.mu.Lock(); defer c.mu.Unlock(); c.closes++; return nil }
func (*memoryConn) LocalAddr() net.Addr       { return nil }
func (*memoryConn) RemoteAddr() net.Addr      { return nil }
func (c *memoryConn) SetDeadline(deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deadline = deadline
	return nil
}
func (c *memoryConn) SetReadDeadline(deadline time.Time) error  { return c.SetDeadline(deadline) }
func (c *memoryConn) SetWriteDeadline(deadline time.Time) error { return c.SetDeadline(deadline) }

func TestStartupSocketDeadlineCancellationAndSingleClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), StartupPingTimeout)
		defer cancel()
		d := newStartupDialer(ctx)
		raw := &memoryConn{}
		d.dial = func(context.Context, string, string) (net.Conn, error) { return raw, nil }
		conn, err := d.DialContext(ctx, "unused", "unused")
		if err != nil {
			t.Fatal(err)
		}
		deadline, _ := ctx.Deadline()
		if !raw.deadline.Equal(deadline) {
			t.Fatal("handshake deadline missing")
		}
		time.Sleep(StartupPingTimeout)
		synctest.Wait()
		if err := d.activate(); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("deadline identity lost")
		}
		conn.Close()
		if raw.closes != 1 {
			t.Fatalf("socket closed %d times", raw.closes)
		}
	})
}

func TestStartupSocketSuccessClearsDeadlineAndRetainsPoolDialer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), StartupPingTimeout)
		d := newStartupDialer(ctx)
		raw := &memoryConn{}
		d.dial = func(context.Context, string, string) (net.Conn, error) { return raw, nil }
		conn, err := d.DialContext(ctx, "unused", "unused")
		if err != nil {
			t.Fatal(err)
		}
		if err := d.activate(); err != nil {
			t.Fatal(err)
		}
		cancel()
		synctest.Wait()
		if !raw.deadline.IsZero() || raw.closes != 0 {
			t.Fatal("startup limit leaked into successful pool")
		}
		conn.Close()
		next := &memoryConn{}
		d.dial = func(context.Context, string, string) (net.Conn, error) { return next, nil }
		conn, err = d.DialContext(context.Background(), "unused", "unused")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if !next.deadline.IsZero() {
			t.Fatal("subsequent connection inherited startup deadline")
		}
	})
}

func TestStartupDialerCancellationBeforeDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := newStartupDialer(ctx)
	defer d.fail(context.Canceled)
	d.dial = func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("dial after cancellation")
		return nil, nil
	}
	if _, err := d.DialContext(ctx, "unused", "unused"); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost")
	}
}

func TestStartupTerminalRejectsLateCancelAndInflightDial(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := newStartupDialer(ctx)
	defer d.fail(nil)
	primary, late := &memoryConn{}, &memoryConn{}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var calls atomic.Int32
	d.dial = func(context.Context, string, string) (net.Conn, error) {
		if calls.Add(1) == 1 {
			return primary, nil
		}
		close(entered)
		<-release // Deliberately model a dial returning after cancellation.
		return late, nil
	}
	conn, err := d.DialContext(ctx, "unused", "unused")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, err := d.DialContext(context.Background(), "unused", "unused")
		done <- err
	}()
	<-entered
	cancel()
	d.fail(context.Canceled)
	d.fail(errors.New("private-backend password"))
	// Equivalent to pq's separately scheduled cancel dial after Ping failed.
	_, err = d.DialContext(context.Background(), "unused", "unused")
	assertPoolError(t, err, "dial", context.Canceled)
	if calls.Load() != 2 {
		t.Fatal("terminal dial reached underlying network")
	}
	close(release)
	assertPoolError(t, <-done, "dial", context.Canceled)
	conn.Close()
	if primary.closes != 1 || late.closes != 1 {
		t.Fatal("socket leak or double close")
	}
	if err := d.activate(); err == nil {
		t.Fatal("failed dialer activated")
	}
}

func TestActiveDialerIgnoresFailureAndBoundsTCP(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		d := newStartupDialer(context.Background())
		raw := &memoryConn{}
		d.dial = func(context.Context, string, string) (net.Conn, error) { return raw, nil }
		conn, err := d.DialContext(context.Background(), "unused", "unused")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if err := d.activate(); err != nil {
			t.Fatal(err)
		}
		d.fail(context.Canceled)
		d.fail(nil)
		synctest.Wait()
		if raw.closes != 0 || !raw.deadline.IsZero() {
			t.Fatal("active socket altered")
		}
		d.dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		start := time.Now()
		_, err = d.DialContext(context.Background(), "unused", "unused")
		assertPoolError(t, err, "dial", context.DeadlineExceeded)
		if time.Since(start) != StartupPingTimeout {
			t.Fatal("unbounded runtime TCP dial")
		}
	})
}

type deadlineFailureConn struct{ memoryConn }

func (c *deadlineFailureConn) SetDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		return errors.New("private-backend password")
	}
	return c.memoryConn.SetDeadline(deadline)
}

func TestStartupActivationFailureIsTerminal(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), StartupPingTimeout)
	defer cancel()
	d := newStartupDialer(ctx)
	raw := &deadlineFailureConn{}
	calls := 0
	d.dial = func(context.Context, string, string) (net.Conn, error) { calls++; return raw, nil }
	conn, err := d.DialContext(ctx, "unused", "unused")
	if err != nil {
		t.Fatal(err)
	}
	assertPoolError(t, d.activate(), "dial", nil)
	d.fail(nil)
	conn.Close()
	_, err = d.DialContext(context.Background(), "unused", "unused")
	assertPoolError(t, err, "dial", nil)
	if calls != 1 || raw.closes != 1 {
		t.Fatal("failed activation leaked or reopened socket")
	}
}

// This is a single-purpose, plaintext PostgreSQL peer: startup, Ping and SELECT 1.
// It is not a TLS server, authentication security test, or database emulator.
func servePoolProbe(peer net.Conn) error {
	return servePoolProbeWithAuth(peer, nil)
}

type poolProbeAuth struct {
	application    string
	hasApplication bool
	password       string
}

func servePoolProbeWithAuth(peer net.Conn, expected *poolProbeAuth) error {
	defer peer.Close()
	readBody := func() ([]byte, error) {
		var size uint32
		if err := binary.Read(peer, binary.BigEndian, &size); err != nil {
			return nil, err
		}
		if size < 4 || size > 4096 {
			return nil, errors.New("invalid probe frame")
		}
		body := make([]byte, size-4)
		_, err := io.ReadFull(peer, body)
		return body, err
	}
	write := func(kind byte, body []byte) error {
		var packet bytes.Buffer
		packet.WriteByte(kind)
		_ = binary.Write(&packet, binary.BigEndian, uint32(len(body)+4))
		packet.Write(body)
		_, err := peer.Write(packet.Bytes())
		return err
	}
	startup, err := readBody()
	if err != nil {
		return err
	}
	if len(startup) < 4 || binary.BigEndian.Uint32(startup) != 196608 {
		return errors.New("expected PostgreSQL startup")
	}
	if expected != nil {
		parameters := make(map[string]string)
		parts := bytes.Split(startup[4:], []byte{0})
		for i := 0; i < len(parts) && len(parts[i]) != 0; i += 2 {
			if i+1 >= len(parts) {
				return errors.New("invalid probe startup parameters")
			}
			parameters[string(parts[i])] = string(parts[i+1])
		}
		application, present := parameters["application_name"]
		if present != expected.hasApplication || application != expected.application {
			return errors.New("application parameter changed")
		}
		// Real pq must answer AuthenticationCleartextPassword with the original
		// synthetic bytes, including an empty value or a final '=' character.
		if err := write('R', []byte{0, 0, 0, 3}); err != nil {
			return err
		}
		var kind [1]byte
		if _, err := io.ReadFull(peer, kind[:]); err != nil {
			return err
		}
		password, err := readBody()
		if err != nil {
			return err
		}
		if kind[0] != 'p' || !bytes.Equal(password, []byte(expected.password+"\x00")) {
			return errors.New("password message changed")
		}
	}
	if err := write('R', []byte{0, 0, 0, 0}); err != nil {
		return err
	} // AuthenticationOk
	if err := write('K', []byte{0, 0, 0, 1, 0, 0, 0, 2}); err != nil {
		return err
	}
	if err := write('Z', []byte{'I'}); err != nil {
		return err
	}
	for {
		var kind [1]byte
		if _, err := io.ReadFull(peer, kind[:]); err != nil {
			return err
		}
		body, err := readBody()
		if err != nil {
			return err
		}
		if kind[0] == 'X' {
			return nil
		}
		if kind[0] != 'Q' {
			return errors.New("unexpected probe message")
		}
		switch string(body) {
		case ";\x00":
			if err := write('I', nil); err != nil {
				return err
			}
		case "SELECT 1\x00":
			var fields bytes.Buffer
			_ = binary.Write(&fields, binary.BigEndian, uint16(1))
			fields.WriteString("value\x00")
			for _, value := range []any{uint32(0), uint16(0), uint32(23), uint16(4), int32(-1), uint16(0)} {
				_ = binary.Write(&fields, binary.BigEndian, value)
			}
			if err := write('T', fields.Bytes()); err != nil {
				return err
			}
			if err := write('D', []byte{0, 1, 0, 0, 0, 1, '1'}); err != nil {
				return err
			}
			if err := write('C', []byte("SELECT 1\x00")); err != nil {
				return err
			}
		default:
			return errors.New("unexpected probe query")
		}
		if err := write('Z', []byte{'I'}); err != nil {
			return err
		}
	}
}

func TestRealPQQueryAfterStartupDeadline(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		done := make(chan error, 1)
		db, err := openPostgresPool(context.Background(), "user=synthetic password=synthetic dbname=synthetic sslmode=disable application_name=", func(context.Context, string, string) (net.Conn, error) {
			calls.Add(1)
			client, peer := net.Pipe()
			go func() { done <- servePoolProbe(peer) }()
			return client, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		time.Sleep(StartupPingTimeout + time.Second)
		synctest.Wait()
		var value int
		if err := db.QueryRowContext(context.Background(), "SELECT 1").Scan(&value); err != nil {
			t.Fatal(err)
		}
		if value != 1 || calls.Load() != 1 {
			t.Fatal("normal query did not reuse startup connection")
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
	})
}

func TestRealPQActiveConnectionStartupTimeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		done := make(chan error, 2)
		db, err := openPostgresPool(context.Background(), "user=synthetic password=synthetic dbname=synthetic sslmode=disable connect_timeout=0 application_name=", func(context.Context, string, string) (net.Conn, error) {
			client, peer := net.Pipe()
			if calls.Add(1) == 1 {
				go func() { done <- servePoolProbe(peer) }()
			} else {
				go func() { defer peer.Close(); _, err := io.Copy(io.Discard, peer); done <- err }()
			}
			return client, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		first, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer first.Close()
		start := time.Now()
		second, err := db.Conn(context.Background())
		if second != nil {
			second.Close()
			t.Fatal("stalled auth connected")
		}
		if err == nil || time.Since(start) != StartupPingTimeout {
			t.Fatal("runtime authentication not bounded")
		}
		first.Close()
		db.Close()
		for range 2 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
		if calls.Load() != 2 {
			t.Fatal("unexpected dial count")
		}
		synctest.Wait()
	})
}

func TestRealPQDSNStartupAndPasswordSemantics(t *testing.T) {
	t.Parallel()
	for _, tc := range poolDSNBoundaryCases() {
		t.Run(tc.name, func(t *testing.T) {
			original, err := pq.NewConnector(tc.dsn)
			if err != nil {
				t.Fatal("invalid synthetic fixture")
			}
			options := poolProbeOptions(original)
			application, present := options["application_name"]
			password, hasPassword := options["password"]
			// An explicit synthetic password (even empty) makes pq skip .pgpass.
			if !hasPassword {
				t.Fatal("fixture must not consult credential files")
			}
			expected := &poolProbeAuth{application: application, hasApplication: present, password: password}
			synctest.Test(t, func(t *testing.T) {
				var calls atomic.Int32
				done := make(chan error, 1)
				db, err := openPostgresPool(context.Background(), tc.dsn, func(context.Context, string, string) (net.Conn, error) {
					if calls.Add(1) != 1 {
						return nil, errors.New("unexpected probe redial")
					}
					client, peer := net.Pipe()
					go func() { done <- servePoolProbeWithAuth(peer, expected) }()
					return client, nil
				})
				if err != nil {
					t.Fatal("pq startup/authentication failed")
				}
				defer db.Close()
				time.Sleep(StartupPingTimeout + time.Second)
				var result int
				if err := db.QueryRowContext(context.Background(), "SELECT 1").Scan(&result); err != nil {
					t.Fatal("query after old deadline failed")
				}
				if result != 1 || calls.Load() != 1 {
					t.Fatal("same connection query failed")
				}
				if err := db.Close(); err != nil {
					t.Fatal("probe connection cleanup failed")
				}
				if err := <-done; err != nil {
					t.Fatal("startup parameter/password verification failed")
				}
				synctest.Wait()
			})
		})
	}
}
