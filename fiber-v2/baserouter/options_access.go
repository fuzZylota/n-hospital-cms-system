package baserouter

import (
	"errors"
	"lib"
	"models"

	"github.com/gofiber/fiber/v2"
)

// optionAdminOnly checks the current database role before any settings read or write.
func optionAdminOnly(utilities *models.Utilities) fiber.Handler {
	return optionAdminOnlyWith(
		func(c *fiber.Ctx) (string, error) {
			actor, err := lib.CheckAuth(c)
			return actor.Uid, err
		},
		func(uid string) (string, bool, bool, error) {
			query := utilities.Orm.Select([]string{"uid", "role", "is_active"})
			query.Table("users")
			query.Where("uid", "=", uid)
			query.Finish()
			if err := query.Execute(); err != nil {
				return "", false, false, err
			}
			rows, err := query.Rows()
			if err != nil {
				return "", false, false, err
			}
			if len(rows) == 0 {
				return "", false, false, nil
			}
			if len(rows) != 1 || lib.String(rows[0]["uid"]) != uid {
				return "", false, false, errors.New("invalid option actor lookup")
			}
			return lib.String(rows[0]["role"]), lib.Bool(rows[0]["is_active"]), true, nil
		},
	)
}

func optionAdminOnlyWith(auth func(*fiber.Ctx) (string, error), lookup func(string) (string, bool, bool, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		uid, err := auth(c)
		if err != nil || uid == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		role, active, found, err := lookup(uid)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		if !found || !active || role != "admin" {
			return c.SendStatus(fiber.StatusForbidden)
		}
		return c.Next()
	}
}
