package panel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// The request list and export use this same current-account/branch decision
// inside their own read-only snapshots. The JWT supplies only the UID.
func appointmentRequestReadScope(ctx context.Context, tx *sql.Tx, userID int64) (string, []int64, int) {
	var role string
	var userBranch sql.NullInt64
	var active bool
	err := tx.QueryRowContext(ctx, "SELECT role, sid, is_active FROM users WHERE uid = $1", userID).Scan(&role, &userBranch, &active)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !active) {
		return "", nil, fiber.StatusNotFound
	}
	if err != nil {
		return "", nil, fiber.StatusServiceUnavailable
	}
	var allowed []int64
	switch role {
	case "admin":
		return role, nil, fiber.StatusOK
	case "moderator", "santral":
		permissionRows, err := tx.QueryContext(ctx, "SELECT sid FROM user_branch_permissions WHERE uid = $1 AND can_view = true", userID)
		if err != nil {
			return "", nil, fiber.StatusServiceUnavailable
		}
		for permissionRows.Next() {
			var sid int64
			if err := permissionRows.Scan(&sid); err != nil {
				permissionRows.Close()
				return "", nil, fiber.StatusServiceUnavailable
			}
			if role == "moderator" || (userBranch.Valid && sid == userBranch.Int64) {
				allowed = append(allowed, sid)
			}
		}
		rowErr := permissionRows.Err()
		closeErr := permissionRows.Close()
		if rowErr != nil || closeErr != nil {
			return "", nil, fiber.StatusServiceUnavailable
		}
		if len(allowed) == 0 {
			return "", nil, fiber.StatusNotFound
		}
		return role, allowed, fiber.StatusOK
	default:
		return "", nil, fiber.StatusNotFound
	}
}

type appointmentRequestExportFilter struct {
	dateStart, dateEnd, statuses string
}

// No patient row leaves the transaction until authorization and all reads pass.
func readAuthorizedAppointmentRequestExport(ctx context.Context, db *sql.DB, uid string, filter appointmentRequestExportFilter) ([]map[string]interface{}, int) {
	userID, err := strconv.ParseInt(uid, 10, 32)
	if err != nil || userID <= 0 {
		return nil, fiber.StatusNotFound
	}
	if db == nil {
		return nil, fiber.StatusServiceUnavailable
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fiber.StatusServiceUnavailable
	}
	defer tx.Rollback()
	role, allowed, status := appointmentRequestReadScope(ctx, tx, userID)
	if status != fiber.StatusOK {
		return nil, status
	}
	conditions := []string{}
	args := []interface{}{}
	if role != "admin" {
		placeholders := make([]string, len(allowed))
		for i, sid := range allowed {
			args = append(args, sid)
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		conditions = append(conditions, "rt.sid IN ("+strings.Join(placeholders, ",")+")")
	}
	if filter.dateStart != "" {
		args = append(args, filter.dateStart)
		conditions = append(conditions, fmt.Sprintf("rt.created_at >= $%d", len(args)))
	}
	if filter.dateEnd != "" {
		args = append(args, filter.dateEnd+" 23:59:59")
		conditions = append(conditions, fmt.Sprintf("rt.created_at <= $%d", len(args)))
	}
	if filter.statuses != "" {
		placeholders := []string{}
		for _, requested := range strings.Split(filter.statuses, ",") {
			args = append(args, strings.TrimSpace(requested))
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		conditions = append(conditions, "rt.status IN ("+strings.Join(placeholders, ",")+")")
	}
	query := `SELECT rt.rrid, rt.patient_first_name, rt.patient_last_name, rt.patient_phone,
		rt.patient_email, rt.message, rt.created_at, rt.status, s.name AS sube_name
		FROM randevu_talepleri rt INNER JOIN subeler s ON rt.sid = s.sid`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY rt.created_at DESC"
	rows, err := appointmentListRows(ctx, tx, query, args...)
	if err != nil {
		return nil, fiber.StatusServiceUnavailable
	}
	if err := tx.Commit(); err != nil {
		return nil, fiber.StatusServiceUnavailable
	}
	return rows, fiber.StatusOK
}
