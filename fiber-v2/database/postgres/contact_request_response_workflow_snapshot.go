package postgres

import (
	"context"
	"database/sql"

	"models/data"
)

const readContactRequestResponseWorkflowSnapshotSQL = `SELECT
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

type contactRequestResponseWorkflowSnapshotRow struct {
	identity        optionIdentityRow
	smtpHost        sql.NullString
	smtpPort        sql.NullInt64
	smtpUsername    sql.NullString
	smtpPassword    sql.NullString
	siteName        sql.NullString
	siteDescription sql.NullString
	contactEmail    sql.NullString
	contactPhone    sql.NullString
	facebookURL     sql.NullString
	twitterURL      sql.NullString
	instagramURL    sql.NullString
	linkedInURL     sql.NullString
	primaryColor    sql.NullString
	siteLogoPath    sql.NullString
}

// ReadContactRequestResponseWorkflowSnapshot reads the active contact-request
// response inputs from one options row and one SQL statement.
func (r *OptionsRepository) ReadContactRequestResponseWorkflowSnapshot(ctx context.Context) (data.ContactRequestResponseWorkflowSnapshot, bool, error) {
	var row contactRequestResponseWorkflowSnapshotRow
	found, err := r.readOne(ctx, readContactRequestResponseWorkflowSnapshot, readContactRequestResponseWorkflowSnapshotSQL, func(rows *sql.Rows) error {
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
			&row.siteLogoPath,
		)
	})
	if err != nil {
		return data.ContactRequestResponseWorkflowSnapshot{}, false, strictContactRequestResponseWorkflowSnapshotError(ctx, err)
	}
	if !found {
		return data.ContactRequestResponseWorkflowSnapshot{}, false, nil
	}
	set, err := activeOptionIdentity(row.identity, readContactRequestResponseWorkflowSnapshot)
	if err != nil {
		return data.ContactRequestResponseWorkflowSnapshot{}, false, strictContactRequestResponseWorkflowSnapshotError(ctx, err)
	}

	return data.ContactRequestResponseWorkflowSnapshot{
		Set:             set,
		SMTPHost:        row.smtpHost.String,
		SMTPPort:        row.smtpPort.Int64,
		SMTPUsername:    row.smtpUsername.String,
		SMTPPassword:    row.smtpPassword.String,
		SiteName:        row.siteName.String,
		SiteDescription: row.siteDescription.String,
		ContactEmail:    row.contactEmail.String,
		ContactPhone:    row.contactPhone.String,
		FacebookURL:     row.facebookURL.String,
		TwitterURL:      row.twitterURL.String,
		InstagramURL:    row.instagramURL.String,
		LinkedInURL:     row.linkedInURL.String,
		PrimaryColor:    row.primaryColor.String,
		SiteLogoPath:    row.siteLogoPath.String,
	}, true, nil
}

func strictContactRequestResponseWorkflowSnapshotError(ctx context.Context, err error) error {
	repositoryErr, ok := err.(*repositoryReadError)
	if !ok || repositoryErr == nil {
		return newStrictContactRequestResponseWorkflowSnapshotError(ctx, readDependency)
	}
	return newStrictContactRequestResponseWorkflowSnapshotError(ctx, repositoryErr.stage)
}

func newStrictContactRequestResponseWorkflowSnapshotError(ctx context.Context, stage readStage) error {
	result := &repositoryReadError{operation: readContactRequestResponseWorkflowSnapshot, stage: stage}
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
