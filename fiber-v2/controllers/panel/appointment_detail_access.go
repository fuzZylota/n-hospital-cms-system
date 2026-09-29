package panel

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// The branch, principal, permission, and detail reads share one PostgreSQL
// snapshot. The final predicate also ties the PII row to the authorized branch.
func readAuthorizedAppointmentDetail(ctx context.Context, db *sql.DB, rid, uid string) ([]map[string]interface{}, string, int) {
	return readAuthorizedAppointment(ctx, db, rid, uid, false)
}

func readAuthorizedAppointmentEdit(ctx context.Context, db *sql.DB, rid, uid string) ([]map[string]interface{}, string, int) {
	return readAuthorizedAppointment(ctx, db, rid, uid, true)
}

func readAuthorizedAppointment(ctx context.Context, db *sql.DB, rid, uid string, edit bool) ([]map[string]interface{}, string, int) {
	appointmentID, err := strconv.ParseInt(rid, 10, 32)
	if err != nil || appointmentID <= 0 || uid == "" {
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
	err = tx.QueryRowContext(ctx, "SELECT sid FROM randevular WHERE rid = $1", appointmentID).Scan(&branchID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !branchID.Valid) {
		return nil, "", fiber.StatusNotFound
	}
	if err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}

	var role string
	var userBranch sql.NullInt64
	var active bool
	err = tx.QueryRowContext(ctx, "SELECT role, sid, is_active FROM users WHERE uid = $1", userID).Scan(&role, &userBranch, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", fiber.StatusNotFound
	}
	if err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	if !active {
		return nil, "", fiber.StatusNotFound
	}

	switch role {
	case "admin":
	case "moderator", "santral":
		if role == "santral" && (!userBranch.Valid || userBranch.Int64 != branchID.Int64) {
			return nil, "", fiber.StatusNotFound
		}
		var allowed bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM user_branch_permissions
			WHERE uid = $1 AND sid = $2 AND can_view = true
		)`, userID, branchID.Int64).Scan(&allowed)
		if err != nil {
			return nil, "", fiber.StatusServiceUnavailable
		}
		if !allowed {
			return nil, "", fiber.StatusNotFound
		}
	default:
		return nil, "", fiber.StatusNotFound
	}

	query := `SELECT r.*, d.title AS doctor_title, d.first_name AS doctor_first_name,
		d.last_name AS doctor_last_name, s.name AS sube_name, s.city AS sube_city,
		b.name AS branch_name, rt.rrid AS related_appointment_rid,
		rt.patient_first_name AS related_appointment_patient_first_name,
		rt.patient_last_name AS related_appointment_patient_last_name,
		rt.patient_phone AS related_appointment_patient_phone,
		rt.patient_email AS related_appointment_patient_email,
		rt.preferred_date AS related_appointment_preferred_date,
		rt.preferred_time AS related_appointment_preferred_time,
		rt.message AS related_appointment_message,
		ak.name AS related_appointment_anlasmali_kurum_name,
		tb.name AS related_appointment_tibbi_birim_name,
		t.name AS related_appointment_tedkik_name
		FROM randevular r
		LEFT JOIN doktorlar d ON r.drid = d.drid
		LEFT JOIN subeler s ON r.sid = s.sid
		LEFT JOIN branslar b ON r.brid = b.brid
		LEFT JOIN randevu_talepleri rt ON r.rrid = rt.rrid
		LEFT JOIN anlasmali_kurumlar ak ON r.akid = ak.akid
		LEFT JOIN tibbi_birimler tb ON r.tbid = tb.tbid
		LEFT JOIN tedkikler t ON r.tid = t.tid
		WHERE r.rid = $1 AND r.sid = $2`
	if edit {
		query = "SELECT r.* FROM randevular r WHERE r.rid = $1 AND r.sid = $2"
	}
	rows, err := tx.QueryContext(ctx, query, appointmentID, branchID.Int64)
	if err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	if !rows.Next() {
		if rows.Err() != nil {
			return nil, "", fiber.StatusServiceUnavailable
		}
		return nil, "", fiber.StatusNotFound
	}
	values := make([]interface{}, len(columns))
	dest := make([]interface{}, len(columns))
	for i := range dest {
		dest[i] = &values[i]
	}
	if err := rows.Scan(dest...); err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	row := make(map[string]interface{}, len(columns))
	for i, column := range columns {
		if bytes, ok := values[i].([]byte); ok {
			row[column] = string(bytes)
		} else {
			row[column] = values[i]
		}
	}
	if err := rows.Close(); err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	if err := tx.Commit(); err != nil {
		return nil, "", fiber.StatusServiceUnavailable
	}
	return []map[string]interface{}{row}, role, fiber.StatusOK
}

func appointmentDetailUnavailable(c *fiber.Ctx, status int) error {
	c.Set("X-Robots-Tag", "noindex")
	c.Set("Cache-Control", "no-store")
	if status == fiber.StatusNotFound {
		return c.Status(status).SendString("Bulunamadı")
	}
	return c.Status(fiber.StatusServiceUnavailable).SendString("Hizmet kullanılamıyor")
}
