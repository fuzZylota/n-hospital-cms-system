package postgres

import (
	"context"
	"database/sql"

	"models/data"
)

const readJobApplicationWorkflowSnapshotSQL = `SELECT
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

type jobApplicationWorkflowSnapshotRow struct {
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
	maxBytes           sql.NullInt64
}

// ReadJobApplicationWorkflowSnapshot reads only AddJobApplication's active
// options in one statement. AccentColor stays empty for legacy parity.
func (r *OptionsRepository) ReadJobApplicationWorkflowSnapshot(ctx context.Context) (data.JobApplicationWorkflowSnapshot, bool, error) {
	var row jobApplicationWorkflowSnapshotRow
	firstRow := true
	found, err := r.readOne(ctx, readJobApplicationWorkflowSnapshot, readJobApplicationWorkflowSnapshotSQL, func(rows *sql.Rows) error {
		if !firstRow {
			// readOne has already found a duplicate; its cardinality error wins
			// even when the duplicate row itself cannot be scanned.
			return nil
		}
		firstRow = false
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
			&row.maxBytes,
		)
	})
	if err != nil {
		return data.JobApplicationWorkflowSnapshot{}, false, strictJobApplicationWorkflowSnapshotError(ctx, err)
	}
	if !found {
		return data.JobApplicationWorkflowSnapshot{}, false, nil
	}
	set, err := activeOptionIdentity(row.identity, readJobApplicationWorkflowSnapshot)
	if err != nil {
		return data.JobApplicationWorkflowSnapshot{}, false, strictJobApplicationWorkflowSnapshotError(ctx, err)
	}
	return data.JobApplicationWorkflowSnapshot{
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
		MaxBytes:           row.maxBytes.Int64,
	}, true, nil
}

func strictJobApplicationWorkflowSnapshotError(ctx context.Context, err error) error {
	repositoryErr, ok := err.(*repositoryReadError)
	if !ok || repositoryErr == nil {
		return newStrictJobApplicationWorkflowSnapshotError(ctx, readDependency)
	}
	return newStrictJobApplicationWorkflowSnapshotError(ctx, repositoryErr.stage)
}

func newStrictJobApplicationWorkflowSnapshotError(ctx context.Context, stage readStage) error {
	result := &repositoryReadError{operation: readJobApplicationWorkflowSnapshot, stage: stage}
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
