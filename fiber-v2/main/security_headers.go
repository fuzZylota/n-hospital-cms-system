package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// cspReportOnly sitenin gerçekten yüklediği kaynaklara göre bilerek geniş
// tutulmuş bir politikadır; yalnızca raporlanır, ENGELLEMEZ. Tema inline
// script/style kullandığı için 'unsafe-inline' gereklidir.
const cspReportOnly = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com https://www.google.com https://www.gstatic.com https://translate.google.com https://translate.googleapis.com https://www.googletagmanager.com; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com https://www.gstatic.com; " +
	"font-src 'self' data: https://fonts.gstatic.com https://cdnjs.cloudflare.com; " +
	"img-src 'self' data: blob: https:; " +
	"connect-src 'self' ws: wss: https://www.google.com https://translate.googleapis.com https://www.google-analytics.com; " +
	"frame-src 'self' https://www.google.com https://maps.google.com https://www.youtube.com https://www.youtube-nocookie.com https://player.vimeo.com; " +
	"media-src 'self' https:; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'"

func isHTTPSRequest(c *fiber.Ctx) bool {
	if c.Protocol() == "https" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(c.Get("X-Forwarded-Proto"), ",")[0]), "https")
}

func setIfAbsent(c *fiber.Ctx, key, value string) {
	if len(c.Response().Header.Peek(key)) == 0 {
		c.Set(key, value)
	}
}

// securityHeaders yanıt başlıklarını ekler; üst katmanın koyduğu başlığı ezmez.
func securityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		setIfAbsent(c, "X-Content-Type-Options", "nosniff")
		setIfAbsent(c, "Referrer-Policy", "strict-origin-when-cross-origin")
		setIfAbsent(c, "Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=(), usb=()")
		setIfAbsent(c, "X-Frame-Options", "SAMEORIGIN")
		setIfAbsent(c, "Content-Security-Policy-Report-Only", cspReportOnly)
		if isHTTPSRequest(c) {
			setIfAbsent(c, "Strict-Transport-Security", "max-age=15552000")
		}
		return err
	}
}
