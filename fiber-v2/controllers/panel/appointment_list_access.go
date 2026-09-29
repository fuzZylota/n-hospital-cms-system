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

type appointmentListResult struct {
	role     string
	rows     []map[string]interface{}
	branches []map[string]interface{}
	count    int
}

type appointmentListFilter struct {
	query, status, branch, sortBy, sortOrder string
	page                                     int
}

// All authorization and list reads use the same read-only snapshot. Both the
// count and PII query receive the identical branch/search/status predicate.
func readAuthorizedAppointmentList(ctx context.Context, db *sql.DB, uid string, filter appointmentListFilter) (appointmentListResult, int) {
	var result appointmentListResult
	userID, err := strconv.ParseInt(uid, 10, 32)
	if err != nil || userID <= 0 {
		return result, fiber.StatusNotFound
	}
	if db == nil {
		return result, fiber.StatusServiceUnavailable
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return result, fiber.StatusServiceUnavailable
	}
	defer tx.Rollback()
	var role string
	var userBranch sql.NullInt64
	var active bool
	err = tx.QueryRowContext(ctx, "SELECT role, sid, is_active FROM users WHERE uid = $1", userID).Scan(&role, &userBranch, &active)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !active) {
		return result, fiber.StatusNotFound
	}
	if err != nil {
		return result, fiber.StatusServiceUnavailable
	}
	result.role = role
	var allowed []int64
	switch role {
	case "admin":
	case "moderator", "santral":
		permissionRows, err := tx.QueryContext(ctx, "SELECT sid FROM user_branch_permissions WHERE uid = $1 AND can_view = true", userID)
		if err != nil {
			return appointmentListResult{}, fiber.StatusServiceUnavailable
		}
		for permissionRows.Next() {
			var sid int64
			if err := permissionRows.Scan(&sid); err != nil {
				permissionRows.Close()
				return appointmentListResult{}, fiber.StatusServiceUnavailable
			}
			if role == "moderator" || (userBranch.Valid && sid == userBranch.Int64) {
				allowed = append(allowed, sid)
			}
		}
		err = permissionRows.Err()
		closeErr := permissionRows.Close()
		if err != nil {
			return appointmentListResult{}, fiber.StatusServiceUnavailable
		}
		if closeErr != nil {
			return appointmentListResult{}, fiber.StatusServiceUnavailable
		}
		if len(allowed) == 0 {
			return appointmentListResult{}, fiber.StatusNotFound
		}
	default:
		return appointmentListResult{}, fiber.StatusNotFound
	}

	branchSQL := ""
	branchArgs := []interface{}{}
	if role != "admin" {
		placeholders := make([]string, len(allowed))
		for i, sid := range allowed {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			branchArgs = append(branchArgs, sid)
		}
		branchSQL = "r.sid IN (" + strings.Join(placeholders, ",") + ")"
	}
	branchWhere := ""
	if branchSQL != "" {
		branchWhere = " WHERE " + branchSQL
	}
	result.branches, err = appointmentListRows(ctx, tx, "SELECT DISTINCT s.name, s.sid FROM randevular r LEFT JOIN subeler s ON r.sid = s.sid"+branchWhere, branchArgs...)
	if err != nil {
		return appointmentListResult{}, fiber.StatusServiceUnavailable
	}

	conditions := []string{}
	args := append([]interface{}{}, branchArgs...)
	if branchSQL != "" {
		conditions = append(conditions, branchSQL)
	}
	if filter.query != "" {
		args = append(args, "%"+filter.query+"%")
		p := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, "(r.patient_first_name ILIKE "+p+" OR r.patient_last_name ILIKE "+p+" OR r.patient_phone ILIKE "+p+" OR r.patient_email ILIKE "+p+" OR r.complaint ILIKE "+p+" OR r.notes ILIKE "+p+")")
	}
	if filter.status != "all" {
		args = append(args, filter.status)
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", len(args)))
	}
	if filter.branch != "all" {
		requested, parseErr := strconv.ParseInt(filter.branch, 10, 32)
		if parseErr != nil || requested <= 0 {
			conditions = append(conditions, "1 = 0")
		} else {
			args = append(args, requested)
			conditions = append(conditions, fmt.Sprintf("r.sid = $%d", len(args)))
		}
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM randevular r"+where, args...).Scan(&result.count); err != nil {
		return appointmentListResult{}, fiber.StatusServiceUnavailable
	}
	// A missing or unauthorized branch must not cause a PII read.
	if result.count > 0 {
		args = append(args, 10, (filter.page-1)*10)
		query := `SELECT r.rid, r.patient_first_name, r.patient_last_name, r.patient_phone,
			r.patient_email, r.appointment_date, r.appointment_time, r.duration,
			r.status, r.payment_status, r.price, r.created_at, r.updated_at,
			d.title, d.first_name AS doctor_first_name, d.last_name AS doctor_last_name,
			s.name AS sube_name, s.city AS sube_city, b.name AS branch_name
			FROM randevular r LEFT JOIN doktorlar d ON r.drid = d.drid
			LEFT JOIN subeler s ON r.sid = s.sid LEFT JOIN branslar b ON r.brid = b.brid`
		query += where + " ORDER BY r." + filter.sortBy + " " + filter.sortOrder + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
		result.rows, err = appointmentListRows(ctx, tx, query, args...)
		if err != nil {
			return appointmentListResult{}, fiber.StatusServiceUnavailable
		}
	}
	if err := tx.Commit(); err != nil {
		return appointmentListResult{}, fiber.StatusServiceUnavailable
	}
	return result, fiber.StatusOK
}

func appointmentListRows(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := []map[string]interface{}{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		dest := make([]interface{}, len(columns))
		for i := range dest {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(columns))
		for i, column := range columns {
			if bytes, ok := values[i].([]byte); ok {
				row[column] = string(bytes)
			} else {
				row[column] = values[i]
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return result, nil
}
