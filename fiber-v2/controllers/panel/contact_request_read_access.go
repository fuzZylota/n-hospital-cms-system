package panel

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// A contact request has no branch or recipient field. Until a product policy
// defines wider access, only a current, active DB admin may read this queue.
func contactRequestReadAccess(ctx context.Context, db *sql.DB, actorUID string, crid *int64) int {
	uid, err := strconv.ParseInt(actorUID, 10, 32)
	if err != nil || uid <= 0 || strconv.FormatInt(uid, 10) != actorUID {
		return fiber.StatusNotFound
	}
	if db == nil {
		return fiber.StatusServiceUnavailable
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return fiber.StatusServiceUnavailable
	}
	defer tx.Rollback()

	var role string
	var active sql.NullBool
	err = tx.QueryRowContext(ctx, "SELECT role, is_active FROM users WHERE uid = $1", uid).Scan(&role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return fiber.StatusNotFound
	}
	if err != nil {
		return fiber.StatusServiceUnavailable
	}
	if !active.Valid || !active.Bool || role != "admin" {
		return fiber.StatusNotFound
	}
	if crid != nil {
		var savedID int64
		err = tx.QueryRowContext(ctx, "SELECT crid FROM contact_requests WHERE crid = $1", *crid).Scan(&savedID)
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.StatusNotFound
		}
		if err != nil || savedID != *crid {
			return fiber.StatusServiceUnavailable
		}
	}
	if err = tx.Commit(); err != nil {
		return fiber.StatusServiceUnavailable
	}
	return fiber.StatusOK
}

func canonicalContactRequestID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 32)
	return id, err == nil && id > 0 && strconv.FormatInt(id, 10) == raw
}

func contactRequestReadFailure(c *fiber.Ctx, status int) error {
	c.Set("X-Robots-Tag", "noindex")
	c.Set("Cache-Control", "no-store")
	if status == fiber.StatusNotFound {
		return c.Status(status).SendString("Not found")
	}
	return c.Status(fiber.StatusServiceUnavailable).SendString("Service unavailable")
}

type contactRequestBrowserStats struct {
	Status   string `json:"status"`
	IsRead   bool   `json:"is_read"`
	Priority string `json:"priority"`
}
