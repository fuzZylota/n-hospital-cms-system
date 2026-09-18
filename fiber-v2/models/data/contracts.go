// Package data defines application-owned data access contracts.
package data

import "context"

// HeaderButtonReader reads the header parents needed by the header-button
// consumer. No parents is a successful result represented by a non-nil empty
// slice; implementations do not return a not-found error for the list.
//
// On failure the result is nil and the infrastructure error is returned without
// exposing SQL, credentials, connection details, or backend-specific types.
// Context cancellation and deadline errors remain identifiable with errors.Is.
type HeaderButtonReader interface {
	ListHeaderParents(ctx context.Context) ([]HeaderParent, error)
}
