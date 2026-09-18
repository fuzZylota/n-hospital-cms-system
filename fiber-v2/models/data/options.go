package data

import "context"

// OptionSetSelection names the option-set state a SiteOptionsReader must
// select. Only ActiveOptionSet and TestingOptionSet are valid. Its zero value
// and every unknown value are invalid; they must never be treated as an active
// selection.
type OptionSetSelection uint8

const (
	// ActiveOptionSet selects the option set marked active.
	ActiveOptionSet OptionSetSelection = iota + 1
	// TestingOptionSet selects the option set marked for authenticated testing.
	TestingOptionSet
)

// OptionSetIdentity identifies the option row actually read from persistence
// and retains that row's active/testing flags. ID follows the package
// identifier contract. IsActive and IsTesting are independent because the
// persistence rules have not been proven to make them mutually exclusive.
type OptionSetIdentity struct {
	ID        string
	IsActive  bool
	IsTesting bool
}

// PublicMedia describes display-only media joined to site options. It carries
// no filesystem path authority: Path is application data for an existing
// public asset, not permission to read an arbitrary local path.
type PublicMedia struct {
	Path    string
	AltText string
	Title   string
}

// SiteOptions is the public and panel-safe projection used to render the site.
// It deliberately excludes SMTP credentials, CAPTCHA secrets, password policy,
// upload policy, and persistence-only media identifiers.
type SiteOptions struct {
	Set                       OptionSetIdentity
	SiteName                  string
	SiteDescription           string
	MaintenanceMode           bool
	Preloader                 string
	FacebookURL               string
	TwitterURL                string
	InstagramURL              string
	LinkedInURL               string
	ContactEmail              string
	ContactPhone              string
	MainPageMetaTitle         string
	MainPageMetaDescription   string
	GoogleAnalytics           string
	PrimaryColor              string
	SecondaryColor            string
	AccentColor               string
	BackgroundColor           string
	FontColor                 string
	FontFamily                string
	ItemsPerPage              int64
	EnableTestimonials        bool
	MaximumSublinksOnMenuItem int64
	ShowDoctorSocialMedia     bool
	ShowDoctorAppointmentFee  bool
	ShowPartnerPictures       bool
	RecaptchaSiteKey          string
	SiteLogo                  PublicMedia
	SiteLightLogo             PublicMedia
	Favicon                   PublicMedia
	DefaultPageMedia          PublicMedia
}

// UploadPolicy contains only the active server-side upload limit needed by
// media-accepting handlers. MaxBytes is a byte count; zero remains a real value
// and is not used as a missing-row marker.
type UploadPolicy struct {
	Set      OptionSetIdentity
	MaxBytes int64
}

// PasswordPolicy contains only the server-side password setting needed by user
// mutations.
type PasswordPolicy struct {
	Set           OptionSetIdentity
	RequireStrong bool
}

// MailDeliveryOptions contains server-internal SMTP settings used to send
// application mail. It must never be serialized or passed to a template.
type MailDeliveryOptions struct {
	Set      OptionSetIdentity
	Host     string
	Port     int64
	Username string
	Password string
}

// CaptchaVerificationOptions contains server-internal CAPTCHA verification
// material. SecretKey must never be serialized or passed to a template.
type CaptchaVerificationOptions struct {
	Set       OptionSetIdentity
	SiteKey   string
	SecretKey string
}

// OptionMediaReferences contains persistence identifiers needed by the option
// media edit flow. A nil pointer means SQL NULL; non-nil identifiers follow the
// package identifier contract.
type OptionMediaReferences struct {
	Set                OptionSetIdentity
	SiteLogoID         *string
	SiteLightLogoID    *string
	FaviconID          *string
	DefaultPageMediaID *string
}

// SiteOptionsReader reads one explicitly selected public/panel-safe option set.
// Only ActiveOptionSet and TestingOptionSet are valid selections. For a zero or
// unknown selection, implementations return zero options, found=false, and a
// safe non-nil error; an invalid selection is never converted to an active
// selection.
//
// When the selected row does not exist, implementations return zero options,
// found=false, and err=nil. In particular, a missing testing set does not cause
// the repository to read the active set. Any testing-to-active compatibility
// fallback belongs in the caller or compatibility layer. On lookup failure,
// implementations return zero options, found=false, and a safe error. Context
// failures follow the package errors.Is contract.
//
// SiteOptions.Set describes the row actually selected and read from
// persistence, not a requested or fallback state. Flag/cardinality rules and
// selected-row consistency will be enforced by the N05B repository.
type SiteOptionsReader interface {
	ReadSiteOptions(ctx context.Context, selection OptionSetSelection) (options SiteOptions, found bool, err error)
}

// UploadPolicyReader reads the active option set's upload policy. It accepts no
// testing selection and performs no active/testing fallback. If no active row
// exists it returns zero policy, found=false, and err=nil. On lookup failure it
// returns zero policy, found=false, and a safe error. Policy.Set describes the
// active row actually read from persistence.
type UploadPolicyReader interface {
	ReadUploadPolicy(ctx context.Context) (policy UploadPolicy, found bool, err error)
}

// PasswordPolicyReader reads the active option set's password policy. It
// accepts no testing selection and performs no active/testing fallback. If no
// active row exists it returns zero policy, found=false, and err=nil. On lookup
// failure it returns zero policy, found=false, and a safe error. Policy.Set
// describes the active row actually read from persistence.
type PasswordPolicyReader interface {
	ReadPasswordPolicy(ctx context.Context) (policy PasswordPolicy, found bool, err error)
}

// MailDeliveryOptionsReader reads the active option set's server-internal mail
// settings. It accepts no testing selection and performs no active/testing
// fallback. If no active row exists it returns zero options, found=false, and
// err=nil. On lookup failure it returns zero options, found=false, and a safe
// error. Options.Set describes the active row actually read from persistence.
type MailDeliveryOptionsReader interface {
	ReadMailDeliveryOptions(ctx context.Context) (options MailDeliveryOptions, found bool, err error)
}

// CaptchaVerificationOptionsReader reads the active option set's
// server-internal CAPTCHA settings. It accepts no testing selection and
// performs no active/testing fallback. If no active row exists it returns zero
// options, found=false, and err=nil. On lookup failure it returns zero options,
// found=false, and a safe error. Options.Set describes the active row actually
// read from persistence.
type CaptchaVerificationOptionsReader interface {
	ReadCaptchaVerificationOptions(ctx context.Context) (options CaptchaVerificationOptions, found bool, err error)
}

// OptionMediaReferencesReader reads the active option set's media identifiers.
// It accepts no testing selection and performs no active/testing fallback. If
// no active row exists it returns zero references, found=false, and err=nil. On
// lookup failure it returns zero references, found=false, and a safe error.
// References.Set describes the active row actually read from persistence.
type OptionMediaReferencesReader interface {
	ReadOptionMediaReferences(ctx context.Context) (references OptionMediaReferences, found bool, err error)
}
