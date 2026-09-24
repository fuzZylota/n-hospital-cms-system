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

const wantJobApplicationWorkflowSnapshotSQL = `SELECT
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
    o.google_recaptcha_site_key,
    o.google_recaptcha_secret_key,
    logo.file_path,
    o.max_upload_size
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
WHERE o.option_set_is_active = TRUE`

var _ data.JobApplicationWorkflowSnapshotReader = (*OptionsRepository)(nil)

func TestOptionsRepositoryJobApplicationWorkflowSnapshotQueryAndMapping(t *testing.T) {
	rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(61, true, true, 2525))
	connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))

	got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
	if err != nil || !found {
		t.Fatal("unexpected job-application snapshot read result")
	}
	want := expectedJobApplicationSnapshot(61, true, true, 2525)
	if got != want {
		t.Fatal("job-application snapshot mapping mismatch")
	}
	if got.AccentColor != "" {
		t.Fatal("legacy AccentColor parity mismatch")
	}
	assertOptionsQuery(t, connector, wantJobApplicationWorkflowSnapshotSQL)
	assertRowsClosed(t, rowsPlan)
}

func TestOptionsRepositoryJobApplicationWorkflowSnapshotLogoAndNullableParity(t *testing.T) {
	t.Run("logo present", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(62, true, false, 587))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SiteLogoPath == "" {
			t.Fatal("joined site-logo path mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("logo NULL", func(t *testing.T) {
		values := jobApplicationSnapshotValues(63, true, false, 587)
		values[18] = nil
		rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SiteLogoPath != "" {
			t.Fatal("NULL site-logo path mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("all nullable workflow values", func(t *testing.T) {
		values := []any{int64(64), true, false}
		for range len(jobApplicationSnapshotColumns()) - len(values) {
			values = append(values, nil)
		}
		rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if err != nil || !found || got != (data.JobApplicationWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "64", IsActive: true}}) {
			t.Fatal("all-NULL workflow mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	for index := 3; index < len(jobApplicationSnapshotColumns()); index++ {
		t.Run("single nullable workflow value", func(t *testing.T) {
			values := jobApplicationSnapshotValues(65, true, false, 2525)
			values[index] = nil
			rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), values)
			_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
			want := expectedJobApplicationSnapshot(65, true, false, 2525)
			clearJobApplicationSnapshotColumn(&want, index)
			if err != nil || !found || got != want {
				t.Fatal("single NULL workflow mapping mismatch")
			}
			if got.AccentColor != "" {
				t.Fatal("NULL mapping changed legacy AccentColor parity")
			}
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestOptionsRepositoryJobApplicationWorkflowSnapshotMissingIdentityAndPortBoundaries(t *testing.T) {
	t.Run("missing active row", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns())
		connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if err != nil || found || got != (data.JobApplicationWorkflowSnapshot{}) {
			t.Fatal("missing job-application snapshot mismatch")
		}
		assertOptionsQuery(t, connector, wantJobApplicationWorkflowSnapshotSQL)
		assertRowsClosed(t, rowsPlan)
	})

	for _, optionID := range []int64{0, -1} {
		rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(optionID, true, false, 2525))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if got != (data.JobApplicationWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("invalid option identity returned workflow data")
		}
		assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: invalid identifier", nil)
		assertRowsClosed(t, rowsPlan)
	}

	rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(66, false, true, 2525))
	_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
	if got != (data.JobApplicationWorkflowSnapshot{}) || found || err == nil {
		t.Fatal("inactive selected row returned workflow data")
	}
	assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: selected row", nil)
	assertRowsClosed(t, rowsPlan)

	for _, port := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
		rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(67, true, false, port))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SMTPPort != port {
			t.Fatal("SMTP port boundary mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	}
}

func TestOptionsRepositoryJobApplicationWorkflowSnapshotMaxBytesBoundaries(t *testing.T) {
	for _, maxBytes := range []int64{1, 0, -1, math.MaxInt64, math.MinInt64} {
		values := jobApplicationSnapshotValues(74, true, false, 2525)
		values[19] = maxBytes
		rowsPlan := dbtest.NewRows(jobApplicationSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if err != nil || !found || got.MaxBytes != maxBytes {
			t.Fatal("max upload byte boundary mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	}
}

func TestOptionsRepositoryJobApplicationWorkflowSnapshotCardinalityAndFailures(t *testing.T) {
	backend := &repositoryBackendError{}
	tests := []struct {
		name      string
		rows      *dbtest.Rows
		wantStage string
	}{
		{
			name: "duplicate active rows",
			rows: dbtest.NewRows(jobApplicationSnapshotColumns(),
				jobApplicationSnapshotValues(68, true, false, 2525),
				jobApplicationSnapshotValues(69, true, false, 2525)),
			wantStage: "cardinality",
		},
		{
			name: "duplicate with malformed second row",
			rows: dbtest.NewRows(jobApplicationSnapshotColumns(),
				jobApplicationSnapshotValues(68, true, false, 2525),
				replaceJobApplicationSnapshotValue(jobApplicationSnapshotValues(69, true, false, 2525), 0, "invalid")),
			wantStage: "cardinality",
		},
		{name: "scan", rows: dbtest.NewRows(jobApplicationSnapshotColumns(), replaceJobApplicationSnapshotValue(jobApplicationSnapshotValues(70, true, false, 2525), 0, "invalid")), wantStage: "scan"},
		{name: "rows error", rows: dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(71, true, false, 2525)).WithErrorAfter(1, backend), wantStage: "rows/close"},
		{name: "close error", rows: dbtest.NewRows(jobApplicationSnapshotColumns()).WithCloseError(backend), wantStage: "rows/close"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.Query(test.rows))
			got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
			if got != (data.JobApplicationWorkflowSnapshot{}) || found || err == nil {
				t.Fatal("repository failure returned workflow data")
			}
			assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: "+test.wantStage, nil)
			assertOptionsQuery(t, connector, wantJobApplicationWorkflowSnapshotSQL)
			assertRowsClosed(t, test.rows)
		})
	}

	connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
	got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
	if got != (data.JobApplicationWorkflowSnapshot{}) || found || err == nil {
		t.Fatal("query failure returned workflow data")
	}
	assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: query", nil)
	if errors.Is(err, backend) {
		t.Fatal("backend error identity escaped")
	}
	assertOptionsQuery(t, connector, wantJobApplicationWorkflowSnapshotSQL)
}

func TestOptionsRepositoryJobApplicationWorkflowSnapshotSafeErrorsAndContext(t *testing.T) {
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
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if got != (data.JobApplicationWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("backend failure returned workflow data")
		}
		assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: query", nil)
		if errors.Is(err, backend) {
			t.Fatal("backend error identity escaped strict boundary")
		}
		assertOptionsQuery(t, connector, wantJobApplicationWorkflowSnapshotSQL)
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
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(ctx)
		if got != (data.JobApplicationWorkflowSnapshot{}) || found || !errors.Is(err, contextCase.want) {
			t.Fatal("completed caller context result mismatch")
		}
		assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: query", contextCase.want)
		if len(connector.Events()) != 0 || connector.Remaining() != 1 {
			t.Fatal("completed caller context reached driver")
		}
	}
}

func TestOptionsRepositoryJobApplicationWorkflowSnapshotCallerContextWinsBackendSentinel(t *testing.T) {
	t.Run("caller canceled wins backend deadline", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		entered := make(chan struct{}, 1)
		backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.DeadlineExceeded}
		connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		done := readJobApplicationWorkflowSnapshotAsync(repository, ctx)
		waitForJobApplicationSnapshotQueryBarrier(t, entered, done, cancel)
		cancel()
		result := waitForJobApplicationSnapshotResult(t, done, cancel)
		assertStrictJobApplicationSnapshotContextResult(t, result, context.Canceled, backend)
		assertOptionsQuery(t, connector, wantJobApplicationWorkflowSnapshotSQL)
	})

	t.Run("caller deadline wins backend canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			entered := make(chan struct{}, 1)
			backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.Canceled}
			connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
			done := readJobApplicationWorkflowSnapshotAsync(repository, ctx)
			waitForJobApplicationSnapshotQueryBarrier(t, entered, done, cancel)
			result := waitForJobApplicationSnapshotResult(t, done, cancel)
			assertStrictJobApplicationSnapshotContextResult(t, result, context.DeadlineExceeded, backend)
			assertOptionsQuery(t, connector, wantJobApplicationWorkflowSnapshotSQL)
		})
	})
}

func TestStrictJobApplicationWorkflowSnapshotErrorUnexpectedInput(t *testing.T) {
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
		err := strictJobApplicationWorkflowSnapshotError(ctx, test.source)
		cancel()
		assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: dependency", test.wantContext)
		if test.source != nil && errors.Is(err, test.source) {
			t.Fatal("unexpected error identity escaped")
		}
	}
}

func TestStrictJobApplicationWorkflowSnapshotCallerBackendContextMatrix(t *testing.T) {
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
			err := strictJobApplicationWorkflowSnapshotError(ctx, newRepositoryReadError(readJobApplicationWorkflowSnapshot, readQuery, backend))
			cancel()
			assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: query", contextCase.want)
			if errors.Is(err, backend) && backend != contextCase.want {
				t.Fatal("backend context identity crossed caller boundary")
			}
		}
	}
}

func TestOptionsRepositoryJobApplicationWorkflowSnapshotConstructorAndPoolReuse(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
		if got != (data.JobApplicationWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("nil dependency returned workflow data")
		}
		assertSafeRepositoryError(t, err, "job application workflow snapshot could not be read: dependency", nil)
	}

	first := dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(72, true, false, 2525))
	second := dbtest.NewRows(jobApplicationSnapshotColumns(), jobApplicationSnapshotValues(73, true, false, 465))
	connector, repository := openOptionsRepository(t, dbtest.Query(first), dbtest.Query(second))
	for index, wantPort := range []int64{2525, 465} {
		got, found, err := repository.ReadJobApplicationWorkflowSnapshot(context.Background())
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

type jobApplicationSnapshotReadResult struct {
	snapshot data.JobApplicationWorkflowSnapshot
	found    bool
	err      error
}

func readJobApplicationWorkflowSnapshotAsync(repository *OptionsRepository, ctx context.Context) <-chan jobApplicationSnapshotReadResult {
	done := make(chan jobApplicationSnapshotReadResult, 1)
	go func() {
		snapshot, found, err := repository.ReadJobApplicationWorkflowSnapshot(ctx)
		done <- jobApplicationSnapshotReadResult{snapshot: snapshot, found: found, err: err}
	}()
	return done
}

func waitForJobApplicationSnapshotQueryBarrier(t *testing.T, entered <-chan struct{}, done <-chan jobApplicationSnapshotReadResult, cleanup context.CancelFunc) {
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
		waitForJobApplicationSnapshotCleanup(t, done)
		t.Fatal("query barrier guard expired")
	}
}

func waitForJobApplicationSnapshotResult(t *testing.T, done <-chan jobApplicationSnapshotReadResult, cleanup context.CancelFunc) jobApplicationSnapshotReadResult {
	t.Helper()
	guard := time.NewTimer(5 * time.Second)
	defer stopSnapshotTestTimer(guard)
	select {
	case result := <-done:
		return result
	case <-guard.C:
		cleanup()
		waitForJobApplicationSnapshotCleanup(t, done)
		t.Fatal("operation result guard expired")
		return jobApplicationSnapshotReadResult{}
	}
}

func waitForJobApplicationSnapshotCleanup(t *testing.T, done <-chan jobApplicationSnapshotReadResult) {
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

func assertStrictJobApplicationSnapshotContextResult(t *testing.T, result jobApplicationSnapshotReadResult, wantContext error, backend *snapshotContextGateBackendError) {
	t.Helper()
	if result.snapshot != (data.JobApplicationWorkflowSnapshot{}) || result.found || result.err == nil {
		t.Fatal("strict context result mismatch")
	}
	assertSafeRepositoryError(t, result.err, "job application workflow snapshot could not be read: query", wantContext)
	if errors.Is(result.err, backend) {
		t.Fatal("backend identity escaped strict boundary")
	}
	var leaked *snapshotContextGateBackendError
	if errors.As(result.err, &leaked) {
		t.Fatal("backend type escaped strict boundary")
	}
}

func jobApplicationSnapshotColumns() []string {
	return []string{
		"oid", "option_set_is_active", "option_set_is_testing_now", "smtp_host", "smtp_port", "smtp_username",
		"smtp_password", "site_name", "site_description", "contact_email", "contact_phone", "facebook_url",
		"twitter_url", "instagram_url", "linkedin_url", "primary_color", "google_recaptcha_site_key",
		"google_recaptcha_secret_key", "logo_path", "max_upload_size",
	}
}

func jobApplicationSnapshotValues(id int64, active, testing bool, smtpPort int64) []any {
	return []any{
		id, active, testing, "smtp.internal.invalid", smtpPort, "mailer", "fixture-password", "Nivgoz",
		"Public description", "contact@example.invalid", "+90 000", "facebook", "twitter", "instagram", "linkedin",
		"primary", "fixture-site-key", "fixture-secret-key", "files/logo.webp", int64(1048576),
	}
}

func expectedJobApplicationSnapshot(id int64, active, testing bool, smtpPort int64) data.JobApplicationWorkflowSnapshot {
	return data.JobApplicationWorkflowSnapshot{
		Set:                data.OptionSetIdentity{ID: strconv.FormatInt(id, 10), IsActive: active, IsTesting: testing},
		SMTPHost:           "smtp.internal.invalid",
		SMTPPort:           smtpPort,
		SMTPUsername:       "mailer",
		SMTPPassword:       "fixture-password",
		SiteName:           "Nivgoz",
		SiteDescription:    "Public description",
		ContactEmail:       "contact@example.invalid",
		ContactPhone:       "+90 000",
		FacebookURL:        "facebook",
		TwitterURL:         "twitter",
		InstagramURL:       "instagram",
		LinkedInURL:        "linkedin",
		PrimaryColor:       "primary",
		RecaptchaSiteKey:   "fixture-site-key",
		RecaptchaSecretKey: "fixture-secret-key",
		SiteLogoPath:       "files/logo.webp",
		AccentColor:        "",
		MaxBytes:           1048576,
	}
}

func clearJobApplicationSnapshotColumn(snapshot *data.JobApplicationWorkflowSnapshot, columnIndex int) {
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
		snapshot.RecaptchaSiteKey = ""
	case 17:
		snapshot.RecaptchaSecretKey = ""
	case 18:
		snapshot.SiteLogoPath = ""
	case 19:
		snapshot.MaxBytes = 0
	}
}

func replaceJobApplicationSnapshotValue(values []any, index int, value any) []any {
	result := append([]any(nil), values...)
	result[index] = value
	return result
}
