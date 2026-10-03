package lib

import (
	"encoding/json"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// Herkese açık yazma uçlarının (randevu, iletişim, iş başvurusu, giriş) kötüye
// kullanım korumaları: IP başına hız sınırı, honeypot ve alan uzunluğu denetimi.

const (
	TooManyAttemptsMessage = "Çok fazla deneme yaptınız. Lütfen birkaç dakika sonra tekrar deneyin."
	HoneypotFieldName      = "website"
)

// NewPublicFormLimiter ajax uçları için IP başına max/expiration sınırı uygular.
// Aşımda ajax sözleşmesine uygun JSON döner: {"status":429,"message":...}.
func NewPublicFormLimiter(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  429,
				"message": TooManyAttemptsMessage,
			})
		},
	})
}

// NewLoginLimiter giriş formu (form post + yönlendirme) için IP başına sınır uygular.
func NewLoginLimiter(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Redirect("/giris?error=too_many_attempts")
		},
	})
}

// formValue JSON veya form gövdesinden (urlencoded/multipart) bir alanı okur.
// Gövdeyi tüketmez; sonraki BodyParser aynı gövdeyi yeniden okuyabilir.
func formValue(c *fiber.Ctx, jsonBody map[string]interface{}, isJSON bool, name string) string {
	if isJSON {
		value, _ := jsonBody[name].(string)
		return value
	}
	return c.FormValue(name)
}

func readBody(c *fiber.Ctx) (map[string]interface{}, bool) {
	if !strings.Contains(strings.ToLower(c.Get(fiber.HeaderContentType)), "json") {
		return nil, false
	}
	var body map[string]interface{}
	if json.Unmarshal(c.Body(), &body) != nil {
		return nil, true
	}
	return body, true
}

// HoneypotGuard gizli "website" alanı doluysa isteği handler'a hiç iletmeden
// başarı görünümlü yanıt döner: kayıt, e-posta ve bildirim yapılmaz. Yalnızca
// tek satır log yazar. Handler gövdelerine dokunulmaz (AST pin testleri korunur).
func HoneypotGuard(operation, successMessage string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body, isJSON := readBody(c)
		if strings.TrimSpace(formValue(c, body, isJSON, HoneypotFieldName)) != "" {
			log.Printf("operation=%s stage=honeypot", operation)
			return c.JSON(fiber.Map{
				"status":  201,
				"message": successMessage,
			})
		}
		return c.Next()
	}
}

// FieldLengthGuard alan başına azami karakter (rune) sayısını denetler; aşımda
// mevcut stile uygun 400 JSON döner.
func FieldLengthGuard(limits map[string]int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body, isJSON := readBody(c)
		for name, max := range limits {
			if utf8.RuneCountInString(formValue(c, body, isJSON, name)) > max {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Alanlardan biri izin verilen uzunluğu aşıyor. Lütfen kontrol edin.",
				})
			}
		}
		return c.Next()
	}
}
