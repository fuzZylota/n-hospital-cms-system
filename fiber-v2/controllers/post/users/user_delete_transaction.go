package users

import (
	"context"
	"database/sql"
	"math"
	"strconv"
)

type userDeleteResult struct {
	status  int
	message string
}

func canonicalDeleteUID(value string) (int64, bool) {
	if len(value) == 0 || value[0] < '1' || value[0] > '9' {
		return 0, false
	}
	for i := 1; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
	}
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0 && id <= math.MaxInt32 && strconv.FormatInt(id, 10) == value
}

func deleteUserServiceFailure() userDeleteResult {
	return userDeleteResult{status: 500, message: "Internal server error"}
}

// The actor and target are locked in UID order by lockEditUsers. Only the
// target's receipts are removed; notification events and other recipients stay.
func runUserDelete(ctx context.Context, tx *sql.Tx, actorID, targetID int64) userDeleteResult {
	actor, target, err := lockEditUsers(ctx, tx, actorID, targetID)
	if err != nil {
		return deleteUserServiceFailure()
	}
	if actor.uid != actorID || !actor.active.Valid || !actor.active.Bool || actor.role != "admin" || actorID == targetID {
		return userDeleteResult{status: 403, message: "Only admins can delete users"}
	}
	if target.uid != targetID {
		return userDeleteResult{status: 404, message: "User not found"}
	}

	receipts, err := tx.ExecContext(ctx, "DELETE FROM notification_receipts WHERE recipient_uid = $1", targetID)
	if err != nil || receipts == nil {
		return deleteUserServiceFailure()
	}
	count, err := receipts.RowsAffected()
	if err != nil || count < 0 {
		return deleteUserServiceFailure()
	}

	user, err := tx.ExecContext(ctx, "DELETE FROM users WHERE uid = $1", targetID)
	if err != nil || user == nil {
		return deleteUserServiceFailure()
	}
	count, err = user.RowsAffected()
	if err != nil || count != 1 {
		return deleteUserServiceFailure()
	}
	return userDeleteResult{status: 201, message: "User deleted successfully"}
}
