package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"database/postgres/internal/dbtest"
	"models/data"
)

var _ data.UserStatusReader = (*UserStatusRepository)(nil)

const wantLookupUserStatusSQL = `SELECT uid, is_active, role
FROM users
WHERE uid = $1`

func TestUserStatusRepositoryQueryAndStates(t *testing.T) {
	for _, test := range []struct {
		name       string
		userID     string
		expectedID int64
		active     bool
		role       string
	}{
		{name: "active admin", userID: "7", expectedID: 7, active: true, role: "admin"},
		{name: "inactive moderator", userID: "12", expectedID: 12, active: false, role: "moderator"},
		{name: "maximum int64", userID: "9223372036854775807", expectedID: int64(9223372036854775807), active: true, role: "ik"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rowsPlan := dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{test.expectedID, test.active, test.role})
			connector, repository := openUserStatusRepository(t, dbtest.Query(rowsPlan))
			got, err := repository.LookupUserStatus(context.Background(), test.userID)
			if err != nil || !got.Found || got.Active != test.active || got.Role != test.role {
				t.Fatal("unexpected user status")
			}
			assertUserStatusQuery(t, connector, test.expectedID)
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestUserStatusRepositoryNullableActivePolicyAndPoolReuse(t *testing.T) {
	nullRows := dbtest.NewRows(
		[]string{"uid", "is_active", "role"},
		[]any{int64(27), nil, "private-null-active-role"},
	)
	validRows := dbtest.NewRows(
		[]string{"uid", "is_active", "role"},
		[]any{int64(27), true, "admin"},
	)
	connector, repository := openUserStatusRepository(t, dbtest.Query(nullRows), dbtest.Query(validRows))

	got, err := repository.LookupUserStatus(context.Background(), "27")
	if got != (data.UserStatus{}) || err == nil {
		t.Fatal("unexpected NULL active result")
	}
	assertSafeRepositoryError(t, err, "user status could not be read: selected row", nil)
	backendSentinel := errors.New("private-null-active-backend-sentinel")
	if errors.Is(err, backendSentinel) {
		t.Fatal("NULL active error exposed a backend sentinel")
	}
	var backend *repositoryBackendError
	if errors.As(err, &backend) {
		t.Fatal("NULL active error exposed a backend type")
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("NULL active error exposed an unwrap chain")
	}
	for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
		printed := fmt.Sprintf(format, err)
		for _, forbidden := range []string{"27", "private-null-active-role", "NULL", "<nil>", "private-null-active-backend-sentinel"} {
			if strings.Contains(printed, forbidden) {
				t.Fatal("NULL active error exposed row or backend detail")
			}
		}
	}
	assertRowsClosed(t, nullRows)

	second, err := repository.LookupUserStatus(context.Background(), "27")
	if err != nil || second != (data.UserStatus{Found: true, Active: true, Role: "admin"}) {
		t.Fatal("repository was not reusable after NULL active row")
	}
	assertUserStatusQueries(t, connector, 27, 2)
	assertRowsClosed(t, validRows)
}

func TestUserStatusRepositoryNullRoleIsSafeScanFailure(t *testing.T) {
	rowsPlan := dbtest.NewRows(
		[]string{"uid", "is_active", "role"},
		[]any{int64(28), true, nil},
	)
	connector, repository := openUserStatusRepository(t, dbtest.Query(rowsPlan))

	got, err := repository.LookupUserStatus(context.Background(), "28")
	if got != (data.UserStatus{}) || err == nil {
		t.Fatal("NULL role returned partial data or no error")
	}
	assertSafeRepositoryError(t, err, "user status could not be read: scan", nil)
	assertUserStatusQuery(t, connector, 28)
	assertRowsClosed(t, rowsPlan)
}

func TestUserStatusRepositoryNotFoundIsDistinctFromInactive(t *testing.T) {
	emptyRows := dbtest.NewRows([]string{"uid", "is_active", "role"})
	connector, repository := openUserStatusRepository(t, dbtest.Query(emptyRows))
	got, err := repository.LookupUserStatus(context.Background(), "8")
	if err != nil || got != (data.UserStatus{Found: false}) {
		t.Fatal("unexpected not-found status")
	}
	assertUserStatusQuery(t, connector, 8)
	assertRowsClosed(t, emptyRows)

	inactiveRows := dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(8), false, "santral"})
	_, inactiveRepository := openUserStatusRepository(t, dbtest.Query(inactiveRows))
	inactive, err := inactiveRepository.LookupUserStatus(context.Background(), "8")
	if err != nil || !inactive.Found || inactive.Active || inactive.Role != "santral" {
		t.Fatal("unexpected inactive status")
	}
}

func TestUserStatusRepositoryRejectsInvalidIdentifiersWithoutQuery(t *testing.T) {
	invalid := []string{
		"", "0", "-1", "+1", "01", "00", " 1", "1 ", "1\t", "1\n", "1.0", "1e2", "abc",
		"9223372036854775808", "18446744073709551615", "１２", "١", "1/../../private",
	}
	for _, userID := range invalid {
		t.Run("invalid identifier", func(t *testing.T) {
			connector, repository := openUserStatusRepository(t, dbtest.Query(dbtest.NewRows([]string{"uid", "is_active", "role"})))
			got, err := repository.LookupUserStatus(context.Background(), userID)
			if got != (data.UserStatus{}) || err == nil {
				t.Fatal("invalid identifier returned data or no error")
			}
			assertSafeRepositoryError(t, err, "user status could not be read: invalid identifier", nil)
			for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
				if userID != "" && strings.Contains(fmt.Sprintf(format, err), userID) {
					t.Fatal("repository error exposed raw identifier")
				}
			}
			if len(connector.Events()) != 0 || connector.Remaining() != 1 {
				t.Fatal("invalid identifier reached driver")
			}
		})
	}
}

func TestUserStatusRepositoryRejectsNULIdentifierWithoutQuery(t *testing.T) {
	connector, repository := openUserStatusRepository(t, dbtest.Query(dbtest.NewRows([]string{"uid", "is_active", "role"})))
	got, err := repository.LookupUserStatus(context.Background(), "1\x002")
	if got != (data.UserStatus{}) || err == nil {
		t.Fatal("NUL identifier returned data or no error")
	}
	assertSafeRepositoryError(t, err, "user status could not be read: invalid identifier", nil)
	if len(connector.Events()) != 0 || connector.Remaining() != 1 {
		t.Fatal("NUL identifier reached driver")
	}
}

func TestUserStatusRepositoryCardinalityAndReturnedIdentifier(t *testing.T) {
	for _, test := range []struct {
		name   string
		values [][]any
		stage  string
	}{
		{name: "duplicate", values: [][]any{{int64(4), true, "admin"}, {int64(4), false, "moderator"}}, stage: "cardinality"},
		{name: "wrong returned uid", values: [][]any{{int64(5), true, "admin"}}, stage: "selected row"},
		{name: "wrong returned uid wins over NULL active", values: [][]any{{int64(5), nil, "admin"}}, stage: "selected row"},
		{name: "non-positive returned uid", values: [][]any{{int64(0), true, "admin"}}, stage: "invalid identifier"},
		{name: "negative returned uid", values: [][]any{{int64(-5), true, "admin"}}, stage: "invalid identifier"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rowsPlan := dbtest.NewRows([]string{"uid", "is_active", "role"}, test.values...)
			connector, repository := openUserStatusRepository(t, dbtest.Query(rowsPlan))
			got, err := repository.LookupUserStatus(context.Background(), "4")
			if got != (data.UserStatus{}) || err == nil {
				t.Fatal("invalid result returned data or no error")
			}
			assertSafeRepositoryError(t, err, "user status could not be read: "+test.stage, nil)
			if test.name == "wrong returned uid" || test.name == "wrong returned uid wins over NULL active" {
				for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
					printed := fmt.Sprintf(format, err)
					for _, forbidden := range []string{"4", "5", "admin", "SELECT", "users"} {
						if strings.Contains(printed, forbidden) {
							t.Fatal("UID mismatch error exposed query or row detail")
						}
					}
				}
			}
			assertUserStatusQuery(t, connector, 4)
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestUserStatusRepositoryFailureStagesAndPriority(t *testing.T) {
	backend := &repositoryBackendError{}
	cardinalityClose := &repositoryBackendError{cause: context.DeadlineExceeded}
	for _, test := range []struct {
		name        string
		step        dbtest.Step
		rows        *dbtest.Rows
		stage       string
		wantContext error
	}{
		{name: "query", step: dbtest.QueryError(backend), stage: "query"},
		{name: "scan first row", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{"bad-id", true, "admin"}), stage: "scan"},
		{name: "scan after successful row", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(4), true, "admin"}, []any{"bad-id", false, "moderator"}), stage: "scan"},
		{name: "iteration", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(4), true, "admin"}).WithErrorAfter(1, backend), stage: "rows/close"},
		{name: "close", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(4), true, "admin"}).WithCloseError(backend), stage: "rows/close"},
		{name: "scan wins over close", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{"bad-id", true, "admin"}).WithCloseError(&repositoryBackendError{cause: context.DeadlineExceeded}), stage: "scan"},
		{name: "iteration context wins over close", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(4), true, "admin"}).WithErrorAfter(1, &repositoryBackendError{cause: context.Canceled}).WithCloseError(&repositoryBackendError{cause: context.DeadlineExceeded}), stage: "rows/close", wantContext: context.Canceled},
		{name: "cardinality wins over close", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(4), true, "admin"}, []any{int64(4), false, "moderator"}).WithCloseError(cardinalityClose), stage: "cardinality"},
		{name: "close context", rows: dbtest.NewRows([]string{"uid", "is_active", "role"}).WithCloseError(&repositoryBackendError{cause: context.DeadlineExceeded}), stage: "rows/close", wantContext: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			step := test.step
			if step == nil {
				step = dbtest.Query(test.rows)
			}
			connector, repository := openUserStatusRepository(t, step)
			got, err := repository.LookupUserStatus(context.Background(), "4")
			if got != (data.UserStatus{}) || err == nil {
				t.Fatal("failure returned data or no error")
			}
			assertSafeRepositoryError(t, err, "user status could not be read: "+test.stage, test.wantContext)
			if errors.Is(err, backend) {
				t.Fatal("backend identity escaped repository")
			}
			assertUserStatusQuery(t, connector, 4)
			if test.rows != nil {
				assertRowsClosed(t, test.rows)
			}
		})
	}
}

func TestUserStatusRepositoryContextIdentityAndFormatting(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "backend type", err: &repositoryBackendError{}},
		{name: "canceled", err: context.Canceled, want: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: context.DeadlineExceeded},
		{name: "wrapped deadline", err: &repositoryBackendError{cause: context.DeadlineExceeded}, want: context.DeadlineExceeded},
		{name: "same canceled text", err: errors.New(context.Canceled.Error())},
		{name: "same deadline text", err: errors.New(context.DeadlineExceeded.Error())},
	} {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openUserStatusRepository(t, dbtest.QueryError(test.err))
			got, err := repository.LookupUserStatus(context.Background(), "41")
			if got != (data.UserStatus{}) || err == nil {
				t.Fatal("query error returned data or no error")
			}
			assertSafeRepositoryError(t, err, "user status could not be read: query", test.want)
			for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
				printed := fmt.Sprintf(format, err)
				for _, forbidden := range []string{"41", "admin", "role", "users", "C:\\private", "credential"} {
					if strings.Contains(printed, forbidden) {
						t.Fatal("repository error exposed input or backend detail")
					}
				}
			}
			assertUserStatusQuery(t, connector, 41)
		})
	}
}

func TestUserStatusRepositoryContextBeforeAndAfterDriverEntry(t *testing.T) {
	t.Run("pre-canceled", func(t *testing.T) {
		connector, repository := openUserStatusRepository(t, dbtest.QueryUntilCanceled(nil))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got, err := repository.LookupUserStatus(ctx, "5")
		if got != (data.UserStatus{}) || !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected pre-canceled result")
		}
		if len(connector.Events()) != 0 || connector.Remaining() != 1 {
			t.Fatal("pre-canceled lookup reached driver")
		}
	})

	for _, test := range []struct {
		name    string
		makeCtx func() (context.Context, context.CancelFunc)
		finish  func(context.CancelFunc)
		want    error
	}{
		{name: "canceled", makeCtx: func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }, finish: func(cancel context.CancelFunc) { cancel() }, want: context.Canceled},
		{name: "deadline", makeCtx: func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 100*time.Millisecond)
		}, finish: func(context.CancelFunc) {}, want: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				entered := make(chan struct{})
				connector, repository := openUserStatusRepository(t, dbtest.QueryUntilCanceled(entered))
				ctx, cancel := test.makeCtx()
				defer cancel()
				type result struct {
					status data.UserStatus
					err    error
				}
				done := make(chan result, 1)
				go func() {
					status, err := repository.LookupUserStatus(ctx, "5")
					done <- result{status: status, err: err}
				}()
				select {
				case <-entered:
				case result := <-done:
					if result.status != (data.UserStatus{}) || result.err == nil {
						t.Fatal("query returned an unexpected result before entry handshake")
					}
					t.Fatal("query returned before entry handshake")
				}
				if ctx.Err() != nil || connector.Remaining() != 0 {
					t.Fatal("context ended before driver entry")
				}
				test.finish(cancel)
				got := <-done
				if got.status != (data.UserStatus{}) {
					t.Fatal("context failure returned partial data")
				}
				assertSafeRepositoryError(t, got.err, "user status could not be read: query", test.want)
				assertUserStatusQuery(t, connector, 5)
				if !errors.Is(connector.Events()[0].ContextErr, test.want) {
					t.Fatal("unexpected recorded context classification")
				}
			})
		})
	}
}

func TestUserStatusRepositoryNilDependencyAndPoolReuse(t *testing.T) {
	for _, repository := range []*UserStatusRepository{nil, NewUserStatusRepository(nil)} {
		got, err := repository.LookupUserStatus(context.Background(), "1")
		if got != (data.UserStatus{}) || err == nil {
			t.Fatal("nil dependency returned data or no error")
		}
		assertSafeRepositoryError(t, err, "user status could not be read: dependency", nil)
	}

	first := dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(1), true, "admin"})
	second := dbtest.NewRows([]string{"uid", "is_active", "role"}, []any{int64(1), false, "moderator"})
	connector, repository := openUserStatusRepository(t, dbtest.Query(first), dbtest.Query(second))
	active, err := repository.LookupUserStatus(context.Background(), "1")
	if err != nil || !active.Found || !active.Active || active.Role != "admin" {
		t.Fatal("first pool-reuse lookup mismatch")
	}
	inactive, err := repository.LookupUserStatus(context.Background(), "1")
	if err != nil || !inactive.Found || inactive.Active || inactive.Role != "moderator" {
		t.Fatal("second pool-reuse lookup mismatch")
	}
	if len(connector.Events()) != 2 || connector.Remaining() != 0 {
		t.Fatal("repository did not reuse borrowed pool")
	}
	assertRowsClosed(t, first)
	assertRowsClosed(t, second)
}

func openUserStatusRepository(t *testing.T, steps ...dbtest.Step) (*dbtest.Connector, *UserStatusRepository) {
	t.Helper()
	connector := dbtest.NewConnector(steps...)
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error("database close failed")
		}
	})
	return connector, NewUserStatusRepository(db)
}

func assertUserStatusQuery(t *testing.T, connector *dbtest.Connector, wantID int64) {
	t.Helper()
	assertUserStatusQueries(t, connector, wantID, 1)
}

func assertUserStatusQueries(t *testing.T, connector *dbtest.Connector, wantID int64, wantCount int) {
	t.Helper()
	events := connector.Events()
	if len(events) != wantCount {
		t.Fatal("unexpected query event count")
	}
	for _, event := range events {
		if event.Kind != dbtest.QueryKind || event.SQL != wantLookupUserStatusSQL {
			t.Fatal("unexpected query")
		}
		if len(event.Args) != 1 || event.Args[0].Name != "" || event.Args[0].Ordinal != 1 || event.Args[0].Value != wantID {
			t.Fatal("unexpected query arguments")
		}
	}
	if connector.Remaining() != 0 {
		t.Fatal("unexpected remaining operations")
	}
}
