package postgres

import (
	"context"
	"database/sql"

	"models/data"
)

const readAppointmentWorkflowSnapshotSQL = `SELECT
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

var _ data.AppointmentWorkflowSnapshotReader = (*OptionsRepository)(nil)

type appointmentWorkflowSnapshotRow struct {
	identity       optionIdentityRow
	smtpHost       sql.NullString
	smtpPort       sql.NullInt64
	smtpUsername   sql.NullString
	smtpPassword   sql.NullString
	siteName       sql.NullString
	contactEmail   sql.NullString
	contactPhone   sql.NullString
	primaryColor   sql.NullString
	secondaryColor sql.NullString
	siteLogoPath   sql.NullString
}

// ReadAppointmentWorkflowSnapshot returns the AddRandevu inputs, also needed
// by the currently unwired EditRandevu, from one active options row.
func (r *OptionsRepository) ReadAppointmentWorkflowSnapshot(ctx context.Context) (data.AppointmentWorkflowSnapshot, bool, error) {
	var row appointmentWorkflowSnapshotRow
	firstRow := true
	found, err := r.readOne(ctx, readAppointmentWorkflowSnapshot, readAppointmentWorkflowSnapshotSQL, func(rows *sql.Rows) error {
		if !firstRow {
			// Cardinality wins even if the second row is malformed.
			return nil
		}
		firstRow = false
		return rows.Scan(
			&row.identity.id, &row.identity.active, &row.identity.testing,
			&row.smtpHost, &row.smtpPort, &row.smtpUsername, &row.smtpPassword,
			&row.siteName, &row.contactEmail, &row.contactPhone,
			&row.primaryColor, &row.secondaryColor, &row.siteLogoPath,
		)
	})
	if err != nil {
		return data.AppointmentWorkflowSnapshot{}, false, strictAppointmentWorkflowSnapshotError(ctx, err)
	}
	if !found {
		return data.AppointmentWorkflowSnapshot{}, false, nil
	}
	set, err := activeOptionIdentity(row.identity, readAppointmentWorkflowSnapshot)
	if err != nil {
		return data.AppointmentWorkflowSnapshot{}, false, strictAppointmentWorkflowSnapshotError(ctx, err)
	}
	return data.AppointmentWorkflowSnapshot{
		Set:      set,
		SMTPHost: row.smtpHost.String, SMTPPort: row.smtpPort.Int64,
		SMTPUsername: row.smtpUsername.String, SMTPPassword: row.smtpPassword.String,
		SiteName: row.siteName.String, ContactEmail: row.contactEmail.String,
		ContactPhone: row.contactPhone.String, PrimaryColor: row.primaryColor.String,
		SecondaryColor: row.secondaryColor.String, SiteLogoPath: row.siteLogoPath.String,
	}, true, nil
}

func strictAppointmentWorkflowSnapshotError(ctx context.Context, err error) error {
	repositoryErr, ok := err.(*repositoryReadError)
	if !ok || repositoryErr == nil {
		return newStrictAppointmentWorkflowSnapshotError(ctx, readDependency)
	}
	return newStrictAppointmentWorkflowSnapshotError(ctx, repositoryErr.stage)
}

func newStrictAppointmentWorkflowSnapshotError(ctx context.Context, stage readStage) error {
	result := &repositoryReadError{operation: readAppointmentWorkflowSnapshot, stage: stage}
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
