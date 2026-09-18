package notificationws

import (
	"log"
	"models/notify"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

var _ Socket = (*websocket.Conn)(nil)

// Handler authenticates while the HTTP context is valid. The upgrader copies
// the immutable typed UID into its own locals map; no Fiber context is retained.
func Handler(hub notify.Hub, authenticate func(*fiber.Ctx) (notify.UserID, error), onText func(Event, []byte), config websocket.Config) fiber.Handler {
	// The library's default recovery logs the panic and calls WriteJSON. Neither
	// is safe for this handler. serve owns cleanup; this is a final safe boundary.
	config.RecoverHandler = func(c *websocket.Conn) {
		if recover() != nil {
			_ = c.Close()
			log.Print("notification websocket: handler failed")
		}
	}
	upgrade := websocket.New(func(c *websocket.Conn) {
		event, ok := c.Locals("notification.event").(Event)
		if !ok {
			event = Unknown
		}
		if err := serveAuthenticated(hub, c, c.Locals(identityKey), event, onText); err != nil {
			log.Print("notification websocket: session failed")
		}
	}, config)
	return func(c *fiber.Ctx) error {
		err := authenticatedUpgrade(func() (any, error) {
			return authenticate(c)
		}, func(uid notify.UserID) {
			c.Locals(identityKey, uid)
		}, func() error {
			c.Locals("notification.event", eventFor(c.Get("Sec-WebSocket-Protocol")))
			return upgrade(c)
		})
		if err == errIdentity {
			return fiber.ErrUnauthorized
		}
		return err
	}
}
