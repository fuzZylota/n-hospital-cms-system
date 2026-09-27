package postgres

import (
	"context"
	"database/sql"

	"models/data"
)

const readAppointmentRequestWorkflowSnapshotSQL = `SELECT
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

var _ data.AppointmentRequestWorkflowSnapshotReader = (*OptionsRepository)(nil)

type appointmentRequestWorkflowSnapshotRow struct {
	identity           optionIdentityRow
	smtpHost           sql.NullString
	smtpPort           sql.NullInt64
	smtpUsername       sql.NullString
	smtpPassword       sql.NullString
	siteName           sql.NullString
	siteDescription    sql.NullString
	contactEmail       sql.NullString
	contactPhone       sql.NullString
	facebookURL        sql.NullString
	twitterURL         sql.NullString
	instagramURL       sql.NullString
	linkedInURL        sql.NullString
	primaryColor       sql.NullString
	secondaryColor     sql.NullString
	recaptchaSiteKey   sql.NullString
	recaptchaSecretKey sql.NullString
	siteLogoPath       sql.NullString
}

// ReadAppointmentRequestWorkflowSnapshot is currently unwired: no production
// caller uses it. AccentColor stays empty for legacy projection parity.
func (r *OptionsRepository) ReadAppointmentRequestWorkflowSnapshot(ctx context.Context) (data.AppointmentRequestWorkflowSnapshot, bool, error) {
	var row appointmentRequestWorkflowSnapshotRow
	firstRow := true
	found, err := r.readOne(ctx, readAppointmentRequestWorkflowSnapshot, readAppointmentRequestWorkflowSnapshotSQL, func(rows *sql.Rows) error {
		if !firstRow {
			// The second row proves a cardinality violation even if malformed.
			return nil
		}
		firstRow = false
		return rows.Scan(
			&row.identity.id, &row.identity.active, &row.identity.testing,
			&row.smtpHost, &row.smtpPort, &row.smtpUsername, &row.smtpPassword,
			&row.siteName, &row.siteDescription, &row.contactEmail, &row.contactPhone,
			&row.facebookURL, &row.twitterURL, &row.instagramURL, &row.linkedInURL,
			&row.primaryColor, &row.secondaryColor,
			&row.recaptchaSiteKey, &row.recaptchaSecretKey, &row.siteLogoPath,
		)
	})
	if err != nil {
		return data.AppointmentRequestWorkflowSnapshot{}, false, strictAppointmentRequestWorkflowSnapshotError(ctx, err)
	}
	if !found {
		return data.AppointmentRequestWorkflowSnapshot{}, false, nil
	}
	set, err := activeOptionIdentity(row.identity, readAppointmentRequestWorkflowSnapshot)
	if err != nil {
		return data.AppointmentRequestWorkflowSnapshot{}, false, strictAppointmentRequestWorkflowSnapshotError(ctx, err)
	}
	return data.AppointmentRequestWorkflowSnapshot{
		Set:      set,
		SMTPHost: row.smtpHost.String, SMTPPort: row.smtpPort.Int64,
		SMTPUsername: row.smtpUsername.String, SMTPPassword: row.smtpPassword.String,
		SiteName: row.siteName.String, SiteDescription: row.siteDescription.String,
		ContactEmail: row.contactEmail.String, ContactPhone: row.contactPhone.String,
		FacebookURL: row.facebookURL.String, TwitterURL: row.twitterURL.String,
		InstagramURL: row.instagramURL.String, LinkedInURL: row.linkedInURL.String,
		PrimaryColor: row.primaryColor.String, SecondaryColor: row.secondaryColor.String,
		RecaptchaSiteKey: row.recaptchaSiteKey.String, RecaptchaSecretKey: row.recaptchaSecretKey.String,
		SiteLogoPath: row.siteLogoPath.String,
		AccentColor:  "",
	}, true, nil
}

func strictAppointmentRequestWorkflowSnapshotError(ctx context.Context, err error) error {
	repositoryErr, ok := err.(*repositoryReadError)
	if !ok || repositoryErr == nil {
		return newStrictAppointmentRequestWorkflowSnapshotError(ctx, readDependency)
	}
	return newStrictAppointmentRequestWorkflowSnapshotError(ctx, repositoryErr.stage)
}

func newStrictAppointmentRequestWorkflowSnapshotError(ctx context.Context, stage readStage) error {
	result := &repositoryReadError{operation: readAppointmentRequestWorkflowSnapshot, stage: stage}
	if ctx != nil {
		switch ctx.Err() {
		case context.Canceled:
			result.contextErr = context.Canceled
		case context.DeadlineExceeded:
			result.contextErr = context.DeadlineExceeded
		}
	}
	return result
}
