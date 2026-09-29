package panel

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Read the current actor, branch permissions, and newest request rows in one snapshot.
func readAuthorizedAppointmentRequestLatest(ctx context.Context, db *sql.DB, uid, since string) ([]map[string]interface{}, int) {
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
	args := []interface{}{}
	conditions := []string{}
	if role != "admin" {
		placeholders := make([]string, len(allowed))
		for i, sid := range allowed {
			args = append(args, sid)
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		conditions = append(conditions, "rt.sid IN ("+strings.Join(placeholders, ",")+")")
	}
	if since != "" {
		args = append(args, since)
		conditions = append(conditions, fmt.Sprintf("rt.created_at > $%d", len(args)))
	}
	query := `SELECT rt.rrid, rt.patient_first_name, rt.patient_last_name,
		rt.patient_phone, rt.message, rt.created_at, rt.status, s.name AS sube_name
		FROM randevu_talepleri rt INNER JOIN subeler s ON rt.sid = s.sid`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY rt.created_at DESC LIMIT 20"
	rows, err := appointmentListRows(ctx, tx, query, args...)
	if err != nil {
		return nil, fiber.StatusServiceUnavailable
	}
	if err := tx.Commit(); err != nil {
		return nil, fiber.StatusServiceUnavailable
	}
	return rows, fiber.StatusOK
}
