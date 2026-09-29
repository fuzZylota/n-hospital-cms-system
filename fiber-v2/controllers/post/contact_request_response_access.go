package post

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
)

type contactResponseTarget struct {
	Crid      int64
	FirstName string
	LastName  string
	Email     string
}

func contactResponseID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 32)
	return id, err == nil && id > 0 && strconv.FormatInt(id, 10) == raw
}

// Lock the current actor before any target or options read. The target has no
// sid: only the current active DB admin is authorized by the temporary policy.
// The caller must commit this short read before starting SMTP. The returned
// target is a request-local value copy; later role/target changes cannot alter
// its recipient and cannot reliably cancel the external delivery.
func authorizeContactResponse(ctx context.Context, tx *sql.Tx, actor string, crid int64) (contactResponseTarget, int) {
	uid, ok := contactResponseID(actor)
	if !ok {
		return contactResponseTarget{}, 403
	}
	var role string
	var active sql.NullBool
	err := tx.QueryRowContext(ctx, "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE", uid).Scan(&role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return contactResponseTarget{}, 403
	}
	if err != nil {
		return contactResponseTarget{}, 503
	}
	if !active.Valid || !active.Bool || role != "admin" {
		return contactResponseTarget{}, 403
	}
	var target contactResponseTarget
	err = tx.QueryRowContext(ctx, "SELECT crid FROM contact_requests WHERE crid = $1 FOR UPDATE", crid).Scan(&target.Crid)
	if errors.Is(err, sql.ErrNoRows) {
		return contactResponseTarget{}, 404
	}
	if err != nil || target.Crid != crid {
		return contactResponseTarget{}, 503
	}
	err = tx.QueryRowContext(ctx, "SELECT first_name, last_name, email FROM contact_requests WHERE crid = $1", target.Crid).Scan(&target.FirstName, &target.LastName, &target.Email)
	if err != nil {
		return contactResponseTarget{}, 503
	}
	return target, 0
}

// SMTP is an external effect: rollback cannot undo a delivered email. Every
// update/affected-row/commit failure is returned to REL-001A as partial success.
func markContactResponse(ctx context.Context, db *sql.DB, crid int64) error {
	if db == nil {
		return errors.New("contact response state unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return errors.New("contact response state unavailable")
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "UPDATE contact_requests SET is_replied = TRUE, response_date = NOW(), updated_at = NOW() WHERE crid = $1", crid)
	if err != nil {
		return errors.New("contact response state unavailable")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return errors.New("contact response state unavailable")
	}
	if err = tx.Commit(); err != nil {
		return errors.New("contact response state unavailable")
	}
	return nil
}
