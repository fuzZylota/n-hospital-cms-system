package postgres

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"database/postgres/internal/dbtest"
	"models/data"
)

const wantContactRequestResponseWorkflowSnapshotSQL = `SELECT
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

var _ data.ContactRequestResponseWorkflowSnapshotReader = (*OptionsRepository)(nil)

func TestOptionsRepositoryContactRequestResponseWorkflowSnapshotActiveSelectionAndMapping(t *testing.T) {
	for _, isTesting := range []bool{false, true} {
		t.Run("testing="+strconv.FormatBool(isTesting), func(t *testing.T) {
			rowsPlan := dbtest.NewRows(contactRequestResponseSnapshotColumns(), contactRequestResponseSnapshotValues(81, true, isTesting, 2525))
			connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))

			got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
			if err != nil || !found {
				t.Fatal("unexpected contact-request response snapshot read result")
			}
			want := expectedContactRequestResponseSnapshot(81, true, isTesting, 2525)
			if got != want {
				t.Fatal("contact-request response snapshot mapping mismatch")
			}
			assertOptionsQuery(t, connector, wantContactRequestResponseWorkflowSnapshotSQL)
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestOptionsRepositoryContactRequestResponseWorkflowSnapshotNoRowAndNullableValues(t *testing.T) {
	t.Run("no active row", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(contactRequestResponseSnapshotColumns())
		connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		if err != nil || found || got != (data.ContactRequestResponseWorkflowSnapshot{}) {
			t.Fatal("missing contact-request response snapshot mismatch")
		}
		assertOptionsQuery(t, connector, wantContactRequestResponseWorkflowSnapshotSQL)
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("nullable workflow values", func(t *testing.T) {
		values := []any{int64(82), true, false}
		for range len(contactRequestResponseSnapshotColumns()) - len(values) {
			values = append(values, nil)
		}
		rowsPlan := dbtest.NewRows(contactRequestResponseSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		want := data.ContactRequestResponseWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "82", IsActive: true}}
		if err != nil || !found || got != want {
			t.Fatal("nullable contact-request response mapping mismatch")
		}
		assertRowsClosed(t, rowsPlan)
	})
}

func TestOptionsRepositoryContactRequestResponseWorkflowSnapshotFailsClosed(t *testing.T) {
	t.Run("duplicate active rows", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(
			contactRequestResponseSnapshotColumns(),
			contactRequestResponseSnapshotValues(83, true, false, 2525),
			contactRequestResponseSnapshotValues(84, true, true, 2525),
		)
		connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("duplicate active rows returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request response workflow snapshot could not be read: cardinality", nil)
		assertOptionsQuery(t, connector, wantContactRequestResponseWorkflowSnapshotSQL)
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("inactive selected row", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(contactRequestResponseSnapshotColumns(), contactRequestResponseSnapshotValues(85, false, true, 2525))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("inactive selected row returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request response workflow snapshot could not be read: selected row", nil)
		assertRowsClosed(t, rowsPlan)
	})

	t.Run("invalid identity", func(t *testing.T) {
		rowsPlan := dbtest.NewRows(contactRequestResponseSnapshotColumns(), contactRequestResponseSnapshotValues(0, true, false, 2525))
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("invalid option identity returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request response workflow snapshot could not be read: invalid identifier", nil)
		assertRowsClosed(t, rowsPlan)
	})
}

func TestOptionsRepositoryContactRequestResponseWorkflowSnapshotQueryAndScanErrors(t *testing.T) {
	backend := &repositoryBackendError{}

	t.Run("query", func(t *testing.T) {
		connector, repository := openOptionsRepository(t, dbtest.QueryError(backend))
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("query failure returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request response workflow snapshot could not be read: query", nil)
		if errors.Is(err, backend) {
			t.Fatal("backend error identity escaped")
		}
		assertOptionsQuery(t, connector, wantContactRequestResponseWorkflowSnapshotSQL)
	})

	t.Run("scan", func(t *testing.T) {
		values := contactRequestResponseSnapshotValues(86, true, false, 2525)
		values[0] = "invalid"
		rowsPlan := dbtest.NewRows(contactRequestResponseSnapshotColumns(), values)
		_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("scan failure returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request response workflow snapshot could not be read: scan", nil)
		assertRowsClosed(t, rowsPlan)
	})
}

func TestOptionsRepositoryContactRequestResponseWorkflowSnapshotDependencies(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		if got != (data.ContactRequestResponseWorkflowSnapshot{}) || found || err == nil {
			t.Fatal("missing dependency returned workflow data")
		}
		assertSafeRepositoryError(t, err, "contact request response workflow snapshot could not be read: dependency", nil)
	}
}

func contactRequestResponseSnapshotColumns() []string {
	return []string{
		"oid", "option_set_is_active", "option_set_is_testing_now", "smtp_host", "smtp_port", "smtp_username",
		"smtp_password", "site_name", "site_description", "contact_email", "contact_phone", "facebook_url",
		"twitter_url", "instagram_url", "linkedin_url", "primary_color", "logo_path",
	}
}

func contactRequestResponseSnapshotValues(id int64, active, testing bool, smtpPort int64) []any {
	return []any{
		id, active, testing, "smtp.internal.invalid", smtpPort, "mailer", "fixture-password", "Nivgoz",
		"Public description", "contact@example.invalid", "+90 000", "facebook", "twitter", "instagram", "linkedin",
		"primary", "files/logo.webp",
	}
}

func expectedContactRequestResponseSnapshot(id int64, active, testing bool, smtpPort int64) data.ContactRequestResponseWorkflowSnapshot {
	return data.ContactRequestResponseWorkflowSnapshot{
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
