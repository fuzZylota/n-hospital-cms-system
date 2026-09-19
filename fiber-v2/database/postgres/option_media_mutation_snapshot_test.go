package postgres

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"database/postgres/internal/dbtest"
	"models/data"
)

const wantOptionMediaMutationSnapshotSQL = `SELECT oid, option_set_is_active, option_set_is_testing_now, max_upload_size, site_logo_mid, site_light_logo_mid, site_favicon_mid, default_page_mid
FROM options
WHERE option_set_is_active = TRUE`

var _ data.OptionMediaMutationSnapshotReader = (*OptionsRepository)(nil)

func TestOptionsRepositoryOptionMediaMutationSnapshotQueryAndMapping(t *testing.T) {
	rowsPlan := dbtest.NewRows(snapshotColumns(), snapshotValues(41, true, true, 5242880, 11, 12, 13, 14))
	connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))

	got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
	if err != nil || !found {
		t.Fatal("unexpected snapshot read result")
	}
	if got.Set.ID != "41" || !got.Set.IsActive || !got.Set.IsTesting || got.MaxBytes != 5242880 ||
		!stringPointerEquals(got.SiteLogoID, "11") || !stringPointerEquals(got.SiteLightLogoID, "12") ||
		!stringPointerEquals(got.FaviconID, "13") || !stringPointerEquals(got.DefaultPageMediaID, "14") {
		t.Fatal("snapshot mapping mismatch")
	}
	assertOptionsQuery(t, connector, wantOptionMediaMutationSnapshotSQL)
	assertRowsClosed(t, rowsPlan)
}

func TestOptionsRepositoryOptionMediaMutationSnapshotMissingAndNulls(t *testing.T) {
	t.Run("missing active row", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(snapshotColumns())
		connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
		if err != nil || found || got != (data.OptionMediaMutationSnapshot{}) {
			t.Fatal("missing snapshot result mismatch")
		}
		assertOptionsQuery(t, connector, wantOptionMediaMutationSnapshotSQL)
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("all nullable fields", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(snapshotColumns(), snapshotValues(42, true, false, nil, nil, nil, nil, nil))
		connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
		if err != nil || !found || got.Set.ID != "42" || got.MaxBytes != 0 || got.SiteLogoID != nil || got.SiteLightLogoID != nil || got.FaviconID != nil || got.DefaultPageMediaID != nil {
			t.Fatal("NULL snapshot mapping mismatch")
		}
		assertOptionsQuery(t, connector, wantOptionMediaMutationSnapshotSQL)
		assertRowsClosed(t, rowsPlan)
	})

	for index := 4; index < len(snapshotColumns()); index++ {
		t.Run("single nullable media field", func(t *testing.T) {
			values := snapshotValues(43, true, false, 10, 21, 22, 23, 24)
			values[index] = nil
			rowsPlan := dbtest.NewRows(snapshotColumns(), values)
			_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
			if err != nil || !found {
				t.Fatal("nullable media read mismatch")
			}
			media := []*string{got.SiteLogoID, got.SiteLightLogoID, got.FaviconID, got.DefaultPageMediaID}
			for mediaIndex, value := range media {
				if (mediaIndex == index-4) != (value == nil) {
					t.Fatal("nullable media field mismatch")
				}
			}
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestOptionsRepositoryOptionMediaMutationSnapshotNumericBoundaries(t *testing.T) {
	for _, maxBytes := range []int64{1, 0, -1, math.MaxInt64, math.MinInt64} {
		rowsPlan := dbtest.NewRows(snapshotColumns(), snapshotValues(44, true, false, maxBytes, nil, nil, nil, nil))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
		if err != nil || !found || got.MaxBytes != maxBytes {
			t.Fatal("numeric boundary mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	}
}

func TestOptionsRepositoryOptionMediaMutationSnapshotRejectsInvalidIdentity(t *testing.T) {
	for _, optionID := range []int64{0, -1} {
		rowsPlan := dbtest.NewRows(snapshotColumns(), snapshotValues(optionID, true, false, 10, nil, nil, nil, nil))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
		if got != (data.OptionMediaMutationSnapshot{}) || found || err == nil {
			t.Fatal("invalid option identity returned data")
		}
		assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: invalid identifier", nil)
		assertRowsClosed(t, rowsPlan)
	}

	rowsPlan := dbtest.NewRows(snapshotColumns(), snapshotValues(45, false, true, 10, nil, nil, nil, nil))
	_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
	if got != (data.OptionMediaMutationSnapshot{}) || found || err == nil {
		t.Fatal("inactive selected row returned data")
	}
	assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: selected row", nil)
	assertRowsClosed(t, rowsPlan)
}

func TestOptionsRepositoryOptionMediaMutationSnapshotRejectsInvalidMediaIdentities(t *testing.T) {
	for mediaIndex := 4; mediaIndex < len(snapshotColumns()); mediaIndex++ {
		for _, invalidID := range []int64{0, -1} {
			values := snapshotValues(46, true, false, 10, nil, nil, nil, nil)
			values[mediaIndex] = invalidID
			rowsPlan := dbtest.NewRows(snapshotColumns(), values)
			_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
			if got != (data.OptionMediaMutationSnapshot{}) || found || err == nil {
				t.Fatal("invalid media identity returned data")
			}
			assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: invalid identifier", nil)
			assertRowsClosed(t, rowsPlan)
		}
	}
}

func TestOptionsRepositoryOptionMediaMutationSnapshotCardinalityAndFailures(t *testing.T) {
	backend := &repositoryBackendError{}
	tests := []struct {
		name      string
		rows      *dbtest.Rows
		wantStage string
	}{
		{
			name: "duplicate", rows: dbtest.NewRows(snapshotColumns(),
				snapshotValues(47, true, false, 10, nil, nil, nil, nil),
				snapshotValues(48, true, false, 20, nil, nil, nil, nil)), wantStage: "cardinality",
		},
		{name: "scan", rows: dbtest.NewRows(snapshotColumns(), []any{"invalid", true, false, int64(10), nil, nil, nil, nil}), wantStage: "scan"},
		{name: "rows error", rows: dbtest.NewRows(snapshotColumns(), snapshotValues(49, true, false, 10, nil, nil, nil, nil)).WithErrorAfter(1, backend), wantStage: "rows/close"},
		{name: "close error", rows: dbtest.NewRows(snapshotColumns()).WithCloseError(backend), wantStage: "rows/close"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.Query(test.rows))
			got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
			if got != (data.OptionMediaMutationSnapshot{}) || found || err == nil {
				t.Fatal("failure returned snapshot data")
			}
			assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: "+test.wantStage, nil)
			assertOptionsQuery(t, connector, wantOptionMediaMutationSnapshotSQL)
			assertRowsClosed(t, test.rows)
		})
	}
}

func TestOptionsRepositoryOptionMediaMutationSnapshotSafeErrorsAndContext(t *testing.T) {
	backgroundCases := []struct {
		name    string
		backend error
	}{
		{name: "backend", backend: &repositoryBackendError{}},
		{name: "canceled sentinel", backend: context.Canceled},
		{name: "wrapped canceled sentinel", backend: &repositoryBackendError{cause: context.Canceled}},
		{name: "deadline sentinel", backend: context.DeadlineExceeded},
		{name: "wrapped deadline sentinel", backend: &repositoryBackendError{cause: context.DeadlineExceeded}},
		{name: "canceled text", backend: errors.New(context.Canceled.Error())},
		{name: "deadline text", backend: errors.New(context.DeadlineExceeded.Error())},
	}
	for _, test := range backgroundCases {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.QueryError(test.backend))
			got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
			if got != (data.OptionMediaMutationSnapshot{}) || found || err == nil {
				t.Fatal("query failure returned snapshot data")
			}
			assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: query", nil)
			if errors.Is(err, test.backend) {
				t.Fatal("backend error identity escaped")
			}
			assertOptionsQuery(t, connector, wantOptionMediaMutationSnapshotSQL)
		})
	}

	for _, contextCase := range []struct {
		name    string
		context func() (context.Context, context.CancelFunc)
		want    error
	}{
		{name: "pre-canceled", context: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, func() {}
		}, want: context.Canceled},
		{name: "expired deadline", context: func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Unix(1, 0))
		}, want: context.DeadlineExceeded},
	} {
		t.Run(contextCase.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.QueryUntilCanceled(nil))
			ctx, cancel := contextCase.context()
			defer cancel()
			got, found, err := repository.ReadOptionMediaMutationSnapshot(ctx)
			if got != (data.OptionMediaMutationSnapshot{}) || found || !errors.Is(err, contextCase.want) {
				t.Fatal("completed context result mismatch")
			}
			assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: query", contextCase.want)
			if len(connector.Events()) != 0 || connector.Remaining() != 1 {
				t.Fatal("completed context reached driver")
			}
		})
	}
}

func TestOptionsRepositoryOptionMediaMutationSnapshotCallerContextWinsBackendSentinel(t *testing.T) {
	t.Run("caller canceled wins backend deadline", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		entered := make(chan struct{}, 1)
		backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.DeadlineExceeded}
		connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		done := readOptionMediaMutationSnapshotAsync(repository, ctx)
		waitForSnapshotQueryBarrier(t, entered, done, cancel)
		cancel()
		result := waitForSnapshotReadResult(t, done, cancel)
		assertStrictSnapshotContextResult(t, result, context.Canceled, backend)
		assertOptionsQuery(t, connector, wantOptionMediaMutationSnapshotSQL)
	})

	t.Run("caller deadline wins backend canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			entered := make(chan struct{}, 1)
			backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.Canceled}
			connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
			done := readOptionMediaMutationSnapshotAsync(repository, ctx)
			waitForSnapshotQueryBarrier(t, entered, done, cancel)
			result := waitForSnapshotReadResult(t, done, cancel)
			assertStrictSnapshotContextResult(t, result, context.DeadlineExceeded, backend)
			assertOptionsQuery(t, connector, wantOptionMediaMutationSnapshotSQL)
		})
	})
}

func TestStrictOptionMediaMutationSnapshotErrorUnexpectedInput(t *testing.T) {
	tests := []struct {
		name        string
		context     func() (context.Context, context.CancelFunc)
		source      error
		wantContext error
	}{
		{name: "plain backend", context: backgroundSnapshotContext, source: &repositoryBackendError{}},
		{name: "nil error", context: backgroundSnapshotContext},
		{name: "canceled caller", context: canceledSnapshotContext, source: &repositoryBackendError{}, wantContext: context.Canceled},
		{name: "deadline caller", context: deadlineSnapshotContext, source: &repositoryBackendError{}, wantContext: context.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := test.context()
			defer cancel()
			err := strictOptionMediaMutationSnapshotError(ctx, test.source)
			assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: dependency", test.wantContext)
			if test.source != nil && errors.Is(err, test.source) {
				t.Fatal("unexpected error identity escaped")
			}
		})
	}
}

func backgroundSnapshotContext() (context.Context, context.CancelFunc) {
	return context.Background(), func() {}
}

func canceledSnapshotContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx, func() {}
}

func deadlineSnapshotContext() (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), time.Unix(1, 0))
}

type snapshotReadResult struct {
	snapshot data.OptionMediaMutationSnapshot
	found    bool
	err      error
}

func readOptionMediaMutationSnapshotAsync(repository *OptionsRepository, ctx context.Context) <-chan snapshotReadResult {
	done := make(chan snapshotReadResult, 1)
	go func() {
		snapshot, found, err := repository.ReadOptionMediaMutationSnapshot(ctx)
		done <- snapshotReadResult{snapshot: snapshot, found: found, err: err}
	}()
	return done
}

func waitForSnapshotQueryBarrier(t *testing.T, entered <-chan struct{}, done <-chan snapshotReadResult, cleanup context.CancelFunc) {
	t.Helper()
	guard := time.NewTimer(5 * time.Second)
	defer stopSnapshotTestTimer(guard)
	select {
	case <-entered:
		return
	case <-done:
		cleanup()
		t.Fatal("operation completed before query barrier")
	case <-guard.C:
		cleanup()
		waitForSnapshotReadCleanup(t, done)
		t.Fatal("query barrier guard expired")
	}
}

func waitForSnapshotReadResult(t *testing.T, done <-chan snapshotReadResult, cleanup context.CancelFunc) snapshotReadResult {
	t.Helper()
	guard := time.NewTimer(5 * time.Second)
	defer stopSnapshotTestTimer(guard)
	select {
	case result := <-done:
		return result
	case <-guard.C:
		cleanup()
		waitForSnapshotReadCleanup(t, done)
		t.Fatal("operation result guard expired")
		return snapshotReadResult{}
	}
}

func waitForSnapshotReadCleanup(t *testing.T, done <-chan snapshotReadResult) {
	t.Helper()
	guard := time.NewTimer(5 * time.Second)
	defer stopSnapshotTestTimer(guard)
	select {
	case <-done:
		return
	case <-guard.C:
		t.Fatal("operation cleanup guard expired")
	}
}

func stopSnapshotTestTimer(timer *time.Timer) {
	if timer.Stop() {
		return
	}
	select {
	case <-timer.C:
	default:
	}
}

func assertStrictSnapshotContextResult(t *testing.T, result snapshotReadResult, wantContext error, backend *snapshotContextGateBackendError) {
	t.Helper()
	if result.snapshot != (data.OptionMediaMutationSnapshot{}) || result.found || result.err == nil {
		t.Fatal("strict context result mismatch")
	}
	assertSafeRepositoryError(t, result.err, "option media mutation snapshot could not be read: query", wantContext)
	if errors.Is(result.err, backend) {
		t.Fatal("backend identity escaped strict boundary")
	}
	var leaked *snapshotContextGateBackendError
	if errors.As(result.err, &leaked) {
		t.Fatal("backend type escaped strict boundary")
	}
}

type snapshotContextGateBackendError struct {
	ctx     context.Context
	entered chan<- struct{}
	cause   error
	once    sync.Once
}

func (*snapshotContextGateBackendError) Error() string {
	return "private backend detail"
}

func (e *snapshotContextGateBackendError) Is(target error) bool {
	e.once.Do(func() {
		// Capacity one lets cancellation cleanup proceed if the guard stops receiving.
		e.entered <- struct{}{}
		<-e.ctx.Done()
	})
	return errors.Is(e.cause, target)
}

func (e *snapshotContextGateBackendError) Unwrap() error {
	return e.cause
}

func TestOptionsRepositoryOptionMediaMutationSnapshotConstructorAndPoolReuse(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
		if got != (data.OptionMediaMutationSnapshot{}) || found || err == nil {
			t.Fatal("nil dependency returned snapshot data")
		}
		assertSafeRepositoryError(t, err, "option media mutation snapshot could not be read: dependency", nil)
	}

	first := dbtest.NewRows(snapshotColumns(), snapshotValues(50, true, false, 10, nil, nil, nil, nil))
	second := dbtest.NewRows(snapshotColumns(), snapshotValues(51, true, false, 20, nil, nil, nil, nil))
	connector, repository := openOptionsRepository(t, dbtest.Query(first), dbtest.Query(second))
	for index, wantMaxBytes := range []int64{10, 20} {
		got, found, err := repository.ReadOptionMediaMutationSnapshot(context.Background())
		if err != nil || !found || got.MaxBytes != wantMaxBytes || got.Set.ID == "" {
			t.Fatal("pool reuse mismatch")
		}
		if len(connector.Events()) != index+1 {
			t.Fatal("unexpected pool reuse query count")
		}
	}
	if connector.Remaining() != 0 {
		t.Fatal("pool reuse left an operation")
	}
	assertRowsClosed(t, first)
	assertRowsClosed(t, second)
}

func snapshotColumns() []string {
	return []string{"oid", "option_set_is_active", "option_set_is_testing_now", "max_upload_size", "site_logo_mid", "site_light_logo_mid", "site_favicon_mid", "default_page_mid"}
}

func snapshotValues(id int64, active, testing bool, maxBytes, siteLogoID, siteLightLogoID, faviconID, defaultPageMediaID any) []any {
	return []any{id, active, testing, maxBytes, siteLogoID, siteLightLogoID, faviconID, defaultPageMediaID}
}
