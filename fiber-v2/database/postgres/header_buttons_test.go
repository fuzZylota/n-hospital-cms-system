package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"database/postgres/internal/dbtest"
	"models/data"
)

func TestHeaderButtonRepositoryQueryAndMapping(t *testing.T) {
	rowsPlan := dbtest.NewRows(
		[]string{"hbid", "title", "parent_id"},
		[]any{int64(7), "Ana Sayfa", nil},
		[]any{int64(12), "Kurumsal", int64(3)},
	)
	connector, repository := openHeaderButtonRepository(t, dbtest.Query(rowsPlan))

	got, err := repository.ListHeaderParents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	parentID := "3"
	want := []data.HeaderParent{
		{ID: "7", Title: "Ana Sayfa", ParentID: nil},
		{ID: "12", Title: "Kurumsal", ParentID: &parentID},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parents = %#v, want %#v", got, want)
	}
	assertHeaderButtonQuery(t, connector)
	if rowsPlan.CloseCount() != 1 {
		t.Fatalf("rows close count = %d, want 1", rowsPlan.CloseCount())
	}
}

func TestHeaderButtonRepositoryEmptyResultIsNonNil(t *testing.T) {
	rowsPlan := dbtest.NewRows([]string{"hbid", "title", "parent_id"})
	connector, repository := openHeaderButtonRepository(t, dbtest.Query(rowsPlan))

	got, err := repository.ListHeaderParents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("empty result = %#v, want non-nil empty slice", got)
	}
	assertHeaderButtonQuery(t, connector)
	if rowsPlan.CloseCount() != 1 {
		t.Fatalf("rows close count = %d, want 1", rowsPlan.CloseCount())
	}
}

func TestHeaderButtonRepositoryKeepsUnorderedParentRows(t *testing.T) {
	rowsPlan := dbtest.NewRows(
		[]string{"hbid", "title", "parent_id"},
		[]any{int64(12), "Later", nil},
		[]any{int64(7), "Earlier", nil},
	)
	connector, repository := openHeaderButtonRepository(t, dbtest.Query(rowsPlan))
	got, err := repository.ListHeaderParents(context.Background())
	if err != nil || !reflect.DeepEqual(got, []data.HeaderParent{{ID: "12", Title: "Later"}, {ID: "7", Title: "Earlier"}}) {
		t.Fatal("repository changed the driver's row order")
	}
	// The exact SQL contract has neither an active-only predicate nor ORDER BY.
	assertHeaderButtonQuery(t, connector)
}

func TestHeaderButtonRepositoryQueryErrorIsSafe(t *testing.T) {
	for _, test := range []struct {
		name        string
		cause       error
		wantContext error
	}{
		{"backend sentinel", errors.New("synthetic backend detail: password=secret SELECT hbid"), nil},
		{"backend type", &headerBackendError{}, nil},
		{"canceled", context.Canceled, context.Canceled},
		{"deadline", context.DeadlineExceeded, context.DeadlineExceeded},
		{"wrapped canceled", &headerBackendError{cause: context.Canceled}, context.Canceled},
		{"wrapped deadline", &headerBackendError{cause: context.DeadlineExceeded}, context.DeadlineExceeded},
		{"text is not cancellation", errors.New(context.Canceled.Error()), nil},
		{"text is not deadline", errors.New(context.DeadlineExceeded.Error()), nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			connector, repository := openHeaderButtonRepository(t, dbtest.QueryError(test.cause))
			got, err := repository.ListHeaderParents(context.Background())
			if got != nil || err == nil {
				t.Fatalf("result = %#v, error = %v", got, err)
			}
			assertSafeHeaderButtonError(t, err, "query")
			assertHeaderContext(t, err, test.wantContext)
			if test.cause != context.Canceled && test.cause != context.DeadlineExceeded && errors.Is(err, test.cause) {
				t.Fatal("repository exposed backend error identity")
			}
			assertHeaderButtonQuery(t, connector)
		})
	}
}

func TestHeaderButtonRepositoryScanErrorReturnsNilAndClosesRows(t *testing.T) {
	rowsPlan := dbtest.NewRows(
		[]string{"hbid", "title", "parent_id"},
		[]any{int64(1), "Successful first row", nil},
		[]any{"not-numeric", "Ana Sayfa", nil},
	)
	connector, repository := openHeaderButtonRepository(t, dbtest.Query(rowsPlan))

	got, err := repository.ListHeaderParents(context.Background())
	if got != nil || err == nil {
		t.Fatalf("result = %#v, error = %v", got, err)
	}
	assertSafeHeaderButtonError(t, err, "scan")
	assertHeaderButtonQuery(t, connector)
	if rowsPlan.CloseCount() != 1 {
		t.Fatalf("rows close count = %d, want 1", rowsPlan.CloseCount())
	}
}

func TestHeaderButtonRepositoryIterationErrorIsSafeAndClosesRows(t *testing.T) {
	iterationErr := &headerBackendError{}
	rowsPlan := dbtest.NewRows(
		[]string{"hbid", "title", "parent_id"},
		[]any{int64(1), "Ana Sayfa", nil},
	).WithErrorAfter(1, iterationErr)
	connector, repository := openHeaderButtonRepository(t, dbtest.Query(rowsPlan))

	got, err := repository.ListHeaderParents(context.Background())
	if got != nil || err == nil || errors.Is(err, iterationErr) {
		t.Fatalf("result = %#v, error = %v", got, err)
	}
	assertSafeHeaderButtonError(t, err, "rows/close")
	assertHeaderContext(t, err, nil)
	assertHeaderButtonQuery(t, connector)
	if rowsPlan.CloseCount() != 1 {
		t.Fatalf("rows close count = %d, want 1", rowsPlan.CloseCount())
	}
}

func TestHeaderButtonRepositoryPreCanceledContext(t *testing.T) {
	connector, repository := openHeaderButtonRepository(t, dbtest.QueryUntilCanceled(nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := repository.ListHeaderParents(ctx)
	if got != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("result = %#v, error = %v", got, err)
	}
	assertSafeHeaderButtonError(t, err, "query")
	if len(connector.Events()) != 0 || connector.Remaining() != 1 {
		t.Fatalf("pre-canceled query reached driver: events=%#v remaining=%d", connector.Events(), connector.Remaining())
	}
}

func TestHeaderButtonRepositoryContextEndsAfterDriverEntry(t *testing.T) {
	for _, test := range []struct {
		name    string
		context func() (context.Context, context.CancelFunc)
		finish  func(context.CancelFunc)
		want    error
	}{
		{
			name: "canceled",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			finish: func(cancel context.CancelFunc) { cancel() },
			want:   context.Canceled,
		},
		{
			name: "deadline",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 100*time.Millisecond)
			},
			finish: func(context.CancelFunc) {},
			want:   context.DeadlineExceeded,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				entered := make(chan struct{})
				connector, repository := openHeaderButtonRepository(t, dbtest.QueryUntilCanceled(entered))
				ctx, cancel := test.context()
				defer cancel()

				type result struct {
					parents []data.HeaderParent
					err     error
				}
				done := make(chan result, 1)
				go func() {
					parents, err := repository.ListHeaderParents(ctx)
					done <- result{parents: parents, err: err}
				}()

				select {
				case <-entered:
				case result := <-done:
					t.Fatalf("query returned before driver entry was observed: %v", result.err)
				}
				if ctx.Err() != nil || connector.Remaining() != 0 || connector.Events()[0].ContextErr != nil {
					t.Fatal("context ended before driver entry handshake")
				}
				test.finish(cancel)

				gotResult := <-done // synctest advances virtual time while blocked.
				if gotResult.parents != nil || !errors.Is(gotResult.err, test.want) {
					t.Fatalf("result = %#v, error = %v, want %v", gotResult.parents, gotResult.err, test.want)
				}
				assertSafeHeaderButtonError(t, gotResult.err, "query")
				assertHeaderContext(t, gotResult.err, test.want)

				assertHeaderButtonQuery(t, connector)
				if !errors.Is(connector.Events()[0].ContextErr, test.want) {
					t.Fatalf("recorded context error = %v, want %v", connector.Events()[0].ContextErr, test.want)
				}
			})
		})
	}
}

func TestHeaderButtonRepositoryNilDependencyIsSafe(t *testing.T) {
	for _, repository := range []*HeaderButtonRepository{nil, NewHeaderButtonRepository(nil)} {
		got, err := repository.ListHeaderParents(context.Background())
		if got != nil || err == nil {
			t.Fatalf("result = %#v, error = %v", got, err)
		}
		assertSafeHeaderButtonError(t, err, "dependency")
		assertHeaderContext(t, err, nil)
	}
}

func TestHeaderButtonRepositoryCloseErrorPriority(t *testing.T) {
	for _, test := range []struct {
		name         string
		values       [][]any
		iterationErr error
		closeErr     error
		stage        string
		wantContext  error
	}{
		{name: "successful rows then close error", values: [][]any{{int64(1), "First", nil}}, closeErr: &headerBackendError{}, stage: "rows/close"},
		{name: "empty then close error", closeErr: &headerBackendError{}, stage: "rows/close"},
		{name: "scan wins over close", values: [][]any{{int64(1), "First", nil}, {"private-row", "Bad", nil}}, closeErr: &headerBackendError{cause: context.DeadlineExceeded}, stage: "scan"},
		{name: "iteration wins over close", values: [][]any{{int64(1), "First", nil}}, iterationErr: &headerBackendError{}, closeErr: &headerBackendError{cause: context.DeadlineExceeded}, stage: "rows/close"},
		{name: "iteration context wins over close context", values: [][]any{{int64(1), "First", nil}}, iterationErr: &headerBackendError{cause: context.Canceled}, closeErr: &headerBackendError{cause: context.DeadlineExceeded}, stage: "rows/close", wantContext: context.Canceled},
		{name: "close context is preserved", closeErr: &headerBackendError{cause: context.DeadlineExceeded}, stage: "rows/close", wantContext: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			rowsPlan := dbtest.NewRows([]string{"hbid", "title", "parent_id"}, test.values...).WithCloseError(test.closeErr)
			if test.iterationErr != nil {
				rowsPlan.WithErrorAfter(len(test.values), test.iterationErr)
			}
			connector, repository := openHeaderButtonRepository(t, dbtest.Query(rowsPlan))
			got, err := repository.ListHeaderParents(context.Background())
			if got != nil || err == nil {
				t.Fatalf("result = %#v, error = %v", got, err)
			}
			assertSafeHeaderButtonError(t, err, test.stage)
			assertHeaderContext(t, err, test.wantContext)
			if errors.Is(err, test.closeErr) || (test.iterationErr != nil && errors.Is(err, test.iterationErr)) {
				t.Fatal("backend identity escaped repository")
			}
			assertHeaderButtonQuery(t, connector)
			if rowsPlan.CloseCount() != 1 {
				t.Fatalf("close count = %d, want 1", rowsPlan.CloseCount())
			}
		})
	}
}

func TestHeaderButtonRepositoryDoesNotClosePool(t *testing.T) {
	first := dbtest.NewRows([]string{"hbid", "title", "parent_id"}, []any{int64(1), "First", nil})
	second := dbtest.NewRows([]string{"hbid", "title", "parent_id"}, []any{int64(2), "Second", nil})
	connector, repository := openHeaderButtonRepository(t, dbtest.Query(first), dbtest.Query(second))
	for _, want := range []data.HeaderParent{{ID: "1", Title: "First"}, {ID: "2", Title: "Second"}} {
		got, err := repository.ListHeaderParents(context.Background())
		if err != nil || !reflect.DeepEqual(got, []data.HeaderParent{want}) {
			t.Fatalf("result = %#v, error = %v, want %#v", got, err, want)
		}
	}
	assertHeaderButtonQueries(t, connector, 2)
	if first.CloseCount() != 1 || second.CloseCount() != 1 {
		t.Fatalf("close counts = %d, %d, want 1, 1", first.CloseCount(), second.CloseCount())
	}
}

func openHeaderButtonRepository(t *testing.T, steps ...dbtest.Step) (*dbtest.Connector, *HeaderButtonRepository) {
	t.Helper()
	connector := dbtest.NewConnector(steps...)
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return connector, NewHeaderButtonRepository(db)
}

func assertHeaderButtonQuery(t *testing.T, connector *dbtest.Connector) {
	t.Helper()
	assertHeaderButtonQueries(t, connector, 1)
}

func assertHeaderButtonQueries(t *testing.T, connector *dbtest.Connector, count int) {
	t.Helper()
	// Independent contract: do not refer to the production SQL constant here.
	const wantSQL = `SELECT hbid, title, parent_id
FROM header_buttons
WHERE parent_id IS NULL`
	events := connector.Events()
	if len(events) != count {
		t.Fatalf("query event count = %d, want %d", len(events), count)
	}
	for _, event := range events {
		if event.Kind != dbtest.QueryKind || event.SQL != wantSQL {
			t.Fatalf("query event = %#v, want exact parent query", event)
		}
		if len(event.Args) != 0 {
			t.Fatalf("query arguments = %#v, want zero arguments", event.Args)
		}
	}
	if connector.Remaining() != 0 {
		t.Fatalf("remaining scripted operations = %d", connector.Remaining())
	}
}

func assertSafeHeaderButtonError(t *testing.T, err error, stage string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a repository error")
	}
	if got, want := err.Error(), "header buttons could not be read: "+stage; got != want {
		t.Fatalf("error text = %q, want %q", got, want)
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("repository exposed an unwrap chain")
	}
	var backend *headerBackendError
	if errors.As(err, &backend) {
		t.Fatal("repository exposed backend error type")
	}
	for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
		printed := fmt.Sprintf(format, err)
		for _, forbidden := range []string{"SELECT", "header_buttons", "password", "secret", "backend detail", "postgres://", "private-row"} {
			if strings.Contains(printed, forbidden) {
				t.Fatalf("error format %s exposes %q", format, forbidden)
			}
		}
	}
}

type headerBackendError struct{ cause error }

func (*headerBackendError) Error() string {
	return "synthetic backend detail: postgres://user:secret@host/db password=secret SELECT header_buttons private-row"
}

func (e *headerBackendError) Unwrap() error { return e.cause }

func assertHeaderContext(t *testing.T, err, want error) {
	t.Helper()
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
		if got := errors.Is(err, sentinel); got != (want == sentinel) {
			t.Fatalf("errors.Is(error, %v) = %v, want %v", sentinel, got, want == sentinel)
		}
	}
}
