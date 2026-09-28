package options

import "strings"

// An empty or whitespace-only edit keeps the server's existing secret value.
func shouldRotateOptionSecret(value string) bool {
	return strings.TrimSpace(value) != ""
}
