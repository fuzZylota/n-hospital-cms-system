package postgres

import (
	"context"
	"database/sql"

	"models/data"
)

const readJobApplicationResponseWorkflowSnapshotSQL = `SELECT
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

var _ data.JobApplicationResponseWorkflowSnapshotReader = (*OptionsRepository)(nil)

type jobApplicationResponseWorkflowSnapshotRow struct {
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

// ReadJobApplicationResponseWorkflowSnapshot reads the response inputs from
// one active option row in one statement.
func (r *OptionsRepository) ReadJobApplicationResponseWorkflowSnapshot(ctx context.Context) (data.JobApplicationResponseWorkflowSnapshot, bool, error) {
	var row jobApplicationResponseWorkflowSnapshotRow
	firstRow := true
	found, err := r.readOne(ctx, readJobApplicationResponseWorkflowSnapshot, readJobApplicationResponseWorkflowSnapshotSQL, func(rows *sql.Rows) error {
		if !firstRow {
			// readOne has detected a duplicate. Cardinality wins even if the
			// second row contains a value that cannot be scanned.
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
			&row.siteLogoPath,
		)
	})
	if err != nil {
		return data.JobApplicationResponseWorkflowSnapshot{}, false, strictJobApplicationResponseWorkflowSnapshotError(ctx, err)
	}
	if !found {
		return data.JobApplicationResponseWorkflowSnapshot{}, false, nil
	}
	set, err := activeOptionIdentity(row.identity, readJobApplicationResponseWorkflowSnapshot)
	if err != nil {
		return data.JobApplicationResponseWorkflowSnapshot{}, false, strictJobApplicationResponseWorkflowSnapshotError(ctx, err)
	}
	return data.JobApplicationResponseWorkflowSnapshot{
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

func strictJobApplicationResponseWorkflowSnapshotError(ctx context.Context, err error) error {
	repositoryErr, ok := err.(*repositoryReadError)
	if !ok || repositoryErr == nil {
		return newStrictJobApplicationResponseWorkflowSnapshotError(ctx, readDependency)
	}
	return newStrictJobApplicationResponseWorkflowSnapshotError(ctx, repositoryErr.stage)
}

func newStrictJobApplicationResponseWorkflowSnapshotError(ctx context.Context, stage readStage) error {
	result := &repositoryReadError{operation: readJobApplicationResponseWorkflowSnapshot, stage: stage}
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
