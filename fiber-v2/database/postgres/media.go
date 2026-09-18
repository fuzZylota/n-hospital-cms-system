package postgres

import (
	"context"
	"database/sql"
	"math"
	"strconv"
	"strings"

	"models/data"
)

const mediaInsertPrefix = "INSERT INTO medias ("

// MediaRepository inserts media data through the supplied application-owned
// pool. It borrows the pool and never opens, registers, pings, or closes it.
type MediaRepository struct {
	db *sql.DB
}

// NewMediaRepository stores db without performing I/O or taking ownership.
func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

var _ data.MediaInserter = (*MediaRepository)(nil)

type preparedMediaInsert struct {
	query string
	args  []any
}

// InsertMedia owns the transaction for one standalone media insert. Package
// domain operations that already own a transaction use insertMediaTx instead.
func (r *MediaRepository) InsertMedia(ctx context.Context, input data.MediaInsertInput) (data.MediaInsertResult, error) {
	prepared, err := prepareMediaInsert(input)
	if err != nil {
		return data.MediaInsertResult{}, err
	}
	if ctx == nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteInvalidInput, nil, nil)
	}
	if r == nil || r.db == nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteDependency, nil, ctx)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteBegin, ctxErr, ctx)
	}

	tx, beginErr := r.db.BeginTx(ctx, nil)
	if beginErr != nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteBegin, beginErr, ctx)
	}

	result, insertErr := insertMediaTx(ctx, tx, prepared)
	if insertErr != nil {
		_ = tx.Rollback()
		return data.MediaInsertResult{}, insertErr
	}
	return commitMediaInsert(ctx, tx, result)
}

func commitMediaInsert(ctx context.Context, tx *sql.Tx, result data.MediaInsertResult) (data.MediaInsertResult, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		_ = tx.Rollback()
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteCommit, ctxErr, ctx)
	}
	if commitErr := tx.Commit(); commitErr != nil {
		// database/sql considers a transaction done after Commit returns, even on
		// error. A speculative second Rollback would not provide a cleanup
		// guarantee and could obscure the actual commit outcome.
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteCommit, commitErr, ctx)
	}
	return result, nil
}

// insertMediaTx is the tx-bound insert core for package-owned domain
// operations. It never begins, commits, or rolls back a transaction.
func insertMediaTx(ctx context.Context, tx *sql.Tx, prepared preparedMediaInsert) (data.MediaInsertResult, error) {
	if ctx == nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteInvalidInput, nil, nil)
	}
	if tx == nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteDependency, nil, ctx)
	}

	row := tx.QueryRowContext(ctx, prepared.query, prepared.args...)
	if queryErr := row.Err(); queryErr != nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteInsert, queryErr, ctx)
	}
	var id int64
	if scanErr := row.Scan(&id); scanErr != nil {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteScan, scanErr, ctx)
	}
	if id <= 0 || id > math.MaxInt32 {
		return data.MediaInsertResult{}, newMediaWriteError(mediaWriteSelectedRow, nil, ctx)
	}
	return data.MediaInsertResult{ID: strconv.FormatInt(id, 10)}, nil
}

func prepareMediaInsert(input data.MediaInsertInput) (preparedMediaInsert, error) {
	if input.FileName == "" || input.FilePath == "" || input.MIMEType == "" || input.FileType == "" || input.TargetID == "" {
		return preparedMediaInsert{}, newMediaWriteError(mediaWriteInvalidInput, nil, nil)
	}
	if !validOptionalState(input.UserID.State) ||
		!validOptionalState(input.Data.State) ||
		!validOptionalState(input.AltText.State) ||
		!validOptionalState(input.Title.State) ||
		!validOptionalState(input.Width.State) ||
		!validOptionalState(input.Height.State) {
		return preparedMediaInsert{}, newMediaWriteError(mediaWriteInvalidInput, nil, nil)
	}
	if input.Width.State == data.FieldValue && (input.Width.Value < math.MinInt32 || input.Width.Value > math.MaxInt32) {
		return preparedMediaInsert{}, newMediaWriteError(mediaWriteInvalidInput, nil, nil)
	}
	if input.Height.State == data.FieldValue && (input.Height.Value < math.MinInt32 || input.Height.Value > math.MaxInt32) {
		return preparedMediaInsert{}, newMediaWriteError(mediaWriteInvalidInput, nil, nil)
	}

	columns := []string{"file_name", "file_path", "file_size", "mime_type", "file_type", "target_id"}
	args := []any{input.FileName, input.FilePath, input.FileSize, input.MIMEType, input.FileType, input.TargetID}

	switch input.UserID.State {
	case data.FieldNull:
		columns = append(columns, "uid")
		args = append(args, nil)
	case data.FieldValue:
		userID, err := parseCanonicalPositiveInt32(input.UserID.Value)
		if err != nil {
			return preparedMediaInsert{}, newMediaWriteError(mediaWriteInvalidInput, nil, nil)
		}
		columns = append(columns, "uid")
		args = append(args, userID)
	}
	appendOptionalString := func(column string, field data.OptionalString) {
		switch field.State {
		case data.FieldNull:
			columns = append(columns, column)
			args = append(args, nil)
		case data.FieldValue:
			columns = append(columns, column)
			args = append(args, field.Value)
		}
	}
	appendOptionalInt64 := func(column string, field data.OptionalInt64) {
		switch field.State {
		case data.FieldNull:
			columns = append(columns, column)
			args = append(args, nil)
		case data.FieldValue:
			columns = append(columns, column)
			args = append(args, field.Value)
		}
	}
	appendOptionalString("data", input.Data)
	appendOptionalString("alt_text", input.AltText)
	appendOptionalString("title", input.Title)
	appendOptionalInt64("width", input.Width)
	appendOptionalInt64("height", input.Height)

	placeholders := make([]string, len(args))
	for index := range placeholders {
		placeholders[index] = "$" + strconv.Itoa(index+1)
	}
	query := mediaInsertPrefix + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ") RETURNING mid"
	return preparedMediaInsert{query: query, args: args}, nil
}

func validOptionalState(state data.OptionalFieldState) bool {
	return state == data.FieldOmitted || state == data.FieldNull || state == data.FieldValue
}

func parseCanonicalPositiveInt32(value string) (int64, error) {
	if len(value) == 0 || value[0] < '1' || value[0] > '9' {
		return 0, strconv.ErrSyntax
	}
	for index := 1; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return 0, strconv.ErrSyntax
		}
	}
	id, err := strconv.ParseInt(value, 10, 32)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
