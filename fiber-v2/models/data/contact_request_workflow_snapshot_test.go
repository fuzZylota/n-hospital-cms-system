package data_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"models/data"
)

type contactRequestWorkflowSnapshotReaderFunc func(context.Context) (data.ContactRequestWorkflowSnapshot, bool, error)

func (f contactRequestWorkflowSnapshotReaderFunc) ReadContactRequestWorkflowSnapshot(ctx context.Context) (data.ContactRequestWorkflowSnapshot, bool, error) {
	return f(ctx)
}

var _ data.ContactRequestWorkflowSnapshotReader = contactRequestWorkflowSnapshotReaderFunc(nil)

func TestContactRequestWorkflowSnapshotHasExactInternalValueOnlyShape(t *testing.T) {
	typeOfSnapshot := reflect.TypeOf(data.ContactRequestWorkflowSnapshot{})
	wantNames := []string{
		"Set", "SMTPHost", "SMTPPort", "SMTPUsername", "SMTPPassword", "SiteName", "SiteDescription",
		"ContactEmail", "ContactPhone", "FacebookURL", "TwitterURL", "InstagramURL", "LinkedInURL",
		"PrimaryColor", "RecaptchaSiteKey", "RecaptchaSecretKey", "SiteLogoPath", "AccentColor",
	}
	wantTypes := []reflect.Type{
		reflect.TypeOf(data.OptionSetIdentity{}),
		reflect.TypeOf(""), reflect.TypeOf(int64(0)), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
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
		case reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Chan, reflect.UnsafePointer:
			t.Fatal("snapshot contains a non-value capability")
		}
	}
	if typeOfSnapshot.ConvertibleTo(reflect.TypeOf(data.SiteOptions{})) {
		t.Fatal("internal snapshot is convertible to public site options")
	}
}

func TestContactRequestWorkflowSnapshotReaderContract(t *testing.T) {
	readerType := reflect.TypeOf((*data.ContactRequestWorkflowSnapshotReader)(nil)).Elem()
	if readerType.NumMethod() != 1 {
		t.Fatal("snapshot reader method count mismatch")
	}
	method := readerType.Method(0)
	if method.Name != "ReadContactRequestWorkflowSnapshot" {
		t.Fatal("snapshot reader method name mismatch")
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if method.Type.NumIn() != 1 || method.Type.In(0) != contextType || method.Type.NumOut() != 3 {
		t.Fatal("snapshot reader signature mismatch")
	}
	if method.Type.Out(0) != reflect.TypeOf(data.ContactRequestWorkflowSnapshot{}) || method.Type.Out(1).Kind() != reflect.Bool || method.Type.Out(2) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatal("snapshot reader result signature mismatch")
	}

	repositoryFailure := errors.New("safe repository failure")
	for _, outcome := range []string{"found", "missing", "failure"} {
		reader := contactRequestWorkflowSnapshotReaderFunc(func(context.Context) (data.ContactRequestWorkflowSnapshot, bool, error) {
			switch outcome {
			case "found":
				return data.ContactRequestWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "1", IsActive: true}}, true, nil
			case "missing":
				return data.ContactRequestWorkflowSnapshot{}, false, nil
			default:
				return data.ContactRequestWorkflowSnapshot{}, false, repositoryFailure
			}
		})
		got, found, err := reader.ReadContactRequestWorkflowSnapshot(context.Background())
		switch outcome {
		case "found":
			if err != nil || !found || got.Set.ID == "" {
				t.Fatal("found outcome mismatch")
			}
		case "missing":
			if err != nil || found || got != (data.ContactRequestWorkflowSnapshot{}) {
				t.Fatal("missing outcome mismatch")
			}
		default:
			if !errors.Is(err, repositoryFailure) || found || got != (data.ContactRequestWorkflowSnapshot{}) {
				t.Fatal("failure outcome mismatch")
			}
		}
	}
}
