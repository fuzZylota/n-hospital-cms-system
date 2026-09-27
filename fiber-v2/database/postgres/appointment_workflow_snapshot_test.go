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

const wantAppointmentWorkflowSnapshotSQL = `SELECT
    o.oid,
    o.option_set_is_active,
    o.option_set_is_testing_now,
    o.smtp_host,
    o.smtp_port,
    o.smtp_username,
    o.smtp_password,
    o.site_name,
    o.contact_email,
    o.contact_phone,
    o.primary_color,
    o.secondary_color,
    logo.file_path
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
WHERE o.option_set_is_active = TRUE`

func appointmentWorkflowColumns() []string {
	return []string{"oid", "option_set_is_active", "option_set_is_testing_now", "smtp_host", "smtp_port", "smtp_username", "smtp_password", "site_name", "contact_email", "contact_phone", "primary_color", "secondary_color", "file_path"}
}

func appointmentWorkflowValues(id int64) []any {
	return []any{id, true, false, "smtp.invalid", int64(2525), "sender", "fixture-password", "site", "contact@invalid", "phone", "primary", "secondary", "files/logo.png"}
}

func appointmentWorkflowExpected(id int64) data.AppointmentWorkflowSnapshot {
	return data.AppointmentWorkflowSnapshot{
		Set:      data.OptionSetIdentity{ID: strconv.FormatInt(id, 10), IsActive: true},
		SMTPHost: "smtp.invalid", SMTPPort: 2525, SMTPUsername: "sender", SMTPPassword: "fixture-password",
		SiteName: "site", ContactEmail: "contact@invalid", ContactPhone: "phone",
		PrimaryColor: "primary", SecondaryColor: "secondary", SiteLogoPath: "files/logo.png",
	}
}

func TestAppointmentWorkflowSnapshotProjectionAndScan(t *testing.T) {
	if readAppointmentWorkflowSnapshotSQL != wantAppointmentWorkflowSnapshotSQL {
		t.Fatal("appointment SQL projection changed")
	}
	for _, unused := range []string{"max_upload_size", "site_description", "facebook_url", "google_recaptcha", "accent_color", "ORDER BY", "LIMIT"} {
		if strings.Contains(readAppointmentWorkflowSnapshotSQL, unused) {
			t.Fatalf("unused or cardinality-changing SQL: %s", unused)
		}
	}
	rows := dbtest.NewRows(appointmentWorkflowColumns(), appointmentWorkflowValues(71))
	connector, repository := openOptionsRepository(t, dbtest.Query(rows))
	got, found, err := repository.ReadAppointmentWorkflowSnapshot(context.Background())
	if err != nil || !found || got != appointmentWorkflowExpected(71) {
		t.Fatal("appointment snapshot scan order or mapping mismatch")
	}
	assertOptionsQuery(t, connector, wantAppointmentWorkflowSnapshotSQL)
	assertRowsClosed(t, rows)
}

func TestAppointmentWorkflowSnapshotNullableAndMissing(t *testing.T) {
	values := appointmentWorkflowValues(72)
	values[2] = true
	rows := dbtest.NewRows(appointmentWorkflowColumns(), values)
	_, repository := openOptionsRepository(t, dbtest.Query(rows))
	got, found, err := repository.ReadAppointmentWorkflowSnapshot(context.Background())
	want := appointmentWorkflowExpected(72)
	want.Set.IsTesting = true
	if err != nil || !found || got != want {
		t.Fatal("active and testing identity flags were not retained independently")
	}
	assertRowsClosed(t, rows)

	values = appointmentWorkflowValues(72)
	values[2] = nil
	rows = dbtest.NewRows(appointmentWorkflowColumns(), values)
	_, repository = openOptionsRepository(t, dbtest.Query(rows))
	got, found, err = repository.ReadAppointmentWorkflowSnapshot(context.Background())
	if err != nil || !found || got != appointmentWorkflowExpected(72) {
		t.Fatal("NULL testing flag did not map to false")
	}
	assertRowsClosed(t, rows)

	for index := 3; index < len(values); index++ {
		row := appointmentWorkflowValues(72)
		row[index] = nil
		rows := dbtest.NewRows(appointmentWorkflowColumns(), row)
		_, repository := openOptionsRepository(t, dbtest.Query(rows))
		got, found, err := repository.ReadAppointmentWorkflowSnapshot(context.Background())
		if err != nil || !found {
			t.Fatalf("nullable column %d rejected", index)
		}
		want := appointmentWorkflowExpected(72)
		field := reflect.ValueOf(&want).Elem().Field(index - 2)
		field.Set(reflect.Zero(field.Type()))
		if got != want {
			t.Fatalf("nullable column %d mapped to the wrong field", index)
		}
		assertRowsClosed(t, rows)
	}

	allNull := []any{int64(73), true, false}
	for len(allNull) < len(appointmentWorkflowColumns()) {
		allNull = append(allNull, nil)
	}
	rows = dbtest.NewRows(appointmentWorkflowColumns(), allNull)
	_, repository = openOptionsRepository(t, dbtest.Query(rows))
	got, found, err = repository.ReadAppointmentWorkflowSnapshot(context.Background())
	if err != nil || !found || got != (data.AppointmentWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "73", IsActive: true}}) {
		t.Fatal("all-NULL mapping mismatch")
	}
	assertRowsClosed(t, rows)

	rows = dbtest.NewRows(appointmentWorkflowColumns())
	_, repository = openOptionsRepository(t, dbtest.Query(rows))
	got, found, err = repository.ReadAppointmentWorkflowSnapshot(context.Background())
	if err != nil || found || got != (data.AppointmentWorkflowSnapshot{}) {
		t.Fatal("missing active row mismatch")
	}
	assertRowsClosed(t, rows)
}

func TestAppointmentWorkflowSnapshotInvalidIdentityAndFailures(t *testing.T) {
	backend := &repositoryBackendError{}
	for _, tc := range []struct {
		name  string
		rows  *dbtest.Rows
		stage string
	}{
		{"zero identity", dbtest.NewRows(appointmentWorkflowColumns(), appointmentWorkflowValues(0)), "invalid identifier"},
		{"negative identity", dbtest.NewRows(appointmentWorkflowColumns(), appointmentWorkflowValues(-1)), "invalid identifier"},
		{"NULL identity", dbtest.NewRows(appointmentWorkflowColumns(), append([]any{nil}, appointmentWorkflowValues(74)[1:]...)), "scan"},
		{"NULL active flag", dbtest.NewRows(appointmentWorkflowColumns(), append([]any{int64(74), nil}, appointmentWorkflowValues(74)[2:]...)), "selected row"},
		{"inactive row", dbtest.NewRows(appointmentWorkflowColumns(), append([]any{int64(74), false}, appointmentWorkflowValues(74)[2:]...)), "selected row"},
		{"duplicate", dbtest.NewRows(appointmentWorkflowColumns(), appointmentWorkflowValues(74), appointmentWorkflowValues(75)), "cardinality"},
		{"malformed duplicate", dbtest.NewRows(appointmentWorkflowColumns(), appointmentWorkflowValues(74), append([]any{"bad"}, appointmentWorkflowValues(75)[1:]...)), "cardinality"},
		{"scan", dbtest.NewRows(appointmentWorkflowColumns(), append([]any{"bad"}, appointmentWorkflowValues(74)[1:]...)), "scan"},
		{"invalid port", dbtest.NewRows(appointmentWorkflowColumns(), func() []any { values := appointmentWorkflowValues(74); values[4] = "invalid"; return values }()), "scan"},
		{"rows error", dbtest.NewRows(appointmentWorkflowColumns(), appointmentWorkflowValues(74)).WithErrorAfter(1, backend), "rows/close"},
		{"close error", dbtest.NewRows(appointmentWorkflowColumns()).WithCloseError(backend), "rows/close"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.Query(tc.rows))
			got, found, err := repository.ReadAppointmentWorkflowSnapshot(context.Background())
			if got != (data.AppointmentWorkflowSnapshot{}) || found || err == nil {
				t.Fatal("failed read returned data")
			}
			assertSafeRepositoryError(t, err, "appointment workflow snapshot could not be read: "+tc.stage, nil)
			assertOptionsQuery(t, connector, wantAppointmentWorkflowSnapshotSQL)
			assertRowsClosed(t, tc.rows)
		})
	}
	connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
	got, found, err := repository.ReadAppointmentWorkflowSnapshot(context.Background())
	if got != (data.AppointmentWorkflowSnapshot{}) || found || err == nil || errors.Is(err, backend) {
		t.Fatal("backend error or data escaped")
	}
	assertSafeRepositoryError(t, err, "appointment workflow snapshot could not be read: query", nil)
	assertOptionsQuery(t, connector, wantAppointmentWorkflowSnapshotSQL)
}

func TestAppointmentWorkflowSnapshotContextAndDependency(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadAppointmentWorkflowSnapshot(context.Background())
		if got != (data.AppointmentWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("nil dependency returned data")
		}
		assertSafeRepositoryError(t, err, "appointment workflow snapshot could not be read: dependency", nil)
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
		got, found, err := repository.ReadAppointmentWorkflowSnapshot(tc.ctx)
		tc.stop()
		if got != (data.AppointmentWorkflowSnapshot{}) || found || !errors.Is(err, tc.want) {
			t.Fatal("caller context not preserved")
		}
		assertSafeRepositoryError(t, err, "appointment workflow snapshot could not be read: query", tc.want)
		if len(connector.Events()) != 0 {
			t.Fatal("completed context reached driver")
		}
	}
	for _, backend := range []error{context.Canceled, context.DeadlineExceeded, &repositoryBackendError{cause: context.Canceled}} {
		_, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		_, _, err := repository.ReadAppointmentWorkflowSnapshot(context.Background())
		assertSafeRepositoryError(t, err, "appointment workflow snapshot could not be read: query", nil)
		if errors.Is(err, backend) {
			t.Fatal("backend context identity escaped")
		}
	}
}
