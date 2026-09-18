package data

import "context"

// UserStatus is the current account state used by the ban/status seam. Found
// distinguishes a missing user from an inactive user. Role is meaningful only
// when Found is true; inactive users still retain their current role.
//
// This DTO prepares a current-state lookup for N05B/N05C. It does not define an
// authorization policy and does not by itself resolve SEC-003.
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
