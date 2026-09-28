package data

import "context"

// UserStatus is the current account state used by the ban/status seam. Found
// distinguishes a missing user from an inactive user. An active account must
// have a nonblank current Role for authorization; inactive users may retain it.
//
// This DTO carries the current-state lookup for N05B/N05C. Callers make the
// authorization decision at their own boundary, including HTTP requests and
// notification delivery; the DTO itself grants no access.
type UserStatus struct {
	Found  bool
	Active bool
	Role   string
}

// UserStatusReader looks up the current status of one user. userID remains a
// string at this boundary and follows the package identifier contract when the
// backing column is numeric. A missing user returns UserStatus{Found: false}
// and nil error. An inactive user returns Found true and Active false. Any
// lookup failure returns the zero result and a non-nil safe error; it must not
// be treated as success or not-found. Context failures follow the package
// errors.Is contract.
type UserStatusReader interface {
	LookupUserStatus(ctx context.Context, userID string) (UserStatus, error)
}
