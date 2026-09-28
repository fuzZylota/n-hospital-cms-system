package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestOptionsReadOmitsSecrets(t *testing.T) {
	typeOf := reflect.TypeOf(OptionsRead{})
	for _, name := range []string{"SMTPPassword", "RecaptchaSecretKey"} {
		if _, found := typeOf.FieldByName(name); found {
			t.Fatalf("read DTO contains %s", name)
		}
	}
	payload, err := json.Marshal(OptionsRead{SiteName: "synthetic-site"})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"smtp_password", "recaptcha_secret_key"} {
		if strings.Contains(string(payload), field) {
			t.Fatalf("read JSON contains %s", field)
		}
	}
}

func TestSettingsTemplatesDoNotRenderSecrets(t *testing.T) {
	dir := filepath.Join("..", "static", "html", "views", "panel", "secenek-sayfalari")
	for _, name := range []string{"secenek.jet", "secenek-duzenle.jet"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, exposure := range []string{"Option.SMTPPassword", "Option.RecaptchaSecretKey", "old_smtp_password", "old_recaptcha_secret_key"} {
			if strings.Contains(string(body), exposure) {
				t.Fatalf("%s contains %s", name, exposure)
			}
		}
		if name == "secenek-duzenle.jet" {
			for _, inputName := range []string{"smtp_password", "recaptcha_secret_key"} {
				input := regexp.MustCompile(`(?s)<input\b[^>]*name="` + inputName + `"[^>]*>`).Find(body)
				if len(input) == 0 || regexp.MustCompile(`\bvalue\s*=`).Match(input) {
					t.Fatalf("%s edit input missing or populated", inputName)
				}
			}
		}
	}
}
