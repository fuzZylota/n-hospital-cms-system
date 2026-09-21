package data_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"models/data"
)

type contactRequestResponseWorkflowSnapshotReaderFunc func(context.Context) (data.ContactRequestResponseWorkflowSnapshot, bool, error)

func (f contactRequestResponseWorkflowSnapshotReaderFunc) ReadContactRequestResponseWorkflowSnapshot(ctx context.Context) (data.ContactRequestResponseWorkflowSnapshot, bool, error) {
	return f(ctx)
}

var _ data.ContactRequestResponseWorkflowSnapshotReader = contactRequestResponseWorkflowSnapshotReaderFunc(nil)

func TestContactRequestResponseWorkflowSnapshotHasExactInternalValueOnlyShape(t *testing.T) {
	typeOfSnapshot := reflect.TypeOf(data.ContactRequestResponseWorkflowSnapshot{})
	wantNames := []string{
		"Set", "SMTPHost", "SMTPPort", "SMTPUsername", "SMTPPassword", "SiteName", "SiteDescription",
		"ContactEmail", "ContactPhone", "FacebookURL", "TwitterURL", "InstagramURL", "LinkedInURL",
		"PrimaryColor", "SiteLogoPath",
	}
	wantTypes := []reflect.Type{
		reflect.TypeOf(data.OptionSetIdentity{}),
		reflect.TypeOf(""), reflect.TypeOf(int64(0)), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""),
	}
	if typeOfSnapshot.NumField() != len(wantNames) {
		t.Fatal("snapshot field count mismatch")
	}
	for index := 0; index < typeOfSnapshot.NumField(); index++ {
		field := typeOfSnapshot.Field(index)
		if field.Name != wantNames[index] || field.Type != wantTypes[index] || field.Tag != "" {
			t.Fatal("snapshot field contract mismatch")
		}
		switch field.Type.Kind() {
		case reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Chan, reflect.UnsafePointer:
			t.Fatal("snapshot contains a non-value capability")
		}
	}
	if typeOfSnapshot.ConvertibleTo(reflect.TypeOf(data.SiteOptions{})) {
		t.Fatal("internal snapshot is convertible to public site options")
	}
	publicType := reflect.TypeOf(data.SiteOptions{})
	for _, secret := range []string{"SMTPHost", "SMTPPort", "SMTPUsername", "SMTPPassword", "RecaptchaSecretKey"} {
		if _, found := publicType.FieldByName(secret); found {
			t.Fatal("secret-bearing field reached public site options")
		}
	}
}

func TestContactRequestResponseWorkflowSnapshotReaderContract(t *testing.T) {
	readerType := reflect.TypeOf((*data.ContactRequestResponseWorkflowSnapshotReader)(nil)).Elem()
	if readerType.NumMethod() != 1 {
		t.Fatal("snapshot reader method count mismatch")
	}
	method := readerType.Method(0)
	if method.Name != "ReadContactRequestResponseWorkflowSnapshot" {
		t.Fatal("snapshot reader method name mismatch")
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if method.Type.NumIn() != 1 || method.Type.In(0) != contextType || method.Type.NumOut() != 3 {
		t.Fatal("snapshot reader signature mismatch")
	}
	if method.Type.Out(0) != reflect.TypeOf(data.ContactRequestResponseWorkflowSnapshot{}) || method.Type.Out(1).Kind() != reflect.Bool || method.Type.Out(2) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatal("snapshot reader result signature mismatch")
	}

	repositoryFailure := errors.New("safe repository failure")
	for _, outcome := range []string{"found", "missing", "failure"} {
		reader := contactRequestResponseWorkflowSnapshotReaderFunc(func(context.Context) (data.ContactRequestResponseWorkflowSnapshot, bool, error) {
			switch outcome {
			case "found":
				return data.ContactRequestResponseWorkflowSnapshot{Set: data.OptionSetIdentity{ID: "1", IsActive: true}}, true, nil
			case "missing":
				return data.ContactRequestResponseWorkflowSnapshot{}, false, nil
			default:
				return data.ContactRequestResponseWorkflowSnapshot{}, false, repositoryFailure
			}
		})
		got, found, err := reader.ReadContactRequestResponseWorkflowSnapshot(context.Background())
		switch outcome {
		case "found":
			if err != nil || !found || got.Set.ID == "" {
				t.Fatal("found outcome mismatch")
			}
		case "missing":
			if err != nil || found || got != (data.ContactRequestResponseWorkflowSnapshot{}) {
				t.Fatal("missing outcome mismatch")
			}
		default:
			if !errors.Is(err, repositoryFailure) || found || got != (data.ContactRequestResponseWorkflowSnapshot{}) {
				t.Fatal("failure outcome mismatch")
			}
		}
	}
}
