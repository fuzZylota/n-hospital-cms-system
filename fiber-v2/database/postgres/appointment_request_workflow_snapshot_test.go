package postgres

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"database/postgres/internal/dbtest"
	"models/data"
)

const wantAppointmentRequestWorkflowSnapshotSQL = `SELECT
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
    o.secondary_color,
    o.google_recaptcha_site_key,
    o.google_recaptcha_secret_key,
    logo.file_path
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
WHERE o.option_set_is_active = TRUE`

func appointmentRequestColumns() []string {
	return []string{"oid", "option_set_is_active", "option_set_is_testing_now", "smtp_host", "smtp_port", "smtp_username", "smtp_password", "site_name", "site_description", "contact_email", "contact_phone", "facebook_url", "twitter_url", "instagram_url", "linkedin_url", "primary_color", "secondary_color", "google_recaptcha_site_key", "google_recaptcha_secret_key", "file_path"}
}

func appointmentRequestValues(id int64) []any {
	return []any{id, true, false, "smtp.invalid", int64(2525), "sender", "fixture-password", "site", "description", "contact@invalid", "phone", "facebook", "twitter", "instagram", "linkedin", "primary", "secondary", "site-key", "fixture-secret", "files/logo.png"}
}

func appointmentRequestExpected(id int64) data.AppointmentRequestWorkflowSnapshot {
	return data.AppointmentRequestWorkflowSnapshot{
		Set:      data.OptionSetIdentity{ID: strconv.FormatInt(id, 10), IsActive: true},
		SMTPHost: "smtp.invalid", SMTPPort: 2525, SMTPUsername: "sender", SMTPPassword: "fixture-password",
		SiteName: "site", SiteDescription: "description", ContactEmail: "contact@invalid", ContactPhone: "phone",
		FacebookURL: "facebook", TwitterURL: "twitter", InstagramURL: "instagram", LinkedInURL: "linkedin",
		PrimaryColor: "primary", SecondaryColor: "secondary", RecaptchaSiteKey: "site-key", RecaptchaSecretKey: "fixture-secret",
		SiteLogoPath: "files/logo.png",
	}
}

func TestAppointmentRequestWorkflowSnapshotProjectionAndScanOrder(t *testing.T) {
	if readAppointmentRequestWorkflowSnapshotSQL != wantAppointmentRequestWorkflowSnapshotSQL {
		t.Fatal("appointment SQL projection changed")
	}
	if strings.Contains(readAppointmentRequestWorkflowSnapshotSQL, "max_upload_size") || strings.Contains(readAppointmentRequestWorkflowSnapshotSQL, "accent_color") {
		t.Fatal("projection contains a value not selected/consumed by the legacy workflow")
	}
	rows := dbtest.NewRows(appointmentRequestColumns(), appointmentRequestValues(71))
	connector, repository := openOptionsRepository(t, dbtest.Query(rows))
	got, found, err := repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
	if err != nil || !found || got != appointmentRequestExpected(71) {
		t.Fatal("appointment snapshot scan order or mapping mismatch")
	}
	assertOptionsQuery(t, connector, wantAppointmentRequestWorkflowSnapshotSQL)
	assertRowsClosed(t, rows)
}

func TestAppointmentRequestWorkflowSnapshotNullableAndMissing(t *testing.T) {
	testingNull := appointmentRequestValues(72)
	testingNull[2] = nil
	rows := dbtest.NewRows(appointmentRequestColumns(), testingNull)
	_, repository := openOptionsRepository(t, dbtest.Query(rows))
	got, found, err := repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
	if err != nil || !found || got != appointmentRequestExpected(72) {
		t.Fatal("NULL testing flag did not map to false")
	}
	assertRowsClosed(t, rows)

	values := appointmentRequestValues(72)
	for index := 3; index < len(values); index++ {
		row := append([]any(nil), values...)
		row[index] = nil
		rows := dbtest.NewRows(appointmentRequestColumns(), row)
		_, repository := openOptionsRepository(t, dbtest.Query(rows))
		got, found, err := repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
		if err != nil || !found {
			t.Fatalf("nullable column %d rejected", index)
		}
		want := appointmentRequestExpected(72)
		// Every projected nullable column maps to exactly one field in order.
		field := reflect.ValueOf(&want).Elem().Field(index - 2)
		field.Set(reflect.Zero(field.Type()))
		if got != want {
			t.Fatalf("nullable column %d mapped to the wrong field", index)
		}
		assertRowsClosed(t, rows)
	}
	allNull := []any{int64(73), true, false}
	for len(allNull) < len(appointmentRequestColumns()) {
		allNull = append(allNull, nil)
	}
	rows = dbtest.NewRows(appointmentRequestColumns(), allNull)
	_, repository = openOptionsRepository(t, dbtest.Query(rows))
	got, found, err = repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
	if err != nil || !found || got != (data.AppointmentRequestWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "73", IsActive: true}}) {
		t.Fatal("all-NULL mapping mismatch")
	}
	assertRowsClosed(t, rows)

	rows = dbtest.NewRows(appointmentRequestColumns())
	_, repository = openOptionsRepository(t, dbtest.Query(rows))
	got, found, err = repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
	if err != nil || found || got != (data.AppointmentRequestWorkflowSnapshot{}) {
		t.Fatal("missing active row mismatch")
	}
	assertRowsClosed(t, rows)
}

func TestAppointmentRequestWorkflowSnapshotInvalidSelectionAndFailures(t *testing.T) {
	backend := &repositoryBackendError{}
	for _, tc := range []struct {
		name  string
		rows  *dbtest.Rows
		stage string
	}{
		{"zero identity", dbtest.NewRows(appointmentRequestColumns(), appointmentRequestValues(0)), "invalid identifier"},
		{"negative identity", dbtest.NewRows(appointmentRequestColumns(), appointmentRequestValues(-1)), "invalid identifier"},
		{"NULL identity", dbtest.NewRows(appointmentRequestColumns(), append([]any{nil}, appointmentRequestValues(74)[1:]...)), "scan"},
		{"NULL active flag", dbtest.NewRows(appointmentRequestColumns(), append([]any{int64(74), nil}, appointmentRequestValues(74)[2:]...)), "selected row"},
		{"inactive row", dbtest.NewRows(appointmentRequestColumns(), append([]any{int64(74), false}, appointmentRequestValues(74)[2:]...)), "selected row"},
		{"duplicate", dbtest.NewRows(appointmentRequestColumns(), appointmentRequestValues(74), appointmentRequestValues(75)), "cardinality"},
		{"malformed duplicate", dbtest.NewRows(appointmentRequestColumns(), appointmentRequestValues(74), append([]any{"bad"}, appointmentRequestValues(75)[1:]...)), "cardinality"},
		{"scan", dbtest.NewRows(appointmentRequestColumns(), append([]any{"bad"}, appointmentRequestValues(74)[1:]...)), "scan"},
		{"invalid port", dbtest.NewRows(appointmentRequestColumns(), func() []any { values := appointmentRequestValues(74); values[4] = "invalid"; return values }()), "scan"},
		{"rows error", dbtest.NewRows(appointmentRequestColumns(), appointmentRequestValues(74)).WithErrorAfter(1, backend), "rows/close"},
		{"close error", dbtest.NewRows(appointmentRequestColumns()).WithCloseError(backend), "rows/close"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.Query(tc.rows))
			got, found, err := repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
			if got != (data.AppointmentRequestWorkflowSnapshot{}) || found || err == nil {
				t.Fatal("failed read returned data")
			}
			assertSafeRepositoryError(t, err, "appointment request workflow snapshot could not be read: "+tc.stage, nil)
			assertOptionsQuery(t, connector, wantAppointmentRequestWorkflowSnapshotSQL)
			assertRowsClosed(t, tc.rows)
		})
	}
	connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
	got, found, err := repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
	if got != (data.AppointmentRequestWorkflowSnapshot{}) || found || err == nil || errors.Is(err, backend) {
		t.Fatal("backend error or data escaped")
	}
	assertSafeRepositoryError(t, err, "appointment request workflow snapshot could not be read: query", nil)
	assertOptionsQuery(t, connector, wantAppointmentRequestWorkflowSnapshotSQL)
}

func TestAppointmentRequestWorkflowSnapshotContextAndDependency(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
		if got != (data.AppointmentRequestWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("nil dependency returned data")
		}
		assertSafeRepositoryError(t, err, "appointment request workflow snapshot could not be read: dependency", nil)
	}
	canceled, stopCanceled := context.WithCancel(context.Background())
	stopCanceled()
	expired, stopExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	for _, tc := range []struct {
		ctx  context.Context
		stop context.CancelFunc
		want error
	}{
		{canceled, stopCanceled, context.Canceled},
		{expired, stopExpired, context.DeadlineExceeded},
	} {
		connector, repository := openOptionsRepository(t, dbtest.QueryUntilCanceled(nil))
		got, found, err := repository.ReadAppointmentRequestWorkflowSnapshot(tc.ctx)
		tc.stop()
		if got != (data.AppointmentRequestWorkflowSnapshot{}) || found || !errors.Is(err, tc.want) {
			t.Fatal("caller context not preserved")
		}
		assertSafeRepositoryError(t, err, "appointment request workflow snapshot could not be read: query", tc.want)
		if len(connector.Events()) != 0 {
			t.Fatal("completed context reached driver")
		}
	}
	for _, backend := range []error{context.Canceled, context.DeadlineExceeded, &repositoryBackendError{cause: context.Canceled}} {
		_, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		_, _, err := repository.ReadAppointmentRequestWorkflowSnapshot(context.Background())
		assertSafeRepositoryError(t, err, "appointment request workflow snapshot could not be read: query", nil)
		if errors.Is(err, backend) {
			t.Fatal("backend context identity escaped")
		}
	}
}
