package data

import "context"

// OptionalFieldState records whether an optional media column is omitted,
// explicitly set to SQL NULL, or assigned a value. The zero value is omitted;
// writers reject every other, unknown state with a safe error.
type OptionalFieldState uint8

const (
	// FieldOmitted leaves the column out of the insert operation.
	FieldOmitted OptionalFieldState = iota
	// FieldNull includes the column with an explicit SQL NULL.
	FieldNull
	// FieldValue includes the column with Value, including empty string or zero.
	FieldValue
)

// OptionalString preserves omitted, explicit NULL, and string value as three
// distinct states. Value is used only when State is FieldValue; an empty Value
// is still a real value.
type OptionalString struct {
	State OptionalFieldState
	Value string
}

// OptionalInt64 preserves omitted, explicit NULL, and integer value as three
// distinct states. Value is used only when State is FieldValue; zero is still a
// real value.
type OptionalInt64 struct {
	State OptionalFieldState
	Value int64
}

// MediaInsertInput is the infrastructure-independent media insert boundary.
// FileName and FilePath are already-validated application metadata, never raw
// client paths or authority to access a physical filesystem location. FileSize
// accepts zero as a real value. TargetID is required and UserID is an optional
// identifier; identifier values remain strings and follow the package
// identifier contract when their backing columns are numeric.
//
// Optional fields carry their desired state directly. The contract never asks
// the repository to compare client-supplied old data.
type MediaInsertInput struct {
	FileName string
	FilePath string
	FileSize int64
	MIMEType string
	FileType string
	TargetID string
	UserID   OptionalString
	Data     OptionalString
	AltText  OptionalString
	Title    OptionalString
	Width    OptionalInt64
	Height   OptionalInt64
}

// MediaInsertResult identifies the inserted media row. ID stays a string at
// this boundary. For a numeric backing column, the writer owns conversion and
// returns the canonical numeric form defined by this package.
type MediaInsertResult struct {
	ID string
}

// MediaInserter inserts one media record without exposing a transaction, SQL
// executor, ORM, or driver type. A failure returns the zero result and a safe
// error. Context failures follow the package errors.Is contract.
type MediaInserter interface {
	InsertMedia(ctx context.Context, input MediaInsertInput) (MediaInsertResult, error)
}
