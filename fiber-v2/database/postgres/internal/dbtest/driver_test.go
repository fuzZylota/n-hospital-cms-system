package dbtest_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"database/postgres/internal/dbtest"
)

func TestConnectorsHaveIndependentState(t *testing.T) {
	first, firstDB := openDB(t, dbtest.Query(dbtest.NewRows([]string{"value"}, []any{"first"})))
	second, secondDB := openDB(t, dbtest.Query(dbtest.NewRows([]string{"value"}, []any{"second"})))

	var firstValue, secondValue string
	if err := firstDB.QueryRowContext(context.Background(), "SELECT first").Scan(&firstValue); err != nil {
		t.Fatal(err)
	}
	if err := secondDB.QueryRowContext(context.Background(), "SELECT second").Scan(&secondValue); err != nil {
		t.Fatal(err)
	}
	if firstValue != "first" || secondValue != "second" {
		t.Fatalf("values = %q, %q", firstValue, secondValue)
	}
	if got := first.Events()[0].SQL; got != "SELECT first" {
		t.Fatalf("first SQL = %q", got)
	}
	if got := second.Events()[0].SQL; got != "SELECT second" {
		t.Fatalf("second SQL = %q", got)
	}
}

func TestParallelConnectorsDoNotCrossRecordings(t *testing.T) {
	for _, id := range []int{1, 2, 3, 4} {
		id := id
		t.Run(fmt.Sprintf("connector-%d", id), func(t *testing.T) {
			t.Parallel()
			connector, db := openDB(t, dbtest.Exec(1))
			query := fmt.Sprintf("UPDATE synthetic_%d SET active = $1", id)
			if _, err := db.ExecContext(context.Background(), query, id); err != nil {
				t.Fatal(err)
			}
			events := connector.Events()
			if len(events) != 1 || events[0].SQL != query || events[0].Args[0].Value != int64(id) {
				t.Fatalf("unexpected events: %#v", events)
			}
		})
	}
}

func TestQueryRecordsSQLAndOrderedArguments(t *testing.T) {
	connector, db := openDB(t, dbtest.Query(dbtest.NewRows([]string{"ok"}, []any{true})))
	payload := []byte("bytes")
	rows, err := db.QueryContext(context.Background(), "SELECT $1, $2, $3", "alpha", sql.Named("enabled", true), payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X'

	event := connector.Events()[0]
	wantArgs := []dbtest.Arg{
		{Name: "", Ordinal: 1, Value: "alpha"},
		{Name: "enabled", Ordinal: 2, Value: true},
		{Name: "", Ordinal: 3, Value: []byte("bytes")},
	}
	if event.SQL != "SELECT $1, $2, $3" || !reflect.DeepEqual(event.Args, wantArgs) {
		t.Fatalf("event = %#v, want SQL and args %#v", event, wantArgs)
	}
}

func TestRowsScanMultipleTypesAndNull(t *testing.T) {
	moment := time.Date(2026, time.September, 18, 10, 30, 0, 0, time.UTC)
	rowsPlan := dbtest.NewRows(
		[]string{"name", "count", "created_at", "active", "payload", "parent_id"},
		[]any{"one", 1, moment, true, []byte("a"), nil},
		[]any{"two", int64(2), moment.Add(time.Hour), false, []byte("b"), "parent"},
	)
	_, db := openDB(t, dbtest.Query(rowsPlan))
	rows, err := db.QueryContext(context.Background(), "SELECT synthetic rows")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var got []struct {
		name     string
		count    int64
		parentID sql.NullString
	}
	for rows.Next() {
		var item struct {
			name     string
			count    int64
			parentID sql.NullString
		}
		var created time.Time
		var active bool
		var payload []byte
		if err := rows.Scan(&item.name, &item.count, &created, &active, &payload, &item.parentID); err != nil {
			t.Fatal(err)
		}
		got = append(got, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name     string
		count    int64
		parentID sql.NullString
	}{
		{name: "one", count: 1},
		{name: "two", count: 2, parentID: sql.NullString{String: "parent", Valid: true}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %#v, want %#v", got, want)
	}
	if rowsPlan.CloseCount() != 1 {
		t.Fatalf("EOF close count = %d", rowsPlan.CloseCount())
	}
}

func TestScanTypeMismatchUsesDatabaseSQLConversionError(t *testing.T) {
	_, db := openDB(t, dbtest.Query(dbtest.NewRows([]string{"count"}, []any{"not-an-integer"})))
	var count int64
	err := db.QueryRowContext(context.Background(), "SELECT count").Scan(&count)
	if err == nil || errors.Is(err, dbtest.ErrInvalidFixture) || !strings.Contains(err.Error(), "converting driver.Value type string") {
		t.Fatalf("expected database/sql conversion error, got %v", err)
	}
}

func TestRowsIterationErrorAndCloseAreVisible(t *testing.T) {
	iterationErr := errors.New("synthetic iteration failure")
	rowsPlan := dbtest.NewRows([]string{"id"}, []any{1}, []any{2}).WithErrorAfter(1, iterationErr)
	_, db := openDB(t, dbtest.Query(rowsPlan))
	rows, err := db.QueryContext(context.Background(), "SELECT ids")
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if !reflect.DeepEqual(ids, []int64{1}) {
		t.Fatalf("rows before error = %v", ids)
	}
	if !errors.Is(rows.Err(), iterationErr) {
		t.Fatalf("Rows.Err() = %v", rows.Err())
	}
	if !rowsPlan.Closed() || rowsPlan.CloseCount() != 1 {
		t.Fatalf("close count = %d", rowsPlan.CloseCount())
	}
}

func TestExplicitRowsCloseIsObserved(t *testing.T) {
	rowsPlan := dbtest.NewRows([]string{"id"}, []any{1})
	_, db := openDB(t, dbtest.Query(rowsPlan))
	rows, err := db.QueryContext(context.Background(), "SELECT id")
	if err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if !rowsPlan.Closed() {
		t.Fatal("rows close was not observed")
	}
	if err := rows.Close(); err != nil || rowsPlan.CloseCount() != 1 {
		t.Fatalf("second close = %v, count = %d", err, rowsPlan.CloseCount())
	}
}

func TestExecResultAndScriptedError(t *testing.T) {
	execErr := errors.New("synthetic exec failure")
	connector, db := openDB(t, dbtest.Exec(3), dbtest.ExecError(execErr))
	result, err := db.ExecContext(context.Background(), "UPDATE first")
	if err != nil {
		t.Fatal(err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 3 {
		t.Fatalf("RowsAffected = %d, %v", affected, err)
	}
	if _, err := db.ExecContext(context.Background(), "UPDATE second"); !errors.Is(err, execErr) {
		t.Fatalf("exec error = %v", err)
	}
	if connector.Remaining() != 0 {
		t.Fatalf("remaining steps = %d", connector.Remaining())
	}
}

func TestTransactionCommitSequence(t *testing.T) {
	connector, db := openDB(t,
		dbtest.Begin(),
		dbtest.Query(dbtest.NewRows([]string{"id"}, []any{7})),
		dbtest.Exec(1),
		dbtest.Commit(),
	)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := tx.QueryRowContext(context.Background(), "SELECT id").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(context.Background(), "UPDATE item", id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	assertKinds(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.ExecKind, dbtest.CommitKind)
}

func TestTransactionQueryErrorRollbackSequence(t *testing.T) {
	queryErr := errors.New("synthetic query failure")
	connector, db := openDB(t, dbtest.Begin(), dbtest.QueryError(queryErr), dbtest.Rollback())
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.QueryContext(context.Background(), "SELECT broken"); !errors.Is(err, queryErr) {
		t.Fatalf("query error = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	assertKinds(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
}

func TestTransactionExecErrorRollbackSequence(t *testing.T) {
	execErr := errors.New("synthetic exec failure")
	connector, db := openDB(t, dbtest.Begin(), dbtest.ExecError(execErr), dbtest.Rollback())
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(context.Background(), "UPDATE broken"); !errors.Is(err, execErr) {
		t.Fatalf("exec error = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	assertKinds(t, connector.Events(), dbtest.BeginKind, dbtest.ExecKind, dbtest.RollbackKind)
}

func TestTransactionLifecycleErrors(t *testing.T) {
	t.Run("begin", func(t *testing.T) {
		wantErr := errors.New("synthetic begin failure")
		connector, db := openDB(t, dbtest.BeginError(wantErr))
		_, err := db.BeginTx(context.Background(), nil)
		if !errors.Is(err, wantErr) {
			t.Fatalf("begin error = %v", err)
		}
		assertKinds(t, connector.Events(), dbtest.BeginKind)
	})
	t.Run("commit", func(t *testing.T) {
		wantErr := errors.New("synthetic commit failure")
		connector, db := openDB(t, dbtest.Begin(), dbtest.CommitError(wantErr))
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); !errors.Is(err, wantErr) {
			t.Fatalf("commit error = %v", err)
		}
		assertKinds(t, connector.Events(), dbtest.BeginKind, dbtest.CommitKind)
	})
	t.Run("rollback", func(t *testing.T) {
		wantErr := errors.New("synthetic rollback failure")
		connector, db := openDB(t, dbtest.Begin(), dbtest.RollbackError(wantErr))
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(); !errors.Is(err, wantErr) {
			t.Fatalf("rollback error = %v", err)
		}
		assertKinds(t, connector.Events(), dbtest.BeginKind, dbtest.RollbackKind)
	})
}

func TestContextBeforeDriverEntry(t *testing.T) {
	for _, kind := range []dbtest.Kind{dbtest.QueryKind, dbtest.ExecKind, dbtest.BeginKind} {
		for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
			t.Run(string(kind)+"/"+want.Error(), func(t *testing.T) {
				t.Parallel()
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if want == context.DeadlineExceeded {
					ctx, cancel = context.WithDeadline(context.Background(), time.Unix(1, 0))
					t.Cleanup(cancel)
				}
				step := dbtest.Begin()
				if kind == dbtest.QueryKind {
					step = dbtest.QueryUntilCanceled(nil)
				} else if kind == dbtest.ExecKind {
					step = dbtest.ExecUntilCanceled(nil)
				}
				connector, db := openDB(t, step)
				if err := callOperation(ctx, db, kind); !errors.Is(err, want) {
					t.Fatalf("error = %v, want %v", err, want)
				}
				if len(connector.Events()) != 0 || connector.Remaining() != 1 {
					t.Fatalf("pre-canceled call reached driver: %v", connector)
				}
			})
		}
	}
}

func TestContextAfterDriverEntry(t *testing.T) {
	for _, kind := range []dbtest.Kind{dbtest.QueryKind, dbtest.ExecKind} {
		for _, mode := range []string{"cancel", "deadline", "safety-timeout", "unread-signal"} {
			t.Run(string(kind)+"/"+mode, func(t *testing.T) {
				t.Parallel()
				synctest.Test(t, func(t *testing.T) {
					entered := make(chan struct{})
					step := dbtest.QueryUntilCanceled(entered)
					if kind == dbtest.ExecKind {
						step = dbtest.ExecUntilCanceled(entered)
					}
					connector, db := openDB(t, step)
					ctx, cancel := context.WithCancel(context.Background())
					t.Cleanup(cancel)
					var want error
					if mode == "deadline" {
						// synctest advances virtual time only when all goroutines
						// block, so this deadline cannot precede the entry handshake.
						ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
						t.Cleanup(cancel)
						want = context.DeadlineExceeded
					} else if mode == "cancel" {
						want = context.Canceled
					}
					done := make(chan error, 1)
					go func() { done <- callOperation(ctx, db, kind) }()
					if mode != "unread-signal" {
						select {
						case <-entered:
						case err := <-done:
							t.Fatalf("operation returned before entry signal: %v", err)
						}
						if ctx.Err() != nil || connector.Events()[0].ContextErr != nil || connector.Remaining() != 0 {
							t.Fatal("unexpected state before cancellation")
						}
					}
					if mode == "cancel" {
						cancel()
					}
					err := <-done // Virtual deadline/safety timer advances here.
					if want != nil {
						if !errors.Is(err, want) {
							t.Fatalf("error = %v, want %v", err, want)
						}
					} else if err == nil || !strings.Contains(err.Error(), "cancellation step timed out") || ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("safety error = %v, ctx.Err() = %v", err, ctx.Err())
					}
					events := connector.Events()
					assertKinds(t, events, kind)
					if !errors.Is(events[0].ContextErr, want) || connector.Remaining() != 0 {
						t.Fatalf("context event = %#v, remaining = %d", events[0], connector.Remaining())
					}
				})
			})
		}
	}
}

func callOperation(ctx context.Context, db *sql.DB, kind dbtest.Kind) error {
	switch kind {
	case dbtest.QueryKind:
		rows, err := db.QueryContext(ctx, "SELECT synthetic")
		if rows != nil {
			rows.Close()
		}
		return err
	case dbtest.ExecKind:
		_, err := db.ExecContext(ctx, "UPDATE synthetic")
		return err
	default:
		tx, err := db.BeginTx(ctx, nil)
		if tx != nil {
			tx.Rollback()
		}
		return err
	}
}

func TestScriptedErrorsDoNotPopulateContextErr(t *testing.T) {
	for _, test := range []struct {
		kind dbtest.Kind
		step dbtest.Step
	}{
		{dbtest.QueryKind, dbtest.QueryError(context.DeadlineExceeded)},
		{dbtest.ExecKind, dbtest.ExecError(context.DeadlineExceeded)},
		{dbtest.BeginKind, dbtest.BeginError(context.DeadlineExceeded)},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			connector, db := openDB(t, test.step)
			if err := callOperation(context.Background(), db, test.kind); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("scripted error = %v", err)
			}
			events := connector.Events()
			assertKinds(t, events, test.kind)
			if events[0].ContextErr != nil || connector.Remaining() != 0 {
				t.Fatalf("scripted error changed context state: %#v", events[0])
			}
		})
	}
}

func TestRecorderAndErrorsDoNotImplicitlyDumpArguments(t *testing.T) {
	const testSecret = "synthetic-test-secret-do-not-print"
	connector, db := openDB(t)
	_, err := db.ExecContext(context.Background(), "UPDATE synthetic SET token = $1", testSecret)
	if err == nil {
		t.Fatal("expected exhausted-script error")
	}
	if strings.Contains(err.Error(), testSecret) || strings.Contains(connector.String(), testSecret) {
		t.Fatal("error or recorder string exposed an argument")
	}
	events := connector.Events()
	if len(events) != 1 || events[0].Args[0].Value != testSecret {
		t.Fatal("synthetic argument was not available for explicit assertion")
	}
}

func TestOpenDBUsesConnectorWithoutRegisteredDriverOrNetwork(t *testing.T) {
	connector, db := openDB(t, dbtest.Query(dbtest.NewRows([]string{"ok"}, []any{true})))
	var ok bool
	if err := db.QueryRowContext(context.Background(), "SELECT true").Scan(&ok); err != nil {
		t.Fatal(err)
	}
	if !ok || connector.Remaining() != 0 {
		t.Fatalf("ok = %v, remaining = %d", ok, connector.Remaining())
	}
	if _, err := connector.Driver().Open("network-address-is-not-accepted"); err == nil {
		t.Fatal("driver.Open unexpectedly accepted a data source name")
	}
}

func TestConcurrentEventSnapshotsAreRaceSafe(t *testing.T) {
	const operations = 20
	steps := make([]dbtest.Step, operations)
	for index := range steps {
		steps[index] = dbtest.Exec(1)
	}
	connector, db := openDB(t, steps...)
	db.SetMaxOpenConns(operations)

	var wait sync.WaitGroup
	for index := 0; index < operations; index++ {
		wait.Add(1)
		go func(id int) {
			defer wait.Done()
			if _, err := db.ExecContext(context.Background(), "UPDATE concurrent", id); err != nil {
				t.Errorf("exec %d: %v", id, err)
			}
			_ = connector.Events()
		}(index)
	}
	wait.Wait()
	if len(connector.Events()) != operations || connector.Remaining() != 0 {
		t.Fatalf("events = %d, remaining = %d", len(connector.Events()), connector.Remaining())
	}
}

func TestRowsFixtureValues(t *testing.T) {
	moment := time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name        string
		value, want any
	}{
		{"null", nil, nil}, {"string", "value", "value"}, {"bool", true, true},
		{"time", moment, moment}, {"bytes", []byte("bytes"), []byte("bytes")},
		{"nil bytes", []byte(nil), []byte(nil)}, {"empty bytes", []byte{}, []byte{}},
		{"int", int(-1), int64(-1)}, {"int8", int8(-2), int64(-2)},
		{"int16", int16(-3), int64(-3)}, {"int32", int32(-4), int64(-4)},
		{"int64 min", int64(math.MinInt64), int64(math.MinInt64)},
		{"uint", uint(1), int64(1)}, {"uint8", uint8(2), int64(2)},
		{"uint16", uint16(3), int64(3)}, {"uint32", uint32(math.MaxUint32), int64(math.MaxUint32)},
		{"uint64 max accepted", uint64(math.MaxInt64), int64(math.MaxInt64)},
		{"float32", float32(1.5), float64(1.5)}, {"float64", float64(2.5), float64(2.5)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, db := openDB(t, dbtest.Query(dbtest.NewRows([]string{"value"}, []any{test.value})))
			var got any
			if err := db.QueryRowContext(context.Background(), "SELECT value").Scan(&got); err != nil {
				t.Fatal(err)
			}
			if !driver.IsValue(got) || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("value = %#v (%T), want %#v (%T)", got, got, test.want, test.want)
			}
		})
	}
}

func TestInvalidFixturesFailBeforeScan(t *testing.T) {
	const privateValue = "synthetic-private-fixture"
	pointer := privateValue
	invalid := []struct {
		name  string
		value any
	}{
		{"uint64 overflow", uint64(math.MaxInt64) + 1},
		{"map", map[string]string{"key": privateValue}},
		{"slice", []string{privateValue}},
		{"struct", struct{ Value string }{privateValue}},
		{"pointer", &pointer}, {"nil pointer", (*string)(nil)},
		{"array", [1]string{privateValue}}, {"complex", complex(1, 2)},
		{"channel", make(chan int)}, {"function", func() {}},
	}
	if uint64(^uint(0)) > math.MaxInt64 {
		invalid = append(invalid, struct {
			name  string
			value any
		}{"uint overflow", ^uint(0)})
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			plan := dbtest.NewRows([]string{"value"}, []any{test.value})
			connector, db := openDB(t, dbtest.Query(plan))
			rows, err := db.QueryContext(context.Background(), "SELECT "+privateValue)
			if rows != nil || !errors.Is(err, dbtest.ErrInvalidFixture) {
				t.Fatalf("expected fixture error before Scan, got rows=%v, error=%v", rows, err)
			}
			if strings.Contains(err.Error(), privateValue) || strings.Contains(err.Error(), "Scan error") {
				t.Fatal("fixture error exposed data or impersonated a Scan error")
			}
			if plan.CloseCount() != 0 || connector.Remaining() != 0 {
				t.Fatal("invalid fixture opened an iterator or failed to consume the query")
			}
			assertKinds(t, connector.Events(), dbtest.QueryKind)
		})
	}
	t.Run("column count", func(t *testing.T) {
		_, db := openDB(t, dbtest.Query(dbtest.NewRows([]string{"value"}, []any{1, 2})))
		rows, err := db.QueryContext(context.Background(), "SELECT value")
		if rows != nil || !errors.Is(err, dbtest.ErrInvalidFixture) || !strings.Contains(err.Error(), "column/value count mismatch") {
			t.Fatalf("column-count fixture error = %v", err)
		}
	})
}

func TestEventSnapshotsDoNotAlias(t *testing.T) {
	payload := []byte("bytes")
	connector, db := openDB(t, dbtest.Exec(1))
	if _, err := db.ExecContext(context.Background(), "UPDATE synthetic", sql.Named("payload", payload), []byte{}, []byte(nil)); err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X'
	want := []dbtest.Event{{Kind: dbtest.ExecKind, SQL: "UPDATE synthetic", Args: []dbtest.Arg{
		{Name: "payload", Ordinal: 1, Value: []byte("bytes")},
		{Ordinal: 2, Value: []byte{}}, {Ordinal: 3, Value: []byte(nil)},
	}}}
	snapshot := connector.Events()
	if !reflect.DeepEqual(snapshot, want) {
		t.Fatalf("snapshot = %#v, want %#v", snapshot, want)
	}
	snapshot[0].Args[0].Value.([]byte)[0] = 'Y'
	snapshot[0].Args[0].Name = "changed"
	snapshot[0].Args[0].Ordinal = 99
	snapshot[0].Args[1].Value = []byte("replacement")
	snapshot[0].SQL = "changed"
	snapshot[0].Kind = dbtest.QueryKind
	snapshot[0].ContextErr = context.Canceled
	if !reflect.DeepEqual(connector.Events(), want) {
		t.Fatal("caller mutation changed recorded events")
	}
	snapshot[0] = dbtest.Event{}
	if !reflect.DeepEqual(connector.Events(), want) {
		t.Fatal("event slice aliases connector state")
	}
}

func TestRowsSnapshotsDoNotAlias(t *testing.T) {
	columns := []string{"payload"}
	payload := []byte("bytes")
	row := []any{payload}
	source := [][]any{row, row}
	plan := dbtest.NewRows(columns, source...)
	payload[0] = 'X'
	row[0] = "changed"
	source[1] = []any{"changed"}
	columns[0] = "changed"
	_, db := openDB(t, dbtest.Query(plan), dbtest.Query(plan), dbtest.Query(plan))
	db.SetMaxOpenConns(2)
	first, err := db.QueryContext(context.Background(), "SELECT first")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := db.QueryContext(context.Background(), "SELECT second")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	payload[1] = 'Y' // Also mutate source bytes after both iterators are open.
	plan.WithErrorAfter(0, errors.New("future iterator only"))
	gotColumns, err := first.Columns()
	if err != nil || !reflect.DeepEqual(gotColumns, []string{"payload"}) {
		t.Fatalf("columns = %v, error = %v", gotColumns, err)
	}
	if !first.Next() {
		t.Fatalf("first row missing: %v", first.Err())
	}
	var raw sql.RawBytes
	if err := first.Scan(&raw); err != nil || string(raw) != "bytes" {
		t.Fatalf("first payload = %q, error = %v", raw, err)
	}
	raw[0] = 'Z' // Exercise the actual driver buffer, not Scan's []byte copy.
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	plan.WithErrorAfter(-1, nil) // Previously opened iterators keep their snapshot.
	third, err := db.QueryContext(context.Background(), "SELECT third")
	if err != nil {
		t.Fatal(err)
	}
	defer third.Close()
	for _, rows := range []*sql.Rows{second, third} {
		count := 0
		for rows.Next() {
			var got []byte
			if err := rows.Scan(&got); err != nil || string(got) != "bytes" {
				t.Fatalf("independent payload = %q, error = %v", got, err)
			}
			count++
		}
		if rows.Err() != nil || count != 2 {
			t.Fatalf("independent iterator count=%d, error=%v", count, rows.Err())
		}
	}
	if plan.CloseCount() != 3 {
		t.Fatalf("shared close count = %d", plan.CloseCount())
	}
}

func TestBeginTxOptions(t *testing.T) {
	for _, opts := range []sql.TxOptions{
		{ReadOnly: true}, {Isolation: sql.LevelReadCommitted},
		{Isolation: sql.LevelSerializable}, {Isolation: sql.IsolationLevel(-1)},
	} {
		t.Run(fmt.Sprintf("readonly=%v/isolation=%d", opts.ReadOnly, opts.Isolation), func(t *testing.T) {
			t.Parallel()
			connector, db := openDB(t, dbtest.Begin(), dbtest.Commit())
			tx, err := db.BeginTx(context.Background(), &opts)
			if tx != nil || err == nil || !strings.Contains(err.Error(), "unsupported transaction option") {
				t.Fatalf("unsupported BeginTx = %v, %v", tx, err)
			}
			if connector.Remaining() != 2 || len(connector.Events()) != 0 {
				t.Fatalf("rejected options consumed/recorded a step: %v", connector)
			}
			tx, err = db.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			assertKinds(t, connector.Events(), dbtest.BeginKind, dbtest.CommitKind)
			if connector.Remaining() != 0 {
				t.Fatalf("remaining = %d", connector.Remaining())
			}
		})
	}
	t.Run("Begin defaults", func(t *testing.T) {
		connector, db := openDB(t, dbtest.Begin(), dbtest.Rollback())
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		assertKinds(t, connector.Events(), dbtest.BeginKind, dbtest.RollbackKind)
	})
}

func TestConcurrentRowsConfigurationSnapshots(t *testing.T) {
	t.Parallel()
	const operations = 40
	before, after := errors.New("before rows"), errors.New("after first row")
	plan := dbtest.NewRows([]string{"id"}, []any{1}).WithErrorAfter(0, before)
	steps := make([]dbtest.Step, operations)
	for index := range steps {
		steps[index] = dbtest.Query(plan)
	}
	connector, db := openDB(t, steps...)
	db.SetMaxOpenConns(operations)
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		<-start
		for index := 0; index < operations; index++ {
			plan.WithErrorAfter(1, after)
			plan.WithErrorAfter(0, before)
		}
	}()
	for index := 0; index < operations; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			rows, err := db.QueryContext(context.Background(), "SELECT id")
			if err != nil {
				t.Error(err)
				return
			}
			defer rows.Close()
			count := 0
			for rows.Next() {
				var id int64
				if err := rows.Scan(&id); err != nil || id != 1 {
					t.Errorf("id = %d, error = %v", id, err)
				}
				count++
			}
			if !((count == 0 && errors.Is(rows.Err(), before)) || (count == 1 && errors.Is(rows.Err(), after))) {
				t.Errorf("inconsistent configuration snapshot: count=%d, error=%v", count, rows.Err())
			}
		}()
	}
	close(start)
	wait.Wait()
	if plan.CloseCount() != operations || len(connector.Events()) != operations || connector.Remaining() != 0 {
		t.Fatalf("close count=%d, recorder=%v", plan.CloseCount(), connector)
	}
}

func TestRowsCloseErrorLifecycle(t *testing.T) {
	closeErr := errors.New("synthetic close failure")
	for _, test := range []struct {
		name     string
		closeErr error
		explicit bool
	}{
		{"EOF error", closeErr, false},
		{"explicit error", closeErr, true},
		{"EOF nil", nil, false},
		{"explicit nil", nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			plan := dbtest.NewRows([]string{"id"}, []any{1}).WithCloseError(test.closeErr)
			_, db := openDB(t, dbtest.Query(plan))
			func() {
				rows, err := db.QueryContext(context.Background(), "SELECT id")
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := rows.Close(); err != nil {
						t.Errorf("deferred repeat Close = %v", err)
					}
				}()
				if test.explicit {
					if err := rows.Close(); !errors.Is(err, test.closeErr) {
						t.Fatalf("first Close = %v, want %v", err, test.closeErr)
					}
				} else {
					count := 0
					for rows.Next() {
						var id int64
						if err := rows.Scan(&id); err != nil || id != 1 {
							t.Fatalf("id=%d error=%v", id, err)
						}
						count++
					}
					if count != 1 {
						t.Fatalf("row count = %d", count)
					}
				}
				if !errors.Is(rows.Err(), test.closeErr) {
					t.Fatalf("Rows.Err = %v, want %v", rows.Err(), test.closeErr)
				}
				if err := rows.Close(); err != nil || plan.CloseCount() != 1 {
					t.Fatalf("repeat Close = %v, count = %d", err, plan.CloseCount())
				}
				if !errors.Is(rows.Err(), test.closeErr) {
					t.Fatal("repeat Close changed Rows.Err")
				}
			}()
			if plan.CloseCount() != 1 {
				t.Fatalf("close count after defer = %d", plan.CloseCount())
			}
		})
	}
}

func TestRowsCloseErrorSnapshots(t *testing.T) {
	firstErr, secondErr := errors.New("first close"), errors.New("second close")
	plan := dbtest.NewRows([]string{"id"}, []any{1}).WithCloseError(firstErr)
	_, db := openDB(t, dbtest.Query(plan), dbtest.Query(plan), dbtest.Query(plan))
	db.SetMaxOpenConns(3)
	first, err := db.QueryContext(context.Background(), "SELECT first")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	plan.WithCloseError(secondErr)
	second, err := db.QueryContext(context.Background(), "SELECT second")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	plan.WithCloseError(nil)
	third, err := db.QueryContext(context.Background(), "SELECT third")
	if err != nil {
		t.Fatal(err)
	}
	defer third.Close()
	for _, test := range []struct {
		rows *sql.Rows
		want error
	}{
		{first, firstErr}, {second, secondErr}, {third, nil},
	} {
		if err := test.rows.Close(); !errors.Is(err, test.want) {
			t.Fatalf("snapshot Close = %v, want %v", err, test.want)
		}
	}
	if plan.CloseCount() != 3 {
		t.Fatalf("close count = %d, want 3", plan.CloseCount())
	}
}

func TestConcurrentRowsCloseErrorSnapshots(t *testing.T) {
	t.Parallel()
	const operations = 40
	firstErr, secondErr := errors.New("first close"), errors.New("second close")
	plan := dbtest.NewRows([]string{"id"}, []any{1})
	steps := make([]dbtest.Step, operations)
	for i := range steps {
		steps[i] = dbtest.Query(plan)
	}
	connector, db := openDB(t, steps...)
	db.SetMaxOpenConns(operations)
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < operations; i++ {
			plan.WithCloseError(firstErr)
			plan.WithCloseError(secondErr)
			plan.WithCloseError(nil)
		}
	}()
	for i := 0; i < operations; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			rows, err := db.QueryContext(context.Background(), "SELECT id")
			if err != nil {
				t.Error(err)
				return
			}
			closeErr := rows.Close()
			if closeErr != nil && !errors.Is(closeErr, firstErr) && !errors.Is(closeErr, secondErr) {
				t.Errorf("unexpected close error = %v", closeErr)
			}
			if !errors.Is(rows.Err(), closeErr) {
				t.Error("close snapshot changed")
			}
			if err := rows.Close(); err != nil {
				t.Errorf("repeat Close = %v", err)
			}
		}()
	}
	close(start)
	wait.Wait()
	if plan.CloseCount() != operations || len(connector.Events()) != operations || connector.Remaining() != 0 {
		t.Fatalf("close count=%d recorder=%v", plan.CloseCount(), connector)
	}
}

type closeDetailError string

func (e closeDetailError) Error() string { return string(e) }

func TestRowsCloseErrorIsNotImplicitlyFormatted(t *testing.T) {
	const detail = "synthetic-private-close-detail"
	closeErr := closeDetailError(detail)
	plan := dbtest.NewRows([]string{"id"}).WithCloseError(closeErr)
	connector, db := openDB(t, dbtest.Query(plan))
	rows, err := db.QueryContext(context.Background(), "SELECT id")
	if err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close = %v", err)
	}
	// Error injection is explicit; fixture/recorder formatting must not dump it.
	for _, value := range []any{plan, connector, connector.Events()} {
		for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
			if strings.Contains(fmt.Sprintf(format, value), detail) {
				t.Fatalf("%T format %s exposed close detail", value, format)
			}
		}
	}
}

func openDB(t *testing.T, steps ...dbtest.Step) (*dbtest.Connector, *sql.DB) {
	t.Helper()
	connector := dbtest.NewConnector(steps...)
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return connector, db
}

func assertKinds(t *testing.T, events []dbtest.Event, want ...dbtest.Kind) {
	t.Helper()
	got := make([]dbtest.Kind, len(events))
	for index, event := range events {
		got[index] = event.Kind
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds = %v, want %v", got, want)
	}
}
