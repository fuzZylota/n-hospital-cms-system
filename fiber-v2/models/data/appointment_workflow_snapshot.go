package data

import "context"

// AppointmentWorkflowSnapshot contains exactly the active option-row values
// used by AddRandevu and needed by EditRandevu (currently unwired).
// SMTPPassword is a server-internal secret; never serialize,
// render, log, or cache it. SiteLogoPath is application data, not filesystem
// authority. This workflow does not consume CAPTCHA options.
type AppointmentWorkflowSnapshot struct {
	Set            OptionSetIdentity
	SMTPHost       string
	SMTPPort       int64
	SMTPUsername   string
	SMTPPassword   string
	SiteName       string
	ContactEmail   string
	ContactPhone   string
	PrimaryColor   string
	SecondaryColor string
	SiteLogoPath   string
}

// AppointmentWorkflowSnapshotReader reads one active option row in one
// statement. Missing returns a zero snapshot, false, nil. Every failure
// returns a zero snapshot, false, and a safe error. No testing fallback occurs.
type AppointmentWorkflowSnapshotReader interface {
	ReadAppointmentWorkflowSnapshot(ctx context.Context) (snapshot AppointmentWorkflowSnapshot, found bool, err error)
}
