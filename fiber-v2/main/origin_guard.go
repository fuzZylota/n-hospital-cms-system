package main

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const originGuardMessage = "Güvenlik doğrulaması başarısız. Sayfayı yenileyip tekrar deneyin."

// requestHost nginx'in ilettiği X-Forwarded-Host'u (varsa) önceler.
func requestHost(c *fiber.Ctx) string {
	if h := strings.TrimSpace(strings.Split(c.Get("X-Forwarded-Host"), ",")[0]); h != "" {
		return strings.ToLower(h)
	}
	if h := c.Get("Host"); h != "" {
		return strings.ToLower(h)
	}
	return strings.ToLower(c.Hostname())
}

// originGuard durum değiştiren isteklerde çapraz site isteklerini reddeder
// (CSRF savunması). Tarayıcı olmayan istemciler (başlık yok) geçer.
func originGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		default:
			return c.Next()
		}
		if site := strings.ToLower(c.Get("Sec-Fetch-Site")); site != "" {
			if site == "same-origin" || site == "none" {
				return c.Next()
			}
			return rejectCrossSite(c)
		}
		origin := c.Get("Origin")
		if origin == "" {
			return c.Next()
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || !strings.EqualFold(u.Host, requestHost(c)) {
			return rejectCrossSite(c)
		}
		return c.Next()
	}
}

func rejectCrossSite(c *fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"status": 403, "message": originGuardMessage})
}
