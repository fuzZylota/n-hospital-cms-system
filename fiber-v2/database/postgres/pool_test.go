package postgres

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lib/pq"
)

type poolConnector struct {
	pings      atomic.Int32
	closes     atomic.Int32
	connectErr error
	ping       func(context.Context) error
}

func (c *poolConnector) Connect(context.Context) (driver.Conn, error) {
	if c.connectErr != nil {
		return nil, c.connectErr
	}
	return &poolConn{owner: c}, nil
}
func (*poolConnector) Driver() driver.Driver { panic("registered driver must not be used") }

type poolConn struct{ owner *poolConnector }

func (*poolConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unsupported") }
func (*poolConn) Begin() (driver.Tx, error)           { return nil, errors.New("unsupported") }
func (c *poolConn) Close() error                      { c.owner.closes.Add(1); return nil }
func (c *poolConn) Ping(ctx context.Context) error {
	c.owner.pings.Add(1)
	return c.owner.ping(ctx)
}

func TestPoolDefaultsAndOwnership(t *testing.T) {
	if MaxOpenConns != 2 || MaxIdleConns != 1 || ConnMaxLifetime != 5*time.Minute || ConnMaxIdleTime != time.Minute || StartupPingTimeout != 5*time.Second {
		t.Fatal("pilot defaults changed")
	}
	firstPing := true
	c := &poolConnector{ping: func(ctx context.Context) error {
		if !firstPing {
			return nil
		}
		firstPing = false
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > StartupPingTimeout || time.Until(deadline) <= 0 {
			t.Error("ping deadline missing")
		}
		return nil
	}}
	db, err := openPool(context.Background(), "synthetic-dsn", func(dsn string) (driver.Connector, error) {
		if dsn != "synthetic-dsn" {
			t.Fatal("DSN changed")
		}
		return c, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if c.pings.Load() != 1 || c.closes.Load() != 0 || db.Stats().MaxOpenConnections != 2 {
		t.Fatal("startup did not leave a limited open pool")
	}
	// Hold the first connection while acquiring the second; release both and
	// observe the configured idle limit through the public database/sql API.
	first, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	second.Close()
	if db.Stats().Idle != 1 || c.closes.Load() != 1 {
		t.Fatal("idle limit not applied")
	}
	// A repository error must not close the borrowed pool.
	_, _ = NewHeaderButtonRepository(db).ListHeaderParents(context.Background())
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatal("repository closed pool", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if c.closes.Load() != 2 {
		t.Fatalf("physical closes=%d", c.closes.Load())
	}
}

func TestPoolFailureStages(t *testing.T) {
	backend := errors.New("postgres://user:password@host/db private-backend")
	for _, tc := range []struct {
		name, dsn, stage                  string
		connectorErr, connectErr, pingErr error
	}{
		{"missing", "  ", "configuration", nil, nil, nil},
		{"connector", "synthetic", "connector", backend, nil, nil},
		{"connect", "synthetic", "ping", nil, backend, nil},
		{"ping", "synthetic", "ping", nil, nil, backend},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &poolConnector{connectErr: tc.connectErr, ping: func(context.Context) error { return tc.pingErr }}
			db, err := openPool(context.Background(), tc.dsn, func(string) (driver.Connector, error) { calls++; return c, tc.connectorErr })
			if db != nil || err == nil {
				t.Fatal("failure returned a pool or no error")
			}
			assertPoolError(t, err, tc.stage, nil)
			if tc.stage == "configuration" && calls != 0 {
				t.Fatal("missing DSN invoked connector")
			}
			wantClose := int32(0)
			if tc.name == "ping" {
				wantClose = 1
			}
			if c.closes.Load() != wantClose {
				t.Fatalf("close=%d want=%d", c.closes.Load(), wantClose)
			}
		})
	}
}

func TestPoolPingContextAndClose(t *testing.T) {
	for _, cancelEarly := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelEarly), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				c := &poolConnector{ping: func(ctx context.Context) error {
					if cancelEarly {
						cancel()
					}
					<-ctx.Done()
					return fmt.Errorf("private-backend: %w", ctx.Err())
				}}
				start := time.Now()
				db, err := openPool(ctx, "synthetic", func(string) (driver.Connector, error) { return c, nil })
				if db != nil {
					t.Fatal("failed pool returned")
				}
				want := context.DeadlineExceeded
				if cancelEarly {
					want = context.Canceled
				}
				assertPoolError(t, err, "ping", want)
				if !cancelEarly && time.Since(start) != StartupPingTimeout {
					t.Fatal("wrong timeout")
				}
				if c.closes.Load() != 1 {
					t.Fatalf("close count=%d", c.closes.Load())
				}
			})
		})
	}
}

func TestPoolLifetimeAndIdleTimeApplied(t *testing.T) {
	for _, idle := range []bool{false, true} {
		t.Run(fmt.Sprint(idle), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := &poolConnector{ping: func(context.Context) error { return nil }}
				db, err := openPool(context.Background(), "synthetic", func(string) (driver.Connector, error) { return c, nil })
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				if idle {
					time.Sleep(ConnMaxIdleTime + time.Second)
					synctest.Wait()
					if db.Stats().MaxIdleTimeClosed != 1 {
						t.Fatal("idle timeout not applied")
					}
				} else {
					conn, err := db.Conn(context.Background())
					if err != nil {
						t.Fatal(err)
					}
					time.Sleep(ConnMaxLifetime + time.Second)
					conn.Close()
					if db.Stats().MaxLifetimeClosed != 1 {
						t.Fatal("lifetime not applied")
					}
				}
			})
		})
	}
}

func TestPoolRealConnectorMalformedDSN(t *testing.T) {
	// Parse failure occurs before dialing; no network or PostgreSQL is used.
	db, err := OpenPool(context.Background(), "invalid-option-without-equals")
	if db != nil {
		t.Fatal("unexpected pool")
	}
	assertPoolError(t, err, "connector", nil)
}

func assertPoolError(t *testing.T, err error, stage string, wantContext error) {
	t.Helper()
	if err == nil || err.Error() != "database pool startup failed: "+stage {
		t.Fatal("wrong safe stage")
	}
	var typed *PoolError
	if !errors.As(err, &typed) || errors.Unwrap(err) != nil {
		t.Fatal("unsafe error structure")
	}
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, sentinel) != (wantContext == sentinel) {
			t.Fatal("context identity lost")
		}
	}
	for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
		for _, secret := range []string{"password", "postgres://", "private-backend", "synthetic", "invalid-option"} {
			if strings.Contains(fmt.Sprintf(format, err), secret) {
				t.Fatal("error leaked detail")
			}
		}
	}
}

type poolDSNCase struct{ name, dsn string }

func poolDSNBoundaryCases() []poolDSNCase {
	const base = "user=synthetic password=synthetic dbname=synthetic sslmode=disable application_name=probe "
	cases := []poolDSNCase{
		{"empty application", "application_name="},
		{"empty application spaces", "application_name=   "},
		{"empty password", "password="},
		{"empty timeout", "connect_timeout="},
		{"zero timeout empty application", "connect_timeout=0 application_name="},
		{"long timeout empty application", "connect_timeout=60 application_name="},
		{"invalid timeout empty application", "connect_timeout=invalid application_name="},
		{"password ending equals", "password=abc="},
		{"quoted empty application", "application_name=''"},
		{"quoted empty password", "password=''"},
		{"quoted application spaces", "application_name='probe with spaces'"},
		{"password apostrophe backslash", `password='probe \' \\ @:/?&='`},
		{"duplicate timeout", "connect_timeout=60 connect_timeout=0"},
		{"empty value trailing whitespace", "application_name= \r\n\t  "},
		{"normal final value", "application_name=final"},
		{"IPv4 keyword", "host=127.0.0.1 port=6543"},
		{"IPv6 keyword", "host=::1 port=6543"},
		{"equals whitespace", "application_name \t= \t'probe name'"},
		{"escaped unquoted whitespace", `password=probe\ space\\slash\'quote=`},
		{"quoted adjacent options", "application_name='probe'password='abc='"},
		{"unicode whitespace", "application_name\u2003=\u00a0'probe göz'\u2003"},
		{"duplicate non-timeout", "application_name=first application_name="},
		{"empty password trailing whitespace", "password=\n\t "},
	}
	for i := range cases {
		cases[i].dsn = base + cases[i].dsn
	}
	return append(cases,
		poolDSNCase{"URI encoded password", "postgres://synthetic:p%40ss%20%27%5C%3D%25@127.0.0.1:6543/db?sslmode=disable&application_name=uri%20app"},
		poolDSNCase{"URI existing timeout IPv6", "postgresql://synthetic:abc%3D@[::1]:6543/db?sslmode=disable&connect_timeout=60&application_name=uri%20app"},
		poolDSNCase{"URI duplicate timeout", "postgres://synthetic:abc%3D@localhost/db?sslmode=disable&connect_timeout=0&connect_timeout=60"},
	)
}

// pq's approved version has no public option accessor. Read-only reflection is
// restricted to tests; option values are never printed, even on mismatch.
func poolProbeOptions(connector *pq.Connector) map[string]string {
	options := reflect.ValueOf(connector).Elem().FieldByName("opts")
	result := make(map[string]string, options.Len())
	for _, key := range options.MapKeys() {
		result[key.String()] = options.MapIndex(key).String()
	}
	return result
}

func assertPoolDSNSemantics(t *testing.T, dsn string) {
	t.Helper()
	original, err := pq.NewConnector(dsn)
	if err != nil {
		t.Fatal("invalid synthetic DSN fixture")
	}
	connector, err := newPoolConnector(dsn)
	if err != nil {
		t.Fatal("valid DSN rejected")
	}
	before, after := poolProbeOptions(original), poolProbeOptions(connector)
	before["connect_timeout"] = "5"
	if !reflect.DeepEqual(before, after) {
		t.Fatal("connection option semantics changed")
	}
}

func TestPoolDSNEmptyBoundaryAndGrammar(t *testing.T) {
	t.Parallel()
	for _, tc := range poolDSNBoundaryCases() {
		t.Run(tc.name, func(t *testing.T) { assertPoolDSNSemantics(t, tc.dsn) })
	}
	// Cross product checks quote/escape grammar against pq's parser, including
	// values that look like option assignments and '=' inside nonempty values.
	for i, value := range []string{"", "''", "abc=", "a'b", `a\ b`, `a\\b`, `'a\'b'`, `'a\\b'`, "' a\t\nb '", "'connect_timeout=0'"} {
		for j, spacing := range []string{"=", " = ", "\t=\n", "\u2003=\u00a0"} {
			t.Run(fmt.Sprintf("grammar-%d-%d", i, j), func(t *testing.T) {
				assertPoolDSNSemantics(t, "user=synthetic password=synthetic application_name"+spacing+value)
			})
		}
	}
}

func TestPoolDSNParseErrorsAreSafe(t *testing.T) {
	t.Parallel()
	for i, dsn := range []string{
		`user=synthetic password='private-backend`,
		`user=synthetic password=private-backend\`,
		`user=synthetic password='private-backend\`,
		`user=synthetic private-backend`,
		`user=synthetic password='private-backend' garbage`,
		`user=synthetic client_encoding=private-backend`,
		`postgres://synthetic:private-backend%zz@localhost/db`,
		`postgresql://[private-backend/db`,
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, originalErr := pq.NewConnector(dsn)
			if originalErr == nil {
				t.Fatal("invalid error fixture")
			}
			connector, err := newPoolConnector(dsn)
			if connector != nil {
				t.Fatal("invalid DSN returned connector")
			}
			assertPoolError(t, err, "connector", nil)
			if errors.Is(err, originalErr) {
				t.Fatal("backend parse error retained")
			}
			var backend *pq.Error
			if errors.As(err, &backend) {
				t.Fatal("backend error exposed")
			}
		})
	}
}

func TestPoolConnectorTimeoutPolicyPreservesDSN(t *testing.T) {
	t.Parallel()
	// Synthetic credentials only. Compare pq's parsed options, not our own parser.
	for _, dsn := range []string{
		`user=synthetic password='space \' quote \\ slash @:/?&=' dbname='db name' sslmode=disable`,
		`user=synthetic password=escaped\ space connect_timeout=0`,
		`user=synthetic connect_timeout=1`,
		`user=synthetic connect_timeout=90 connect_timeout=0`,
		`user=synthetic connect_timeout=-1`,
		`user=synthetic connect_timeout=invalid`,
		`postgres://synthetic:p%40ss%3A%2F%3F%26%3D%27%5C%20word@localhost/db%20name?sslmode=disable`,
		`postgresql://synthetic:p%25ss@localhost/db?connect_timeout=0`,
		`postgres://synthetic:p%2Bss@localhost/db?connect_timeout=1`,
		`postgres://synthetic:p%23ss@localhost/db?connect_timeout=90`,
		`postgres://synthetic:p%26ss@localhost/db?connect_timeout=invalid`,
	} {
		t.Run(fmt.Sprint(len(dsn)), func(t *testing.T) {
			original, err := pq.NewConnector(dsn)
			if err != nil {
				t.Fatal("invalid test fixture")
			}
			connector, err := newPoolConnector(dsn)
			if err != nil {
				t.Fatal(err)
			}
			// The approved pq version has no public option accessor. Reflection is
			// test-only, read-only and never prints credentials or option values.
			before := reflect.ValueOf(original).Elem().FieldByName("opts")
			after := reflect.ValueOf(connector).Elem().FieldByName("opts")
			for _, key := range before.MapKeys() {
				if key.String() != "connect_timeout" && before.MapIndex(key).String() != after.MapIndex(key).String() {
					t.Fatal("non-timeout connection option changed")
				}
			}
			key := reflect.ValueOf("connect_timeout")
			if after.MapIndex(key).String() != "5" {
				t.Fatal("timeout override not applied")
			}
			wantLen := before.Len()
			if !before.MapIndex(key).IsValid() {
				wantLen++
			}
			if after.Len() != wantLen {
				t.Fatal("unexpected option added")
			}
		})
	}
	for _, dsn := range []string{
		`user=synthetic password='unterminated`,
		`user=synthetic password=trailing\`,
		`user=synthetic invalid-option`,
		`postgres://synthetic:bad%zz@localhost/db`,
		`postgresql://[invalid/db`,
	} {
		connector, err := newPoolConnector(dsn)
		if connector != nil {
			t.Fatal("malformed DSN accepted")
		}
		assertPoolError(t, err, "connector", nil)
	}
}
