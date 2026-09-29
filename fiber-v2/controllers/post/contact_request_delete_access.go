package post

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
)

func contactDeleteID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 32)
	return id, err == nil && id > 0 && strconv.FormatInt(id, 10) == raw
}

// Only the current active DB admin can delete a real contact request. There
// is no branch column. Actor/target locks and DELETE share this request's Tx;
// no PII or notification/receipt tables are read or changed by this workflow.
func deleteContactRequestTransaction(ctx context.Context, db *sql.DB, actor string, crid int64) (status int) {
	uid, ok := contactDeleteID(actor)
	if !ok {
		return 403
	}
	if db == nil {
		return 503
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 503
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			status = 503
		}
	}()
	var role string
	var active sql.NullBool
	err = tx.QueryRowContext(ctx, "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE", uid).Scan(&role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return 403
	}
	if err != nil {
		return 503
	}
	if !active.Valid || !active.Bool || role != "admin" {
		return 403
	}
	var savedID int64
	err = tx.QueryRowContext(ctx, "SELECT crid FROM contact_requests WHERE crid = $1 FOR UPDATE", crid).Scan(&savedID)
	if errors.Is(err, sql.ErrNoRows) {
		return 404
	}
	if err != nil || savedID != crid {
		return 503
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM contact_requests WHERE crid = $1", savedID)
	if err != nil {
		return 503
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 503
	}
	if affected == 0 {
		return 404
	}
	if affected != 1 {
		return 503
	}
	if err := tx.Commit(); err != nil {
		return 503
	}
	return 201
}
