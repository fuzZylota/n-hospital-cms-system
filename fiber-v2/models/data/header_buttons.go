package data

// HeaderParent is the data needed to identify a top-level header button.
// ParentID is nil for a top-level button.
type HeaderParent struct {
	ID       string
	Title    string
	ParentID *string
}
