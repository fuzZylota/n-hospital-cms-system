package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestPersonalNotificationSchemaContract(t *testing.T) {
	fresh, err := os.ReadFile("../../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	upgrade, err := os.ReadFile("../../migrations/20260929_01_personal_notification_contract.sql")
	if err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{"new installation": string(fresh), "additive upgrade": string(upgrade)} {
		for _, clause := range []string{
			"delivery_model VARCHAR(10) NOT NULL DEFAULT 'legacy'",
			"event_kind VARCHAR(32)", "subject_id INTEGER",
			"delivery_model = 'legacy' AND event_kind IS NULL AND subject_id IS NULL",
			"event_kind IS NOT NULL AND subject_id IS NOT NULL",
			"event_kind IN ('request_created', 'application_created', 'contact_created')",
			"subject_id > 0",
			"CONSTRAINT notification_receipts_pkey PRIMARY KEY (nid, recipient_uid)",
			"REFERENCES notifications(nid) ON DELETE RESTRICT",
			"REFERENCES users(uid) ON DELETE RESTRICT",
			"is_read BOOLEAN NOT NULL DEFAULT FALSE",
			"CONSTRAINT notification_receipts_read_state CHECK",
			"CREATE UNIQUE INDEX idx_notifications_personal_event_subject",
			"ON notifications(event_kind, subject_id) WHERE delivery_model = 'personal'",
			"CREATE INDEX idx_notifications_created_nid",
			"CREATE INDEX idx_notification_receipts_user_unread",
			"ON notification_receipts(recipient_uid, is_read, nid DESC)",
		} {
			if !strings.Contains(source, clause) {
				t.Errorf("%s lacks %q", name, clause)
			}
		}
	}
	if strings.Contains(string(upgrade), "UPDATE notifications") || strings.Contains(string(upgrade), "ALTER COLUMN is_read") ||
		strings.Contains(string(upgrade), "DROP TABLE") || strings.Contains(string(upgrade), "DROP SCHEMA") ||
		strings.Contains(string(upgrade), "ON DELETE CASCADE") || strings.Contains(string(upgrade), "ON DELETE SET NULL") {
		t.Fatal("upgrade rewrites or deletes legacy notification rows")
	}
}
