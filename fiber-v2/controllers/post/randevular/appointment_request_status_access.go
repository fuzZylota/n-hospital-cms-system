package randevular

import (
	"database/sql"
	"errors"
	"fmt"
	"lib"
	"log"
	"models"
	"post/notificationevent"
	"time"

	"github.com/gofiber/fiber/v2"
)

// This is the existing UI cycle, not a new lifecycle policy.
var requestStatusCycle = map[string]struct{ next, timestamp string }{
	"yeni":               {"randevu-verildi", "randevu_verildi_at"},
	"randevu-verildi":    {"randevu-verilemedi", "randevu_verilemedi_at"},
	"randevu-verilemedi": {"hasta-arandi", "hasta_arandi_at"},
	"hasta-arandi":       {"ulasilamadi", "ulasilamadi_at"},
	"ulasilamadi":        {"gelmedi", "gelmedi_at"},
	"gelmedi":            {"hasta-vazgecti", "hasta_vazgecti_at"},
	"hasta-vazgecti":     {"yeni", ""},
}

func requestStatusFailure(c *fiber.Ctx, status int) error {
	message := "Server Hatası: Lütfen daha sonra tekrar deneyin."
	switch status {
	case 400:
		message = "Invalid status"
	case 404:
		message = "Randevu talebi bulunamadı"
	case 409:
		message = "Randevu talebi durumu değişti"
	}
	return c.Status(status).JSON(fiber.Map{"status": status, "message": message})
}

func toggleRandevuRequestStatus(utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"status": 401, "message": "Unauthorized"})
		}
		rrid := c.Params("rrid")
		requestID, valid := editPositiveID(rrid)
		if !valid {
			return requestStatusFailure(c, 404)
		}
		userID, valid := editPositiveID(ourUser.Uid)
		if !valid {
			return requestStatusFailure(c, 404)
		}
		var input struct {
			Status string `json:"status" form:"status"`
		}
		if err := c.BodyParser(&input); err != nil {
			return requestStatusFailure(c, 400)
		}
		transition, valid := requestStatusCycle[input.Status]
		if !valid {
			return requestStatusFailure(c, 400)
		}
		if utilities == nil || utilities.Orm == nil || utilities.Orm.Pool == nil {
			return requestStatusFailure(c, 503)
		}
		tx, err := utilities.Orm.Pool.BeginTx(c.UserContext(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			log.Printf("operation=ToggleRandevuRequestStatus stage=transaction_begin")
			return requestStatusFailure(c, 503)
		}
		finished := false
		defer func() {
			if !finished {
				if err := tx.Rollback(); err != nil {
					log.Printf("operation=ToggleRandevuRequestStatus stage=transaction_rollback")
				}
			}
		}()
		reject := func(status int) error {
			finished = true
			if err := tx.Rollback(); err != nil {
				log.Printf("operation=ToggleRandevuRequestStatus stage=transaction_rollback")
				return requestStatusFailure(c, 503)
			}
			return requestStatusFailure(c, status)
		}

		var role string
		var active sql.NullBool
		err = tx.QueryRowContext(c.UserContext(), "SELECT role, is_active FROM users WHERE uid = $1 FOR UPDATE", userID).Scan(&role, &active)
		if errors.Is(err, sql.ErrNoRows) {
			return reject(404)
		}
		if err != nil || !active.Valid {
			log.Printf("operation=ToggleRandevuRequestStatus stage=user_read")
			return reject(503)
		}
		if !active.Bool || role != "admin" {
			return reject(404)
		}

		var sid sql.NullInt64
		var currentStatus sql.NullString
		err = tx.QueryRowContext(c.UserContext(), "SELECT sid, status FROM randevu_talepleri WHERE rrid = $1 FOR UPDATE", requestID).Scan(&sid, &currentStatus)
		if errors.Is(err, sql.ErrNoRows) {
			return reject(404)
		}
		if err != nil {
			log.Printf("operation=ToggleRandevuRequestStatus stage=request_read")
			return reject(503)
		}
		if !currentStatus.Valid || currentStatus.String != input.Status {
			return reject(409)
		}

		query := "UPDATE randevu_talepleri SET status = $1, last_modified_uid = $2 WHERE rrid = $3 AND sid IS NOT DISTINCT FROM $4 AND status = $5"
		args := []any{transition.next, userID, requestID, sid, currentStatus.String}
		if transition.timestamp != "" {
			query = fmt.Sprintf("UPDATE randevu_talepleri SET status = $1, last_modified_uid = $2, %s = $3 WHERE rrid = $4 AND sid IS NOT DISTINCT FROM $5 AND status = $6", transition.timestamp)
			args = []any{transition.next, userID, time.Now(), requestID, sid, currentStatus.String}
		}
		result, err := tx.ExecContext(c.UserContext(), query, args...)
		if err != nil {
			log.Printf("operation=ToggleRandevuRequestStatus stage=request_update")
			return reject(503)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			log.Printf("operation=ToggleRandevuRequestStatus stage=affected_rows")
			return reject(503)
		}
		if affected == 0 {
			return reject(404)
		}
		if affected != 1 {
			log.Printf("operation=ToggleRandevuRequestStatus stage=affected_rows")
			return reject(503)
		}
		err = tx.Commit()
		finished = true
		if err != nil {
			log.Printf("operation=ToggleRandevuRequestStatus stage=transaction_commit")
			return requestStatusFailure(c, 503)
		}
		go func() {
			if err := notificationevent.Publish(utilities.NotificationHub, notificationevent.Status{Type: "randevu_talebi_status", Rrid: rrid, NewStatus: transition.next}, notificationevent.CurrentRecipients(utilities.UserStatusReader, notificationevent.Recipient)); err != nil {
				log.Printf("operation=ToggleRandevuRequestStatus stage=notification_publish")
			}
		}()
		return c.JSON(fiber.Map{"status": 201, "message": "Randevu talebi başarıyla güncellendi.", "new_status": transition.next})
	}
}
