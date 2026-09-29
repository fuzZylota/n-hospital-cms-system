package panel

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type appointmentRequestListFilter struct {
	query, status, branch, sid, sortBy, sortOrder string
	page                                          int
}

type appointmentRequestListResult struct {
	role                      string
	rows, branches            []map[string]interface{}
	count, today, week, month int64
	itemsPerPage              int64
}

// Authorization, counters, branch choices, and patient rows share one snapshot.
func readAuthorizedAppointmentRequestList(ctx context.Context, db *sql.DB, uid string, filter appointmentRequestListFilter) (appointmentRequestListResult, int) {
	var result appointmentRequestListResult
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
	role, allowed, accessStatus := appointmentRequestReadScope(ctx, tx, userID)
	if accessStatus != fiber.StatusOK {
		return appointmentRequestListResult{}, accessStatus
	}
	result.role = role

	// The existing panel setting controls pagination; read it after authorization.
	if err := tx.QueryRowContext(ctx, "SELECT items_per_page FROM options WHERE option_set_is_active = true LIMIT 1").Scan(&result.itemsPerPage); err != nil || result.itemsPerPage <= 0 || result.itemsPerPage > 1000 {
		return appointmentRequestListResult{}, fiber.StatusServiceUnavailable
	}
	base := " FROM randevu_talepleri rt INNER JOIN subeler s ON rt.sid = s.sid"
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
	branchWhere := ""
	if len(conditions) > 0 {
		branchWhere = " WHERE " + strings.Join(conditions, " AND ")
	}
	result.branches, err = appointmentListRows(ctx, tx, "SELECT DISTINCT s.name, s.sid"+base+branchWhere, args...)
	if err != nil {
		return appointmentRequestListResult{}, fiber.StatusServiceUnavailable
	}
	if filter.query != "" {
		args = append(args, "%"+filter.query+"%")
		p := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, "(rt.patient_first_name ILIKE "+p+" OR rt.patient_last_name ILIKE "+p+" OR rt.patient_phone ILIKE "+p+" OR rt.patient_email ILIKE "+p+" OR rt.message ILIKE "+p+")")
	}
	if filter.status != "all" {
		args = append(args, filter.status)
		conditions = append(conditions, fmt.Sprintf("rt.status = $%d", len(args)))
	}
	for _, requestedBranch := range []string{filter.branch, filter.sid} {
		if requestedBranch == "" || requestedBranch == "all" {
			continue
		}
		requested, parseErr := strconv.ParseInt(requestedBranch, 10, 32)
		if parseErr != nil || requested <= 0 {
			conditions = append(conditions, "1 = 0")
		} else {
			args = append(args, requested)
			conditions = append(conditions, fmt.Sprintf("rt.sid = $%d", len(args)))
		}
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	counts := `SELECT COUNT(*) AS length,
		COUNT(*) FILTER (WHERE rt.created_at >= date_trunc('day', now())) AS today_length,
		COUNT(*) FILTER (WHERE rt.created_at >= date_trunc('week', now())) AS this_week_length,
		COUNT(*) FILTER (WHERE rt.created_at >= date_trunc('month', now())) AS this_month_length`
	if err := tx.QueryRowContext(ctx, counts+base+where, args...).Scan(&result.count, &result.today, &result.week, &result.month); err != nil {
		return appointmentRequestListResult{}, fiber.StatusServiceUnavailable
	}
	if result.count > 0 {
		args = append(args, result.itemsPerPage, int64(filter.page-1)*result.itemsPerPage)
		query := `SELECT rt.rrid, rt.patient_first_name, rt.patient_last_name, rt.patient_phone,
			rt.patient_email, rt.preferred_date, rt.preferred_time, rt.message, rt.drid,
			rt.sid, rt.created_at, rt.updated_at, rt.status, s.name AS sube_name`
		query += base + where + " ORDER BY rt." + filter.sortBy + " " + filter.sortOrder + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
		result.rows, err = appointmentListRows(ctx, tx, query, args...)
		if err != nil {
			return appointmentRequestListResult{}, fiber.StatusServiceUnavailable
		}
	}
	if err := tx.Commit(); err != nil {
		return appointmentRequestListResult{}, fiber.StatusServiceUnavailable
	}
	return result, fiber.StatusOK
}
