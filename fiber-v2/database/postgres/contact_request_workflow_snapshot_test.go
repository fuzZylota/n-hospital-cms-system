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

const wantContactRequestWorkflowSnapshotSQL = `SELECT
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
    logo.file_path
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
WHERE o.option_set_is_active = TRUE`

var _ data.ContactRequestWorkflowSnapshotReader = (*OptionsRepository)(nil)

func TestOptionsRepositoryContactRequestWorkflowSnapshotQueryAndMapping(t *testing.T) {
	rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(61, true, true, 2525))
	connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))

	got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
	if err != nil || !found {
		t.Fatal("unexpected contact-request snapshot read result")
	}
	want := expectedContactRequestSnapshot(61, true, true, 2525)
	if got != want {
		t.Fatal("contact-request snapshot mapping mismatch")
	}
	if got.AccentColor != "" {
		t.Fatal("legacy AccentColor parity mismatch")
	}
	assertOptionsQuery(t, connector, wantContactRequestWorkflowSnapshotSQL)
	assertRowsClosed(t, rowsPlan)
}

func TestOptionsRepositoryContactRequestWorkflowSnapshotLogoAndNullableParity(t *testing.T) {
	t.Run("logo present", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(62, true, false, 587))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SiteLogoPath == "" {
			t.Fatal("joined site-logo path mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("logo NULL", func(t *testing.T) {
		values := contactRequestSnapshotValues(63, true, false, 587)
		values[len(values)-1] = nil
		rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SiteLogoPath != "" {
			t.Fatal("NULL site-logo path mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("all nullable workflow values", func(t *testing.T) {
		values := []any{int64(64), true, false}
		for range len(contactRequestSnapshotColumns()) - len(values) {
			values = append(values, nil)
		}
		rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if err != nil || !found || got != (data.ContactRequestWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "64", IsActive: true}}) {
			t.Fatal("all-NULL workflow mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})

	for index := 3; index < len(contactRequestSnapshotColumns()); index++ {
		t.Run("single nullable workflow value", func(t *testing.T) {
			values := contactRequestSnapshotValues(65, true, false, 2525)
			values[index] = nil
			rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), values)
			_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
			want := expectedContactRequestSnapshot(65, true, false, 2525)
			clearContactRequestSnapshotColumn(&want, index)
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

func TestOptionsRepositoryContactRequestWorkflowSnapshotMissingIdentityAndPortBoundaries(t *testing.T) {
	t.Run("missing active row", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns())
		connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if err != nil || found || got != (data.ContactRequestWorkflowSnapshot{}) {
			t.Fatal("missing contact-request snapshot mismatch")
		}
		assertOptionsQuery(t, connector, wantContactRequestWorkflowSnapshotSQL)
		assertRowsClosed(t, rowsPlan)
	})

	for _, optionID := range []int64{0, -1} {
		rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(optionID, true, false, 2525))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("invalid option identity returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: invalid identifier", nil)
		assertRowsClosed(t, rowsPlan)
	}

	rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(66, false, true, 2525))
	_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
	if got != (data.ContactRequestWorkflowSnapshot{}) || found || err == nil {
		t.Fatal("inactive selected row returned workflow data")
	}
	assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: selected row", nil)
	assertRowsClosed(t, rowsPlan)

	for _, port := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
		rowsPlan := dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(67, true, false, port))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if err != nil || !found || got.SMTPPort != port {
			t.Fatal("SMTP port boundary mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	}
}

func TestOptionsRepositoryContactRequestWorkflowSnapshotCardinalityAndFailures(t *testing.T) {
	backend := &repositoryBackendError{}
	tests := []struct {
		name      string
		rows      *dbtest.Rows
		wantStage string
	}{
		{
			name: "duplicate active rows",
			rows: dbtest.NewRows(contactRequestSnapshotColumns(),
				contactRequestSnapshotValues(68, true, false, 2525),
				contactRequestSnapshotValues(69, true, false, 2525)),
			wantStage: "cardinality",
		},
		{name: "scan", rows: dbtest.NewRows(contactRequestSnapshotColumns(), replaceContactRequestSnapshotValue(contactRequestSnapshotValues(70, true, false, 2525), 0, "invalid")), wantStage: "scan"},
		{name: "rows error", rows: dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(71, true, false, 2525)).WithErrorAfter(1, backend), wantStage: "rows/close"},
		{name: "close error", rows: dbtest.NewRows(contactRequestSnapshotColumns()).WithCloseError(backend), wantStage: "rows/close"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.Query(test.rows))
			got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
			if got != (data.ContactRequestWorkflowSnapshot{}) || found || err == nil {
				t.Fatal("repository failure returned workflow data")
			}
			assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: "+test.wantStage, nil)
			assertOptionsQuery(t, connector, wantContactRequestWorkflowSnapshotSQL)
			assertRowsClosed(t, test.rows)
		})
	}

	connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
	got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
	if got != (data.ContactRequestWorkflowSnapshot{}) || found || err == nil {
		t.Fatal("query failure returned workflow data")
	}
	assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: query", nil)
	if errors.Is(err, backend) {
		t.Fatal("backend error identity escaped")
	}
	assertOptionsQuery(t, connector, wantContactRequestWorkflowSnapshotSQL)
}

func TestOptionsRepositoryContactRequestWorkflowSnapshotSafeErrorsAndContext(t *testing.T) {
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
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("backend failure returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: query", nil)
		if errors.Is(err, backend) {
			t.Fatal("backend error identity escaped strict boundary")
		}
		assertOptionsQuery(t, connector, wantContactRequestWorkflowSnapshotSQL)
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
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(ctx)
		if got != (data.ContactRequestWorkflowSnapshot{}) || found || !errors.Is(err, contextCase.want) {
			t.Fatal("completed caller context result mismatch")
		}
		assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: query", contextCase.want)
		if len(connector.Events()) != 0 || connector.Remaining() != 1 {
			t.Fatal("completed caller context reached driver")
		}
	}
}

func TestOptionsRepositoryContactRequestWorkflowSnapshotCallerContextWinsBackendSentinel(t *testing.T) {
	t.Run("caller canceled wins backend deadline", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		entered := make(chan struct{}, 1)
		backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.DeadlineExceeded}
		connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		done := readContactRequestWorkflowSnapshotAsync(repository, ctx)
		waitForContactRequestSnapshotQueryBarrier(t, entered, done, cancel)
		cancel()
		result := waitForContactRequestSnapshotResult(t, done, cancel)
		assertStrictContactRequestSnapshotContextResult(t, result, context.Canceled, backend)
		assertOptionsQuery(t, connector, wantContactRequestWorkflowSnapshotSQL)
	})

	t.Run("caller deadline wins backend canceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			entered := make(chan struct{}, 1)
			backend := &snapshotContextGateBackendError{ctx: ctx, entered: entered, cause: context.Canceled}
			connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
			done := readContactRequestWorkflowSnapshotAsync(repository, ctx)
			waitForContactRequestSnapshotQueryBarrier(t, entered, done, cancel)
			result := waitForContactRequestSnapshotResult(t, done, cancel)
			assertStrictContactRequestSnapshotContextResult(t, result, context.DeadlineExceeded, backend)
			assertOptionsQuery(t, connector, wantContactRequestWorkflowSnapshotSQL)
		})
	})
}

func TestStrictContactRequestWorkflowSnapshotErrorUnexpectedInput(t *testing.T) {
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
		err := strictContactRequestWorkflowSnapshotError(ctx, test.source)
		cancel()
		assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: dependency", test.wantContext)
		if test.source != nil && errors.Is(err, test.source) {
			t.Fatal("unexpected error identity escaped")
		}
	}
}

func TestOptionsRepositoryContactRequestWorkflowSnapshotConstructorAndPoolReuse(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("nil dependency returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request workflow snapshot could not be read: dependency", nil)
	}

	first := dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(72, true, false, 2525))
	second := dbtest.NewRows(contactRequestSnapshotColumns(), contactRequestSnapshotValues(73, true, false, 465))
	connector, repository := openOptionsRepository(t, dbtest.Query(first), dbtest.Query(second))
	for index, wantPort := range []int64{2525, 465} {
		got, found, err := repository.ReadContactRequestWorkflowSnapshot(context.Background())
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

type contactRequestSnapshotReadResult struct {
	snapshot data.ContactRequestWorkflowSnapshot
	found    bool
	err      error
}

func readContactRequestWorkflowSnapshotAsync(repository *OptionsRepository, ctx context.Context) <-chan contactRequestSnapshotReadResult {
	done := make(chan contactRequestSnapshotReadResult, 1)
	go func() {
		snapshot, found, err := repository.ReadContactRequestWorkflowSnapshot(ctx)
		done <- contactRequestSnapshotReadResult{snapshot: snapshot, found: found, err: err}
	}()
	return done
}

func waitForContactRequestSnapshotQueryBarrier(t *testing.T, entered <-chan struct{}, done <-chan contactRequestSnapshotReadResult, cleanup context.CancelFunc) {
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
		waitForContactRequestSnapshotCleanup(t, done)
		t.Fatal("query barrier guard expired")
	}
}

func waitForContactRequestSnapshotResult(t *testing.T, done <-chan contactRequestSnapshotReadResult, cleanup context.CancelFunc) contactRequestSnapshotReadResult {
	t.Helper()
	guard := time.NewTimer(5 * time.Second)
	defer stopSnapshotTestTimer(guard)
	select {
	case result := <-done:
		return result
	case <-guard.C:
		cleanup()
		waitForContactRequestSnapshotCleanup(t, done)
		t.Fatal("operation result guard expired")
		return contactRequestSnapshotReadResult{}
	}
}

func waitForContactRequestSnapshotCleanup(t *testing.T, done <-chan contactRequestSnapshotReadResult) {
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

func assertStrictContactRequestSnapshotContextResult(t *testing.T, result contactRequestSnapshotReadResult, wantContext error, backend *snapshotContextGateBackendError) {
	t.Helper()
	if result.snapshot != (data.ContactRequestWorkflowSnapshot{}) || result.found || result.err == nil {
		t.Fatal("strict context result mismatch")
	}
	assertSafeRepositoryError(t, result.err, "contact request workflow snapshot could not be read: query", wantContext)
	if errors.Is(result.err, backend) {
		t.Fatal("backend identity escaped strict boundary")
	}
	var leaked *snapshotContextGateBackendError
	if errors.As(result.err, &leaked) {
		t.Fatal("backend type escaped strict boundary")
	}
}

func contactRequestSnapshotColumns() []string {
	return []string{
		"oid", "option_set_is_active", "option_set_is_testing_now", "smtp_host", "smtp_port", "smtp_username",
		"smtp_password", "site_name", "site_description", "contact_email", "contact_phone", "facebook_url",
		"twitter_url", "instagram_url", "linkedin_url", "primary_color", "google_recaptcha_site_key",
		"google_recaptcha_secret_key", "logo_path",
	}
}

func contactRequestSnapshotValues(id int64, active, testing bool, smtpPort int64) []any {
	return []any{
		id, active, testing, "smtp.internal.invalid", smtpPort, "mailer", "fixture-password", "Nivgoz",
		"Public description", "contact@example.invalid", "+90 000", "facebook", "twitter", "instagram", "linkedin",
		"primary", "fixture-site-key", "fixture-secret-key", "files/logo.webp",
	}
}

func expectedContactRequestSnapshot(id int64, active, testing bool, smtpPort int64) data.ContactRequestWorkflowSnapshot {
	return data.ContactRequestWorkflowSnapshot{
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
	}
}

func clearContactRequestSnapshotColumn(snapshot *data.ContactRequestWorkflowSnapshot, columnIndex int) {
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
	}
}

func replaceContactRequestSnapshotValue(values []any, index int, value any) []any {
	result := append([]any(nil), values...)
	result[index] = value
	return result
}
