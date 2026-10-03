package randevular

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Set only by synthetic HTTP tests; production sends through lib.SendEmail.
var addAppointmentMailHook func(*models.EmailInfos) error

func createAppointmentFailure(c *fiber.Ctx, status int) error {
	message := "Server Hatası: Lütfen daha sonra tekrar deneyin."
	switch status {
	case 400:
		message = "Geçersiz randevu bilgisi"
	case 403:
		message = "Forbidden"
	case 404:
		message = "Randevu talebi bulunamadı"
	case 409:
		message = "Randevu talebi veya saat uygun değil"
	}
	return c.Status(status).JSON(fiber.Map{"status": status, "message": message})
}

// A blank request ID is a manual appointment. A nonblank ID must identify one
// locked request; the client-supplied branch never identifies its owner.
func createRequestID(value string) (sql.NullInt64, int) {
	if value == "" {
		return sql.NullInt64{}, 0
	}
	id, ok := editPositiveID(value)
	if !ok {
		return sql.NullInt64{}, 400
	}
	return sql.NullInt64{Int64: id, Valid: true}, 0
}

func authorizeCreateAppointment(ctx context.Context, tx *sql.Tx, uid string, inputs models.Randevular, requestID sql.NullInt64) (int64, sql.NullInt64, int) {
	userID, ok := editPositiveID(uid)
	if !ok {
		return 0, sql.NullInt64{}, 403
	}
	var role string
	var active sql.NullBool
	err := tx.QueryRowContext(ctx, "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE", userID).Scan(&role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, sql.NullInt64{}, 403
	}
	if err != nil || !active.Valid {
		return 0, sql.NullInt64{}, 503
	}
	if !active.Bool || !isAppointmentWriteRole(role) {
		return 0, sql.NullInt64{}, 403
	}

	branchID, ok := editPositiveID(inputs.Sid)
	if !ok {
		return 0, sql.NullInt64{}, 400
	}
	// Admin dışı roller yalnız yetkili oldukları şubede oluşturabilir.
	if status := authorizeBranchWrite(ctx, tx, userID, role, sql.NullInt64{Int64: branchID, Valid: true}); status != 0 {
		return 0, sql.NullInt64{}, status
	}
	if requestID.Valid {
		var requestSID sql.NullInt64
		err = tx.QueryRowContext(ctx, "SELECT sid FROM randevu_talepleri WHERE rrid = $1 FOR UPDATE", requestID.Int64).Scan(&requestSID)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, sql.NullInt64{}, 404
		}
		if err != nil {
			return 0, sql.NullInt64{}, 503
		}
		if !requestSID.Valid || requestSID.Int64 <= 0 || requestSID.Int64 != branchID {
			return 0, sql.NullInt64{}, 409
		}
		var linked bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM randevular WHERE rrid = $1)", requestID.Int64).Scan(&linked); err != nil {
			return 0, sql.NullInt64{}, 503
		}
		if linked {
			return 0, sql.NullInt64{}, 409
		}
	}

	var branchExists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM subeler WHERE sid = $1)", branchID).Scan(&branchExists); err != nil {
		return 0, sql.NullInt64{}, 503
	}
	if !branchExists {
		return 0, sql.NullInt64{}, 400
	}

	var doctorID sql.NullInt64
	if inputs.Drid != "" && inputs.Drid != "0" {
		id, ok := editPositiveID(inputs.Drid)
		if !ok {
			return 0, sql.NullInt64{}, 400
		}
		doctorID = sql.NullInt64{Int64: id, Valid: true}
		var doctorExists bool
		// sid is the structured primary branch. calistigi_subeler_text is not
		// an authoritative, queryable secondary-branch relationship.
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM doktorlar WHERE drid = $1 AND sid = $2 AND is_active = true)", id, branchID).Scan(&doctorExists); err != nil {
			return 0, sql.NullInt64{}, 503
		}
		if !doctorExists {
			return 0, sql.NullInt64{}, 400
		}
	}
	return branchID, doctorID, 0
}

func checkCreateAppointmentSlot(ctx context.Context, tx *sql.Tx, inputs models.Randevular, doctorID sql.NullInt64) int {
	if inputs.Duration <= 0 {
		return 400
	}
	if !doctorID.Valid {
		return 0
	}
	start := inputs.AppointmentTime.Format("15:04:05")
	end := inputs.AppointmentTime.Add(time.Duration(inputs.Duration) * time.Minute).Format("15:04:05")
	var conflicts int64
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM randevular
		WHERE drid = $1 AND appointment_date = $2
		AND (appointment_time + (duration || ' minutes')::interval) > $3
		AND appointment_time < $4`, doctorID.Int64, inputs.AppointmentDate.Format("2006-01-02"), start, end).Scan(&conflicts)
	if err != nil {
		return 503
	}
	if conflicts > 0 {
		return 409
	}
	return 0
}

func insertCreatedAppointment(ctx context.Context, tx *sql.Tx, columns []string, values []any) (int64, error) {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query := fmt.Sprintf("INSERT INTO randevular (%s) VALUES (%s) RETURNING rid", strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	var rid int64
	if err := tx.QueryRowContext(ctx, query, values...).Scan(&rid); err != nil {
		return 0, err
	}
	if rid <= 0 {
		return 0, errors.New("invalid appointment insert result")
	}
	return rid, nil
}
