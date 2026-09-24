package data

import "context"

// JobApplicationWorkflowSnapshot contains only the active option-row values
// consumed by AddJobApplication. SMTPPassword and RecaptchaSecretKey are
// server-internal secrets and must never be rendered, logged, or cached.
// SiteLogoPath is application data, not filesystem authority. AccentColor
// remains empty because the legacy query does not select accent_color.
type JobApplicationWorkflowSnapshot struct {
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
	RecaptchaSiteKey   string
	RecaptchaSecretKey string
	SiteLogoPath       string
	AccentColor        string
	MaxBytes           int64
}

// JobApplicationWorkflowSnapshotReader reads one active option set in one
// statement. A missing active row returns a zero snapshot and found=false.
// Every failure returns a zero snapshot, found=false, and a safe error.
type JobApplicationWorkflowSnapshotReader interface {
	ReadJobApplicationWorkflowSnapshot(ctx context.Context) (snapshot JobApplicationWorkflowSnapshot, found bool, err error)
}
