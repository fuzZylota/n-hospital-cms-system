package data_test

import (
	"context"
	"reflect"
	"testing"

	"models/data"
)

func TestJobApplicationWorkflowSnapshotExactShape(t *testing.T) {
	snapshot := reflect.TypeOf(data.JobApplicationWorkflowSnapshot{})
	wantNames := []string{
		"Set", "SMTPHost", "SMTPPort", "SMTPUsername", "SMTPPassword", "SiteName", "SiteDescription",
		"ContactEmail", "ContactPhone", "FacebookURL", "TwitterURL", "InstagramURL", "LinkedInURL",
		"PrimaryColor", "RecaptchaSiteKey", "RecaptchaSecretKey", "SiteLogoPath", "AccentColor", "MaxBytes",
	}
	wantTypes := []reflect.Type{
		reflect.TypeOf(data.OptionSetIdentity{}),
		reflect.TypeOf(""), reflect.TypeOf(int64(0)), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""), reflect.TypeOf(""),
		reflect.TypeOf(int64(0)),
	}
	if snapshot.NumField() != 19 || snapshot.NumField() != len(wantNames) || len(wantNames) != len(wantTypes) {
		t.Fatal("job snapshot field count mismatch")
	}
	for index := range wantNames {
		field := snapshot.Field(index)
		if field.Name != wantNames[index] || field.Type != wantTypes[index] || field.Tag != "" || field.PkgPath != "" {
			t.Fatal("job snapshot field contract mismatch")
		}
		switch field.Type.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Array, reflect.Chan, reflect.Func, reflect.UnsafePointer:
			t.Fatal("job snapshot contains a reference or container")
		}
	}
	if snapshot.ConvertibleTo(reflect.TypeOf(data.SiteOptions{})) {
		t.Fatal("job snapshot crosses public options boundary")
	}
}

func TestJobApplicationWorkflowSnapshotReaderExactSignature(t *testing.T) {
	reader := reflect.TypeOf((*data.JobApplicationWorkflowSnapshotReader)(nil)).Elem()
	if reader.NumMethod() != 1 {
		t.Fatal("job snapshot reader method count mismatch")
	}
	method := reader.Method(0)
	if method.Name != "ReadJobApplicationWorkflowSnapshot" || method.Type.NumIn() != 1 || method.Type.NumOut() != 3 {
		t.Fatal("job snapshot reader signature mismatch")
	}
	if method.Type.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() ||
		method.Type.Out(0) != reflect.TypeOf(data.JobApplicationWorkflowSnapshot{}) ||
		method.Type.Out(1) != reflect.TypeOf(true) ||
		method.Type.Out(2) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatal("job snapshot reader types mismatch")
	}
}
