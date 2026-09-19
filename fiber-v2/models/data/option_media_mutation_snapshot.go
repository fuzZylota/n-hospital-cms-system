package data

import "context"

// OptionMediaMutationSnapshot contains the active option-row values needed by
// the option-media mutation flow. All fields come from one selected row and
// carry no filesystem, transaction, or serialization authority. A nil media
// identifier means SQL NULL; non-nil identifiers follow the package identifier
// contract. MaxBytes preserves the stored value, including zero and negatives.
type OptionMediaMutationSnapshot struct {
	Set                OptionSetIdentity
	MaxBytes           int64
	SiteLogoID         *string
	SiteLightLogoID    *string
	FaviconID          *string
	DefaultPageMediaID *string
}

// OptionMediaMutationSnapshotReader reads one consistent snapshot of the
// active option set. It accepts no testing selection and performs no fallback.
// A missing active row returns a zero snapshot, found=false, and err=nil. Read
// failures return a zero snapshot and a safe error. Snapshot.Set identifies the
// active row that supplied every returned value.
type OptionMediaMutationSnapshotReader interface {
	ReadOptionMediaMutationSnapshot(ctx context.Context) (snapshot OptionMediaMutationSnapshot, found bool, err error)
}
