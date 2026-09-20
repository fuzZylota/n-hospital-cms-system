package postgres

import (
	"context"
	"database/sql"

	"models/data"
)

const readContactRequestWorkflowSnapshotSQL = `SELECT
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

type contactRequestWorkflowSnapshotRow struct {
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
	recaptchaSiteKey   sql.NullString
	recaptchaSecretKey sql.NullString
	siteLogoPath       sql.NullString
}

// ReadContactRequestWorkflowSnapshot reads the active contact-request inputs
// in one SQL statement. AccentColor is intentionally not selected and remains
// empty to preserve the current legacy projection behavior.
func (r *OptionsRepository) ReadContactRequestWorkflowSnapshot(ctx context.Context) (data.ContactRequestWorkflowSnapshot, bool, error) {
	var row contactRequestWorkflowSnapshotRow
	found, err := r.readOne(ctx, readContactRequestWorkflowSnapshot, readContactRequestWorkflowSnapshotSQL, func(rows *sql.Rows) error {
		return rows.Scan(
			&row.identity.id,
			&row.identity.active,
			&row.identity.testing,
			&row.smtpHost,
			&row.smtpPort,
			&row.smtpUsername,
			&row.smtpPassword,
			&row.siteName,
			&row.siteDescription,
			&row.contactEmail,
			&row.contactPhone,
			&row.facebookURL,
			&row.twitterURL,
			&row.instagramURL,
			&row.linkedInURL,
			&row.primaryColor,
			&row.recaptchaSiteKey,
			&row.recaptchaSecretKey,
			&row.siteLogoPath,
		)
	})
	if err != nil {
		return data.ContactRequestWorkflowSnapshot{}, false, strictContactRequestWorkflowSnapshotError(ctx, err)
	}
	if !found {
		return data.ContactRequestWorkflowSnapshot{}, false, nil
	}
	set, err := activeOptionIdentity(row.identity, readContactRequestWorkflowSnapshot)
	if err != nil {
		return data.ContactRequestWorkflowSnapshot{}, false, strictContactRequestWorkflowSnapshotError(ctx, err)
	}

	return data.ContactRequestWorkflowSnapshot{
		Set:                set,
		SMTPHost:           row.smtpHost.String,
		SMTPPort:           row.smtpPort.Int64,
		SMTPUsername:       row.smtpUsername.String,
		SMTPPassword:       row.smtpPassword.String,
		SiteName:           row.siteName.String,
		SiteDescription:    row.siteDescription.String,
		ContactEmail:       row.contactEmail.String,
		ContactPhone:       row.contactPhone.String,
		FacebookURL:        row.facebookURL.String,
		TwitterURL:         row.twitterURL.String,
		InstagramURL:       row.instagramURL.String,
		LinkedInURL:        row.linkedInURL.String,
		PrimaryColor:       row.primaryColor.String,
		RecaptchaSiteKey:   row.recaptchaSiteKey.String,
		RecaptchaSecretKey: row.recaptchaSecretKey.String,
		SiteLogoPath:       row.siteLogoPath.String,
		AccentColor:        "",
	}, true, nil
}

func strictContactRequestWorkflowSnapshotError(ctx context.Context, err error) error {
	repositoryErr, ok := err.(*repositoryReadError)
	if !ok || repositoryErr == nil {
		return newStrictContactRequestWorkflowSnapshotError(ctx, readDependency)
	}
	return newStrictContactRequestWorkflowSnapshotError(ctx, repositoryErr.stage)
}

func newStrictContactRequestWorkflowSnapshotError(ctx context.Context, stage readStage) error {
	result := &repositoryReadError{operation: readContactRequestWorkflowSnapshot, stage: stage}
	if ctx == nil {
		return result
	}
	switch ctx.Err() {
	case context.Canceled:
		result.contextErr = context.Canceled
	case context.DeadlineExceeded:
		result.contextErr = context.DeadlineExceeded
	}
	return result
}
