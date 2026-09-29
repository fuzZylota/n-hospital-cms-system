package users

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	lib "lib"
	"log"
	"models"
	"regexp"
	"users/passwordpolicy"

	"github.com/gofiber/fiber/v2"
	"strings"
)

func collectUserEditFields(c *fiber.Ctx) (map[string][]string, error) {
	fields := make(map[string][]string)
	contentType := c.Get(fiber.HeaderContentType)

	if strings.HasPrefix(contentType, fiber.MIMEApplicationJSON) {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return nil, err
		}

		for key, value := range body {
			switch typedValue := value.(type) {
			case []interface{}:
				for _, item := range typedValue {
					switch item.(type) {
					case string, bool, float64:
						fields[key] = append(fields[key], fmt.Sprint(item))
					default:
						return nil, errors.New("invalid user edit field value")
					}
				}
			case nil:
				fields[key] = []string{""}
			case string, bool, float64:
				fields[key] = []string{fmt.Sprint(typedValue)}
			default:
				return nil, errors.New("invalid user edit field value")
			}
		}

		return fields, nil
	}

	if strings.HasPrefix(contentType, fiber.MIMEMultipartForm) {
		form, err := c.MultipartForm()
		if err != nil {
			return nil, err
		}
		if len(form.File) != 0 {
			return nil, errors.New("file fields are not allowed in user edit requests")
		}
		for key, values := range form.Value {
			fields[key] = append([]string(nil), values...)
		}
		return fields, nil
	}

	c.Context().PostArgs().VisitAll(func(key []byte, value []byte) {
		field := string(key)
		fields[field] = append(fields[field], string(value))
	})

	return fields, nil
}

func AddUser(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can add users",
			})
		}

		inputs := models.Users{}
		c.BodyParser(&inputs)

		Orm := utilities.Orm

		if OurUser.Role != "admin" {
			checkIfUserIsAdmin := Orm.Count("users")
			checkIfUserIsAdmin.Where("uid", "=", OurUser.Uid)
			checkIfUserIsAdmin.And("role", "=", "admin")
			checkIfUserIsAdmin.Finish()

			err = checkIfUserIsAdmin.Execute()

			if err != nil {
				log.Printf("Cannot check if user is admin: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if checkIfUserIsAdmin.Length() == 0 {
				log.Printf("User is not admin")
				return c.JSON(fiber.Map{
					"status":  403,
					"message": "Only admins can add users",
				})
			}
		}

		if inputs.Password != inputs.PasswordConfirm {
			return c.Redirect("/panel/kullanici-ekle?error=password_and_password_confirm_do_not_match")
		}

		CheckIfEmailOrPhoneExists := Orm.Count("users")
		CheckIfEmailOrPhoneExists.Where("email", "=", inputs.Email)
		CheckIfEmailOrPhoneExists.Or("phone", "=", inputs.Phone)
		CheckIfEmailOrPhoneExists.Finish()

		err = CheckIfEmailOrPhoneExists.Execute()

		if err != nil {
			log.Printf("Cannot check if email or phone exists: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if CheckIfEmailOrPhoneExists.Length() > 0 {
			log.Printf("Email or phone already exists")
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Email or phone already exists",
			})
		}

		passwordPolicy, err := passwordpolicy.Read(c.UserContext(), utilities.PasswordPolicyReader)
		if err != nil {
			log.Print("Cannot get options")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if passwordPolicy.RequireStrong {
			if len(inputs.Password) < 8 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password is too short",
				})
			}

			if len(inputs.Password) > 32 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password is too long",
				})
			}

			matched, err := regexp.MatchString("[A-Z]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				log.Printf("Password must contain at least one uppercase letter")
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one uppercase letter",
				})
			}

			matched, err = regexp.MatchString("[a-z]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one lowercase letter",
				})
			}

			matched, err = regexp.MatchString("[0-9]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one number",
				})
			}

			matched, err = regexp.MatchString("[!@#$%^&*()]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one special character",
				})
			}
		} else {
			if len(inputs.Password) < 6 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password is too short",
				})
			}
		}

		hashedPassword, err := lib.HashPassword(inputs.Password)
		if err != nil {
			log.Printf("Cannot hash password: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		columns := []string{"name", "surname", "password", "email", "phone", "role", "is_active", "timezone"}
		values := []interface{}{inputs.Name, inputs.Surname, hashedPassword, inputs.Email, inputs.Phone, inputs.Role, inputs.IsActive, inputs.Timezone}

		if inputs.Sid != "" {
			columns = append(columns, "sid")
			values = append(values, inputs.Sid)
		}

		InsertUser := Orm.Insert(columns, values)
		InsertUser.Table("users")
		InsertUser.Finish()
		err = InsertUser.Execute()
		if err != nil {
			log.Printf("Cannot insert user: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "User added successfully",
		})
	}
}

func EditUser(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authenticatedUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{"status": 403, "message": "Forbidden"})
		}
		fields, err := collectUserEditFields(c)
		if err != nil {
			return c.JSON(fiber.Map{"status": 400, "message": "Bad request"})
		}
		inputs := models.UsersEdit{}
		if err = c.BodyParser(&inputs); err != nil {
			return c.JSON(fiber.Map{"status": 400, "message": "Bad request"})
		}
		if utilities == nil || utilities.Orm == nil || utilities.Orm.Pool == nil {
			log.Printf("operation=EditUser stage=transaction_dependency")
			return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
		}
		tx, err := utilities.Orm.Pool.BeginTx(c.UserContext(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			log.Printf("operation=EditUser stage=transaction_begin")
			return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
		}
		finished := false
		defer func() {
			if !finished {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Printf("operation=EditUser stage=transaction_rollback")
				}
			}
		}()
		result := runUserEdit(c.UserContext(), tx, authenticatedUser.Uid, c.Params("uid"), fields, inputs)
		if result.status != 201 {
			if result.status == 500 {
				log.Printf("operation=EditUser stage=%s", result.stage)
			}
			rollbackErr := tx.Rollback()
			finished = true
			if rollbackErr != nil {
				log.Printf("operation=EditUser stage=transaction_rollback")
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
			return c.JSON(fiber.Map{"status": result.status, "message": result.message})
		}
		if err = tx.Commit(); err != nil {
			finished = true
			// database/sql marks the transaction done on a failed Commit.
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				log.Printf("operation=EditUser stage=transaction_rollback")
			}
			log.Printf("operation=EditUser stage=transaction_commit")
			return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
		}
		finished = true
		return c.JSON(fiber.Map{"status": 201, "message": "User edited successfully"})
	}
}

func ChangeUserPassword(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		inputs := models.UsersEdit{}
		c.BodyParser(&inputs)

		UserUid := c.Params("uid")

		Orm := utilities.Orm

		if OurUser.Role != "admin" {
			if OurUser.Uid != UserUid {
				return c.JSON(fiber.Map{
					"status":  403,
					"message": "Only admins can change user password",
				})
			}
		}

		if inputs.Password != inputs.PasswordConfirm {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Password and password confirm do not match",
			})
		}

		passwordPolicy, err := passwordpolicy.Read(c.UserContext(), utilities.PasswordPolicyReader)
		if err != nil {
			log.Print("Cannot get options")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if passwordPolicy.RequireStrong {
			if len(inputs.Password) < 8 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password is too short",
				})
			}

			if len(inputs.Password) > 32 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password is too long",
				})
			}

			matched, err := regexp.MatchString("[A-Z]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one uppercase letter",
				})
			}

			matched, err = regexp.MatchString("[a-z]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one lowercase letter",
				})
			}

			matched, err = regexp.MatchString("[0-9]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one number",
				})
			}

			matched, err = regexp.MatchString("[!@#$%^&*()]", inputs.Password)

			if err != nil {
				log.Printf("Cannot match password: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if !matched {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password must contain at least one special character",
				})
			}
		} else {
			if len(inputs.Password) < 6 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Password is too short",
				})
			}
		}

		hashedPassword, err := lib.HashPassword(inputs.Password)
		if err != nil {
			log.Printf("Cannot hash password: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		UpdateUser := Orm.Update()
		UpdateUser.Table("users")
		UpdateUser.Set("password", hashedPassword)
		UpdateUser.Where("uid", "=", UserUid)
		UpdateUser.Finish()

		err = UpdateUser.Execute()

		if err != nil {
			log.Printf("Cannot update user password: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "User password changed successfully",
		})
	}
}

func DeleteUser(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		UserUid := c.Params("uid")

		Orm := utilities.Orm

		if OurUser.Role != "admin" || OurUser.Uid == UserUid {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can delete users",
			})
		}

		DeleteUser := Orm.Delete()
		DeleteUser.Table("users")
		DeleteUser.Where("uid", "=", UserUid)
		DeleteUser.Finish()

		err = DeleteUser.Execute()

		if err != nil {
			log.Printf("Cannot delete user: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "User deleted successfully",
		})
	}
}

func BanUnbanUser(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			log.Printf("err: %v", err)
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		UserUid := c.Params("uid")

		inputs := models.BanUserInputs{}
		err = c.BodyParser(&inputs)
		if err != nil {
			log.Printf("Cannot parse body: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Bad request",
			})
		}

		Orm := utilities.Orm

		if OurUser.Role != "admin" || OurUser.Uid == UserUid {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can ban/unban users",
			})
		}

		UpdateUser := Orm.Update()
		UpdateUser.Table("users")
		UpdateUser.Set("is_active", inputs.IsActive)
		UpdateUser.Where("uid", "=", UserUid)
		UpdateUser.Finish()

		err = UpdateUser.Execute()

		if err != nil {
			log.Printf("Cannot update user: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "User banned/unbanned successfully",
		})
	}
}
