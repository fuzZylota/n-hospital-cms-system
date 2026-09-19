package users

import (
	"encoding/json"
	"errors"
	"fmt"
	lib "lib"
	"log"
	"models"
	"regexp"
	"users/passwordpolicy"

	"github.com/gofiber/fiber/v2"
	"strconv"
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

		targetUID := c.Params("uid")
		Orm := utilities.Orm

		lookupActor := func(uid string) (userEditActor, error) {
			GetActor := Orm.Select([]string{"uid", "role", "is_active"})
			GetActor.Table("users")
			GetActor.Where("uid", "=", uid)
			GetActor.Finish()
			if lookupErr := GetActor.Execute(); lookupErr != nil {
				return userEditActor{}, lookupErr
			}
			actorRows, lookupErr := GetActor.Rows()
			if lookupErr != nil {
				return userEditActor{}, lookupErr
			}
			if len(actorRows) == 0 {
				return userEditActor{}, nil
			}
			if len(actorRows) != 1 {
				return userEditActor{}, errors.New("multiple editing users found")
			}
			return userEditActor{
				UID:      lib.String(actorRows[0]["uid"]),
				Role:     lib.String(actorRows[0]["role"]),
				IsActive: lib.Bool(actorRows[0]["is_active"]),
			}, nil
		}

		decision, err := decideUserEditRequest(authenticatedUser.Uid, targetUID, fields, lookupActor)
		if err != nil {
			if errors.Is(err, errInvalidUserEditUID) || errors.Is(err, errInvalidUserProfile) {
				return c.JSON(fiber.Map{"status": 400, "message": "Invalid user profile"})
			}
			if errors.Is(err, errUserEditActorLookup) {
				log.Printf("Cannot load editing user: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
			return c.JSON(fiber.Map{"status": 403, "message": "Only admins can edit protected user fields"})
		}
		fields = decision.Fields
		policy := decision.Policy
		targetUIDInt := decision.TargetUID

		inputs := models.UsersEdit{}
		if err = c.BodyParser(&inputs); err != nil {
			return c.JSON(fiber.Map{"status": 400, "message": "Bad request"})
		}
		if !policy.IsAdmin {
			if value, submitted := submittedFieldValue(fields, "name"); submitted {
				inputs.Name = value
			}
			if value, submitted := submittedFieldValue(fields, "surname"); submitted {
				inputs.Surname = value
			}
			if value, submitted := submittedFieldValue(fields, "email"); submitted {
				inputs.Email = value
			}
			if value, submitted := submittedFieldValue(fields, "phone"); submitted {
				inputs.Phone = value
			}
			if value, submitted := submittedFieldValue(fields, "timezone"); submitted {
				inputs.Timezone = value
			}
		}

		permissionChanges := []branchPermissionChange{}
		if policy.UpdatePermissions {
			permissionChanges, err = parseBranchPermissionChanges(targetUIDInt, fields)
			if err != nil {
				return c.JSON(fiber.Map{"status": 400, "message": "Invalid branch permissions"})
			}

			GetActiveBranches := Orm.Select([]string{"sid"})
			GetActiveBranches.Table("subeler")
			GetActiveBranches.Where("is_active", "=", true)
			GetActiveBranches.Finish()
			if err = GetActiveBranches.Execute(); err != nil {
				log.Printf("Cannot load active branches: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
			activeBranchRows, rowsErr := GetActiveBranches.Rows()
			if rowsErr != nil {
				log.Printf("Cannot read active branches: %v\n", rowsErr)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
			activeBranches := make(map[int]struct{}, len(activeBranchRows))
			for _, row := range activeBranchRows {
				branchID, conversionErr := strconv.Atoi(lib.String(row["sid"]))
				if conversionErr != nil || branchID <= 0 {
					log.Printf("Cannot parse active branch id")
					return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
				}
				activeBranches[branchID] = struct{}{}
			}
			for _, change := range permissionChanges {
				if _, ok := activeBranches[change.BranchID]; !ok {
					return c.JSON(fiber.Map{"status": 400, "message": "Invalid branch permissions"})
				}
			}
		}

		GetTarget := Orm.Select([]string{"uid", "email", "phone", "name", "surname", "role", "is_active", "timezone", "sid"})
		GetTarget.Table("users")
		GetTarget.Where("uid", "=", targetUID)
		GetTarget.Finish()
		if err = GetTarget.Execute(); err != nil {
			log.Printf("Cannot load target user: %v\n", err)
			return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
		}
		targetRows, err := GetTarget.Rows()
		if err != nil {
			log.Printf("Cannot read target user: %v\n", err)
			return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
		}
		if len(targetRows) != 1 {
			return c.JSON(fiber.Map{"status": 404, "message": "User not found"})
		}
		currentUser := targetRows[0]

		if _, submitted := fields["email"]; submitted && inputs.Email != lib.String(currentUser["email"]) {
			CheckIfEmailExists := Orm.Count("users")
			CheckIfEmailExists.Where("email", "=", inputs.Email)
			CheckIfEmailExists.And("uid", "!=", targetUID)
			CheckIfEmailExists.Finish()
			if err = CheckIfEmailExists.Execute(); err != nil {
				log.Printf("Cannot check if email exists: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
			if CheckIfEmailExists.Length() > 0 {
				return c.JSON(fiber.Map{"status": 403, "message": "Email already exists"})
			}
		}

		if _, submitted := fields["phone"]; submitted && inputs.Phone != lib.String(currentUser["phone"]) {
			CheckIfPhoneExists := Orm.Count("users")
			CheckIfPhoneExists.Where("phone", "=", inputs.Phone)
			CheckIfPhoneExists.And("uid", "!=", targetUID)
			CheckIfPhoneExists.Finish()
			if err = CheckIfPhoneExists.Execute(); err != nil {
				log.Printf("Cannot check if phone exists: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
			if CheckIfPhoneExists.Length() > 0 {
				return c.JSON(fiber.Map{"status": 403, "message": "Phone already exists"})
			}
		}

		UpdateUser := Orm.Update()
		UpdateUser.Table("users")
		userChanged := false
		if _, submitted := fields["email"]; submitted && inputs.Email != lib.String(currentUser["email"]) {
			UpdateUser.Set("email", inputs.Email)
			userChanged = true
		}
		if _, submitted := fields["phone"]; submitted && inputs.Phone != lib.String(currentUser["phone"]) {
			UpdateUser.Set("phone", inputs.Phone)
			userChanged = true
		}
		if _, submitted := fields["name"]; submitted && inputs.Name != lib.String(currentUser["name"]) {
			UpdateUser.Set("name", inputs.Name)
			userChanged = true
		}
		if _, submitted := fields["surname"]; submitted && inputs.Surname != lib.String(currentUser["surname"]) {
			UpdateUser.Set("surname", inputs.Surname)
			userChanged = true
		}
		if _, submitted := fields["timezone"]; submitted && inputs.Timezone != lib.String(currentUser["timezone"]) {
			UpdateUser.Set("timezone", inputs.Timezone)
			userChanged = true
		}

		if policy.IsAdmin {
			if _, submitted := fields["role"]; submitted && inputs.Role != lib.String(currentUser["role"]) {
				UpdateUser.Set("role", inputs.Role)
				userChanged = true
			}
			if _, submitted := fields["is_active"]; submitted && inputs.IsActive != lib.Bool(currentUser["is_active"]) {
				UpdateUser.Set("is_active", inputs.IsActive)
				userChanged = true
			}
			if _, submitted := fields["sid"]; submitted && inputs.Sid != lib.String(currentUser["sid"]) {
				UpdateUser.Set("sid", inputs.Sid)
				userChanged = true
			}
		}

		if !userChanged && len(permissionChanges) == 0 {
			return c.JSON(fiber.Map{"status": 400, "message": "Nothing changed"})
		}

		if userChanged {
			UpdateUser.Where("uid", "=", targetUID)
			UpdateUser.Finish()
			if err = UpdateUser.Execute(); err != nil {
				log.Printf("Cannot update user: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
		}

		for _, change := range permissionChanges {
			CheckPermission := Orm.Select([]string{"id"})
			CheckPermission.Table("user_branch_permissions")
			CheckPermission.Where("uid", "=", change.TargetUID)
			CheckPermission.And("sid", "=", change.BranchID)
			CheckPermission.Finish()
			if err = CheckPermission.Execute(); err != nil {
				log.Printf("Cannot check branch permission: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
			permissionRows, rowsErr := CheckPermission.Rows()
			if rowsErr != nil {
				log.Printf("Cannot read branch permission: %v\n", rowsErr)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}

			if len(permissionRows) > 0 {
				UpdatePermission := Orm.Update()
				UpdatePermission.Table("user_branch_permissions")
				if change.CanView != nil {
					UpdatePermission.Set("can_view", *change.CanView)
				}
				if change.CanDelete != nil {
					UpdatePermission.Set("can_delete", *change.CanDelete)
				}
				UpdatePermission.Where("uid", "=", change.TargetUID)
				UpdatePermission.And("sid", "=", change.BranchID)
				UpdatePermission.Finish()
				if err = UpdatePermission.Execute(); err != nil {
					log.Printf("Cannot update branch permission: %v\n", err)
					return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
				}
				continue
			}

			canView := false
			canDelete := false
			if change.CanView != nil {
				canView = *change.CanView
			}
			if change.CanDelete != nil {
				canDelete = *change.CanDelete
			}
			InsertPermission := Orm.Insert(
				[]string{"uid", "sid", "can_view", "can_delete"},
				[]interface{}{change.TargetUID, change.BranchID, canView, canDelete},
			)
			InsertPermission.Table("user_branch_permissions")
			InsertPermission.Finish()
			if err = InsertPermission.Execute(); err != nil {
				log.Printf("Cannot insert branch permission: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Internal server error"})
			}
		}

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
