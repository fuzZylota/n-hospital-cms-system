package postgres

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"database/postgres/internal/dbtest"
	"models/data"
)

const wantJobApplicationResponseWorkflowSnapshotSQL = `SELECT
    o.oid,
    o.option_set_is_active,
    o.option_set_is_testing_now,
    o.smtp_host,
    o.smtp_port,
    o.smtp_username,
    o.smtp_password,
    o.site_name,
    o.site_description,
    o.contact_email,
    o.contact_phone,
    o.facebook_url,
    o.twitter_url,
    o.instagram_url,
    o.linkedin_url,
    o.primary_color,
    logo.file_path
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
WHERE o.option_set_is_active = TRUE`

var _ data.JobApplicationResponseWorkflowSnapshotReader = (*OptionsRepository)(nil)

func TestOptionsRepositoryJobApplicationResponseWorkflowSnapshotQueryAndMapping(t *testing.T) {
	rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(61, true, true, 2525))
	connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))

	got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
	if err != nil || !found {
		t.Fatal("unexpected job-application response snapshot read result")
	}
	want := expectedJobApplicationResponseSnapshot(61, true, true, 2525)
	if got != want {
		t.Fatal("job-application response snapshot mapping mismatch")
	}
	assertOptionsQuery(t, connector, wantJobApplicationResponseWorkflowSnapshotSQL)
	assertRowsClosed(t, rowsPlan)
}

func TestOptionsRepositoryJobApplicationResponseWorkflowSnapshotLogoAndNullableParity(t *testing.T) {
	t.Run("logo present", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(62, true, false, 587))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SiteLogoPath == "" {
			t.Fatal("joined site-logo path mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("logo NULL", func(t *testing.T) {
		values := jobApplicationResponseSnapshotValues(63, true, false, 587)
		values[16] = nil
		rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SiteLogoPath != "" {
			t.Fatal("NULL site-logo path mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("all nullable workflow values", func(t *testing.T) {
		values := []any{int64(64), true, false}
		for range len(jobApplicationResponseSnapshotColumns()) - len(values) {
			values = append(values, nil)
		}
		rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if err != nil || !found || got != (data.JobApplicationResponseWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "64", IsActive: true}}) {
			t.Fatal("all-NULL workflow mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	for index := 3; index < len(jobApplicationResponseSnapshotColumns()); index++ {
		t.Run("single nullable workflow value", func(t *testing.T) {
			values := jobApplicationResponseSnapshotValues(65, true, false, 2525)
			values[index] = nil
			rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), values)
			_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
			want := expectedJobApplicationResponseSnapshot(65, true, false, 2525)
			clearJobApplicationResponseSnapshotColumn(&want, index)
			if err != nil || !found || got != want {
				t.Fatal("single NULL workflow mapping mismatch")
			}
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestOptionsRepositoryJobApplicationResponseWorkflowSnapshotMissingIdentityAndPortBoundaries(t *testing.T) {
	t.Run("missing active row", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns())
		connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if err != nil || found || got != (data.JobApplicationResponseWorkflowSnapshot{}) {
			t.Fatal("missing job-application response snapshot mismatch")
		}
		assertOptionsQuery(t, connector, wantJobApplicationResponseWorkflowSnapshotSQL)
		assertRowsClosed(t, rowsPlan)
	})

	for _, optionID := range []int64{0, -1} {
		rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(optionID, true, false, 2525))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if got != (data.JobApplicationResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("invalid option identity returned workflow data")
		}
		assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: invalid identifier", nil)
		assertRowsClosed(t, rowsPlan)
	}

	rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(66, false, true, 2525))
	_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
	if got != (data.JobApplicationResponseWorkflowSnapshot{}) || found || err == nil {
		t.Fatal("inactive selected row returned workflow data")
	}
	assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: selected row", nil)
	assertRowsClosed(t, rowsPlan)

	for _, port := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
		rowsPlan := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(67, true, false, port))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SMTPPort != port {
			t.Fatal("SMTP port boundary mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	}
}

func TestOptionsRepositoryJobApplicationResponseWorkflowSnapshotCardinalityAndFailures(t *testing.T) {
	backend := &repositoryBackendError{}
	tests := []struct {
		name      string
		rows      *dbtest.Rows
		wantStage string
	}{
		{
			name: "duplicate active rows",
			rows: dbtest.NewRows(jobApplicationResponseSnapshotColumns(),
				jobApplicationResponseSnapshotValues(68, true, false, 2525),
				jobApplicationResponseSnapshotValues(69, true, false, 2525)),
			wantStage: "cardinality",
		},
		{
			name: "duplicate with malformed second row",
			rows: dbtest.NewRows(jobApplicationResponseSnapshotColumns(),
				jobApplicationResponseSnapshotValues(68, true, false, 2525),
				replaceJobApplicationResponseSnapshotValue(jobApplicationResponseSnapshotValues(69, true, false, 2525), 0, "invalid")),
			wantStage: "cardinality",
		},
		{name: "scan", rows: dbtest.NewRows(jobApplicationResponseSnapshotColumns(), replaceJobApplicationResponseSnapshotValue(jobApplicationResponseSnapshotValues(70, true, false, 2525), 0, "invalid")), wantStage: "scan"},
		{name: "rows error", rows: dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(71, true, false, 2525)).WithErrorAfter(1, backend), wantStage: "rows/close"},
		{name: "close error", rows: dbtest.NewRows(jobApplicationResponseSnapshotColumns()).WithCloseError(backend), wantStage: "rows/close"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.Query(test.rows))
			got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
			if got != (data.JobApplicationResponseWorkflowSnapshot{}) || found || err == nil {
				t.Fatal("repository failure returned workflow data")
			}
			assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: "+test.wantStage, nil)
			assertOptionsQuery(t, connector, wantJobApplicationResponseWorkflowSnapshotSQL)
			assertRowsClosed(t, test.rows)
		})
	}

	connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
	got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
	if got != (data.JobApplicationResponseWorkflowSnapshot{}) || found || err == nil {
		t.Fatal("query failure returned workflow data")
	}
	assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: query", nil)
	if errors.Is(err, backend) {
		t.Fatal("backend error identity escaped")
	}
	assertOptionsQuery(t, connector, wantJobApplicationResponseWorkflowSnapshotSQL)
}

func TestOptionsRepositoryJobApplicationResponseWorkflowSnapshotSafeErrorsAndContext(t *testing.T) {
	for _, backend := range []error{
		&repositoryBackendError{},
		context.Canceled,
		&repositoryBackendError{cause: context.Canceled},
		context.DeadlineExceeded,
		&repositoryBackendError{cause: context.DeadlineExceeded},
		errors.New(context.Canceled.Error()),
		errors.New(context.DeadlineExceeded.Error()),
	} {
		connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if got != (data.JobApplicationResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("backend failure returned workflow data")
		}
		assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: query", nil)
		if errors.Is(err, backend) {
			t.Fatal("backend error identity escaped strict boundary")
		}
		assertOptionsQuery(t, connector, wantJobApplicationResponseWorkflowSnapshotSQL)
	}

	for _, contextCase := range []struct {
		makeContext func() (context.Context, context.CancelFunc)
		want        error
	}{
		{makeContext: canceledSnapshotContext, want: context.Canceled},
		{makeContext: deadlineSnapshotContext, want: context.DeadlineExceeded},
	} {
		connector, repository := openOptionsRepository(t, dbtest.QueryUntilCanceled(nil))
		ctx, cancel := contextCase.makeContext()
		defer cancel()
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(ctx)
		if got != (data.JobApplicationResponseWorkflowSnapshot{}) || found || !errors.Is(err, contextCase.want) {
			t.Fatal("completed caller context result mismatch")
		}
		assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: query", contextCase.want)
		if len(connector.Events()) != 0 || connector.Remaining() != 1 {
			t.Fatal("completed caller context reached driver")
		}
	}
}

func TestOptionsRepositoryJobApplicationResponseWorkflowSnapshotCallerContextWinsBackendSentinel(t *testing.T) {
	t.Run("caller canceled wins backend deadline", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		entered := make(chan struct{}, 1)
		backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.DeadlineExceeded}
		connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		done := readJobApplicationResponseWorkflowSnapshotAsync(repository, ctx)
		waitForJobApplicationResponseSnapshotQueryBarrier(t, entered, done, cancel)
		cancel()
		result := waitForJobApplicationResponseSnapshotResult(t, done, cancel)
		assertStrictJobApplicationResponseSnapshotContextResult(t, result, context.Canceled, backend)
		assertOptionsQuery(t, connector, wantJobApplicationResponseWorkflowSnapshotSQL)
	})

	t.Run("caller deadline wins backend canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			entered := make(chan struct{}, 1)
			backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.Canceled}
			connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
			done := readJobApplicationResponseWorkflowSnapshotAsync(repository, ctx)
			waitForJobApplicationResponseSnapshotQueryBarrier(t, entered, done, cancel)
			result := waitForJobApplicationResponseSnapshotResult(t, done, cancel)
			assertStrictJobApplicationResponseSnapshotContextResult(t, result, context.DeadlineExceeded, backend)
			assertOptionsQuery(t, connector, wantJobApplicationResponseWorkflowSnapshotSQL)
		})
	})
}

func TestStrictJobApplicationResponseWorkflowSnapshotErrorUnexpectedInput(t *testing.T) {
	var typedNil *repositoryReadError
	for _, test := range []struct {
		makeContext func() (context.Context, context.CancelFunc)
		source      error
		wantContext error
	}{
		{makeContext: backgroundSnapshotContext, source: &repositoryBackendError{}},
		{makeContext: backgroundSnapshotContext},
		{makeContext: backgroundSnapshotContext, source: typedNil},
		{makeContext: canceledSnapshotContext, source: &repositoryBackendError{}, wantContext: context.Canceled},
		{makeContext: deadlineSnapshotContext, source: &repositoryBackendError{}, wantContext: context.DeadlineExceeded},
	} {
		ctx, cancel := test.makeContext()
		err := strictJobApplicationResponseWorkflowSnapshotError(ctx, test.source)
		cancel()
		assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: dependency", test.wantContext)
		if test.source != nil && errors.Is(err, test.source) {
			t.Fatal("unexpected error identity escaped")
		}
	}
}

func TestStrictJobApplicationResponseWorkflowSnapshotCallerBackendContextMatrix(t *testing.T) {
	contexts := []struct {
		makeContext func() (context.Context, context.CancelFunc)
		want        error
	}{
		{makeContext: backgroundSnapshotContext},
		{makeContext: canceledSnapshotContext, want: context.Canceled},
		{makeContext: deadlineSnapshotContext, want: context.DeadlineExceeded},
	}
	backends := []error{
		&repositoryBackendError{},
		context.Canceled,
		context.DeadlineExceeded,
		&repositoryBackendError{cause: context.Canceled},
		&repositoryBackendError{cause: context.DeadlineExceeded},
	}
	for _, contextCase := range contexts {
		for _, backend := range backends {
			ctx, cancel := contextCase.makeContext()
			err := strictJobApplicationResponseWorkflowSnapshotError(ctx, newRepositoryReadError(readJobApplicationResponseWorkflowSnapshot, readQuery, backend))
			cancel()
			assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: query", contextCase.want)
			if errors.Is(err, backend) && backend != contextCase.want {
				t.Fatal("backend context identity crossed caller boundary")
			}
		}
	}
}

func TestOptionsRepositoryJobApplicationResponseWorkflowSnapshotConstructorAndPoolReuse(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if got != (data.JobApplicationResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("nil dependency returned workflow data")
		}
		assertSafeRepositoryError(t, err, "job application response workflow snapshot could not be read: dependency", nil)
	}

	first := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(72, true, false, 2525))
	second := dbtest.NewRows(jobApplicationResponseSnapshotColumns(), jobApplicationResponseSnapshotValues(73, true, false, 465))
	connector, repository := openOptionsRepository(t, dbtest.Query(first), dbtest.Query(second))
	for index, wantPort := range []int64{2525, 465} {
		got, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SMTPPort != wantPort || got.Set.ID == "" {
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

type jobApplicationResponseSnapshotReadResult struct {
	snapshot data.JobApplicationResponseWorkflowSnapshot
	found    bool
	err      error
}

func readJobApplicationResponseWorkflowSnapshotAsync(repository *OptionsRepository, ctx context.Context) <-chan jobApplicationResponseSnapshotReadResult {
	done := make(chan jobApplicationResponseSnapshotReadResult, 1)
	go func() {
		snapshot, found, err := repository.ReadJobApplicationResponseWorkflowSnapshot(ctx)
		done <- jobApplicationResponseSnapshotReadResult{snapshot: snapshot, found: found, err: err}
	}()
	return done
}

func waitForJobApplicationResponseSnapshotQueryBarrier(t *testing.T, entered <-chan struct{}, done <-chan jobApplicationResponseSnapshotReadResult, cleanup context.CancelFunc) {
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
		waitForJobApplicationResponseSnapshotCleanup(t, done)
		t.Fatal("query barrier guard expired")
	}
}

func waitForJobApplicationResponseSnapshotResult(t *testing.T, done <-chan jobApplicationResponseSnapshotReadResult, cleanup context.CancelFunc) jobApplicationResponseSnapshotReadResult {
	t.Helper()
	guard := time.NewTimer(5 * time.Second)
	defer stopSnapshotTestTimer(guard)
	select {
	case result := <-done:
		return result
	case <-guard.C:
		cleanup()
		waitForJobApplicationResponseSnapshotCleanup(t, done)
		t.Fatal("operation result guard expired")
		return jobApplicationResponseSnapshotReadResult{}
	}
}

func waitForJobApplicationResponseSnapshotCleanup(t *testing.T, done <-chan jobApplicationResponseSnapshotReadResult) {
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

func assertStrictJobApplicationResponseSnapshotContextResult(t *testing.T, result jobApplicationResponseSnapshotReadResult, wantContext error, backend *snapshotContextGateBackendError) {
	t.Helper()
	if result.snapshot != (data.JobApplicationResponseWorkflowSnapshot{}) || result.found || result.err == nil {
		t.Fatal("strict context result mismatch")
	}
	assertSafeRepositoryError(t, result.err, "job application response workflow snapshot could not be read: query", wantContext)
	if errors.Is(result.err, backend) {
		t.Fatal("backend identity escaped strict boundary")
	}
	var leaked *snapshotContextGateBackendError
	if errors.As(result.err, &leaked) {
		t.Fatal("backend type escaped strict boundary")
	}
}

func jobApplicationResponseSnapshotColumns() []string {
	return []string{
		"oid", "option_set_is_active", "option_set_is_testing_now", "smtp_host", "smtp_port", "smtp_username",
		"smtp_password", "site_name", "site_description", "contact_email", "contact_phone", "facebook_url",
		"twitter_url", "instagram_url", "linkedin_url", "primary_color", "logo_path",
	}
}

func jobApplicationResponseSnapshotValues(id int64, active, testing bool, smtpPort int64) []any {
	return []any{
		id, active, testing, "smtp.internal.invalid", smtpPort, "mailer", "fixture-password", "Nivgoz",
		"Public description", "contact@example.invalid", "+90 000", "facebook", "twitter", "instagram", "linkedin",
		"primary", "files/logo.webp",
	}
}

func expectedJobApplicationResponseSnapshot(id int64, active, testing bool, smtpPort int64) data.JobApplicationResponseWorkflowSnapshot {
	return data.JobApplicationResponseWorkflowSnapshot{
		Set:             data.OptionSetIdentity{ID: strconv.FormatInt(id, 10), IsActive: active, IsTesting: testing},
		SMTPHost:        "smtp.internal.invalid",
		SMTPPort:        smtpPort,
		SMTPUsername:    "mailer",
		SMTPPassword:    "fixture-password",
		SiteName:        "Nivgoz",
		SiteDescription: "Public description",
		ContactEmail:    "contact@example.invalid",
		ContactPhone:    "+90 000",
		FacebookURL:     "facebook",
		TwitterURL:      "twitter",
		InstagramURL:    "instagram",
		LinkedInURL:     "linkedin",
		PrimaryColor:    "primary",
		SiteLogoPath:    "files/logo.webp",
	}
}

func clearJobApplicationResponseSnapshotColumn(snapshot *data.JobApplicationResponseWorkflowSnapshot, columnIndex int) {
	switch columnIndex {
	case 3:
		snapshot.SMTPHost = ""
	case 4:
		snapshot.SMTPPort = 0
	case 5:
		snapshot.SMTPUsername = ""
	case 6:
		snapshot.SMTPPassword = ""
	case 7:
		snapshot.SiteName = ""
	case 8:
		snapshot.SiteDescription = ""
	case 9:
		snapshot.ContactEmail = ""
	case 10:
		snapshot.ContactPhone = ""
	case 11:
		snapshot.FacebookURL = ""
	case 12:
		snapshot.TwitterURL = ""
	case 13:
		snapshot.InstagramURL = ""
	case 14:
		snapshot.LinkedInURL = ""
	case 15:
		snapshot.PrimaryColor = ""
	case 16:
		snapshot.SiteLogoPath = ""
	}
}

func replaceJobApplicationResponseSnapshotValue(values []any, index int, value any) []any {
	result := append([]any(nil), values...)
	result[index] = value
	return result
}
