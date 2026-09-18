// Package parentread owns the existing parent-selector JSON response decision.
package parentread

import (
	"context"
	"models/data"
	"time"
)

// Button preserves the complete legacy models.HeaderButton JSON shape,
// including zero-valued fields. The transport contract is checked statically.
type Button struct {
	Hbid       string    `json:"hbid"`
	Title      string    `json:"title"`
	Url        string    `json:"url"`
	Target     string    `json:"target"`
	Icon       string    `json:"icon"`
	SortOrder  int64     `json:"sort_order"`
	IsActive   bool      `json:"is_active"`
	ButtonType string    `json:"button_type"`
	ParentId   string    `json:"parent_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Response performs no read before authentication, never falls back, and
// returns only fixed error messages. The caller retains Fiber's HTTP behavior.
func Response(ctx context.Context, authenticated bool, reader data.HeaderButtonReader) map[string]any {
	if !authenticated {
		return map[string]any{"status": 403, "message": "Forbidden"}
	}
	if reader == nil {
		return failure()
	}
	parents, err := reader.ListHeaderParents(ctx)
	if err != nil {
		return failure()
	}
	buttons := make([]Button, 0, len(parents))
	for _, parent := range parents {
		button := Button{Hbid: parent.ID, Title: parent.Title}
		if parent.ParentID != nil {
			button.ParentId = *parent.ParentID
		}
		buttons = append(buttons, button)
	}
	return map[string]any{"status": 200, "message": "Header buttons fetched successfully", "header_buttons": buttons}
}

func failure() map[string]any {
	return map[string]any{"status": 500, "message": "Internal server error"}
}
