package userstatus

import (
	"context"
	"errors"
	"strings"

	"models/data"
)

// ShouldExpireAuthCookies returns the current role only for an active account.
// Callers must stop on errors; only a definitively inactive or missing account
// should lose its cookie.
func ShouldExpireAuthCookies(ctx context.Context, reader data.UserStatusReader, userID string) (bool, string, error) {
	if reader == nil || userID == "" {
		return false, "", errors.New("user status lookup unavailable")
	}

	status, err := reader.LookupUserStatus(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if !status.Found || !status.Active {
		return true, "", nil
	}
	if strings.TrimSpace(status.Role) == "" {
		return false, "", errors.New("current user role unavailable")
	}
	return false, status.Role, nil
}
