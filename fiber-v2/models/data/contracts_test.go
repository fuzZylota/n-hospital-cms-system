package data_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"models/data"
)

type headerButtonReaderFunc func(context.Context) ([]data.HeaderParent, error)

func (f headerButtonReaderFunc) ListHeaderParents(ctx context.Context) ([]data.HeaderParent, error) {
	return f(ctx)
}

var _ data.HeaderButtonReader = headerButtonReaderFunc(nil)

func TestHeaderParentContract(t *testing.T) {
	parentID := "7"
	parent := data.HeaderParent{
		ID:       "12",
		Title:    "Kurumsal",
		ParentID: &parentID,
	}

	var id string = parent.ID
	var title string = parent.Title
	var nullableParentID *string = parent.ParentID
	if id != "12" || title != "Kurumsal" || nullableParentID == nil || *nullableParentID != "7" {
		t.Fatalf("unexpected header parent: %#v", parent)
	}

	typeOfParent := reflect.TypeOf(parent)
	wantFields := []struct {
		name   string
		typeOf reflect.Type
	}{
		{name: "ID", typeOf: reflect.TypeOf("")},
		{name: "Title", typeOf: reflect.TypeOf("")},
		{name: "ParentID", typeOf: reflect.TypeOf((*string)(nil))},
	}
	if typeOfParent.NumField() != len(wantFields) {
		t.Fatalf("HeaderParent has %d fields, want %d", typeOfParent.NumField(), len(wantFields))
	}
	for index, want := range wantFields {
		field := typeOfParent.Field(index)
		if field.Name != want.name || field.Type != want.typeOf {
			t.Errorf("field %d = %s %s, want %s %s", index, field.Name, field.Type, want.name, want.typeOf)
		}
		if field.Tag != "" {
			t.Errorf("field %s has infrastructure tag %q", field.Name, field.Tag)
		}
	}
}

func TestHeaderButtonReaderOutcomes(t *testing.T) {
	repositoryFailure := errors.New("repository unavailable")
	normalResult := []data.HeaderParent{{ID: "1", Title: "Ana Sayfa"}}

	tests := []struct {
		name      string
		context   func() context.Context
		reader    data.HeaderButtonReader
		want      []data.HeaderParent
		wantError error
		wantEmpty bool
	}{
		{
			name:    "normal list",
			context: context.Background,
			reader: headerButtonReaderFunc(func(context.Context) ([]data.HeaderParent, error) {
				return normalResult, nil
			}),
			want: normalResult,
		},
		{
			name:    "empty list",
			context: context.Background,
			reader: headerButtonReaderFunc(func(context.Context) ([]data.HeaderParent, error) {
				return []data.HeaderParent{}, nil
			}),
			want:      []data.HeaderParent{},
			wantEmpty: true,
		},
		{
			name: "context canceled",
			context: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			reader:    headerButtonReaderFunc(contextAwareResult),
			wantError: context.Canceled,
		},
		{
			name: "context deadline exceeded",
			context: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			reader:    headerButtonReaderFunc(contextAwareResult),
			wantError: context.DeadlineExceeded,
		},
		{
			name:    "repository error",
			context: context.Background,
			reader: headerButtonReaderFunc(func(context.Context) ([]data.HeaderParent, error) {
				return nil, repositoryFailure
			}),
			wantError: repositoryFailure,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.reader.ListHeaderParents(test.context())
			if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("result = %#v, want %#v", got, test.want)
			}
			if test.wantEmpty && got == nil {
				t.Fatal("empty result is nil; contract requires a non-nil empty slice")
			}
		})
	}
}

func contextAwareResult(ctx context.Context) ([]data.HeaderParent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []data.HeaderParent{}, nil
}
