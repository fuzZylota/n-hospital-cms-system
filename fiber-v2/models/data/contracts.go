// Package data defines application-owned, infrastructure-independent data
// access contracts.
//
// Identifiers stay strings at this application boundary. When an identifier is
// backed by a numeric database column, the concrete repository owns strict
// base-10 conversion and canonicalization. A canonical numeric identifier has
// no sign, whitespace, or leading zeroes and is greater than zero. Empty
// strings are not identifiers. Nullable identifiers use nil to represent SQL
// NULL; a non-nil pointer must point to a valid identifier.
//
// Reader and writer implementations return only safe application errors. They
// must not expose SQL text, backend names, credentials, connection details, or
// row contents. When cancellation or a deadline causes failure, wrapping must
// preserve context.Canceled or context.DeadlineExceeded for errors.Is.
package data

import "context"

// HeaderButtonReader reads the header parents needed by the header-button
// consumer. No parents is a successful result represented by a non-nil empty
// slice; implementations do not return a not-found error for the list.
//
// On failure the result is nil and a safe application error is returned without
// exposing SQL, credentials, connection details, or backend-specific types.
// Context cancellation and deadline errors remain identifiable with errors.Is.
type HeaderButtonReader interface {
	ListHeaderParents(ctx context.Context) ([]HeaderParent, error)
}
