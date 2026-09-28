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

var (
	_ data.SiteOptionsReader                = (*OptionsRepository)(nil)
	_ data.UploadPolicyReader               = (*OptionsRepository)(nil)
	_ data.PasswordPolicyReader             = (*OptionsRepository)(nil)
	_ data.MailDeliveryOptionsReader        = (*OptionsRepository)(nil)
	_ data.CaptchaVerificationOptionsReader = (*OptionsRepository)(nil)
	_ data.OptionMediaReferencesReader      = (*OptionsRepository)(nil)
)

const wantActiveSiteOptionsSQL = `SELECT
    o.oid,
    o.option_set_is_active,
    o.option_set_is_testing_now,
    o.site_name,
    o.site_description,
    o.maintenance_mode,
    o.preloader,
    o.facebook_url,
    o.twitter_url,
    o.instagram_url,
    o.linkedin_url,
    o.contact_email,
    o.contact_phone,
    o.main_page_meta_title,
    o.main_page_meta_description,
    o.google_analytics,
    o.primary_color,
    o.secondary_color,
    o.accent_color,
    o.background_color,
    o.font_color,
    o.font_family,
    o.items_per_page,
    o.enable_testimonials,
    o.maximum_sublinks_on_a_menu_item,
    o.show_doctor_social_media,
    o.show_doctor_appointment_fee,
    o.show_anlasmali_kurum_pictures,
    o.google_recaptcha_site_key,
    logo.file_path,
    logo.alt_text,
    logo.title,
    light_logo.file_path,
    light_logo.alt_text,
    light_logo.title,
    favicon.file_path,
    default_media.file_path,
    default_media.alt_text,
    default_media.title
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
LEFT JOIN medias light_logo ON o.site_light_logo_mid = light_logo.mid
LEFT JOIN medias favicon ON o.site_favicon_mid = favicon.mid
LEFT JOIN medias default_media ON o.default_page_mid = default_media.mid
WHERE o.option_set_is_active = TRUE`

const wantTestingSiteOptionsSQL = `SELECT
    o.oid,
    o.option_set_is_active,
    o.option_set_is_testing_now,
    o.site_name,
    o.site_description,
    o.maintenance_mode,
    o.preloader,
    o.facebook_url,
    o.twitter_url,
    o.instagram_url,
    o.linkedin_url,
    o.contact_email,
    o.contact_phone,
    o.main_page_meta_title,
    o.main_page_meta_description,
    o.google_analytics,
    o.primary_color,
    o.secondary_color,
    o.accent_color,
    o.background_color,
    o.font_color,
    o.font_family,
    o.items_per_page,
    o.enable_testimonials,
    o.maximum_sublinks_on_a_menu_item,
    o.show_doctor_social_media,
    o.show_doctor_appointment_fee,
    o.show_anlasmali_kurum_pictures,
    o.google_recaptcha_site_key,
    logo.file_path,
    logo.alt_text,
    logo.title,
    light_logo.file_path,
    light_logo.alt_text,
    light_logo.title,
    favicon.file_path,
    default_media.file_path,
    default_media.alt_text,
    default_media.title
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
LEFT JOIN medias light_logo ON o.site_light_logo_mid = light_logo.mid
LEFT JOIN medias favicon ON o.site_favicon_mid = favicon.mid
LEFT JOIN medias default_media ON o.default_page_mid = default_media.mid
WHERE o.option_set_is_testing_now = TRUE`

func TestOptionsRepositorySiteOptionsQueriesAndMapping(t *testing.T) {
	for _, test := range []struct {
		name      string
		selection data.OptionSetSelection
		active    bool
		testing   bool
		wantSQL   string
	}{
		{name: "active", selection: data.ActiveOptionSet, active: true, wantSQL: wantActiveSiteOptionsSQL},
		{name: "testing", selection: data.TestingOptionSet, testing: true, wantSQL: wantTestingSiteOptionsSQL},
		{name: "active and testing flags coexist", selection: data.ActiveOptionSet, active: true, testing: true, wantSQL: wantActiveSiteOptionsSQL},
	} {
		t.Run(test.name, func(t *testing.T) {
			rowsPlan := dbtest.NewRows(siteOptionsColumns(), siteOptionsValues(17, test.active, test.testing))
			connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadSiteOptions(context.Background(), test.selection)
			if err != nil || !found {
				t.Fatal("unexpected site options read result")
			}
			want := expectedSiteOptions(test.active, test.testing)
			if !reflect.DeepEqual(got, want) {
				t.Fatal("site options mapping mismatch")
			}
			assertOptionsQuery(t, connector, test.wantSQL)
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestOptionsRepositoryInvalidSelectionDoesNotQuery(t *testing.T) {
	for _, selection := range []data.OptionSetSelection{0, 3, 255} {
		connector, repository := openOptionsRepository(t, dbtest.Query(dbtest.NewRows(siteOptionsColumns())))
		got, found, err := repository.ReadSiteOptions(context.Background(), selection)
		if got != (data.SiteOptions{}) || found || err == nil {
			t.Fatal("invalid selection returned data or no error")
		}
		assertSafeRepositoryError(t, err, "site options could not be read: invalid selection", nil)
		if len(connector.Events()) != 0 || connector.Remaining() != 1 {
			t.Fatal("invalid selection reached database")
		}
	}
}

func TestOptionsRepositoryTestingMissingHasNoFallback(t *testing.T) {
	rowsPlan := dbtest.NewRows(siteOptionsColumns())
	connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadSiteOptions(context.Background(), data.TestingOptionSet)
	if got != (data.SiteOptions{}) || found || err != nil {
		t.Fatal("unexpected missing testing result")
	}
	assertOptionsQuery(t, connector, wantTestingSiteOptionsSQL)
	assertRowsClosed(t, rowsPlan)
}

func TestOptionsRepositorySiteOptionsNullsMapToZeroValues(t *testing.T) {
	values := []any{int64(23), true, false}
	for len(values) < len(siteOptionsColumns()) {
		values = append(values, nil)
	}
	rowsPlan := dbtest.NewRows(siteOptionsColumns(), values)
	connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
	want := data.SiteOptions{Set: data.OptionSetIdentity{ID: "23", IsActive: true}}
	if err != nil || !found || !reflect.DeepEqual(got, want) {
		t.Fatal("NULL mapping mismatch")
	}
	assertOptionsQuery(t, connector, wantActiveSiteOptionsSQL)
	assertRowsClosed(t, rowsPlan)
}

func TestOptionsRepositorySiteOptionsCardinalityAndSelectedFlag(t *testing.T) {
	for _, test := range []struct {
		name   string
		values [][]any
		stage  string
	}{
		{name: "multiple active rows", values: [][]any{siteOptionsValues(1, true, false), siteOptionsValues(2, true, false)}, stage: "cardinality"},
		{name: "active flag mismatch", values: [][]any{siteOptionsValues(1, false, true)}, stage: "selected row"},
		{name: "invalid database identifier", values: [][]any{siteOptionsValues(0, true, false)}, stage: "invalid identifier"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rowsPlan := dbtest.NewRows(siteOptionsColumns(), test.values...)
			connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
			if got != (data.SiteOptions{}) || found || err == nil {
				t.Fatal("failure returned data or no error")
			}
			assertSafeRepositoryError(t, err, "site options could not be read: "+test.stage, nil)
			assertOptionsQuery(t, connector, wantActiveSiteOptionsSQL)
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestOptionsRepositoryTestingFlagMismatch(t *testing.T) {
	rowsPlan := dbtest.NewRows(siteOptionsColumns(), siteOptionsValues(9, true, false))
	connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadSiteOptions(context.Background(), data.TestingOptionSet)
	if got != (data.SiteOptions{}) || found || err == nil {
		t.Fatal("testing flag mismatch returned data or no error")
	}
	assertSafeRepositoryError(t, err, "site options could not be read: selected row", nil)
	assertOptionsQuery(t, connector, wantTestingSiteOptionsSQL)
}

func TestOptionsRepositoryInternalReaders(t *testing.T) {
	type internalCase struct {
		name    string
		wantSQL string
		columns []string
		values  []any
		invoke  func(*testing.T, *OptionsRepository) (bool, error)
	}
	tests := []internalCase{
		{
			name: "upload policy", wantSQL: `SELECT oid, option_set_is_active, option_set_is_testing_now, max_upload_size
FROM options
WHERE option_set_is_active = TRUE`,
			columns: []string{"oid", "option_set_is_active", "option_set_is_testing_now", "max_upload_size"},
			values:  []any{int64(31), true, true, int64(5242880)},
			invoke: func(t *testing.T, repository *OptionsRepository) (bool, error) {
				got, found, err := repository.ReadUploadPolicy(context.Background())
				if err == nil {
					if found && !(got.Set.ID == "31" && got.Set.IsActive && got.Set.IsTesting && got.MaxBytes == 5242880) {
						t.Fatal("upload policy mapping mismatch")
					}
					if !found && got != (data.UploadPolicy{}) {
						t.Fatal("empty upload policy was not zero")
					}
				} else if got != (data.UploadPolicy{}) {
					t.Fatal("failed upload policy read returned partial data")
				}
				return found, err
			},
		},
		{
			name: "password policy", wantSQL: `SELECT oid, option_set_is_active, option_set_is_testing_now, require_strong_password
FROM options
WHERE option_set_is_active = TRUE`,
			columns: []string{"oid", "option_set_is_active", "option_set_is_testing_now", "require_strong_password"},
			values:  []any{int64(31), true, false, true},
			invoke: func(t *testing.T, repository *OptionsRepository) (bool, error) {
				got, found, err := repository.ReadPasswordPolicy(context.Background())
				if err == nil {
					if found && !(got.Set.ID == "31" && got.Set.IsActive && !got.Set.IsTesting && got.RequireStrong) {
						t.Fatal("password policy mapping mismatch")
					}
					if !found && got != (data.PasswordPolicy{}) {
						t.Fatal("empty password policy was not zero")
					}
				} else if got != (data.PasswordPolicy{}) {
					t.Fatal("failed password policy read returned partial data")
				}
				return found, err
			},
		},
		{
			name: "mail delivery options", wantSQL: `SELECT oid, option_set_is_active, option_set_is_testing_now, smtp_host, smtp_port, smtp_username, smtp_password
FROM options
WHERE option_set_is_active = TRUE`,
			columns: []string{"oid", "option_set_is_active", "option_set_is_testing_now", "smtp_host", "smtp_port", "smtp_username", "smtp_password"},
			values:  []any{int64(31), true, false, "mail.invalid", int64(2525), "synthetic-user", "synthetic-password"},
			invoke: func(t *testing.T, repository *OptionsRepository) (bool, error) {
				got, found, err := repository.ReadMailDeliveryOptions(context.Background())
				if err == nil {
					if found && !(got.Set.ID == "31" && got.Set.IsActive && got.Host == "mail.invalid" && got.Port == 2525 && got.Username == "synthetic-user" && got.Password == "synthetic-password") {
						t.Fatal("mail delivery options mapping mismatch")
					}
					if !found && got != (data.MailDeliveryOptions{}) {
						t.Fatal("empty mail delivery options were not zero")
					}
				} else if got != (data.MailDeliveryOptions{}) {
					t.Fatal("failed mail delivery options read returned partial data")
				}
				return found, err
			},
		},
		{
			name: "captcha verification options", wantSQL: `SELECT oid, option_set_is_active, option_set_is_testing_now, google_recaptcha_site_key, google_recaptcha_secret_key
FROM options
WHERE option_set_is_active = TRUE`,
			columns: []string{"oid", "option_set_is_active", "option_set_is_testing_now", "google_recaptcha_site_key", "google_recaptcha_secret_key"},
			values:  []any{int64(31), true, false, "synthetic-site-key", "synthetic-secret-key"},
			invoke: func(t *testing.T, repository *OptionsRepository) (bool, error) {
				got, found, err := repository.ReadCaptchaVerificationOptions(context.Background())
				if err == nil {
					if found && !(got.Set.ID == "31" && got.Set.IsActive && got.SiteKey == "synthetic-site-key" && got.SecretKey == "synthetic-secret-key") {
						t.Fatal("captcha verification options mapping mismatch")
					}
					if !found && got != (data.CaptchaVerificationOptions{}) {
						t.Fatal("empty captcha verification options were not zero")
					}
				} else if got != (data.CaptchaVerificationOptions{}) {
					t.Fatal("failed captcha verification options read returned partial data")
				}
				return found, err
			},
		},
		{
			name: "option media references", wantSQL: `SELECT oid, option_set_is_active, option_set_is_testing_now, site_logo_mid, site_light_logo_mid, site_favicon_mid, default_page_mid
FROM options
WHERE option_set_is_active = TRUE`,
			columns: []string{"oid", "option_set_is_active", "option_set_is_testing_now", "site_logo_mid", "site_light_logo_mid", "site_favicon_mid", "default_page_mid"},
			values:  []any{int64(31), true, false, int64(4), nil, int64(6), int64(7)},
			invoke: func(t *testing.T, repository *OptionsRepository) (bool, error) {
				got, found, err := repository.ReadOptionMediaReferences(context.Background())
				if err == nil {
					if found && !(got.Set.ID == "31" && got.Set.IsActive && stringPointerEquals(got.SiteLogoID, "4") && got.SiteLightLogoID == nil && stringPointerEquals(got.FaviconID, "6") && stringPointerEquals(got.DefaultPageMediaID, "7")) {
						t.Fatal("option media references mapping mismatch")
					}
					if !found && got != (data.OptionMediaReferences{}) {
						t.Fatal("empty option media references were not zero")
					}
				} else if got != (data.OptionMediaReferences{}) {
					t.Fatal("failed option media references read returned partial data")
				}
				return found, err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name+"/normal", func(t *testing.T) {
			rowsPlan := dbtest.NewRows(test.columns, test.values)
			connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			found, err := test.invoke(t, repository)
			if err != nil || !found {
				t.Fatal("unexpected normal read result")
			}
			assertOptionsQuery(t, connector, test.wantSQL)
			assertRowsClosed(t, rowsPlan)
		})
		t.Run(test.name+"/empty", func(t *testing.T) {
			rowsPlan := dbtest.NewRows(test.columns)
			connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			found, err := test.invoke(t, repository)
			if err != nil || found {
				t.Fatal("unexpected empty read result")
			}
			assertOptionsQuery(t, connector, test.wantSQL)
			assertRowsClosed(t, rowsPlan)
		})
		t.Run(test.name+"/query error", func(t *testing.T) {
			backendErr := &repositoryBackendError{}
			connector, repository := openOptionsRepository(t, dbtest.QueryError(backendErr))
			found, err := test.invoke(t, repository)
			if found || err == nil {
				t.Fatal("unexpected query error result")
			}
			assertSafeRepositoryError(t, err, operationErrorPrefix(test.name)+"query", nil)
			if errors.Is(err, backendErr) {
				t.Fatal("backend error identity escaped")
			}
			assertOptionsQuery(t, connector, test.wantSQL)
		})
		t.Run(test.name+"/cardinality", func(t *testing.T) {
			rowsPlan := dbtest.NewRows(test.columns, test.values, test.values)
			connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			found, err := test.invoke(t, repository)
			if found || err == nil {
				t.Fatal("unexpected cardinality result")
			}
			assertSafeRepositoryError(t, err, operationErrorPrefix(test.name)+"cardinality", nil)
			assertOptionsQuery(t, connector, test.wantSQL)
			assertRowsClosed(t, rowsPlan)
		})
		t.Run(test.name+"/inactive selected row", func(t *testing.T) {
			values := append([]any(nil), test.values...)
			values[1] = false
			rowsPlan := dbtest.NewRows(test.columns, values)
			connector, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			found, err := test.invoke(t, repository)
			if found || err == nil {
				t.Fatal("unexpected selected-row result")
			}
			assertSafeRepositoryError(t, err, operationErrorPrefix(test.name)+"selected row", nil)
			assertOptionsQuery(t, connector, test.wantSQL)
			assertRowsClosed(t, rowsPlan)
		})
	}
}

func TestOptionsRepositoryInternalReaderRejectsBadActiveIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []any
		stage  string
	}{
		{name: "inactive selected row", values: []any{int64(3), false, true, int64(10)}, stage: "selected row"},
		{name: "zero option id", values: []any{int64(0), true, false, int64(10)}, stage: "invalid identifier"},
		{name: "non-positive option id", values: []any{int64(-3), true, false, int64(10)}, stage: "invalid identifier"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rowsPlan := dbtest.NewRows([]string{"oid", "active", "testing", "max"}, test.values)
			_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
			got, found, err := repository.ReadUploadPolicy(context.Background())
			if got != (data.UploadPolicy{}) || found || err == nil {
				t.Fatal("invalid active identity returned data or no error")
			}
			assertSafeRepositoryError(t, err, "upload policy could not be read: "+test.stage, nil)
		})
	}
}

func TestOptionsRepositoryMediaReferencesRejectNonPositiveID(t *testing.T) {
	rowsPlan := dbtest.NewRows(
		[]string{"oid", "active", "testing", "logo", "light", "favicon", "default"},
		[]any{int64(3), true, false, int64(0), nil, nil, nil},
	)
	_, repository := openOptionsRepository(t, dbtest.Query(rowsPlan))
	got, found, err := repository.ReadOptionMediaReferences(context.Background())
	if got != (data.OptionMediaReferences{}) || found || err == nil {
		t.Fatal("invalid media identifier returned data or no error")
	}
	assertSafeRepositoryError(t, err, "option media references could not be read: invalid identifier", nil)
}

func TestOptionsRepositoryFailureStagesAndPriority(t *testing.T) {
	backend := &repositoryBackendError{}
	for _, test := range []struct {
		name        string
		rows        *dbtest.Rows
		wantStage   string
		wantContext error
	}{
		{name: "scan first row", rows: dbtest.NewRows(siteOptionsColumns(), replaceSiteValue(siteOptionsValues(1, true, false), 0, "bad-id")), wantStage: "scan"},
		{name: "scan after successful row", rows: dbtest.NewRows(siteOptionsColumns(), siteOptionsValues(1, true, false), replaceSiteValue(siteOptionsValues(2, true, false), 0, "bad-id")), wantStage: "scan"},
		{name: "iteration", rows: dbtest.NewRows(siteOptionsColumns(), siteOptionsValues(1, true, false)).WithErrorAfter(1, backend), wantStage: "rows/close"},
		{name: "close", rows: dbtest.NewRows(siteOptionsColumns(), siteOptionsValues(1, true, false)).WithCloseError(backend), wantStage: "rows/close"},
		{name: "scan wins over close", rows: dbtest.NewRows(siteOptionsColumns(), replaceSiteValue(siteOptionsValues(1, true, false), 0, "bad-id")).WithCloseError(&repositoryBackendError{cause: context.DeadlineExceeded}), wantStage: "scan"},
		{name: "iteration context wins over close", rows: dbtest.NewRows(siteOptionsColumns(), siteOptionsValues(1, true, false)).WithErrorAfter(1, &repositoryBackendError{cause: context.Canceled}).WithCloseError(&repositoryBackendError{cause: context.DeadlineExceeded}), wantStage: "rows/close", wantContext: context.Canceled},
		{name: "close context", rows: dbtest.NewRows(siteOptionsColumns()).WithCloseError(&repositoryBackendError{cause: context.DeadlineExceeded}), wantStage: "rows/close", wantContext: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.Query(test.rows))
			got, found, err := repository.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
			if got != (data.SiteOptions{}) || found || err == nil {
				t.Fatal("failure returned data or no error")
			}
			assertSafeRepositoryError(t, err, "site options could not be read: "+test.wantStage, test.wantContext)
			assertOptionsQuery(t, connector, wantActiveSiteOptionsSQL)
			assertRowsClosed(t, test.rows)
		})
	}
}

func TestOptionsRepositoryQueryErrorsPreserveOnlyContextIdentity(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "backend", err: &repositoryBackendError{}},
		{name: "canceled", err: context.Canceled, want: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: context.DeadlineExceeded},
		{name: "wrapped canceled", err: &repositoryBackendError{cause: context.Canceled}, want: context.Canceled},
		{name: "same cancellation text", err: errors.New(context.Canceled.Error())},
		{name: "same deadline text", err: errors.New(context.DeadlineExceeded.Error())},
	} {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openOptionsRepository(t, dbtest.QueryError(test.err))
			got, found, err := repository.ReadSiteOptions(context.Background(), data.ActiveOptionSet)
			if got != (data.SiteOptions{}) || found || err == nil {
				t.Fatal("query failure returned data or no error")
			}
			assertSafeRepositoryError(t, err, "site options could not be read: query", test.want)
			assertOptionsQuery(t, connector, wantActiveSiteOptionsSQL)
		})
	}
}

func TestOptionsRepositoryContextBeforeAndAfterDriverEntry(t *testing.T) {
	t.Run("pre-canceled", func(t *testing.T) {
		connector, repository := openOptionsRepository(t, dbtest.QueryUntilCanceled(nil))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, found, err := repository.ReadSiteOptions(ctx, data.ActiveOptionSet)
		if found || !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected pre-canceled result")
		}
		if len(connector.Events()) != 0 || connector.Remaining() != 1 {
			t.Fatal("pre-canceled request reached driver")
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
				connector, repository := openOptionsRepository(t, dbtest.QueryUntilCanceled(entered))
				ctx, cancel := test.makeCtx()
				defer cancel()
				done := make(chan error, 1)
				go func() {
					_, _, err := repository.ReadSiteOptions(ctx, data.ActiveOptionSet)
					done <- err
				}()
				select {
				case <-entered:
				case err := <-done:
					if err == nil {
						t.Fatal("query returned without an error before entry handshake")
					}
					t.Fatal("query returned before entry handshake")
				}
				if ctx.Err() != nil || connector.Remaining() != 0 {
					t.Fatal("context ended before driver entry")
				}
				test.finish(cancel)
				err := <-done
				assertSafeRepositoryError(t, err, "site options could not be read: query", test.want)
				assertOptionsQuery(t, connector, wantActiveSiteOptionsSQL)
			})
		})
	}
}

func TestOptionsRepositoryNilDependencyAndPoolReuse(t *testing.T) {
	for _, repository := range []*OptionsRepository{nil, NewOptionsRepository(nil)} {
		got, found, err := repository.ReadUploadPolicy(context.Background())
		if got != (data.UploadPolicy{}) || found || err == nil {
			t.Fatal("nil dependency returned data or no error")
		}
		assertSafeRepositoryError(t, err, "upload policy could not be read: dependency", nil)
	}

	first := dbtest.NewRows([]string{"oid", "active", "testing", "max"}, []any{int64(1), true, false, int64(10)})
	second := dbtest.NewRows([]string{"oid", "active", "testing", "max"}, []any{int64(2), true, false, int64(20)})
	connector, repository := openOptionsRepository(t, dbtest.Query(first), dbtest.Query(second))
	for _, want := range []struct {
		id  string
		max int64
	}{{"1", 10}, {"2", 20}} {
		got, found, err := repository.ReadUploadPolicy(context.Background())
		if err != nil || !found || got.Set.ID != want.id || got.MaxBytes != want.max {
			t.Fatal("pool reuse mismatch")
		}
	}
	if len(connector.Events()) != 2 || connector.Remaining() != 0 {
		t.Fatal("repository did not complete two queries on the borrowed pool")
	}
	assertRowsClosed(t, first)
	assertRowsClosed(t, second)
}

func TestOptionsRepositoryUploadPolicyReadsOnSuppliedTransaction(t *testing.T) {
	columns := []string{"oid", "option_set_is_active", "option_set_is_testing_now", "max_upload_size"}
	wantSQL := `SELECT oid, option_set_is_active, option_set_is_testing_now, max_upload_size
FROM options
WHERE option_set_is_active = TRUE
LIMIT 1`
	backendErr := &repositoryBackendError{}
	for _, test := range []struct {
		name      string
		query     dbtest.Step
		wantFound bool
		wantID    string
		wantBytes int64
		wantStage string
	}{
		{"positive", dbtest.Query(dbtest.NewRows(columns, []any{int64(31), true, false, int64(5242880)})), true, "31", 5242880, ""},
		{"zero identifier", dbtest.Query(dbtest.NewRows(columns, []any{int64(0), true, false, int64(5242880)})), true, "0", 5242880, ""},
		{"zero byte limit", dbtest.Query(dbtest.NewRows(columns, []any{int64(31), true, false, int64(0)})), true, "31", 0, ""},
		{"negative byte limit", dbtest.Query(dbtest.NewRows(columns, []any{int64(31), true, false, int64(-1)})), true, "31", -1, ""},
		{"inactive zero identifier", dbtest.Query(dbtest.NewRows(columns, []any{int64(0), false, false, int64(5242880)})), false, "", 0, "invalid identifier"},
		{"negative identifier", dbtest.Query(dbtest.NewRows(columns, []any{int64(-1), true, false, int64(5242880)})), false, "", 0, "invalid identifier"},
		{"missing", dbtest.Query(dbtest.NewRows(columns)), false, "", 0, ""},
		{"query error", dbtest.QueryError(backendErr), false, "", 0, "query"},
	} {
		t.Run(test.name, func(t *testing.T) {
			poolConnector, repository := openOptionsRepository(t)
			txConnector := dbtest.NewConnector(dbtest.Begin(), test.query, dbtest.Rollback())
			db := sql.OpenDB(txConnector)
			t.Cleanup(func() { _ = db.Close() })
			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal("cannot begin synthetic transaction")
			}
			got, found, err := repository.ReadUploadPolicyTx(context.Background(), tx)
			if test.wantStage != "" {
				if got != (data.UploadPolicy{}) || found {
					t.Fatal("failed transaction read returned partial data")
				}
				assertSafeRepositoryError(t, err, "upload policy could not be read: "+test.wantStage, nil)
			} else if err != nil || found != test.wantFound || got.MaxBytes != test.wantBytes {
				t.Fatal("transaction policy mapping changed")
			} else if found && (got.Set.ID != test.wantID || !got.Set.IsActive || got.Set.IsTesting) {
				t.Fatal("transaction policy selected the wrong active set")
			} else if !found && got != (data.UploadPolicy{}) {
				t.Fatal("missing transaction policy returned partial data")
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal("cannot roll back synthetic transaction")
			}
			if len(poolConnector.Events()) != 0 {
				t.Fatal("transaction read used the repository pool")
			}
			events := txConnector.Events()
			if len(events) != 3 || events[0].Kind != dbtest.BeginKind || events[1].Kind != dbtest.QueryKind || events[1].SQL != wantSQL || events[2].Kind != dbtest.RollbackKind || txConnector.Remaining() != 0 {
				t.Fatal("transaction query or rollback order changed")
			}
		})
	}

	poolConnector, repository := openOptionsRepository(t)
	got, found, err := repository.ReadUploadPolicyTx(context.Background(), nil)
	if got != (data.UploadPolicy{}) || found {
		t.Fatal("nil transaction returned policy data")
	}
	assertSafeRepositoryError(t, err, "upload policy could not be read: dependency", nil)
	if len(poolConnector.Events()) != 0 {
		t.Fatal("nil transaction fell back to the pool")
	}
}

func siteOptionsColumns() []string {
	return []string{
		"oid", "option_set_is_active", "option_set_is_testing_now", "site_name", "site_description",
		"maintenance_mode", "preloader", "facebook_url", "twitter_url", "instagram_url", "linkedin_url",
		"contact_email", "contact_phone", "main_page_meta_title", "main_page_meta_description", "google_analytics",
		"primary_color", "secondary_color", "accent_color", "background_color", "font_color", "font_family",
		"items_per_page", "enable_testimonials", "maximum_sublinks_on_a_menu_item", "show_doctor_social_media",
		"show_doctor_appointment_fee", "show_anlasmali_kurum_pictures", "google_recaptcha_site_key",
		"logo_path", "logo_alt_text", "logo_title", "light_logo_path", "light_logo_alt_text", "light_logo_title",
		"favicon_path", "default_page_media_path", "default_page_media_alt_text", "default_page_media_title",
	}
}

func siteOptionsValues(id int64, active, testing bool) []any {
	return []any{
		id, active, testing, "Nivgoz", "Public description", true, "spinner", "facebook", "twitter", "instagram", "linkedin",
		"public@example.invalid", "+90 000", "Meta title", "Meta description", "analytics", "primary", "secondary", "accent",
		"background", "font", "Poppins", int64(25), true, int64(7), true, false, true, "public-site-key",
		"/logo", "logo alt", "logo title", "/light", "light alt", "light title", "/favicon", "/default", "default alt", "default title",
	}
}

func expectedSiteOptions(active, testing bool) data.SiteOptions {
	return data.SiteOptions{
		Set:      data.OptionSetIdentity{ID: "17", IsActive: active, IsTesting: testing},
		SiteName: "Nivgoz", SiteDescription: "Public description", MaintenanceMode: true, Preloader: "spinner",
		FacebookURL: "facebook", TwitterURL: "twitter", InstagramURL: "instagram", LinkedInURL: "linkedin",
		ContactEmail: "public@example.invalid", ContactPhone: "+90 000", MainPageMetaTitle: "Meta title",
		MainPageMetaDescription: "Meta description", GoogleAnalytics: "analytics", PrimaryColor: "primary",
		SecondaryColor: "secondary", AccentColor: "accent", BackgroundColor: "background", FontColor: "font",
		FontFamily: "Poppins", ItemsPerPage: 25, EnableTestimonials: true, MaximumSublinksOnMenuItem: 7,
		ShowDoctorSocialMedia: true, ShowDoctorAppointmentFee: false, ShowPartnerPictures: true,
		RecaptchaSiteKey: "public-site-key", SiteLogo: data.PublicMedia{Path: "/logo", AltText: "logo alt", Title: "logo title"},
		SiteLightLogo:    data.PublicMedia{Path: "/light", AltText: "light alt", Title: "light title"},
		Favicon:          data.PublicMedia{Path: "/favicon"},
		DefaultPageMedia: data.PublicMedia{Path: "/default", AltText: "default alt", Title: "default title"},
	}
}

func replaceSiteValue(values []any, index int, value any) []any {
	result := append([]any(nil), values...)
	result[index] = value
	return result
}

func openOptionsRepository(t *testing.T, steps ...dbtest.Step) (*dbtest.Connector, *OptionsRepository) {
	t.Helper()
	connector := dbtest.NewConnector(steps...)
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error("database close failed")
		}
	})
	return connector, NewOptionsRepository(db)
}

func assertOptionsQuery(t *testing.T, connector *dbtest.Connector, wantSQL string) {
	t.Helper()
	events := connector.Events()
	if len(events) != 1 {
		t.Fatal("unexpected query event count")
	}
	event := events[0]
	if event.Kind != dbtest.QueryKind || event.SQL != wantSQL {
		t.Fatal("unexpected query")
	}
	if len(event.Args) != 0 {
		t.Fatal("unexpected query argument count")
	}
	if connector.Remaining() != 0 {
		t.Fatal("unexpected remaining operations")
	}
}

func assertRowsClosed(t *testing.T, rows *dbtest.Rows) {
	t.Helper()
	if rows.CloseCount() != 1 {
		t.Fatal("unexpected rows close count")
	}
}

func stringPointerEquals(value *string, want string) bool {
	return value != nil && *value == want
}

func operationErrorPrefix(name string) string {
	switch name {
	case "upload policy":
		return "upload policy could not be read: "
	case "password policy":
		return "password policy could not be read: "
	case "mail delivery options":
		return "mail delivery options could not be read: "
	case "captcha verification options":
		return "captcha verification options could not be read: "
	case "option media references":
		return "option media references could not be read: "
	default:
		return "repository data could not be read: "
	}
}

type repositoryBackendError struct{ cause error }

func (*repositoryBackendError) Error() string {
	return "backend detail: postgres://user:credential@host/private SELECT role raw-uid synthetic-password synthetic-secret-key"
}

func (e *repositoryBackendError) Unwrap() error { return e.cause }

func assertSafeRepositoryError(t *testing.T, err error, wantText string, wantContext error) {
	t.Helper()
	if err == nil || err.Error() != wantText {
		t.Fatal("unexpected repository error stage")
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("repository error exposed an unwrap chain")
	}
	var backend *repositoryBackendError
	if errors.As(err, &backend) {
		t.Fatal("repository error exposed backend type")
	}
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, sentinel) != (wantContext == sentinel) {
			t.Fatal("unexpected context classification")
		}
	}
	for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
		printed := fmt.Sprintf(format, err)
		for _, forbidden := range []string{"SELECT", "postgres://", "credential", "private", "raw-uid", "synthetic-password", "synthetic-secret-key"} {
			if strings.Contains(printed, forbidden) {
				t.Fatal("repository error exposed forbidden backend detail")
			}
		}
	}
}
