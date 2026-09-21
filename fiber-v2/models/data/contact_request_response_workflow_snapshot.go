package data

import "context"

// ContactRequestResponseWorkflowSnapshot contains only the active option-row
// values consumed while responding to a contact request. It is a
// secret-bearing, server-internal value: SMTPPassword must remain request-local
// and must never be serialized, rendered, logged, or cached. SiteLogoPath is
// application data used by the legacy workflow and grants no filesystem
// authority.
type ContactRequestResponseWorkflowSnapshot struct {
	Set             OptionSetIdentity
	SMTPHost        string
	SMTPPort        int64
	SMTPUsername    string
	SMTPPassword    string
	SiteName        string
	SiteDescription string
	ContactEmail    string
	ContactPhone    string
	FacebookURL     string
	TwitterURL      string
	InstagramURL    string
	LinkedInURL     string
	PrimaryColor    string
	SiteLogoPath    string
}

// ContactRequestResponseWorkflowSnapshotReader reads one consistent snapshot
// from the active option set. It accepts no testing selection and performs no
// active/testing fallback. A missing active row returns a zero snapshot,
// found=false, and err=nil. Every failure returns a zero snapshot, found=false,
// and a safe error. Snapshot.Set identifies the active row that supplied every
// returned value and independently retains that row's testing flag.
type ContactRequestResponseWorkflowSnapshotReader interface {
	ReadContactRequestResponseWorkflowSnapshot(ctx context.Context) (snapshot ContactRequestResponseWorkflowSnapshot, found bool, err error)
}
