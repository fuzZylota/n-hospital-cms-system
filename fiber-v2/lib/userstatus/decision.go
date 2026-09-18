package userstatus

import (
	"context"

	"models/data"
)

// ShouldExpireAuthCookies reduces the current account lookup to the only
// decision needed by the existing ban middleware. Lookup failures remain
// fail-open and are deliberately not returned or logged here.
func ShouldExpireAuthCookies(ctx context.Context, reader data.UserStatusReader, userID string) bool {
	if reader == nil || userID == "" {
		return false
	}

	status, err := reader.LookupUserStatus(ctx, userID)
	if err != nil {
		return false
	}
	return !status.Found || !status.Active
}
