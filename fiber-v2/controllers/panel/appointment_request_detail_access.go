package panel

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Resolve the real request branch before any patient data is read. The joined
// definitive appointment is constrained to that same authorized branch.
func readAuthorizedAppointmentRequestDetail(ctx context.Context, db *sql.DB, rrid, uid string) ([]map[string]interface{}, string, int) {
	requestID, err := strconv.ParseInt(rrid, 10, 32)
	if err != nil || requestID <= 0 {
		return nil, "", fiber.StatusNotFound
	}
	userID, err := strconv.ParseInt(uid, 10, 32)
	if err != nil || userID <= 0 {
		return nil, "", fiber.StatusNotFound
	}
	if db == nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	defer tx.Rollback()

	var branchID sql.NullInt64
	err = tx.QueryRowContext(ctx, "SELECT sid FROM randevu_talepleri WHERE rrid = $1", requestID).Scan(&branchID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !branchID.Valid) {
		return nil, "", fiber.StatusNotFound
	}
	if err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	role, allowed, status := appointmentRequestReadScope(ctx, tx, userID)
	if status != fiber.StatusOK {
		return nil, "", status
	}
	if role != "admin" {
		permitted := false
		for _, sid := range allowed {
			if sid == branchID.Int64 {
				permitted = true
				break
			}
		}
		if !permitted {
			return nil, "", fiber.StatusNotFound
		}
	}

	query := `SELECT rt.*, d.title AS doctor_title, d.first_name AS doctor_first_name,
		d.last_name AS doctor_last_name, s.name AS sube_name, s.city AS sube_city,
		r.rid AS related_appointment_rid, r.appointment_date AS related_appointment_date,
		r.appointment_time AS related_appointment_time, r.status AS related_appointment_status,
		u.name AS user_first_name, u.surname AS user_surname, u.email AS user_email,
		u.role AS user_role
		FROM randevu_talepleri rt
		LEFT JOIN doktorlar d ON rt.drid = d.drid
		LEFT JOIN subeler s ON rt.sid = s.sid
		LEFT JOIN randevular r ON rt.rrid = r.rrid AND r.sid = rt.sid
		LEFT JOIN users u ON u.uid = rt.last_modified_uid
		WHERE rt.rrid = $1 AND rt.sid = $2`
	rows, err := appointmentListRows(ctx, tx, query, requestID, branchID.Int64)
	if err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	if len(rows) == 0 {
		return nil, "", fiber.StatusNotFound
	}
	if err := tx.Commit(); err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	return rows, role, fiber.StatusOK
}
