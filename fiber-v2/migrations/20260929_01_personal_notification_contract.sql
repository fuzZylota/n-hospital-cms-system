-- Additive, one-shot upgrade for an existing database. Do not run schema.sql:
-- that file drops the public schema. Apply only after inspecting the actual DB.
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

ALTER TABLE notifications
    ADD COLUMN delivery_model VARCHAR(10) NOT NULL DEFAULT 'legacy',
    ADD COLUMN event_kind VARCHAR(32),
    ADD COLUMN subject_id INTEGER;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_delivery_contract CHECK (
        (delivery_model = 'legacy' AND event_kind IS NULL AND subject_id IS NULL)
        OR (delivery_model = 'personal'
            AND event_kind IS NOT NULL AND subject_id IS NOT NULL
            AND event_kind IN ('request_created', 'application_created', 'contact_created')
            AND subject_id > 0)
    );

-- RESTRICT is deliberate: no receipt or read history disappears implicitly.
-- The current DeleteUser path will fail for recipients with receipts; deletion
-- of a referenced notification/user needs a separate lifecycle decision.
CREATE TABLE notification_receipts (
    nid INTEGER NOT NULL REFERENCES notifications(nid) ON DELETE RESTRICT,
    recipient_uid INTEGER NOT NULL REFERENCES users(uid) ON DELETE RESTRICT,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT notification_receipts_pkey PRIMARY KEY (nid, recipient_uid),
    CONSTRAINT notification_receipts_read_state CHECK (
        (is_read = FALSE AND read_at IS NULL)
        OR (is_read = TRUE AND read_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_notifications_personal_event_subject
    ON notifications(event_kind, subject_id) WHERE delivery_model = 'personal';
CREATE INDEX idx_notifications_created_nid ON notifications(created_at DESC, nid DESC);
CREATE INDEX idx_notification_receipts_user_unread
    ON notification_receipts(recipient_uid, is_read, nid DESC);

COMMIT;
