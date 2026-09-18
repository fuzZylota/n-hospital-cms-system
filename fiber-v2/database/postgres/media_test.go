package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"database/postgres/internal/dbtest"
	"models/data"
)

const (
	requiredMediaInsertSQL = "INSERT INTO medias (file_name, file_path, file_size, mime_type, file_type, target_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING mid"
	allMediaInsertSQL      = "INSERT INTO medias (file_name, file_path, file_size, mime_type, file_type, target_id, uid, data, alt_text, title, width, height) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING mid"
)

var _ data.MediaInserter = (*MediaRepository)(nil)

func TestMediaRepositoryNilReceiverAndDependency(t *testing.T) {
	input := validMediaInput()
	for _, test := range []struct {
		name       string
		repository *MediaRepository
	}{
		{name: "nil receiver"},
		{name: "nil database", repository: NewMediaRepository(nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.repository.InsertMedia(context.Background(), input)
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteDependency)
		})
	}
}

func TestMediaRepositoryRequiredOnlyExactInsert(t *testing.T) {
	rows := dbtest.NewRows([]string{"mid"}, []any{int64(41)})
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(rows), dbtest.Commit(),
	)

	result, err := repository.InsertMedia(context.Background(), validMediaInput())
	if err != nil {
		t.Fatal("unexpected repository error")
	}
	if result.ID != "41" {
		t.Fatal("insert did not return the canonical identifier")
	}
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.CommitKind)
	event := connector.Events()[1]
	assertMediaQuery(t, event, requiredMediaInsertSQL, []any{
		"fixture.webp", "files/fixture.webp", int64(0), "image/webp", "image", "external-key",
	})
	if rows.CloseCount() != 1 {
		t.Fatal("unexpected returning rows close count")
	}
}

func TestMediaRepositoryOptionalStateMatrix(t *testing.T) {
	type optionalCase struct {
		name   string
		column string
		set    func(*data.MediaInsertInput, data.OptionalFieldState)
		value  any
	}
	cases := []optionalCase{
		{
			name: "user id", column: "uid", value: int64(23),
			set: func(input *data.MediaInsertInput, state data.OptionalFieldState) {
				input.UserID = data.OptionalString{State: state, Value: "23"}
				if state != data.FieldValue {
					input.UserID.Value = "2147483648"
				}
			},
		},
		{
			name: "data", column: "data", value: "",
			set: func(input *data.MediaInsertInput, state data.OptionalFieldState) {
				input.Data = data.OptionalString{State: state, Value: ""}
				if state != data.FieldValue {
					input.Data.Value = "ignored-data"
				}
			},
		},
		{
			name: "alt text", column: "alt_text", value: "",
			set: func(input *data.MediaInsertInput, state data.OptionalFieldState) {
				input.AltText = data.OptionalString{State: state, Value: ""}
				if state != data.FieldValue {
					input.AltText.Value = "ignored-alt"
				}
			},
		},
		{
			name: "title", column: "title", value: "fixture title",
			set: func(input *data.MediaInsertInput, state data.OptionalFieldState) {
				input.Title = data.OptionalString{State: state, Value: "fixture title"}
				if state != data.FieldValue {
					input.Title.Value = "ignored-title"
				}
			},
		},
		{
			name: "width", column: "width", value: int64(0),
			set: func(input *data.MediaInsertInput, state data.OptionalFieldState) {
				input.Width = data.OptionalInt64{State: state, Value: 0}
				if state != data.FieldValue {
					input.Width.Value = math.MaxInt64
				}
			},
		},
		{
			name: "height", column: "height", value: int64(0),
			set: func(input *data.MediaInsertInput, state data.OptionalFieldState) {
				input.Height = data.OptionalInt64{State: state, Value: 0}
				if state != data.FieldValue {
					input.Height.Value = math.MinInt64
				}
			},
		},
	}

	for _, field := range cases {
		field := field
		for _, state := range []data.OptionalFieldState{data.FieldOmitted, data.FieldNull, data.FieldValue} {
			state := state
			t.Run(field.name+"/"+optionalStateName(state), func(t *testing.T) {
				input := validMediaInput()
				field.set(&input, state)
				connector, repository := openMediaRepository(t,
					dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{7})), dbtest.Commit(),
				)
				result, err := repository.InsertMedia(context.Background(), input)
				if err != nil || result.ID != "7" {
					t.Fatal("unexpected optional insert outcome")
				}
				event := connector.Events()[1]
				if state == data.FieldOmitted {
					assertMediaQuery(t, event, requiredMediaInsertSQL, requiredMediaArgs())
					return
				}
				query := "INSERT INTO medias (file_name, file_path, file_size, mime_type, file_type, target_id, " + field.column + ") VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING mid"
				wantArgs := append(requiredMediaArgs(), field.value)
				if state == data.FieldNull {
					wantArgs[len(wantArgs)-1] = nil
				}
				assertMediaQuery(t, event, query, wantArgs)
			})
		}
	}
}

func TestMediaRepositoryOptionalOrderAndAllFields(t *testing.T) {
	input := validMediaInput()
	input.UserID = data.OptionalString{State: data.FieldValue, Value: "19"}
	input.Data = data.OptionalString{State: data.FieldNull, Value: "ignored-data"}
	input.AltText = data.OptionalString{State: data.FieldValue, Value: ""}
	input.Title = data.OptionalString{State: data.FieldValue, Value: "fixture title"}
	input.Width = data.OptionalInt64{State: data.FieldValue, Value: 0}
	input.Height = data.OptionalInt64{State: data.FieldValue, Value: 720}
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{91})), dbtest.Commit(),
	)

	result, err := repository.InsertMedia(context.Background(), input)
	if err != nil || result.ID != "91" {
		t.Fatal("unexpected all-fields insert outcome")
	}
	assertMediaQuery(t, connector.Events()[1], allMediaInsertSQL, append(requiredMediaArgs(),
		int64(19), nil, "", "fixture title", int64(0), int64(720),
	))
}

func TestMediaRepositoryMultipleOptionalFieldsKeepContractOrder(t *testing.T) {
	input := validMediaInput()
	input.Data = data.OptionalString{State: data.FieldValue, Value: "fixture data"}
	input.Title = data.OptionalString{State: data.FieldNull}
	input.Height = data.OptionalInt64{State: data.FieldValue, Value: 5}
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{3})), dbtest.Commit(),
	)

	_, err := repository.InsertMedia(context.Background(), input)
	if err != nil {
		t.Fatal("unexpected repository error")
	}
	const query = "INSERT INTO medias (file_name, file_path, file_size, mime_type, file_type, target_id, data, title, height) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING mid"
	assertMediaQuery(t, connector.Events()[1], query, append(requiredMediaArgs(), "fixture data", nil, int64(5)))
}

func TestMediaRepositoryRejectsUnknownOptionalStateBeforeBegin(t *testing.T) {
	setters := []struct {
		name string
		set  func(*data.MediaInsertInput)
	}{
		{name: "user id", set: func(input *data.MediaInsertInput) { input.UserID.State = 99 }},
		{name: "data", set: func(input *data.MediaInsertInput) { input.Data.State = 99 }},
		{name: "alt text", set: func(input *data.MediaInsertInput) { input.AltText.State = 99 }},
		{name: "title", set: func(input *data.MediaInsertInput) { input.Title.State = 99 }},
		{name: "width", set: func(input *data.MediaInsertInput) { input.Width.State = 99 }},
		{name: "height", set: func(input *data.MediaInsertInput) { input.Height.State = 99 }},
	}
	for _, test := range setters {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openMediaRepository(t, dbtest.Begin())
			input := validMediaInput()
			test.set(&input)
			result, err := repository.InsertMedia(context.Background(), input)
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteInvalidInput)
			assertNoMediaEvents(t, connector)
		})
	}
}

func TestMediaRepositoryRequiredInputValidationBeforeBegin(t *testing.T) {
	tests := []struct {
		name string
		set  func(*data.MediaInsertInput)
	}{
		{name: "file name", set: func(input *data.MediaInsertInput) { input.FileName = "" }},
		{name: "file path", set: func(input *data.MediaInsertInput) { input.FilePath = "" }},
		{name: "mime type", set: func(input *data.MediaInsertInput) { input.MIMEType = "" }},
		{name: "file type", set: func(input *data.MediaInsertInput) { input.FileType = "" }},
		{name: "target id", set: func(input *data.MediaInsertInput) { input.TargetID = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openMediaRepository(t, dbtest.Begin())
			input := validMediaInput()
			test.set(&input)
			result, err := repository.InsertMedia(context.Background(), input)
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteInvalidInput)
			assertNoMediaEvents(t, connector)
		})
	}
}

func TestMediaRepositoryTargetIDIsRequiredTextNotNumeric(t *testing.T) {
	input := validMediaInput()
	input.TargetID = "external-key-007"
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{6})), dbtest.Commit(),
	)
	_, err := repository.InsertMedia(context.Background(), input)
	if err != nil {
		t.Fatal("unexpected repository error")
	}
	event := connector.Events()[1]
	if len(event.Args) != 6 || event.Args[5].Value != "external-key-007" {
		t.Fatal("target identifier was not preserved as schema-backed text")
	}
}

func TestMediaRepositoryUserIDCanonicalPositiveIntegerValidation(t *testing.T) {
	invalid := []string{"", "0", "-1", "+1", "01", " 1", "1 ", "1.0", "abc", "2147483648", "9223372036854775808"}
	for index, value := range invalid {
		t.Run("case-"+safeTestIndex(index), func(t *testing.T) {
			connector, repository := openMediaRepository(t, dbtest.Begin())
			input := validMediaInput()
			input.UserID = data.OptionalString{State: data.FieldValue, Value: value}
			result, err := repository.InsertMedia(context.Background(), input)
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteInvalidInput)
			assertNoMediaEvents(t, connector)
		})
	}

	input := validMediaInput()
	input.UserID = data.OptionalString{State: data.FieldValue, Value: "2147483647"}
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{2})), dbtest.Commit(),
	)
	_, err := repository.InsertMedia(context.Background(), input)
	if err != nil {
		t.Fatal("unexpected repository error")
	}
	if connector.Events()[1].Args[6].Value != int64(2147483647) {
		t.Fatal("user identifier did not reach database/sql as a numeric value")
	}
}

func TestMediaRepositoryIntegerOptionalSchemaBounds(t *testing.T) {
	tests := []struct {
		name string
		set  func(*data.MediaInsertInput)
	}{
		{name: "width above", set: func(input *data.MediaInsertInput) {
			input.Width = data.OptionalInt64{State: data.FieldValue, Value: math.MaxInt32 + 1}
		}},
		{name: "width below", set: func(input *data.MediaInsertInput) {
			input.Width = data.OptionalInt64{State: data.FieldValue, Value: math.MinInt32 - 1}
		}},
		{name: "height above", set: func(input *data.MediaInsertInput) {
			input.Height = data.OptionalInt64{State: data.FieldValue, Value: math.MaxInt32 + 1}
		}},
		{name: "height below", set: func(input *data.MediaInsertInput) {
			input.Height = data.OptionalInt64{State: data.FieldValue, Value: math.MinInt32 - 1}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openMediaRepository(t, dbtest.Begin())
			input := validMediaInput()
			test.set(&input)
			result, err := repository.InsertMedia(context.Background(), input)
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteInvalidInput)
			assertNoMediaEvents(t, connector)
		})
	}
}

func TestMediaRepositoryBeginFailure(t *testing.T) {
	backendErr := backendMediaError{code: 11}
	connector, repository := openMediaRepository(t, dbtest.BeginError(backendErr))
	result, err := repository.InsertMedia(context.Background(), validMediaInput())
	assertZeroMediaResult(t, result)
	assertMediaStage(t, err, mediaWriteBegin)
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind)
	assertBackendMediaErrorHidden(t, err, backendErr)
}

func TestMediaRepositoryInsertFailureRollsBack(t *testing.T) {
	backendErr := backendMediaError{code: 12}
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.QueryError(backendErr), dbtest.Rollback(),
	)
	result, err := repository.InsertMedia(context.Background(), validMediaInput())
	assertZeroMediaResult(t, result)
	assertMediaStage(t, err, mediaWriteInsert)
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
	assertBackendMediaErrorHidden(t, err, backendErr)
}

func TestMediaRepositoryReturningScanFailureRollsBackAndClosesRows(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
	}{
		{name: "type mismatch", value: "not-an-integer"},
		{name: "null", value: nil},
		{name: "int64 overflow", value: "9223372036854775808"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := dbtest.NewRows([]string{"mid"}, []any{test.value})
			connector, repository := openMediaRepository(t,
				dbtest.Begin(), dbtest.Query(rows), dbtest.Rollback(),
			)
			result, err := repository.InsertMedia(context.Background(), validMediaInput())
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteScan)
			assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
			if rows.CloseCount() != 1 {
				t.Fatal("unexpected returning rows close count")
			}
		})
	}
}

func TestMediaRepositoryReturningBackendFailureIsHidden(t *testing.T) {
	backendErr := backendMediaError{code: 18}
	rows := dbtest.NewRows([]string{"mid"}, []any{1}).WithErrorAfter(0, backendErr)
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(rows), dbtest.Rollback(),
	)
	result, err := repository.InsertMedia(context.Background(), validMediaInput())
	assertZeroMediaResult(t, result)
	assertMediaStage(t, err, mediaWriteScan)
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
	assertBackendMediaErrorHidden(t, err, backendErr)
}

func TestMediaRepositoryInvalidReturnedIdentifierRollsBack(t *testing.T) {
	for index, returnedID := range []int64{0, -1, math.MaxInt32 + 1} {
		t.Run("case-"+safeTestIndex(index), func(t *testing.T) {
			connector, repository := openMediaRepository(t,
				dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{returnedID})), dbtest.Rollback(),
			)
			result, err := repository.InsertMedia(context.Background(), validMediaInput())
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteSelectedRow)
			assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
		})
	}
}

func TestMediaRepositoryCommitFailureReturnsZeroWithoutSecondRollbackClaim(t *testing.T) {
	backendErr := backendMediaError{code: 13}
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{15})), dbtest.CommitError(backendErr),
	)
	result, err := repository.InsertMedia(context.Background(), validMediaInput())
	assertZeroMediaResult(t, result)
	assertMediaStage(t, err, mediaWriteCommit)
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.CommitKind)
	assertBackendMediaErrorHidden(t, err, backendErr)
}

func TestMediaRepositoryRollbackFailureDoesNotReplacePrimaryError(t *testing.T) {
	primaryErr := backendMediaError{code: 14}
	rollbackErr := backendMediaError{code: 15}
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.QueryError(primaryErr), dbtest.RollbackError(rollbackErr),
	)
	result, err := repository.InsertMedia(context.Background(), validMediaInput())
	assertZeroMediaResult(t, result)
	assertMediaStage(t, err, mediaWriteInsert)
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
	assertBackendMediaErrorHidden(t, err, primaryErr)
	assertBackendMediaErrorHidden(t, err, rollbackErr)
}

func TestMediaRepositoryPreCanceledContextsDoNotBegin(t *testing.T) {
	tests := []struct {
		name string
		ctx  func(t *testing.T) context.Context
		want error
	}{
		{
			name: "canceled",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			want: context.Canceled,
		},
		{
			name: "deadline",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
				t.Cleanup(cancel)
				return ctx
			},
			want: context.DeadlineExceeded,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connector, repository := openMediaRepository(t, dbtest.Begin())
			result, err := repository.InsertMedia(test.ctx(t), validMediaInput())
			assertZeroMediaResult(t, result)
			if !errors.Is(err, test.want) {
				t.Fatal("unexpected context identity")
			}
			assertMediaStage(t, err, mediaWriteBegin)
			assertNoMediaEvents(t, connector)
		})
	}
}

func TestMediaRepositoryContextCancellationDuringInsertRollsBack(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				entered := make(chan struct{})
				connector, repository := openMediaRepository(t,
					dbtest.Begin(), dbtest.QueryUntilCanceled(entered), dbtest.Rollback(),
				)
				ctx, cancel := context.WithCancel(context.Background())
				if mode == "deadline" {
					ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				}
				t.Cleanup(cancel)

				type insertOutcome struct {
					result data.MediaInsertResult
					err    error
				}
				done := make(chan insertOutcome, 1)
				go func() {
					result, err := repository.InsertMedia(ctx, validMediaInput())
					done <- insertOutcome{result: result, err: err}
				}()
				select {
				case <-entered:
				case <-done:
					t.Fatal("repository returned before query entry")
				}
				events := connector.Events()
				assertMediaEvents(t, events, dbtest.BeginKind, dbtest.QueryKind)
				if events[0].ContextErr != nil || events[1].ContextErr != nil {
					t.Fatal("unexpected context state before cancellation")
				}
				if connector.Remaining() != 1 {
					t.Fatal("unexpected transaction script state before cancellation")
				}
				if mode == "cancel" {
					cancel()
				}

				outcome := <-done
				synctest.Wait()
				assertZeroMediaResult(t, outcome.result)
				want := context.Canceled
				if mode == "deadline" {
					want = context.DeadlineExceeded
				}
				if !errors.Is(outcome.err, want) {
					t.Fatal("unexpected context identity")
				}
				assertMediaStage(t, outcome.err, mediaWriteInsert)
				events = connector.Events()
				assertMediaEvents(t, events, dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
				if !errors.Is(events[1].ContextErr, want) {
					t.Fatal("unexpected query context event")
				}
				if connector.Remaining() != 0 {
					t.Fatal("unexpected transaction script state after rollback")
				}
			})
		})
	}
}

func TestCommitMediaInsertCanceledBeforeCommitRollsBack(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			prepared, err := prepareMediaInsert(validMediaInput())
			if err != nil {
				t.Fatal("unexpected preparation error")
			}
			connector, db := openMediaDB(t,
				dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{31})), dbtest.Rollback(),
			)
			txContext := context.Background()
			tx, err := db.BeginTx(txContext, nil)
			if err != nil {
				t.Fatal("unexpected begin error")
			}
			inserted, err := insertMediaTx(txContext, tx, prepared)
			if err != nil || inserted.ID != "31" {
				t.Fatal("unexpected tx-bound insert outcome")
			}
			ctx, cancel := context.WithCancel(txContext)
			t.Cleanup(cancel)
			if mode == "deadline" {
				deadlineCtx, deadlineCancel := context.WithDeadline(txContext, time.Unix(1, 0))
				t.Cleanup(deadlineCancel)
				ctx = deadlineCtx
			} else {
				cancel()
			}
			result, err := commitMediaInsert(ctx, tx, inserted)
			assertZeroMediaResult(t, result)
			assertMediaStage(t, err, mediaWriteCommit)
			want := context.Canceled
			if mode == "deadline" {
				want = context.DeadlineExceeded
			}
			if !errors.Is(err, want) {
				t.Fatal("unexpected pre-commit context identity")
			}
			assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
		})
	}
}

func TestMediaRepositoryBackendContextSentinelWithoutCanceledContextIsHidden(t *testing.T) {
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.QueryError(context.Canceled), dbtest.Rollback(),
	)
	_, err := repository.InsertMedia(context.Background(), validMediaInput())
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("backend sentinel was exposed without an actual canceled context")
	}
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
}

func TestMediaWriteErrorAllFormattersAreSafe(t *testing.T) {
	backendErr := backendMediaError{code: 16}
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.QueryError(backendErr), dbtest.Rollback(),
	)
	_, err := repository.InsertMedia(context.Background(), validMediaInput())
	want := "media could not be inserted: insert"
	for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q"} {
		formatted := fmt.Sprintf(format, err)
		if formatted != want {
			t.Fatal("unexpected safe error formatting")
		}
		if strings.Contains(formatted, backendErr.Error()) || strings.Contains(formatted, "fixture.webp") {
			t.Fatal("formatted error exposed protected details")
		}
	}
	if connector.Remaining() != 0 {
		t.Fatal("formatter test did not consume the transaction script")
	}
}

func TestMediaRepositoryPoolRemainsReusable(t *testing.T) {
	connector, repository := openMediaRepository(t,
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{1})), dbtest.Commit(),
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{2})), dbtest.Commit(),
	)
	first, firstErr := repository.InsertMedia(context.Background(), validMediaInput())
	second, secondErr := repository.InsertMedia(context.Background(), validMediaInput())
	if firstErr != nil || secondErr != nil || first.ID != "1" || second.ID != "2" {
		t.Fatal("unexpected borrowed pool reuse outcome")
	}
	assertMediaEvents(t, connector.Events(),
		dbtest.BeginKind, dbtest.QueryKind, dbtest.CommitKind,
		dbtest.BeginKind, dbtest.QueryKind, dbtest.CommitKind,
	)
}

func TestInsertMediaTxUsesOnlyCallerTransaction(t *testing.T) {
	prepared, err := prepareMediaInsert(validMediaInput())
	if err != nil {
		t.Fatal("unexpected preparation error")
	}
	connector, db := openMediaDB(t,
		dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"mid"}, []any{27})), dbtest.Commit(),
	)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("unexpected caller begin error")
	}
	result, err := insertMediaTx(context.Background(), tx, prepared)
	if err != nil || result.ID != "27" {
		t.Fatal("unexpected tx-bound insert outcome")
	}
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind)
	if err := tx.Commit(); err != nil {
		t.Fatal("unexpected caller commit error")
	}
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.CommitKind)
}

func TestInsertMediaTxLeavesFailureCleanupToCaller(t *testing.T) {
	prepared, err := prepareMediaInsert(validMediaInput())
	if err != nil {
		t.Fatal("unexpected preparation error")
	}
	connector, db := openMediaDB(t,
		dbtest.Begin(), dbtest.QueryError(backendMediaError{code: 17}), dbtest.Rollback(),
	)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("unexpected caller begin error")
	}
	result, err := insertMediaTx(context.Background(), tx, prepared)
	assertZeroMediaResult(t, result)
	assertMediaStage(t, err, mediaWriteInsert)
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind)
	if err := tx.Rollback(); err != nil {
		t.Fatal("unexpected caller rollback error")
	}
	assertMediaEvents(t, connector.Events(), dbtest.BeginKind, dbtest.QueryKind, dbtest.RollbackKind)
}

func TestInsertMediaTxNilTransactionIsSafe(t *testing.T) {
	prepared, err := prepareMediaInsert(validMediaInput())
	if err != nil {
		t.Fatal("unexpected preparation error")
	}
	result, err := insertMediaTx(context.Background(), nil, prepared)
	assertZeroMediaResult(t, result)
	assertMediaStage(t, err, mediaWriteDependency)
}

func validMediaInput() data.MediaInsertInput {
	return data.MediaInsertInput{
		FileName: "fixture.webp",
		FilePath: "files/fixture.webp",
		FileSize: 0,
		MIMEType: "image/webp",
		FileType: "image",
		TargetID: "external-key",
	}
}

func requiredMediaArgs() []any {
	return []any{"fixture.webp", "files/fixture.webp", int64(0), "image/webp", "image", "external-key"}
}

func optionalStateName(state data.OptionalFieldState) string {
	switch state {
	case data.FieldOmitted:
		return "omitted"
	case data.FieldNull:
		return "null"
	case data.FieldValue:
		return "value"
	default:
		return "unknown"
	}
}

func openMediaRepository(t *testing.T, steps ...dbtest.Step) (*dbtest.Connector, *MediaRepository) {
	t.Helper()
	connector, db := openMediaDB(t, steps...)
	return connector, NewMediaRepository(db)
}

func openMediaDB(t *testing.T, steps ...dbtest.Step) (*dbtest.Connector, *sql.DB) {
	t.Helper()
	connector := dbtest.NewConnector(steps...)
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error("unexpected test database close error")
		}
	})
	return connector, db
}

func assertMediaQuery(t *testing.T, event dbtest.Event, wantSQL string, wantArgs []any) {
	t.Helper()
	if event.Kind != dbtest.QueryKind {
		t.Fatal("media insert was not recorded as a query")
	}
	if event.SQL != wantSQL {
		t.Fatal("unexpected SQL")
	}
	if len(event.Args) != len(wantArgs) {
		t.Fatal("unexpected argument count")
	}
	for index := range wantArgs {
		if event.Args[index].Ordinal != index+1 || event.Args[index].Name != "" {
			t.Fatalf("unexpected argument metadata at index %d", index)
		}
		if !reflect.DeepEqual(event.Args[index].Value, wantArgs[index]) {
			t.Fatalf("unexpected argument at index %d", index)
		}
	}
}

func assertMediaEvents(t *testing.T, events []dbtest.Event, want ...dbtest.Kind) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatal("unexpected transaction events")
	}
	for index := range want {
		if events[index].Kind != want[index] {
			t.Fatalf("unexpected transaction event at index %d", index)
		}
	}
}

func assertNoMediaEvents(t *testing.T, connector *dbtest.Connector) {
	t.Helper()
	if len(connector.Events()) != 0 {
		t.Fatal("validation failure reached database/sql")
	}
	if connector.Remaining() != 1 {
		t.Fatal("validation failure consumed the transaction script")
	}
}

func assertZeroMediaResult(t *testing.T, result data.MediaInsertResult) {
	t.Helper()
	if result != (data.MediaInsertResult{}) {
		t.Fatal("failure returned a non-zero media result")
	}
}

func assertMediaStage(t *testing.T, err error, want mediaWriteStage) {
	t.Helper()
	var writeErr *mediaWriteError
	if !errors.As(err, &writeErr) {
		t.Fatal("unexpected repository error")
	}
	if writeErr.stage != want {
		t.Fatal("unexpected repository error stage")
	}
}

type backendMediaError struct {
	code int
}

func (backendMediaError) Error() string {
	return "synthetic backend failure"
}

func assertBackendMediaErrorHidden(t *testing.T, err error, backendErr backendMediaError) {
	t.Helper()
	if errors.Is(err, backendErr) {
		t.Fatal("media error exposed the backend cause through errors.Is")
	}
	var exposed backendMediaError
	if errors.As(err, &exposed) {
		t.Fatal("media error exposed the backend type through errors.As")
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("media error exposed a backend chain through errors.Unwrap")
	}
	if strings.Contains(err.Error(), backendErr.Error()) {
		t.Fatal("media error text exposed the backend cause")
	}
}

func safeTestIndex(index int) string {
	return strconv.Itoa(index)
}
