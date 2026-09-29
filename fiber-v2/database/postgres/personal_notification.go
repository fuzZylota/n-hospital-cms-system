package postgres

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"

	"models/data"
)

var (
	ErrInvalidPersonalNotification = errors.New("invalid personal notification")
	ErrPersonalNotificationStorage = errors.New("personal notification storage failed")
)

const insertPersonalNotificationSQL = `INSERT INTO notifications
    (message, notification_type, notification_level, sid, link,
     delivery_model, event_kind, subject_id)
    VALUES ($1, $2, $3, $4, $5, 'personal', $6, $7)
    RETURNING nid`

const insertNotificationReceiptSQL = `INSERT INTO notification_receipts (nid, recipient_uid)
    VALUES ($1, $2)`

// PersonalNotificationRepository owns the future personal-event insert path.
// No current producer, panel query, or detail GET calls it in this phase.
type PersonalNotificationRepository struct{ db *sql.DB }

func NewPersonalNotificationRepository(db *sql.DB) *PersonalNotificationRepository {
	return &PersonalNotificationRepository{db: db}
}

var _ data.PersonalNotificationWriter = (*PersonalNotificationRepository)(nil)

// CreatePersonalNotification saves one event and every recipient atomically.
// Backend errors are intentionally not returned or logged with message/UID data.
func (r *PersonalNotificationRepository) CreatePersonalNotification(ctx context.Context, input data.PersonalNotification) (int64, error) {
	if !validPersonalNotification(input) {
		return 0, ErrInvalidPersonalNotification
	}
	if r == nil || r.db == nil {
		return 0, ErrPersonalNotificationStorage
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, ErrPersonalNotificationStorage
	}
	defer tx.Rollback()

	var branchID any
	if input.BranchID != nil {
		branchID = *input.BranchID
	}
	var nid int64
	if err := tx.QueryRowContext(ctx, insertPersonalNotificationSQL,
		input.Message, input.NotificationType, input.NotificationLevel,
		branchID, input.Link, string(input.Kind), input.SubjectID).Scan(&nid); err != nil || nid <= 0 {
		return 0, ErrPersonalNotificationStorage
	}
	for _, uid := range input.RecipientUIDs {
		result, err := tx.ExecContext(ctx, insertNotificationReceiptSQL, nid, uid)
		if err != nil {
			return 0, ErrPersonalNotificationStorage
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return 0, ErrPersonalNotificationStorage
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, ErrPersonalNotificationStorage
	}
	return nid, nil
}

func validPersonalNotification(input data.PersonalNotification) bool {
	switch input.Kind {
	case data.RequestCreated, data.ApplicationCreated, data.ContactCreated:
	default:
		return false
	}
	if input.SubjectID <= 0 || input.SubjectID > math.MaxInt32 ||
		strings.TrimSpace(input.Message) == "" ||
		len(input.Link) == 0 || len(input.Link) > 200 || !strings.HasPrefix(input.Link, "/panel/") ||
		len(input.RecipientUIDs) == 0 {
		return false
	}
	switch input.NotificationType {
	case "success", "warning", "danger", "info":
	default:
		return false
	}
	switch input.NotificationLevel {
	case "moderator", "admin", "santral", "ik", "all":
	default:
		return false
	}
	if input.BranchID != nil && (*input.BranchID <= 0 || *input.BranchID > math.MaxInt32) {
		return false
	}
	seen := make(map[int64]struct{}, len(input.RecipientUIDs))
	for _, uid := range input.RecipientUIDs {
		if uid <= 0 || uid > math.MaxInt32 {
			return false
		}
		if _, duplicate := seen[uid]; duplicate {
			return false
		}
		seen[uid] = struct{}{}
	}
	return true
}
