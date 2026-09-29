package randevular

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"models"
	"strconv"
	"strings"
)

// Set only by this package's synthetic HTTP tests. Production uses lib.SendEmail.
var editAppointmentMailHook func(*models.EmailInfos) error

func editPositiveID(value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 32)
	return id, err == nil && id > 0 && strconv.FormatInt(id, 10) == value
}

func validEditAppointmentID(routeRID, bodyRID string) (int64, bool) {
	id, ok := editPositiveID(routeRID)
	return id, ok && routeRID == bodyRID
}

// Both rows remain locked until the appointment update commits or rolls back.
// Neither an old JWT role nor client-supplied old_sid grants write access.
func authorizeEditAppointment(ctx context.Context, tx *sql.Tx, uid string, rid int64) (sql.NullInt64, int) {
	userID, ok := editPositiveID(uid)
	if !ok {
		return sql.NullInt64{}, 403
	}
	var role string
	var active sql.NullBool
	err := tx.QueryRowContext(ctx, "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE", userID).Scan(&role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, 403
	}
	if err != nil || !active.Valid {
		return sql.NullInt64{}, 503
	}
	if !active.Bool || role != "admin" {
		return sql.NullInt64{}, 403
	}
	var sid sql.NullInt64
	err = tx.QueryRowContext(ctx, "SELECT sid FROM randevular WHERE rid = $1 FOR UPDATE", rid).Scan(&sid)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, 404
	}
	if err != nil {
		return sql.NullInt64{}, 503
	}
	return sid, 0
}

func sameEditBranch(input string, current sql.NullInt64) bool {
	if input == "" {
		return !current.Valid
	}
	id, ok := editPositiveID(input)
	return ok && current.Valid && id == current.Int64
}

func validateEditDestination(ctx context.Context, tx *sql.Tx, sid, drid string) int {
	if sid == "" {
		if drid != "" {
			return 400
		}
		return 0
	}
	branchID, ok := editPositiveID(sid)
	if !ok {
		return 400
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM subeler WHERE sid = $1)", branchID).Scan(&exists); err != nil {
		return 503
	}
	if !exists {
		return 400
	}
	if drid == "" {
		return 0
	}
	doctorID, ok := editPositiveID(drid)
	if !ok {
		return 400
	}
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM doktorlar WHERE drid = $1 AND sid = $2)", doctorID, branchID).Scan(&exists); err != nil {
		return 503
	}
	if !exists {
		return 400
	}
	return 0
}

type editAppointmentUpdate struct {
	ctx  context.Context
	tx   *sql.Tx
	rid  int64
	sid  sql.NullInt64
	sets []string
	args []any
	res  sql.Result
}

func newEditAppointmentUpdate(ctx context.Context, tx *sql.Tx, rid int64, sid sql.NullInt64) *editAppointmentUpdate {
	return &editAppointmentUpdate{ctx: ctx, tx: tx, rid: rid, sid: sid}
}

func (u *editAppointmentUpdate) Set(column string, value any) {
	if column == "updated_at" {
		u.sets = append(u.sets, "updated_at = NOW()")
		return
	}
	// Every caller passes a source-code constant from the existing edit field list.
	u.args = append(u.args, value)
	u.sets = append(u.sets, fmt.Sprintf("%s = $%d", column, len(u.args)))
}

func (u *editAppointmentUpdate) Execute() error {
	if len(u.sets) == 0 {
		return errors.New("empty appointment edit")
	}
	u.args = append(u.args, u.rid, u.sid)
	query := fmt.Sprintf("UPDATE randevular SET %s WHERE rid = $%d AND sid IS NOT DISTINCT FROM $%d", strings.Join(u.sets, ", "), len(u.args)-1, len(u.args))
	var err error
	u.res, err = u.tx.ExecContext(u.ctx, query, u.args...)
	return err
}

func (u *editAppointmentUpdate) RowsAffected() (int64, error) {
	return u.res.RowsAffected()
}
