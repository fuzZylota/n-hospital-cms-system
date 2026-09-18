package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"
)

// Pilot limits are per process, not evidence of production capacity.
const (
	MaxOpenConns       = 2
	MaxIdleConns       = 1
	ConnMaxLifetime    = 5 * time.Minute
	ConnMaxIdleTime    = time.Minute
	StartupPingTimeout = 5 * time.Second
)

// OpenPool returns a pinged pool owned by the composition root. Repositories
// borrow it; only the composition root closes a successfully returned pool.
func OpenPool(ctx context.Context, dsn string) (*sql.DB, error) {
	return openPostgresPool(ctx, dsn, (&net.Dialer{}).DialContext)
}

// The socket seam permits exercising pq's real startup code with net.Pipe.
func openPostgresPool(ctx context.Context, dsn string, dial func(context.Context, string, string) (net.Conn, error)) (*sql.DB, error) {
	ctx, cancel := context.WithTimeout(ctx, StartupPingTimeout)
	defer cancel()
	dialer := newStartupDialer(ctx)
	dialer.dial = dial
	db, err := openPool(ctx, dsn, func(dsn string) (driver.Connector, error) {
		connector, err := newPoolConnector(dsn)
		if err != nil {
			return nil, err
		}
		connector.Dialer(dialer)
		return connector, nil
	}, dialer.fail)
	if err != nil {
		dialer.fail(err)
		return nil, err
	}
	// Remove startup-only deadlines before handing a reusable pool to callers.
	if err := dialer.activate(); err != nil {
		_ = db.Close()
		return nil, poolError("ping", err)
	}
	return db, nil
}

// Owned connections always use a five-second connection-establishment budget,
// overriding even an explicit zero/longer DSN timeout. pq clears this deadline
// after authentication; query I/O is not subject to a global five-second limit.
// pq itself converts URI escaping; canonical keyword values retain their decoded
// semantics, including a final empty value that cannot safely precede an append.
func newPoolConnector(dsn string) (*pq.Connector, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		var err error
		dsn, err = pq.ParseURL(dsn)
		if err != nil {
			return nil, poolError("connector", nil)
		}
	}
	options, err := parsePoolKeywords(dsn)
	if err != nil {
		return nil, err
	}
	options["connect_timeout"] = strconv.Itoa(int(StartupPingTimeout / time.Second))
	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var canonical strings.Builder
	escape := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	for _, key := range keys {
		// Parsed keys contain neither whitespace nor '='. Only values use the
		// quote/escape grammar; quoting every value also preserves empty values.
		canonical.WriteString(key)
		canonical.WriteString("='")
		canonical.WriteString(escape.Replace(options[key]))
		canonical.WriteString("' ")
	}
	connector, err := pq.NewConnector(canonical.String())
	if err != nil {
		return nil, poolError("connector", nil)
	}
	return connector, nil
}

// parsePoolKeywords is confined to pq connection options, not general config.
// Like pq v1.10.9, it treats whitespace after '=' as a separator before the
// value, an end-of-input value as empty, and duplicate keys as last-value-wins.
// Backslash quotes one rune in either value form; '=' inside a value is data.
func parsePoolKeywords(dsn string) (map[string]string, error) {
	input := []rune(dsn)
	pos := 0
	skipSpace := func() {
		for pos < len(input) && unicode.IsSpace(input[pos]) {
			pos++
		}
	}
	options := make(map[string]string)
	for {
		skipSpace()
		if pos == len(input) {
			return options, nil
		}
		start := pos
		for pos < len(input) && !unicode.IsSpace(input[pos]) && input[pos] != '=' {
			pos++
		}
		key := string(input[start:pos])
		skipSpace()
		if pos == len(input) || input[pos] != '=' {
			return nil, poolError("connector", nil)
		}
		pos++
		skipSpace()
		quoted := pos < len(input) && input[pos] == '\''
		if quoted {
			pos++
		}
		var value strings.Builder
		closed := !quoted
		for pos < len(input) {
			ch := input[pos]
			if quoted && ch == '\'' {
				pos++
				closed = true
				break
			}
			if !quoted && unicode.IsSpace(ch) {
				break
			}
			pos++
			if ch == '\\' {
				if pos == len(input) {
					return nil, poolError("connector", nil)
				}
				ch = input[pos]
				pos++
			}
			value.WriteRune(ch)
		}
		if !closed {
			return nil, poolError("connector", nil)
		}
		options[key] = value.String()
	}
}

// The connector seam exercises real database/sql without a network or registry.
func openPool(ctx context.Context, dsn string, connectorFor func(string) (driver.Connector, error), onFailure ...func(error)) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, poolError("configuration", nil)
	}
	connector, err := connectorFor(dsn)
	if err != nil || connector == nil {
		return nil, poolError("connector", err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(MaxOpenConns)
	db.SetMaxIdleConns(MaxIdleConns)
	db.SetConnMaxLifetime(ConnMaxLifetime)
	db.SetConnMaxIdleTime(ConnMaxIdleTime)
	ctx, cancel := context.WithTimeout(ctx, StartupPingTimeout)
	defer cancel()
	err = db.PingContext(ctx)
	contextErr := startupContextError(ctx)
	if err != nil || contextErr != nil {
		if contextErr != nil {
			err = contextErr
		}
		for _, fail := range onFailure {
			fail(err)
		}
		_ = db.Close()
		return nil, poolError("ping", err)
	}
	return db, nil
}

func startupContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// A socket deadline can fire just before the context timer is dispatched.
	// Preserve deadline identity in either scheduling order without retaining
	// the driver's network error or its connection details.
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

// PoolError contains only a fixed stage and standard context identity.
// It deliberately has no backend cause or unwrap chain.
type PoolError struct {
	stage      string
	contextErr error
}

func poolError(stage string, cause error) error {
	e := &PoolError{stage: stage}
	if errors.Is(cause, context.Canceled) {
		e.contextErr = context.Canceled
	} else if errors.Is(cause, context.DeadlineExceeded) {
		e.contextErr = context.DeadlineExceeded
	}
	return e
}

func (e *PoolError) Error() string { return "database pool startup failed: " + e.stage }
func (e *PoolError) Is(target error) bool {
	return (target == context.Canceled || target == context.DeadlineExceeded) && e.contextErr == target
}
