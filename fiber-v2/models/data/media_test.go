package data_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"models/data"
)

type mediaInserterFunc func(context.Context, data.MediaInsertInput) (data.MediaInsertResult, error)

func (f mediaInserterFunc) InsertMedia(ctx context.Context, input data.MediaInsertInput) (data.MediaInsertResult, error) {
	return f(ctx, input)
}

var _ data.MediaInserter = mediaInserterFunc(nil)

func TestMediaInsertDTOBoundaries(t *testing.T) {
	stringType := reflect.TypeOf("")
	int64Type := reflect.TypeOf(int64(0))
	optionalStringType := reflect.TypeOf(data.OptionalString{})
	optionalInt64Type := reflect.TypeOf(data.OptionalInt64{})

	assertStructFields(t, data.OptionalString{}, []fieldContract{
		{name: "State", typeOf: reflect.TypeOf(data.FieldOmitted)},
		{name: "Value", typeOf: stringType},
	})
	assertStructFields(t, data.OptionalInt64{}, []fieldContract{
		{name: "State", typeOf: reflect.TypeOf(data.FieldOmitted)},
		{name: "Value", typeOf: int64Type},
	})
	assertStructFields(t, data.MediaInsertInput{}, []fieldContract{
		{name: "FileName", typeOf: stringType},
		{name: "FilePath", typeOf: stringType},
		{name: "FileSize", typeOf: int64Type},
		{name: "MIMEType", typeOf: stringType},
		{name: "FileType", typeOf: stringType},
		{name: "TargetID", typeOf: stringType},
		{name: "UserID", typeOf: optionalStringType},
		{name: "Data", typeOf: optionalStringType},
		{name: "AltText", typeOf: optionalStringType},
		{name: "Title", typeOf: optionalStringType},
		{name: "Width", typeOf: optionalInt64Type},
		{name: "Height", typeOf: optionalInt64Type},
	})
	assertStructFields(t, data.MediaInsertResult{}, []fieldContract{
		{name: "ID", typeOf: stringType},
	})

	if _, exists := reflect.TypeOf(data.MediaInsertInput{}).FieldByName("OldData"); exists {
		t.Fatal("MediaInsertInput must not carry client-supplied old data")
	}
}

func TestOptionalStringOmittedNullAndValueMatrix(t *testing.T) {
	tests := []struct {
		name  string
		field data.OptionalString
		state data.OptionalFieldState
		value string
	}{
		{name: "zero value is omitted", field: data.OptionalString{}, state: data.FieldOmitted},
		{name: "explicit omitted", field: data.OptionalString{State: data.FieldOmitted, Value: "ignored"}, state: data.FieldOmitted, value: "ignored"},
		{name: "explicit null", field: data.OptionalString{State: data.FieldNull, Value: "ignored"}, state: data.FieldNull, value: "ignored"},
		{name: "real empty string", field: data.OptionalString{State: data.FieldValue, Value: ""}, state: data.FieldValue, value: ""},
		{name: "real non-empty string", field: data.OptionalString{State: data.FieldValue, Value: "caption"}, state: data.FieldValue, value: "caption"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.field.State != test.state || test.field.Value != test.value {
				t.Fatalf("field = %#v, want state %v and value %q", test.field, test.state, test.value)
			}
		})
	}
}

func TestOptionalInt64OmittedNullAndValueMatrix(t *testing.T) {
	tests := []struct {
		name  string
		field data.OptionalInt64
		state data.OptionalFieldState
		value int64
	}{
		{name: "zero value is omitted", field: data.OptionalInt64{}, state: data.FieldOmitted},
		{name: "explicit omitted", field: data.OptionalInt64{State: data.FieldOmitted, Value: 99}, state: data.FieldOmitted, value: 99},
		{name: "explicit null", field: data.OptionalInt64{State: data.FieldNull, Value: 99}, state: data.FieldNull, value: 99},
		{name: "real zero", field: data.OptionalInt64{State: data.FieldValue, Value: 0}, state: data.FieldValue, value: 0},
		{name: "real positive value", field: data.OptionalInt64{State: data.FieldValue, Value: 640}, state: data.FieldValue, value: 640},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.field.State != test.state || test.field.Value != test.value {
				t.Fatalf("field = %#v, want state %v and value %d", test.field, test.state, test.value)
			}
		})
	}
}

func TestMediaInserterOutcomesAndContextIdentity(t *testing.T) {
	repositoryFailure := errors.New("safe media insert failure")
	input := data.MediaInsertInput{FileName: "logo.webp", TargetID: "17"}

	tests := []struct {
		name      string
		context   func() context.Context
		inserter  data.MediaInserter
		want      data.MediaInsertResult
		wantError error
	}{
		{
			name:    "success returns string identifier",
			context: context.Background,
			inserter: mediaInserterFunc(func(_ context.Context, got data.MediaInsertInput) (data.MediaInsertResult, error) {
				if !reflect.DeepEqual(got, input) {
					t.Fatalf("input = %#v, want %#v", got, input)
				}
				return data.MediaInsertResult{ID: "123"}, nil
			}),
			want: data.MediaInsertResult{ID: "123"},
		},
		{
			name:    "writer error has zero result",
			context: context.Background,
			inserter: mediaInserterFunc(func(context.Context, data.MediaInsertInput) (data.MediaInsertResult, error) {
				return data.MediaInsertResult{}, repositoryFailure
			}),
			wantError: repositoryFailure,
		},
		{
			name: "context canceled",
			context: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			inserter:  mediaInserterFunc(contextAwareMediaInsert),
			wantError: context.Canceled,
		},
		{
			name: "context deadline exceeded",
			context: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			inserter:  mediaInserterFunc(contextAwareMediaInsert),
			wantError: context.DeadlineExceeded,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.inserter.InsertMedia(test.context(), input)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("result = %#v, want %#v", got, test.want)
			}
		})
	}
}

func contextAwareMediaInsert(ctx context.Context, _ data.MediaInsertInput) (data.MediaInsertResult, error) {
	if err := ctx.Err(); err != nil {
		return data.MediaInsertResult{}, err
	}
	return data.MediaInsertResult{ID: "1"}, nil
}

func TestIdentifierBoundaryAndCanonicalNumericExamples(t *testing.T) {
	var optionID string = data.OptionSetIdentity{ID: "42"}.ID
	var mediaID string = data.MediaInsertResult{ID: "42"}.ID
	var targetID string = data.MediaInsertInput{TargetID: "42"}.TargetID
	if optionID != "42" || mediaID != "42" || targetID != "42" {
		t.Fatal("identifiers did not remain strings at the application boundary")
	}

	for _, id := range []string{"1", "42", "9223372036854775807"} {
		if !isCanonicalPositiveDecimal(id) {
			t.Errorf("expected canonical identifier %q", id)
		}
	}
	for _, id := range []string{"", "0", "01", "+1", "-1", " 1", "1 ", "1.0", "abc"} {
		if isCanonicalPositiveDecimal(id) {
			t.Errorf("expected non-canonical identifier %q", id)
		}
	}
}

func isCanonicalPositiveDecimal(id string) bool {
	if id == "" || id[0] < '1' || id[0] > '9' {
		return false
	}
	for index := 1; index < len(id); index++ {
		if id[index] < '0' || id[index] > '9' {
			return false
		}
	}
	return true
}
