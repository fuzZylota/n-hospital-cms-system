package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"models/data"
)

const listHeaderParentsSQL = `SELECT hbid, title, parent_id
FROM header_buttons
WHERE parent_id IS NULL`

// HeaderButtonRepository reads header-button data from PostgreSQL.
type HeaderButtonRepository struct {
	db *sql.DB
}

// NewHeaderButtonRepository creates a repository that uses the supplied,
// application-owned database pool. It does not open or close the pool.
func NewHeaderButtonRepository(db *sql.DB) *HeaderButtonRepository {
	return &HeaderButtonRepository{db: db}
}

var _ data.HeaderButtonReader = (*HeaderButtonRepository)(nil)

// ListHeaderParents returns the top-level header buttons used by the parent
// selector. Numeric identifiers from the repository schema are converted to API
// strings here; production schema parity must be verified before wiring.
func (r *HeaderButtonRepository) ListHeaderParents(ctx context.Context) (parents []data.HeaderParent, err error) {
	if r == nil || r.db == nil {
		return nil, newHeaderButtonReadError(headerButtonDependency, nil)
	}

	rows, queryErr := r.db.QueryContext(ctx, listHeaderParentsSQL)
	if queryErr != nil {
		return nil, newHeaderButtonReadError(headerButtonQuery, queryErr)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			parents = nil
			err = newHeaderButtonReadError(headerButtonClose, closeErr)
		}
	}()

	parents = make([]data.HeaderParent, 0)
	for rows.Next() {
		var (
			id       int64
			title    string
			parentID sql.NullInt64
		)
		if scanErr := rows.Scan(&id, &title, &parentID); scanErr != nil {
			return nil, newHeaderButtonReadError(headerButtonScan, scanErr)
		}

		parent := data.HeaderParent{
			ID:    strconv.FormatInt(id, 10),
			Title: title,
		}
		if parentID.Valid {
			value := strconv.FormatInt(parentID.Int64, 10)
			parent.ParentID = &value
		}
		parents = append(parents, parent)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		// database/sql also reports automatic EOF-close failures through Err.
		// Its public API cannot distinguish those from iteration failures.
		return nil, newHeaderButtonReadError(headerButtonRowsClose, rowsErr)
	}

	return parents, nil
}

type headerButtonReadStage uint8

const (
	headerButtonDependency headerButtonReadStage = iota
	headerButtonQuery
	headerButtonScan
	headerButtonRowsClose
	headerButtonClose
)

type headerButtonReadError struct {
	stage      headerButtonReadStage
	contextErr error // Only a standard context sentinel, never the backend cause.
}

func newHeaderButtonReadError(stage headerButtonReadStage, cause error) error {
	err := &headerButtonReadError{stage: stage}
	switch {
	case errors.Is(cause, context.Canceled):
		err.contextErr = context.Canceled
	case errors.Is(cause, context.DeadlineExceeded):
		err.contextErr = context.DeadlineExceeded
	}
	return err
}

func (e *headerButtonReadError) Error() string {
	switch e.stage {
	case headerButtonDependency:
		return "header buttons could not be read: dependency"
	case headerButtonQuery:
		return "header buttons could not be read: query"
	case headerButtonScan:
		return "header buttons could not be read: scan"
	case headerButtonRowsClose:
		return "header buttons could not be read: rows/close"
	case headerButtonClose:
		return "header buttons could not be read: close"
	default:
		return "header buttons could not be read"
	}
}

// Is preserves only context classification without exposing a backend chain.
func (e *headerButtonReadError) Is(target error) bool {
	return (target == context.Canceled || target == context.DeadlineExceeded) && e.contextErr == target
}
