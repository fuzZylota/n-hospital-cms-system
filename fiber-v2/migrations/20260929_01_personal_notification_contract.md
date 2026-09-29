# Personal notification contract — additive first phase

`schema.sql` begins with `DROP SCHEMA public CASCADE`. Neither it nor
`migrate.sh` / `production-migrate.sh` is an upgrade path for an existing DB.
The one-shot SQL file beside this note is the proposed additive upgrade. No
application wiring or legacy notification data rewrite belongs to this phase.

Before any approved execution, inspect the actual PostgreSQL schema, existing
constraint/index names, table sizes, privileges, application version, backups,
and lock budget. The repository schema is not evidence of production parity.
The migration runs once: it is not idempotent, and a second application fails
on existing columns/table/indexes rather than silently accepting schema drift.
It runs transactionally; an error before COMMIT rolls it back. `lock_timeout`
of five seconds limits each wait to **acquire** a lock, not the time a lock is
held. `statement_timeout` of 60 seconds applies to each statement, not the
whole transaction. `ALTER TABLE` and the added CHECK may scan or lock existing
rows; ordinary `CREATE INDEX` in the same transaction can block writes, and
locks can be held until COMMIT. These settings do not prove an acceptable
total outage window. Table size, PostgreSQL version, current schema and traffic
must be checked before a separately approved rollout; large tables may need a
different, online migration sequence.

On an **approved disposable** DB with the repository's current base schema,
the future verification commands are:

```sh
psql -X -v ON_ERROR_STOP=1 --dbname="$APPROVED_DISPOSABLE_DB_URL" \
  --file=fiber-v2/migrations/20260929_01_personal_notification_contract.sql
psql -X -v ON_ERROR_STOP=1 --dbname="$APPROVED_DISPOSABLE_DB_URL" \
  --file=fiber-v2/migrations/verify_20260929_01_personal_notification_contract.sql
```

The second file creates only synthetic rows inside a transaction, checks legacy
defaults, personal event constraints, duplicate rejection, separate recipient
read states, and both `ON DELETE RESTRICT` foreign keys, then rolls back. Do not
run either command against a real or production DB without separate approval.
The Go fake and static schema tests do **not** prove PostgreSQL constraint,
lock, timing, or migration behavior.

The registered `POST /backend/user/:uid/delete` calls `DeleteUser`, which
directly deletes from `users` (`baserouter/baserouter.go` and
`controllers/post/users/users.go`). A recipient with a receipt will make that
DELETE fail under the chosen `ON DELETE RESTRICT`; a referenced notification
also cannot be cleaned up by a direct DELETE. No notification DELETE endpoint
or cleanup job was found in the current source. `CASCADE` would silently erase
personal read history, and `SET NULL` would detach its owner from the
`(nid, recipient_uid)` primary key. This first phase retains `RESTRICT` and
does not alter those live deletion paths. Account/notification retention,
deletion and error handling need a separate product and application decision.
Do not wire a producer to the owned writer until that decision is implemented
and tested, including the existing user-delete response.

Rollback before migration COMMIT is PostgreSQL transaction rollback. After
COMMIT, do not drop the added columns/table while any personal notification or
receipt may exist: doing so would lose personal read history. Roll back to a
compatible application version that leaves the additive schema in place;
reversing the schema requires a later, separately approved data-retention plan.

The source defaults existing and still-running legacy writers to `legacy` with
`event_kind` and `subject_id` NULL. No existing row gets a fabricated recipient
UID. Producers, panel list/count, and `notification=true` GET writes remain
legacy until their later coordinated package is approved and deployed.

The partial unique index prevents a second personal row for the same
`(event_kind, subject_id)` for the three explicitly allowed creation events.
Equal numeric IDs in different event kinds remain distinct; any future event
kind needs an explicit CHECK/contract migration before use. A positive subject
ID and known event kind establish the stored link's shape, but PostgreSQL
cannot use one ordinary foreign key to
reference the three different source tables selected by `event_kind`. Actual
source-object existence and recipient eligibility need producer-side checks in
the later integration package; neither is claimed by this first schema phase.
