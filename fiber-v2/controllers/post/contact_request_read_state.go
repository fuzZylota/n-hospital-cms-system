package post

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
)

func contactReadStateID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 32)
	return id, err == nil && id > 0 && strconv.FormatInt(id, 10) == raw
}

// The panel sends a JSON boolean target state. Missing/null/coerced values do
// not describe a target state; unrelated body fields cannot select a target.
func contactReadStateInput(body []byte) (bool, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return false, false
	}
	switch string(bytes.TrimSpace(fields["is_read"])) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

// No branch exists on contact_requests. Check only a current active DB admin,
// then lock the real target without reading PII. All changes use a local Tx.
// A role change after the actor lock waits until this short operation ends.
func setContactRequestReadState(ctx context.Context, db *sql.DB, actor string, crid int64, desired bool) (status int) {
	uid, ok := contactReadStateID(actor)
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
	var current sql.NullBool
	err = tx.QueryRowContext(ctx, "SELECT crid, is_read FROM contact_requests WHERE crid = $1 FOR UPDATE", crid).Scan(&savedID, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return 404
	}
	if err != nil || savedID != crid {
		return 503
	}
	// Repeating an already-applied value succeeds without even changing updated_at.
	if !current.Valid || current.Bool != desired {
		result, err := tx.ExecContext(ctx, "UPDATE contact_requests SET is_read = $1, updated_at = NOW() WHERE crid = $2", desired, savedID)
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
	}
	if err := tx.Commit(); err != nil {
		return 503
	}
	return 201
}
