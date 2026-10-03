package lib

import (
	"html"
	"strings"
)

// EscapeEmailText HTML-escapes a user-controlled value before it is placed
// into an HTML e-mail body (<, >, &, ", ' are escaped).
func EscapeEmailText(value string) string {
	return html.EscapeString(value)
}

// SanitizeHeaderValue removes CR, LF and NUL so a value used in an e-mail
// header (Subject, To, From, Reply-To) cannot inject extra headers.
func SanitizeHeaderValue(value string) string {
	return strings.NewReplacer("\r", "", "\n", "", "\x00", "").Replace(value)
}
