-- Run only on an approved disposable PostgreSQL database after the migration.
-- All synthetic rows are rolled back. This is NOT a production verification.
BEGIN;
DO $verify$
DECLARE
    suffix TEXT := txid_current()::TEXT;
    first_uid INTEGER;
    second_uid INTEGER;
    legacy_nid INTEGER;
    personal_nid INTEGER;
    different_kind_nid INTEGER;
    subject INTEGER := (txid_current() % 2000000000)::INTEGER + 1;
    legacy_model TEXT;
    legacy_kind TEXT;
    legacy_subject INTEGER;
    legacy_read BOOLEAN;
    first_read BOOLEAN;
    second_read BOOLEAN;
BEGIN
    INSERT INTO users (name, surname, password, email, phone, role, is_active)
    VALUES ('Contract', 'One', 'synthetic-unusable', 'contract-' || suffix || '-one@example.invalid',
            'contract-' || suffix || '-one', 'admin', TRUE)
    RETURNING uid INTO first_uid;
    INSERT INTO users (name, surname, password, email, phone, role, is_active)
    VALUES ('Contract', 'Two', 'synthetic-unusable', 'contract-' || suffix || '-two@example.invalid',
            'contract-' || suffix || '-two', 'admin', TRUE)
    RETURNING uid INTO second_uid;

    INSERT INTO notifications (message, notification_type, notification_level, link, is_read)
    VALUES ('synthetic legacy', 'info', 'admin', '/panel/contract-legacy', TRUE)
    RETURNING nid INTO legacy_nid;
    SELECT delivery_model, event_kind, subject_id, is_read
    INTO legacy_model, legacy_kind, legacy_subject, legacy_read
    FROM notifications WHERE nid = legacy_nid;
    IF legacy_model <> 'legacy' OR legacy_kind IS NOT NULL OR legacy_subject IS NOT NULL
       OR legacy_read IS NOT TRUE THEN
        RAISE EXCEPTION 'legacy default failed';
    END IF;

    BEGIN
        INSERT INTO notifications (message, delivery_model, event_kind, subject_id)
        VALUES ('synthetic invalid', 'personal', NULL, subject);
        RAISE EXCEPTION 'personal NULL kind was accepted';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
    BEGIN
        INSERT INTO notifications (message, delivery_model, event_kind, subject_id)
        VALUES ('synthetic invalid', 'personal', 'contact_created', NULL);
        RAISE EXCEPTION 'personal NULL subject was accepted';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
    BEGIN
        INSERT INTO notifications (message, delivery_model, event_kind, subject_id)
        VALUES ('synthetic invalid', 'personal', 'contact_created', 0);
        RAISE EXCEPTION 'personal zero subject was accepted';
    EXCEPTION WHEN check_violation THEN NULL;
    END;

    INSERT INTO notifications (message, notification_type, notification_level, link,
                               delivery_model, event_kind, subject_id)
    VALUES ('synthetic personal', 'info', 'admin', '/panel/contract-personal',
            'personal', 'contact_created', subject)
    RETURNING nid INTO personal_nid;
    -- The same numeric source ID in a different creation-event namespace is distinct.
    INSERT INTO notifications (message, delivery_model, event_kind, subject_id)
    VALUES ('synthetic other kind', 'personal', 'application_created', subject)
    RETURNING nid INTO different_kind_nid;
    IF different_kind_nid = personal_nid THEN
        RAISE EXCEPTION 'different event kinds were merged';
    END IF;
    BEGIN
        INSERT INTO notifications (message, delivery_model, event_kind, subject_id)
        VALUES ('synthetic duplicate', 'personal', 'contact_created', subject);
        RAISE EXCEPTION 'duplicate event was accepted';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;

    INSERT INTO notification_receipts (nid, recipient_uid)
    VALUES (personal_nid, first_uid), (personal_nid, second_uid);
    INSERT INTO notification_receipts (nid, recipient_uid)
    VALUES (different_kind_nid, first_uid);
    BEGIN
        INSERT INTO notification_receipts (nid, recipient_uid)
        VALUES (personal_nid, first_uid);
        RAISE EXCEPTION 'duplicate recipient was accepted';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;
    UPDATE notification_receipts SET is_read = TRUE, read_at = CURRENT_TIMESTAMP
    WHERE nid = personal_nid AND recipient_uid = first_uid;
    SELECT is_read INTO first_read FROM notification_receipts
    WHERE nid = personal_nid AND recipient_uid = first_uid;
    SELECT is_read INTO second_read FROM notification_receipts
    WHERE nid = personal_nid AND recipient_uid = second_uid;
    IF first_read IS NOT TRUE OR second_read IS NOT FALSE THEN
        RAISE EXCEPTION 'recipients did not retain independent state';
    END IF;

    BEGIN
        DELETE FROM notifications WHERE nid = personal_nid;
        RAISE EXCEPTION 'notification FK RESTRICT failed';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;
    BEGIN
        DELETE FROM users WHERE uid = first_uid;
        RAISE EXCEPTION 'user FK RESTRICT failed';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;
END;
$verify$;
ROLLBACK;
