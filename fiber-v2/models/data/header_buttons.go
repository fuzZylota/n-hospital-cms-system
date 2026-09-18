package data

// HeaderParent is the data needed to identify a top-level header button. ID is
// a string at the application boundary and follows the package identifier
// contract when backed by a numeric column. ParentID is nil for SQL NULL (a
// top-level button); a non-nil ParentID follows the same identifier contract.
type HeaderParent struct {
	ID       string
	Title    string
	ParentID *string
}
