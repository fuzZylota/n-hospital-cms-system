package data

import "context"

// ContactRequestWorkflowSnapshot contains only the active option-row values
// consumed by the public contact-request workflow. It is a secret-bearing,
// server-internal value: SMTPPassword and RecaptchaSecretKey must never be
// serialized, rendered, logged, or cached. SiteLogoPath is application data
// used by the legacy workflow and grants no filesystem authority.
//
// AccentColor deliberately remains the empty string. The legacy caller reads
// that field without selecting accent_color, so an empty value preserves the
// existing behavior until a separately approved workflow change.
type ContactRequestWorkflowSnapshot struct {
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
}

// ContactRequestWorkflowSnapshotReader reads one consistent snapshot from the
// active option set without a testing or active fallback. A missing active row
// returns a zero snapshot, found=false, and err=nil. Every failure returns a
// zero snapshot, found=false, and a safe error.
type ContactRequestWorkflowSnapshotReader interface {
	ReadContactRequestWorkflowSnapshot(ctx context.Context) (snapshot ContactRequestWorkflowSnapshot, found bool, err error)
}
