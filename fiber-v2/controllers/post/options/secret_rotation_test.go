package options

import (
	"os"
	"strings"
	"testing"
)

func TestShouldRotateOptionSecret(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
		want  bool
	}{
		{"empty keeps current", "", false},
		{"blank keeps current", "  ", false},
		{"new value rotates", "synthetic-new-value", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRotateOptionSecret(tt.value); got != tt.want {
				t.Fatalf("rotation = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEditOptionUsesServerSideSecretRotation(t *testing.T) {
	source, err := os.ReadFile("options.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"SMTPPassword", "RecaptchaSecretKey"} {
		if !strings.Contains(string(source), "if shouldRotateOptionSecret(inputs."+input+") {") {
			t.Fatalf("%s does not use empty-preserving rotation", input)
		}
	}
	for _, old := range []string{"OldSMTPPassword", "OldRecaptchaSecretKey"} {
		if strings.Contains(string(source), old) {
			t.Fatalf("edit handler still trusts client-provided %s", old)
		}
	}
}

func TestEditOptionRejectsMalformedInputAndTransactionFailures(t *testing.T) {
	source, err := os.ReadFile("options.go")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "func EditOption(")
	end := strings.Index(string(source), "func DeleteOption(")
	if start < 0 || end <= start {
		t.Fatal("EditOption handler not found")
	}
	edit := string(source[start:end])
	parse := strings.Index(edit, "if err := c.BodyParser(&inputs); err != nil {")
	begin := strings.Index(edit, "if err := Orm.Begin(); err != nil {")
	commit := strings.Index(edit, "if err := Orm.Commit(); err != nil {")
	if parse < 0 || begin <= parse || commit <= begin {
		t.Fatal("edit handler does not stop on parse, begin, and commit errors")
	}
	if strings.Contains(edit, "updateOption.Query") || strings.Contains(edit, `log.Printf("Cannot update option: %v`) {
		t.Fatal("edit handler may log secret-bearing update details")
	}
}
