package notificationws

import (
	"models/notify"
	"strconv"
)

// authenticatedUserID accepts only the server's named UID value. This policy
// belongs to the application boundary, not the transport-independent hub.
func authenticatedUserID(value any) (notify.UserID, error) {
	uid, ok := value.(notify.UserID)
	if !ok || len(uid) == 0 || len(uid) > 19 || uid[0] < '1' || uid[0] > '9' {
		return "", errIdentity
	}
	for i := 1; i < len(uid); i++ {
		if uid[i] < '0' || uid[i] > '9' {
			return "", errIdentity
		}
	}
	number, err := strconv.ParseInt(string(uid), 10, 64)
	if err != nil || number <= 0 {
		return "", errIdentity
	}
	canonical := strconv.FormatInt(number, 10)
	if canonical != string(uid) {
		return "", errIdentity
	}
	// FormatInt provides an immutable value independent of request buffers.
	return notify.UserID(canonical), nil
}

// authenticatedUpgrade is the production HTTP gate. No local or upgrade side
// effect is allowed until both authentication and UID validation succeed.
// The Fiber wrapper retains its typed auth callback; any also lets this gate
// reject wrong-type values without coupling the policy to Fiber.
func authenticatedUpgrade(authenticate func() (any, error), setIdentity func(notify.UserID), upgrade func() error) error {
	value, err := authenticate()
	if err != nil {
		return errIdentity
	}
	uid, err := authenticatedUserID(value)
	if err != nil {
		return err
	}
	setIdentity(uid)
	return upgrade()
}
