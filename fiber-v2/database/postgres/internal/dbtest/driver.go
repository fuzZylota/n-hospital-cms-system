package dbtest

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Kind identifies a recorded database operation.
type Kind string

const (
	QueryKind    Kind = "query"
	ExecKind     Kind = "exec"
	BeginKind    Kind = "begin"
	CommitKind   Kind = "commit"
	RollbackKind Kind = "rollback"
)

// Arg is a copied database/sql argument in call order.
type Arg struct {
	Name    string
	Ordinal int
	Value   any
}

// Event records an attempted scripted operation, including failed operations.
// Unsupported BeginTx options are rejected before recording or consuming a step.
// ContextErr is only the observed ctx.Err(): initially at entry, refreshed on
// return. Script/helper errors never populate it. Commit/Rollback have no context.
type Event struct {
	Kind       Kind
	SQL        string
	Args       []Arg
	ContextErr error
}

// Step is one expected driver operation. Values are created by the step
// constructors in this package.
type Step interface {
	stepKind() Kind
}

type scriptedStep struct {
	kind           Kind
	rows           *Rows
	rowsAffected   int64
	err            error
	waitForContext bool
	entered        chan<- struct{}
}

func (s scriptedStep) stepKind() Kind { return s.kind }

// Query returns a successful query step backed by rows.
func Query(rows *Rows) Step {
	return scriptedStep{kind: QueryKind, rows: rows}
}

// QueryError returns a query step that fails with err.
func QueryError(err error) Step {
	return scriptedStep{kind: QueryKind, err: err}
}

// QueryUntilCanceled waits for the query context to finish, with a bounded
// safety timeout, and then returns the context error. If entered is non-nil,
// the step sends one signal after recording/consumption, before waiting. Use a
// local channel per step; signaling is also bounded by context/safety timeout.
func QueryUntilCanceled(entered chan<- struct{}) Step {
	return scriptedStep{kind: QueryKind, waitForContext: true, entered: entered}
}

// Exec returns a successful exec step with the given affected-row count.
func Exec(rowsAffected int64) Step {
	return scriptedStep{kind: ExecKind, rowsAffected: rowsAffected}
}

// ExecError returns an exec step that fails with err.
func ExecError(err error) Step {
	return scriptedStep{kind: ExecKind, err: err}
}

// ExecUntilCanceled waits for the exec context to finish, with a bounded
// safety timeout, and then returns the context error. The optional entered
// channel follows QueryUntilCanceled's entry-signal contract.
func ExecUntilCanceled(entered chan<- struct{}) Step {
	return scriptedStep{kind: ExecKind, waitForContext: true, entered: entered}
}

// Begin returns a successful transaction-begin step.
func Begin() Step { return scriptedStep{kind: BeginKind} }

// BeginError returns a transaction-begin step that fails with err.
func BeginError(err error) Step { return scriptedStep{kind: BeginKind, err: err} }

// Commit returns a successful transaction-commit step.
func Commit() Step { return scriptedStep{kind: CommitKind} }

// CommitError returns a transaction-commit step that fails with err.
func CommitError(err error) Step { return scriptedStep{kind: CommitKind, err: err} }

// Rollback returns a successful transaction-rollback step.
func Rollback() Step { return scriptedStep{kind: RollbackKind} }

// RollbackError returns a transaction-rollback step that fails with err.
func RollbackError(err error) Step { return scriptedStep{kind: RollbackKind, err: err} }

// Connector is an isolated database/sql connector, script, and recorder.
// Pass it to sql.OpenDB; it never registers a global driver.
type Connector struct {
	mu     sync.Mutex
	steps  []scriptedStep
	events []Event
}

// NewConnector creates a connector with steps consumed in call order.
func NewConnector(steps ...Step) *Connector {
	connector := &Connector{steps: make([]scriptedStep, len(steps))}
	for index, step := range steps {
		connector.steps[index] = step.(scriptedStep)
	}
	return connector
}

// Connect creates an in-memory scripted connection.
func (c *Connector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &conn{connector: c}, nil
}

// Driver implements driver.Connector. Its Open method deliberately cannot be
// used; sql.OpenDB calls Connect directly.
func (c *Connector) Driver() driver.Driver { return connectorDriver{} }

// Events returns a deep copy of recorded calls in invocation order.
func (c *Connector) Events() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	events := make([]Event, len(c.events))
	for index, event := range c.events {
		events[index] = cloneEvent(event)
	}
	return events
}

// Remaining reports the number of unconsumed scripted operations.
func (c *Connector) Remaining() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.steps)
}

// String intentionally reports only counts, never SQL text or arguments.
func (c *Connector) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf("dbtest connector: %d event(s), %d step(s) remaining", len(c.events), len(c.steps))
}

func (c *Connector) next(kind Kind, event Event) (scriptedStep, int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	eventIndex := len(c.events)
	c.events = append(c.events, cloneEvent(event))
	if len(c.steps) == 0 {
		return scriptedStep{}, eventIndex, errors.New("dbtest: unexpected operation; script is exhausted")
	}
	step := c.steps[0]
	if step.kind != kind {
		return scriptedStep{}, eventIndex, fmt.Errorf("dbtest: unexpected %s operation; next step is %s", kind, step.kind)
	}
	c.steps = c.steps[1:]
	return step, eventIndex, nil
}

func (c *Connector) finishContext(eventIndex int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if eventIndex >= 0 && eventIndex < len(c.events) {
		c.events[eventIndex].ContextErr = err
	}
}

func cloneEvent(event Event) Event {
	cloned := event
	cloned.Args = make([]Arg, len(event.Args))
	for index, arg := range event.Args {
		cloned.Args[index] = arg
		if value, ok := arg.Value.([]byte); ok {
			cloned.Args[index].Value = bytes.Clone(value)
		}
	}
	return cloned
}

type connectorDriver struct{}

func (connectorDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("dbtest: use sql.OpenDB with the connector")
}

type conn struct {
	connector *Connector
	closed    bool
	mu        sync.Mutex
}

var (
	_ driver.Conn           = (*conn)(nil)
	_ driver.QueryerContext = (*conn)(nil)
	_ driver.ExecerContext  = (*conn)(nil)
	_ driver.ConnBeginTx    = (*conn)(nil)
)

func (c *conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("dbtest: prepared statements are not supported")
}

func (c *conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if err := c.connectionError(); err != nil {
		return nil, err
	}
	if opts.ReadOnly || opts.Isolation != driver.IsolationLevel(0) {
		return nil, errors.New("dbtest: unsupported transaction option")
	}
	event := Event{Kind: BeginKind, ContextErr: ctx.Err()}
	step, eventIndex, err := c.connector.next(BeginKind, event)
	defer func() { c.connector.finishContext(eventIndex, ctx.Err()) }()
	if err != nil {
		return nil, err
	}
	if step.err != nil {
		return nil, step.err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &tx{connector: c.connector}, nil
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := c.connectionError(); err != nil {
		return nil, err
	}
	event := Event{Kind: QueryKind, SQL: query, Args: copyArgs(args), ContextErr: ctx.Err()}
	step, eventIndex, err := c.connector.next(QueryKind, event)
	defer func() { c.connector.finishContext(eventIndex, ctx.Err()) }()
	if err != nil {
		return nil, err
	}
	if step.waitForContext {
		return nil, waitForContext(ctx, step.entered)
	}
	if step.err != nil {
		return nil, step.err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if step.rows == nil {
		return nil, fmt.Errorf("%w: query step has no rows", ErrInvalidFixture)
	}
	return step.rows.open()
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := c.connectionError(); err != nil {
		return nil, err
	}
	event := Event{Kind: ExecKind, SQL: query, Args: copyArgs(args), ContextErr: ctx.Err()}
	step, eventIndex, err := c.connector.next(ExecKind, event)
	defer func() { c.connector.finishContext(eventIndex, ctx.Err()) }()
	if err != nil {
		return nil, err
	}
	if step.waitForContext {
		return nil, waitForContext(ctx, step.entered)
	}
	if step.err != nil {
		return nil, step.err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return driver.RowsAffected(step.rowsAffected), nil
}

func (c *conn) connectionError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("dbtest: connection is closed")
	}
	return nil
}

type tx struct {
	connector *Connector
}

func (t *tx) Commit() error {
	return t.finish(CommitKind)
}

func (t *tx) Rollback() error {
	return t.finish(RollbackKind)
}

func (t *tx) finish(kind Kind) error {
	step, _, err := t.connector.next(kind, Event{Kind: kind})
	if err != nil {
		return err
	}
	return step.err
}

func copyArgs(values []driver.NamedValue) []Arg {
	args := make([]Arg, len(values))
	for index, value := range values {
		copiedValue := value.Value
		if payload, ok := copiedValue.([]byte); ok {
			copiedValue = bytes.Clone(payload)
		}
		args[index] = Arg{Name: value.Name, Ordinal: value.Ordinal, Value: copiedValue}
	}
	return args
}

func waitForContext(ctx context.Context, entered chan<- struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ctx.Done() == nil {
		return errors.New("dbtest: cancellation step requires a cancellable context")
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	if entered != nil {
		select {
		case entered <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return errors.New("dbtest: cancellation step timed out")
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return errors.New("dbtest: cancellation step timed out")
	}
}
