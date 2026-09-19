package data_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"models/data"
)

type optionMediaMutationSnapshotReaderFunc func(context.Context) (data.OptionMediaMutationSnapshot, bool, error)

func (f optionMediaMutationSnapshotReaderFunc) ReadOptionMediaMutationSnapshot(ctx context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	return f(ctx)
}

var _ data.OptionMediaMutationSnapshotReader = optionMediaMutationSnapshotReaderFunc(nil)

func TestOptionMediaMutationSnapshotHasExactValueOnlyShape(t *testing.T) {
	typeOfSnapshot := reflect.TypeOf(data.OptionMediaMutationSnapshot{})
	wantNames := []string{"Set", "MaxBytes", "SiteLogoID", "SiteLightLogoID", "FaviconID", "DefaultPageMediaID"}
	wantTypes := []reflect.Type{
		reflect.TypeOf(data.OptionSetIdentity{}),
		reflect.TypeOf(int64(0)),
		reflect.TypeOf((*string)(nil)),
		reflect.TypeOf((*string)(nil)),
		reflect.TypeOf((*string)(nil)),
		reflect.TypeOf((*string)(nil)),
	}
	if typeOfSnapshot.NumField() != len(wantNames) {
		t.Fatal("snapshot field count mismatch")
	}
	for index := 0; index < typeOfSnapshot.NumField(); index++ {
		field := typeOfSnapshot.Field(index)
		if field.Name != wantNames[index] {
			t.Fatal("snapshot field order or name mismatch")
		}
		if field.Type != wantTypes[index] {
			t.Fatal("snapshot field type mismatch")
		}
		if field.Tag != "" {
			t.Fatal("snapshot field tag mismatch")
		}
		switch field.Type.Kind() {
		case reflect.Func, reflect.Interface, reflect.Map, reflect.Chan, reflect.UnsafePointer:
			t.Fatal("snapshot contains a non-value capability")
		}
	}
}

func TestOptionMediaMutationSnapshotReaderContract(t *testing.T) {
	readerType := reflect.TypeOf((*data.OptionMediaMutationSnapshotReader)(nil)).Elem()
	if readerType.NumMethod() != 1 {
		t.Fatal("snapshot reader method count mismatch")
	}
	method := readerType.Method(0)
	if method.Name != "ReadOptionMediaMutationSnapshot" {
		t.Fatal("snapshot reader method name mismatch")
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if method.Type.NumIn() != 1 || method.Type.In(0) != contextType || method.Type.NumOut() != 3 {
		t.Fatal("snapshot reader signature mismatch")
	}
	if method.Type.Out(0) != reflect.TypeOf(data.OptionMediaMutationSnapshot{}) || method.Type.Out(1).Kind() != reflect.Bool || method.Type.Out(2) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatal("snapshot reader result signature mismatch")
	}

	repositoryFailure := errors.New("safe repository failure")
	for _, outcome := range []string{"found", "missing", "failure"} {
		reader := optionMediaMutationSnapshotReaderFunc(func(context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
			switch outcome {
			case "found":
				return data.OptionMediaMutationSnapshot{Set: data.OptionSetIdentity{ID: "1", IsActive: true}}, true, nil
			case "missing":
				return data.OptionMediaMutationSnapshot{}, false, nil
			default:
				return data.OptionMediaMutationSnapshot{}, false, repositoryFailure
			}
		})
		got, found, err := reader.ReadOptionMediaMutationSnapshot(context.Background())
		switch outcome {
		case "found":
			if err != nil || !found || got.Set.ID == "" {
				t.Fatal("found outcome mismatch")
			}
		case "missing":
			if err != nil || found || got != (data.OptionMediaMutationSnapshot{}) {
				t.Fatal("missing outcome mismatch")
			}
		default:
			if !errors.Is(err, repositoryFailure) || found || got != (data.OptionMediaMutationSnapshot{}) {
				t.Fatal("failure outcome mismatch")
			}
		}
	}
}
