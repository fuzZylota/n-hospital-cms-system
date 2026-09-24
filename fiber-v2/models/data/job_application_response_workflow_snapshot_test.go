package data_test

import (
	"context"
	"reflect"
	"testing"

	"models/data"
)

func TestJobApplicationResponseWorkflowSnapshotExactShape(t *testing.T) {
	snapshot := reflect.TypeOf(data.JobApplicationResponseWorkflowSnapshot{})
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
	if snapshot.NumField() != 15 || len(wantNames) != 15 || len(wantTypes) != 15 {
		t.Fatal("response snapshot field count mismatch")
	}
	for index := range wantNames {
		field := snapshot.Field(index)
		if field.Name != wantNames[index] || field.Type != wantTypes[index] || field.Tag != "" || field.PkgPath != "" {
			t.Fatal("response snapshot field contract mismatch")
		}
		switch field.Type.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Array, reflect.Chan, reflect.Func, reflect.UnsafePointer:
			t.Fatal("response snapshot contains a reference or container")
		}
	}
	if snapshot.ConvertibleTo(reflect.TypeOf(data.SiteOptions{})) {
		t.Fatal("response snapshot crosses public options boundary")
	}
	public := reflect.TypeOf(data.SiteOptions{})
	for _, secret := range []string{"SMTPHost", "SMTPPort", "SMTPUsername", "SMTPPassword", "RecaptchaSecretKey"} {
		if _, found := public.FieldByName(secret); found {
			t.Fatal("secret-bearing field reached public site options")
		}
	}
}

func TestJobApplicationResponseWorkflowSnapshotReaderExactSignature(t *testing.T) {
	reader := reflect.TypeOf((*data.JobApplicationResponseWorkflowSnapshotReader)(nil)).Elem()
	if reader.NumMethod() != 1 {
		t.Fatal("response snapshot reader method count mismatch")
	}
	method := reader.Method(0)
	if method.Name != "ReadJobApplicationResponseWorkflowSnapshot" || method.Type.NumIn() != 1 || method.Type.NumOut() != 3 {
		t.Fatal("response snapshot reader signature mismatch")
	}
	if method.Type.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() ||
		method.Type.Out(0) != reflect.TypeOf(data.JobApplicationResponseWorkflowSnapshot{}) ||
		method.Type.Out(1) != reflect.TypeOf(true) ||
		method.Type.Out(2) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatal("response snapshot reader types mismatch")
	}
}
