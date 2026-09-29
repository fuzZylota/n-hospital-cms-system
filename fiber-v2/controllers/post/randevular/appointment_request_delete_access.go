package randevular

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v2"
)

func requestDeleteFailure(c *fiber.Ctx, status int) error {
	message := "Server Hatası: Lütfen daha sonra tekrar deneyin."
	if status == 404 {
		message = "Randevu talebi bulunamadı"
	}
	return c.Status(status).JSON(fiber.Map{"status": status, "message": message})
}

// The user, request and permission rows stay locked through the conditional delete.
// A null request branch is deletable only by an active admin, as before.
func authorizeDeleteRequest(ctx context.Context, tx *sql.Tx, uid string, rrid int64) (sql.NullInt64, int) {
	userID, ok := editPositiveID(uid)
	if !ok {
		return sql.NullInt64{}, 404
	}
	var role string
	var active sql.NullBool
	var userSID sql.NullInt64
	err := tx.QueryRowContext(ctx, "SELECT role, is_active, sid FROM users WHERE uid = $1 FOR UPDATE", userID).Scan(&role, &active, &userSID)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, 404
	}
	if err != nil || !active.Valid {
		return sql.NullInt64{}, 503
	}
	if !active.Bool || (role != "admin" && role != "moderator" && role != "santral") {
		return sql.NullInt64{}, 404
	}

	var requestSID sql.NullInt64
	err = tx.QueryRowContext(ctx, "SELECT sid FROM randevu_talepleri WHERE rrid = $1 FOR UPDATE", rrid).Scan(&requestSID)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, 404
	}
	if err != nil {
		return sql.NullInt64{}, 503
	}
	if role == "admin" {
		return requestSID, 0
	}
	if !requestSID.Valid || requestSID.Int64 <= 0 {
		return sql.NullInt64{}, 404
	}
	if role == "santral" && (!userSID.Valid || userSID.Int64 != requestSID.Int64) {
		return sql.NullInt64{}, 404
	}

	var canDelete sql.NullBool
	err = tx.QueryRowContext(ctx, "SELECT can_delete FROM user_branch_permissions WHERE uid = $1 AND sid = $2 FOR UPDATE", userID, requestSID.Int64).Scan(&canDelete)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullInt64{}, 404
	}
	if err != nil {
		return sql.NullInt64{}, 503
	}
	if !canDelete.Valid || !canDelete.Bool {
		return sql.NullInt64{}, 404
	}
	return requestSID, 0
}
