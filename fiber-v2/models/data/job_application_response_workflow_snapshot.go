package data

import "context"

// JobApplicationResponseWorkflowSnapshot is the server-internal, secret-bearing
// active option row consumed by RespondToJobApplication. SMTPPassword must stay
// request-local and must not be serialized, rendered, logged, or cached.
// SiteLogoPath is application data, not filesystem authority.
type JobApplicationResponseWorkflowSnapshot struct {
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

// JobApplicationResponseWorkflowSnapshotReader reads a single active option
// row. Missing returns the zero snapshot, false, nil. Every failure returns a
// zero snapshot, false, and a safe error. No testing fallback is performed.
type JobApplicationResponseWorkflowSnapshotReader interface {
	ReadJobApplicationResponseWorkflowSnapshot(ctx context.Context) (snapshot JobApplicationResponseWorkflowSnapshot, found bool, err error)
}
