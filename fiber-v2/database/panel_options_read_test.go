package database

import (
	"models"
	"testing"
)

func TestRedactPanelOptionSecrets(t *testing.T) {
	option := models.Options{
		SiteName:           "synthetic-site",
		SMTPPassword:       "synthetic-mail-value",
		RecaptchaSecretKey: "synthetic-captcha-value",
	}
	redactPanelOptionSecrets(&option)
	if option.SMTPPassword != "" || option.RecaptchaSecretKey != "" {
		t.Fatal("panel settings still contain secret fields")
	}
	if option.SiteName != "synthetic-site" {
		t.Fatal("redaction altered a public setting")
	}
}
