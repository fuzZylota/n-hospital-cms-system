package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"models"
	"strconv"
	"strings"
)

type userEditResult struct {
	status  int
	message string
	stage   string
}

func editFailure(status int, message, stage string) userEditResult {
	return userEditResult{status: status, message: message, stage: stage}
}

func editServiceFailure(stage string) userEditResult {
	return editFailure(500, "Internal server error", stage)
}

type lockedEditUser struct {
	uid    int64
	role   string
	active sql.NullBool
	sid    sql.NullInt64
}

type editTargetProfile struct {
	email, phone, name, surname, timezone string
}

// Both user rows are locked in UID order, including when the actor edits self.
// The policy lookup below uses these locked values, not a JWT role or a shared ORM Tx.
func lockEditUsers(ctx context.Context, tx *sql.Tx, actorID, targetID int64) (lockedEditUser, lockedEditUser, error) {
	rows, err := tx.QueryContext(ctx, `SELECT uid, role, is_active, sid FROM users
WHERE uid IN ($1, $2) ORDER BY uid FOR UPDATE`, actorID, targetID)
	if err != nil {
		return lockedEditUser{}, lockedEditUser{}, err
	}
	var actor, target lockedEditUser
	count := 0
	for rows.Next() {
		var user lockedEditUser
		if err = rows.Scan(&user.uid, &user.role, &user.active, &user.sid); err != nil {
			break
		}
		count++
		if user.uid == actorID {
			actor = user
		}
		if user.uid == targetID {
			target = user
		}
	}
	if err == nil {
		err = rows.Err()
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err == nil && (count > 2 || actor.uid != actorID && actor.uid != 0 || target.uid != targetID && target.uid != 0 || !actor.active.Valid && actor.uid != 0) {
		err = errors.New("invalid locked user rows")
	}
	return actor, target, err
}

func oneEditRow(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	if result == nil {
		return errors.New("missing mutation result")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("unexpected affected row count")
	}
	return nil
}

func editValueChanged(fields map[string][]string, name, current string, next string) bool {
	_, submitted := fields[name]
	return submitted && next != current
}

func runUserEdit(ctx context.Context, tx *sql.Tx, actorUID, targetUID string, fields map[string][]string, inputs models.UsersEdit) userEditResult {
	actorID, err := strconv.ParseInt(actorUID, 10, 64)
	if err != nil || actorID <= 0 {
		return editFailure(403, "Only admins can edit protected user fields", "actor_uid")
	}
	targetID, err := strconv.ParseInt(targetUID, 10, 64)
	if err != nil || targetID <= 0 || strconv.FormatInt(targetID, 10) != targetUID {
		return editFailure(400, "Invalid user profile", "target_uid")
	}
	actor, target, err := lockEditUsers(ctx, tx, actorID, targetID)
	if err != nil {
		return editServiceFailure("user_lock")
	}
	decision, err := decideUserEditRequest(actorUID, targetUID, fields, func(uid string) (userEditActor, error) {
		if uid != actorUID || actor.uid == 0 {
			return userEditActor{}, nil
		}
		return userEditActor{UID: strconv.FormatInt(actor.uid, 10), Role: actor.role, IsActive: actor.active.Bool}, nil
	})
	if err != nil {
		if errors.Is(err, errInvalidUserEditUID) || errors.Is(err, errInvalidUserProfile) {
			return editFailure(400, "Invalid user profile", "profile_validation")
		}
		return editFailure(403, "Only admins can edit protected user fields", "authorization")
	}
	if target.uid == 0 {
		return editFailure(404, "User not found", "target_missing")
	}
	fields = decision.Fields
	policy := decision.Policy
	for _, name := range []string{"name", "surname", "email", "phone", "timezone"} {
		if value, submitted := submittedFieldValue(fields, name); submitted {
			switch name {
			case "name":
				inputs.Name = value
			case "surname":
				inputs.Surname = value
			case "email":
				inputs.Email = value
			case "phone":
				inputs.Phone = value
			case "timezone":
				inputs.Timezone = value
			}
		}
	}
	if policy.IsAdmin {
		for _, name := range []string{"role", "sid"} {
			if _, present := fields[name]; present {
				value, ok := submittedFieldValue(fields, name)
				if !ok {
					return editFailure(400, "Invalid user profile", "protected_input")
				}
				if name == "role" {
					inputs.Role = value
				} else {
					inputs.Sid = value
				}
			}
		}
		if values, present := fields["is_active"]; present {
			if len(values) != 1 {
				return editFailure(400, "Invalid user profile", "protected_input")
			}
			inputs.IsActive, err = parseSubmittedBool(values)
			if err != nil {
				return editFailure(400, "Invalid user profile", "protected_input")
			}
		}
	}
	changes := []branchPermissionChange{}
	if policy.UpdatePermissions {
		changes, err = parseBranchPermissionChanges(decision.TargetUID, fields)
		if err != nil {
			return editFailure(400, "Invalid branch permissions", "permission_input")
		}
		// The parser sorts branch IDs; lock the branch rows in that same order.
		for _, change := range changes {
			var sid int64
			err = tx.QueryRowContext(ctx, `SELECT sid FROM subeler WHERE sid = $1 AND is_active = TRUE FOR SHARE`, change.BranchID).Scan(&sid)
			if errors.Is(err, sql.ErrNoRows) {
				return editFailure(400, "Invalid branch permissions", "branch_missing")
			}
			if err != nil || sid != int64(change.BranchID) {
				return editServiceFailure("branch_read")
			}
		}
	}
	var profile editTargetProfile
	err = tx.QueryRowContext(ctx, `SELECT email, phone, name, surname, timezone FROM users WHERE uid = $1`, targetID).
		Scan(&profile.email, &profile.phone, &profile.name, &profile.surname, &profile.timezone)
	if err != nil {
		return editServiceFailure("target_read")
	}
	for _, check := range []struct {
		changed       bool
		column, value string
	}{
		{editValueChanged(fields, "email", profile.email, inputs.Email), "email", inputs.Email},
		{editValueChanged(fields, "phone", profile.phone, inputs.Phone), "phone", inputs.Phone},
	} {
		if !check.changed {
			continue
		}
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM users WHERE %s = $1 AND uid <> $2)", check.column)
		if err = tx.QueryRowContext(ctx, query, check.value, targetID).Scan(&exists); err != nil {
			return editServiceFailure("uniqueness_read")
		}
		if exists {
			return editFailure(403, strings.Title(check.column)+" already exists", "duplicate_profile")
		}
	}
	sets := []string{}
	args := []any{}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	for _, field := range []struct{ name, old, next string }{
		{"email", profile.email, inputs.Email}, {"phone", profile.phone, inputs.Phone},
		{"name", profile.name, inputs.Name}, {"surname", profile.surname, inputs.Surname},
		{"timezone", profile.timezone, inputs.Timezone},
	} {
		if editValueChanged(fields, field.name, field.old, field.next) {
			add(field.name, field.next)
		}
	}
	if policy.IsAdmin {
		if editValueChanged(fields, "role", target.role, inputs.Role) {
			add("role", inputs.Role)
		}
		if _, submitted := fields["is_active"]; submitted && inputs.IsActive != target.active.Bool {
			add("is_active", inputs.IsActive)
		}
		oldSID := ""
		if target.sid.Valid {
			oldSID = strconv.FormatInt(target.sid.Int64, 10)
		}
		if editValueChanged(fields, "sid", oldSID, inputs.Sid) {
			if inputs.Sid == "" {
				add("sid", nil)
			} else {
				add("sid", inputs.Sid)
			}
		}
	}
	if len(sets) == 0 && len(changes) == 0 {
		return editFailure(400, "Nothing changed", "nothing_changed")
	}
	if len(sets) > 0 {
		args = append(args, targetID, target.role, target.sid)
		query := fmt.Sprintf("UPDATE users SET %s WHERE uid = $%d AND role = $%d AND sid IS NOT DISTINCT FROM $%d", strings.Join(sets, ", "), len(args)-2, len(args)-1, len(args))
		if err = oneEditRow(tx.ExecContext(ctx, query, args...)); err != nil {
			return editServiceFailure("user_update")
		}
	}
	for _, change := range changes {
		rows, readErr := tx.QueryContext(ctx, `SELECT id FROM user_branch_permissions WHERE uid = $1 AND sid = $2 FOR UPDATE`, targetID, change.BranchID)
		if readErr != nil {
			return editServiceFailure("permission_read")
		}
		var permissionID int64
		count := 0
		for rows.Next() {
			if readErr = rows.Scan(&permissionID); readErr != nil {
				break
			}
			count++
		}
		if readErr == nil {
			readErr = rows.Err()
		}
		if closeErr := rows.Close(); readErr == nil {
			readErr = closeErr
		}
		if readErr != nil || count > 1 {
			return editServiceFailure("permission_read")
		}
		if count == 0 {
			canView, canDelete := false, false
			if change.CanView != nil {
				canView = *change.CanView
			}
			if change.CanDelete != nil {
				canDelete = *change.CanDelete
			}
			if err = oneEditRow(tx.ExecContext(ctx, `INSERT INTO user_branch_permissions (uid, sid, can_view, can_delete) VALUES ($1, $2, $3, $4)`, targetID, change.BranchID, canView, canDelete)); err != nil {
				return editServiceFailure("permission_insert")
			}
			continue
		}
		permissionSets := []string{}
		permissionArgs := []any{}
		if change.CanView != nil {
			permissionArgs = append(permissionArgs, *change.CanView)
			permissionSets = append(permissionSets, fmt.Sprintf("can_view = $%d", len(permissionArgs)))
		}
		if change.CanDelete != nil {
			permissionArgs = append(permissionArgs, *change.CanDelete)
			permissionSets = append(permissionSets, fmt.Sprintf("can_delete = $%d", len(permissionArgs)))
		}
		permissionArgs = append(permissionArgs, permissionID, targetID, change.BranchID)
		query := fmt.Sprintf("UPDATE user_branch_permissions SET %s WHERE id = $%d AND uid = $%d AND sid = $%d", strings.Join(permissionSets, ", "), len(permissionArgs)-2, len(permissionArgs)-1, len(permissionArgs))
		if err = oneEditRow(tx.ExecContext(ctx, query, permissionArgs...)); err != nil {
			return editServiceFailure("permission_update")
		}
	}
	return userEditResult{status: 201, message: "User edited successfully"}
}
