package users

import (
	"fmt"
	lib "lib"
	"log"
	"models"
	"regexp"

	"database"

	"github.com/gofiber/fiber/v2"
	"strconv"
	"strings"
)

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

		fmt.Printf("user is admin\n")

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

		Options := database.Options{}
		GetOptions, err := Options.FetchOptionsForBackend(Orm, []string{"require_strong_password"}, []string{})
		if err != nil {
			log.Printf("Cannot get options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if GetOptions.Options.RequireStrongPassword {
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
				fmt.Printf("Password must contain at least one uppercase letter\n")
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
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		inputs := models.UsersEdit{}
		c.BodyParser(&inputs)

                // ---- permsChanged: Şube Yetkileri değişti mi? ----
                permsChanged := false
                if strings.Contains(string(c.Body()), "perm_view_") || strings.Contains(string(c.Body()), "perm_delete_") { permsChanged = true }
                if mf, err := c.MultipartForm(); err == nil && mf != nil {
                        for k := range mf.Value {
                                if strings.HasPrefix(k, "perm_view_") || strings.HasPrefix(k, "perm_delete_") {
                                        permsChanged = true
                                        break
                                }
                        }
                } else {
                        // multipart değilse de form değerlerinden anlamaya çalış
                        // (checkbox işaretliyse gelir, işaretsizse gelmez)
                        // bu durumda permsChanged false kalabilir, ama en azından crash olmaz
                }


		Orm := utilities.Orm

		if OurUser.Role != "admin" {
			if OurUser.Uid != inputs.Uid {
				return c.JSON(fiber.Map{
					"status":  403,
					"message": "Only admins can edit users",
				})
			}
		}

		if OurUser.Role != "admin" && (inputs.Role != inputs.OldRole) {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can edit users role",
			})
		}

		if OurUser.Role != "admin" && (inputs.IsActive != inputs.OldIsActive) {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can edit users ban status",
			})
		}

		if OurUser.Role != "admin" && (inputs.Sid != inputs.OldSid) {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can edit users sube status",
			})
		}

		if inputs.Email != inputs.OldEmail {
			CheckIfEmailExists := Orm.Count("users")
			CheckIfEmailExists.Where("email", "=", inputs.Email)
			CheckIfEmailExists.Finish()

			err = CheckIfEmailExists.Execute()

			if err != nil {
				log.Printf("Cannot check if email exists: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if CheckIfEmailExists.Length() > 0 {
				return c.JSON(fiber.Map{
					"status":  403,
					"message": "Email already exists",
				})
			}
		}

		if inputs.Phone != inputs.OldPhone {
			CheckIfPhoneExists := Orm.Count("users")
			CheckIfPhoneExists.Where("phone", "=", inputs.Phone)
			CheckIfPhoneExists.Finish()

			err = CheckIfPhoneExists.Execute()

			if err != nil {
				log.Printf("Cannot check if phone exists: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if CheckIfPhoneExists.Length() > 0 {
				return c.JSON(fiber.Map{
					"status":  403,
					"message": "Phone already exists",
				})
			}
		}

		UpdateUser := Orm.Update()
		UpdateUser.Table("users")
		SomethingSet := false

		{
			if inputs.Email != inputs.OldEmail {
				UpdateUser.Set("email", inputs.Email)
				SomethingSet = true
			}

			if inputs.Phone != inputs.OldPhone {
				UpdateUser.Set("phone", inputs.Phone)
				SomethingSet = true
			}

			if inputs.Name != inputs.OldName {
				UpdateUser.Set("name", inputs.Name)
				SomethingSet = true
			}

			if inputs.Surname != inputs.OldSurname {
				UpdateUser.Set("surname", inputs.Surname)
				SomethingSet = true
			}

			if inputs.Role != inputs.OldRole {
				UpdateUser.Set("role", inputs.Role)
				SomethingSet = true
			}

			if inputs.IsActive != inputs.OldIsActive {
				UpdateUser.Set("is_active", inputs.IsActive)
				SomethingSet = true
			}

			if inputs.Timezone != inputs.OldTimezone {
				UpdateUser.Set("timezone", inputs.Timezone)
				SomethingSet = true
			}

			if inputs.Sid != inputs.OldSid {
				UpdateUser.Set("sid", inputs.Sid)
				SomethingSet = true
			}
		}

		if !SomethingSet && !permsChanged {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Nothing changed",
			})
		}

		if SomethingSet {
		        		UpdateUser.Where("uid", "=", inputs.Uid)
		        		UpdateUser.Finish()
		        
		        		err = UpdateUser.Execute()
		        
		        		if err != nil {
		        			log.Printf("Cannot update user: %v\n", err)
		        			return c.JSON(fiber.Map{
		        				"status":  500,
		        				"message": "Internal server error",
		        			})
		        		}
		}
                // ---- ŞUBE YETKİLERİ KAYDI (user_branch_permissions) ----
                // Formdan perm_view_<sid> / perm_delete_<sid> checkboxlarını okuyup DB'ye yazar.
                // Checkbox işaretli değilse FormValue boş gelir.
                // uid: en doğrusu formdan gelen inputs.Uid (hidden uid alanı)
                uidInt, errUid := strconv.Atoi(inputs.Uid)
                if errUid != nil || uidInt == 0 {
                        // fallback: route parametrelerinden dene
                        uidStr := c.Params("uid")
                        if uidStr == "" {
                                uidStr = c.Params("user")
                        }
                        if uidStr == "" {
                                uidStr = c.Params("kullanici")
                        }
                        uidInt, _ = strconv.Atoi(uidStr)
                }
                // aktif şubeleri çek
                GetSubelerForPerm := Orm.Select([]string{"sid"})
                GetSubelerForPerm.Table("subeler")
                GetSubelerForPerm.Where("is_active", "=", true)
                GetSubelerForPerm.Finish()
                _ = GetSubelerForPerm.Execute()
                subeRows, _ := GetSubelerForPerm.Rows()

                for _, r := range subeRows {
                        sidStr := lib.String(r["sid"])
                        sidInt, _ := strconv.Atoi(sidStr)
                        // perm_view_<sid> / perm_delete_<sid> multi-value oku (hidden=0 + checkbox=1)
                        hasOne := func(vals []string) bool {
                                for _, v := range vals {
                                        if v == "1" || v == "true" || v == "on" {
                                                return true
                                        }
                                }
                                return false
                        }

                        // JSON body'yi bir kez parse et
                        var jsonBody map[string]interface{}
                        _ = c.BodyParser(&jsonBody)

                        readAll := func(key string) []string {
                                // JSON body'den oku
                                if jsonBody != nil {
                                        if val, ok := jsonBody[key]; ok {
                                                switch v := val.(type) {
                                                case string:
                                                        return []string{v}
                                                case bool:
                                                        if v {
                                                                return []string{"1"}
                                                        }
                                                        return []string{"0"}
                                                case float64:
                                                        if v == 1 {
                                                                return []string{"1"}
                                                        }
                                                        return []string{"0"}
                                                }
                                        }
                                }
                                // multipart ise
                                if mf, err := c.MultipartForm(); err == nil && mf != nil {
                                        if vals, ok := mf.Value[key]; ok {
                                                return vals
                                        }
                                }
                                // urlencoded ise
                                bvals := c.Context().PostArgs().PeekMulti(key)
                                out := make([]string, 0, len(bvals))
                                for _, bv := range bvals {
                                        out = append(out, string(bv))
                                }
                                return out
                        }

                        canView := hasOne(readAll("perm_view_" + sidStr))
                        canDelete := hasOne(readAll("perm_delete_" + sidStr))



                        // var mı?
                        CheckPerm := Orm.Select([]string{"id"})
                        CheckPerm.Table("user_branch_permissions")
                        CheckPerm.Where("uid", "=", uidInt)
                        CheckPerm.And("sid", "=", sidInt)
                        CheckPerm.Finish()
                        err = CheckPerm.Execute()
                                if err != nil {
                                        log.Printf("perm CheckPerm execute error: %v\n", err)
                                }
                        permRows, _ := CheckPerm.Rows()

                        if len(permRows) > 0 {
                                Upd := Orm.Update()
                                Upd.Table("user_branch_permissions")
                                Upd.Set("can_view", canView)
                                Upd.Set("can_delete", canDelete)
                                Upd.Where("uid", "=", uidInt)
                                Upd.And("sid", "=", sidInt)
                                Upd.Finish()
                                err = Upd.Execute()
                                  if err != nil {
                                          log.Printf("perm update execute error: %v\n", err)
                                  }
                        } else {
                                Ins := Orm.Insert(
                                        []string{"uid", "sid", "can_view", "can_delete"},
                                        []interface{}{uidInt, sidInt, canView, canDelete},
                                )
                                Ins.Table("user_branch_permissions")
                                Ins.Finish()
                                err = Ins.Execute()
                                  if err != nil {
                                          log.Printf("perm insert execute error: %v\n", err)
                                  }
                        }
                }
                // ---- ŞUBE YETKİLERİ KAYDI SON ----



		return c.JSON(fiber.Map{
			"status":  201,
			"message": "User edited successfully",
		})
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

		Options := database.Options{}
		GetOptions, err := Options.FetchOptionsForBackend(Orm, []string{"require_strong_password"}, []string{})
		if err != nil {
			log.Printf("Cannot get options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if GetOptions.Options.RequireStrongPassword {
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
		fmt.Printf("OurUser: %v\n", OurUser)

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
			fmt.Printf("err: %v\n", err)
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
