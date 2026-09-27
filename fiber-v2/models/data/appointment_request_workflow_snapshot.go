package data

import "context"

// AppointmentRequestWorkflowSnapshot is the server-internal active option-row
// input consumed by AddRandevuRequest. SMTPPassword and RecaptchaSecretKey are
// secrets: never serialize, render, log, or cache this value. SiteLogoPath is
// application data, not filesystem authority. The legacy query does not select
// accent_color, so AccentColor remains empty for behavioral parity.
type AppointmentRequestWorkflowSnapshot struct {
	Set                OptionSetIdentity
	SMTPHost           string
	SMTPPort           int64
	SMTPUsername       string
	SMTPPassword       string
	SiteName           string
	SiteDescription    string
	ContactEmail       string
	ContactPhone       string
	FacebookURL        string
	TwitterURL         string
	InstagramURL       string
	LinkedInURL        string
	PrimaryColor       string
	SecondaryColor     string
	RecaptchaSiteKey   string
	RecaptchaSecretKey string
	SiteLogoPath       string
	AccentColor        string
}

// AppointmentRequestWorkflowSnapshotReader reads one active option set in one
// statement. Missing returns a zero snapshot and found=false; any failure
// returns a zero snapshot, found=false, and a safe error.
type AppointmentRequestWorkflowSnapshotReader interface {
	ReadAppointmentRequestWorkflowSnapshot(ctx context.Context) (snapshot AppointmentRequestWorkflowSnapshot, found bool, err error)
}
