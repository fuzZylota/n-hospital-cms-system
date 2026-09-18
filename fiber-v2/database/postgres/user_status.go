package postgres

import (
	"context"
	"database/sql"
	"strconv"

	"models/data"
)

const lookupUserStatusSQL = `SELECT uid, is_active, role
FROM users
WHERE uid = $1`

// UserStatusRepository reads current user state from the supplied pool.
type UserStatusRepository struct {
	db *sql.DB
}

// NewUserStatusRepository borrows db. It does not open, register, or close it.
func NewUserStatusRepository(db *sql.DB) *UserStatusRepository {
	return &UserStatusRepository{db: db}
}

var _ data.UserStatusReader = (*UserStatusRepository)(nil)

func (r *UserStatusRepository) LookupUserStatus(ctx context.Context, userID string) (result data.UserStatus, err error) {
	id, parseErr := parseCanonicalPositiveID(userID)
	if parseErr != nil {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readInvalidIdentifier, nil)
	}
	if r == nil || r.db == nil {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readDependency, nil)
	}

	rows, queryErr := r.db.QueryContext(ctx, lookupUserStatusSQL, id)
	if queryErr != nil {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readQuery, queryErr)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			result = data.UserStatus{}
			err = newRepositoryReadError(readUserStatus, readRowsClose, closeErr)
		}
	}()

	if !rows.Next() {
		if rowsErr := rows.Err(); rowsErr != nil {
			return data.UserStatus{}, newRepositoryReadError(readUserStatus, readRowsClose, rowsErr)
		}
		return data.UserStatus{Found: false}, nil
	}

	var returnedID int64
	var active sql.NullBool
	var role string
	if scanErr := rows.Scan(&returnedID, &active, &role); scanErr != nil {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readScan, scanErr)
	}
	if rows.Next() {
		if scanErr := rows.Scan(&returnedID, &active, &role); scanErr != nil {
			return data.UserStatus{}, newRepositoryReadError(readUserStatus, readScan, scanErr)
		}
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readCardinality, nil)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readRowsClose, rowsErr)
	}
	if returnedID <= 0 {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readInvalidIdentifier, nil)
	}
	if returnedID != id {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readSelectedRow, nil)
	}
	if !active.Valid {
		return data.UserStatus{}, newRepositoryReadError(readUserStatus, readSelectedRow, nil)
	}
	return data.UserStatus{Found: true, Active: active.Bool, Role: role}, nil
}

func parseCanonicalPositiveID(value string) (int64, error) {
	if len(value) == 0 || value[0] < '1' || value[0] > '9' {
		return 0, strconv.ErrSyntax
	}
	for index := 1; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return 0, strconv.ErrSyntax
		}
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
