package post

import (
	"database"
	"encoding/json"
	"fmt"
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	wsb "github.com/Necoo33/fiber-ws-broadcaster"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func GreetPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, World!",
		})
	}
}

func AuthenticationController(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.GetJWT(c)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=error_while_getting_jwt")
		}

		if ourUser.Uid != "" {
			log.Printf("User is already authenticated")
			return c.Redirect("/panel")
		}

		inputs := models.AuthInputs{}

		c.BodyParser(&inputs)

		checkIfEmailOrPhoneExists := utilities.Orm.Select([]string{"*"})
		checkIfEmailOrPhoneExists.Table("users")
		checkIfEmailOrPhoneExists.Where("email", "=", inputs.EmailOrPhone)
		checkIfEmailOrPhoneExists.Or("phone", "=", inputs.EmailOrPhone)
		checkIfEmailOrPhoneExists.Finish()
		err = checkIfEmailOrPhoneExists.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		rows, err := checkIfEmailOrPhoneExists.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		if len(rows) == 0 {
			return c.Redirect("/giris?error=user_not_found")
		}

		CheckIfUserIsActive := checkIfEmailOrPhoneExists.Count("users")
		CheckIfUserIsActive.Where("uid", "=", rows[0]["uid"])
		CheckIfUserIsActive.And("is_active", "=", true)
		CheckIfUserIsActive.Finish()

		err = CheckIfUserIsActive.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		if CheckIfUserIsActive.Length() == 0 {
			return c.Redirect("/giris?error=user_is_not_active")
		}

		decryptedPassword, err := lib.Decrypt(rows[0]["password"].(string))

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		if string(decryptedPassword) != inputs.Password {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=password_is_incorrect")
		}

		fmt.Printf("rows[0]['is_active']: %v\n", rows[0]["is_active"])

		authenticatedUser := models.AuthenticatedUser{
			Remember:  inputs.Remember,
			Uid:       lib.String(rows[0]["uid"]),
			Email:     lib.String(rows[0]["email"]),
			Phone:     lib.String(rows[0]["phone"]),
			Name:      lib.String(rows[0]["name"]),
			Surname:   lib.String(rows[0]["surname"]),
			Role:      lib.String(rows[0]["role"]),
			Timezone:  lib.String(rows[0]["timezone"]),
			IsActive:  lib.Bool(rows[0]["is_active"]),
			LastLogin: time.Now(),
			CreatedAt: lib.Time(rows[0]["created_at"]),
		}

		updateLastLogin := checkIfEmailOrPhoneExists.Update()
		updateLastLogin.Table("users")
		updateLastLogin.Set("last_login", "NOW()")
		updateLastLogin.Where("uid", "=", authenticatedUser.Uid)

		err = updateLastLogin.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		affectedRows, err := updateLastLogin.RowsAffected()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		if affectedRows == 0 {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		tokenString, err := lib.CreateJWT(authenticatedUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris?error=internal_server_error")
		}

		cookieName := os.Getenv("AUTH_COOKIE_NAME")

		if cookieName == "" {
			cookieName = "n-hospital-auth"
		}

		cookie := fiber.Cookie{
			Name:     cookieName,
			Value:    tokenString,
			HTTPOnly: true,
		}

		if inputs.Remember {
			cookie.MaxAge = 7200 * 12 * 30
		} else {
			cookie.MaxAge = 7200
		}

		c.Cookie(&cookie)

		switch authenticatedUser.Role {
		case "admin", "moderator":
			return c.Redirect("/panel")
		case "santral":
			return c.Redirect("/panel/randevu-talepleri")
		case "ik":
			return c.Redirect("/panel/iletisim-istekleri")
		}

		return c.Redirect("/panel")
	}
}

func LogoutController(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.Redirect("/giris?error=user_not_authenticated")
		}

		cookieName := os.Getenv("AUTH_COOKIE_NAME")
		if cookieName == "" {
			cookieName = "n-hospital-auth"
		}

		c.Cookie(&fiber.Cookie{
			Name:     cookieName,
			Value:    "",
			Expires:  time.Now().Add(-time.Hour),
			HTTPOnly: true,
			MaxAge:   0,
		})

		return c.Redirect("/")
	}
}

func AddCustomMedia(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(utilities.Orm, []string{}, []string{})
		if err != nil {
			log.Printf("Cannot get options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("ROOT_DIR is not set")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		uploadDir := filepath.Join(RootDir, "static", "uploads")

		FileInfos := []models.File{}

		// First try to get files with standard "file" name (for single file uploads)
		file, err := c.FormFile("file")
		if err == nil {
			// Single file upload
			// Check file size
			maxFileSize := GetOptions.Options.MaxUploadSize
			if file.Size > maxFileSize {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": fmt.Sprintf("Dosya boyutu çok büyük. Maksimum dosya boyutu: %d bytes", maxFileSize),
				})
			}

			uniquePathResponse, err := lib.UniqueFilePath(uploadDir + "/" + file.Filename)
			if err != nil {
				log.Printf("Cannot generate unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save file to uploads directory
			err = lib.SaveFileWithBufferingWithRenaming(uploadDir, uniquePathResponse.BaseName, *file)
			if err != nil {
				log.Printf("Cannot save file: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Get file info
			fileInfo, err := os.Stat(uniquePathResponse.FilePath)
			if err != nil {
				log.Printf("Cannot get file info: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			FileInfos = append(FileInfos, models.File{
				Name: file.Filename,
				Url:  uniquePathResponse.FilePath,
				Size: fileInfo.Size(),
			})
		} else {
			// Multiple file upload with numbered names
			for i := 1; i <= 10; i++ {
				file, err := c.FormFile("file" + strconv.Itoa(i))
				if err != nil {
					continue
				}

				// Check file size
				maxFileSize := GetOptions.Options.MaxUploadSize
				if file.Size > maxFileSize {
					continue
				}

				filePath := filepath.Join(RootDir, uploadDir, file.Filename)

				uniquePathResponse, err := lib.UniqueFilePath(filePath + "/" + file.Filename)
				if err != nil {
					log.Printf("Cannot generate unique file path: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				OurUploadDir := filepath.Join(RootDir, "static", "uploads")

				// Save file to uploads directory
				err = lib.SaveFileWithBufferingWithRenaming(OurUploadDir, uniquePathResponse.BaseName, *file)
				if err != nil {
					log.Printf("Cannot save file: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				// Get file info
				fileInfo, err := os.Stat(uniquePathResponse.FilePath)
				if err != nil {
					log.Printf("Cannot get file info: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				FileInfos = append(FileInfos, models.File{
					Name: file.Filename,
					Url:  uniquePathResponse.FilePath,
					Size: fileInfo.Size(),
				})
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Dosya başarıyla yüklendi.",
			"data":    FileInfos,
		})
	}
}

func DeleteCustomMedia(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		inputs := models.DeleteFileInputs{}
		c.BodyParser(&inputs)

		if inputs.FileName == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Dosya adı zorunludur.",
			})
		}

		RootDir := os.Getenv("ROOT_DIRECTORY")
		if RootDir == "" {
			// Try to get current working directory as fallback
			wd, err := os.Getwd()
			if err != nil {
				log.Printf("Cannot get current working directory: %v", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
			RootDir = wd
			log.Printf("Using current working directory as ROOT_DIRECTORY: %s", RootDir)
		}

		// Construct full file path
		filePath := filepath.Join(RootDir, "static/uploads", inputs.FileName)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Dosya bulunamadı.",
			})
		}

		// Delete the file
		err = lib.DeleteFile(filePath)
		if err != nil {
			log.Printf("Cannot delete file: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Dosya başarıyla silindi.",
		})
	}
}

func GetFiles(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		RootDir := os.Getenv("ROOT_DIRECTORY")
		if RootDir == "" {
			// Try to get current working directory as fallback
			wd, err := os.Getwd()
			if err != nil {
				log.Printf("Cannot get current working directory: %v", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
			RootDir = wd
			log.Printf("Using current working directory as ROOT_DIRECTORY: %s", RootDir)
		}

		// Read uploads directory
		uploadsDir := filepath.Join(RootDir, "static/uploads")
		entries, err := lib.ReadDirectory(uploadsDir)
		if err != nil {
			log.Printf("Cannot read uploads directory: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		var files []fiber.Map
		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(uploadsDir, entry.Name())
				fileInfo, err := os.Stat(filePath)
				if err != nil {
					continue
				}

				// Determine file type
				fileType := "other"
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				switch ext {
				case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg":
					fileType = "image"
				case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm":
					fileType = "video"
				case ".mp3", ".wav", ".ogg", ".flac":
					fileType = "audio"
				case ".pdf":
					fileType = "pdf"
				case ".doc", ".docx":
					fileType = "document"
				case ".xls", ".xlsx":
					fileType = "spreadsheet"
				case ".ppt", ".pptx":
					fileType = "presentation"
				case ".zip", ".rar", ".7z", ".tar", ".gz":
					fileType = "archive"
				}

				files = append(files, fiber.Map{
					"name": entry.Name(),
					"size": fileInfo.Size(),
					"type": fileType,
					"date": fileInfo.ModTime().Format("2006-01-02T15:04:05Z"),
				})
			}
		}

		return c.JSON(fiber.Map{
			"status": 200,
			"data":   files,
		})
	}
}

func AddExpertiseArea(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			return c.Redirect("/panel/uzmanliklar/uzmanlik-ekle?error=only_admins_can_add_expertise")
		}

		inputs := models.Uzmanliklar{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.Name == "" {
			return c.Redirect("/panel/uzmanliklar/uzmanlik-ekle?error=name_required")
		}

		columns := []string{"name", "is_active"}
		values := []interface{}{inputs.Name, inputs.IsActive}

		// Add optional fields if provided
		if inputs.UrlName != "" {
			columns = append(columns, "url_name")
			values = append(values, inputs.UrlName)
		}
		if inputs.Description != "" {
			columns = append(columns, "description")
			values = append(values, inputs.Description)
		}
		if inputs.Icon != "" {
			columns = append(columns, "icon")
			values = append(values, inputs.Icon)
		}

		Orm := utilities.Orm

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/uzmanliklar/uzmanlik-ekle?error=internal_server_error")
		}

		insertUzmanlik := Orm.Insert(columns, values)
		insertUzmanlik.Table("uzmanliklar")
		insertUzmanlik.Returning("uzid")
		insertUzmanlik.Finish()
		err = insertUzmanlik.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert expertise: %v\n", err)
			return c.Redirect("/panel/uzmanliklar/uzmanlik-ekle?error=internal_server_error")
		}

		uzid, err := insertUzmanlik.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/uzmanliklar/uzmanlik-ekle?error=internal_server_error")
		}

		Orm.Commit()

		return c.Redirect("/panel/uzmanliklar/" + uzid)
	}
}

func EditExpertiseArea(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		ExpertiseId := c.Params("eid")
		Orm := utilities.Orm

		inputs := models.UzmanliklarEdit{}
		c.BodyParser(&inputs)
		inputs.Uzid = ExpertiseId

		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Uzmanlık adı zorunludur.",
			})
		}
		if inputs.UrlName == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "URL adı zorunludur.",
			})
		}

		// Check if UrlName already exists (excluding current record)
		if inputs.UrlName != inputs.OldUrlName {
			CheckIfUrlNameExists := Orm.Count("uzmanliklar")
			CheckIfUrlNameExists.Where("url_name", "=", inputs.UrlName)
			CheckIfUrlNameExists.And("uzid", "!=", ExpertiseId)
			CheckIfUrlNameExists.Finish()
			err = CheckIfUrlNameExists.Execute()

			if err != nil {
				log.Printf("%v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if CheckIfUrlNameExists.Length() > 0 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Bu URL adı zaten kullanılıyor.",
				})
			}
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		updateExpertise := Orm.Update()
		updateExpertise.Table("uzmanliklar")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			updateExpertise.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateExpertise.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateExpertise.Set("description", inputs.Description)
			SomethingSet = true
		}

		if inputs.Icon != inputs.OldIcon {
			updateExpertise.Set("icon", inputs.Icon)
			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateExpertise.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		if SomethingSet {
			updateExpertise.Set("updated_at", "NOW()")
			updateExpertise.Where("uzid", "=", ExpertiseId)
			updateExpertise.Finish()
			err = updateExpertise.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update expertise: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Uzmanlık alanı başarıyla güncellendi.",
		})
	}
}

func DeleteExpertiseArea(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		ExpertiseId := c.Params("eid")
		if ExpertiseId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Expertise ID is required",
			})
		}

		Orm := utilities.Orm

		// Check if expertise exists
		CheckExpertise := Orm.Select([]string{"uzid"})
		CheckExpertise.Table("uzmanliklar")
		CheckExpertise.Where("uzid", "=", ExpertiseId)
		CheckExpertise.Finish()
		err = CheckExpertise.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		expertiseRows, err := CheckExpertise.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(expertiseRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Expertise area not found",
			})
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete the expertise
		DeleteExpertise := Orm.Delete()
		DeleteExpertise.Table("uzmanliklar")
		DeleteExpertise.Where("uzid", "=", ExpertiseId)
		DeleteExpertise.Finish()
		err = DeleteExpertise.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteExpertise.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Uzmanlık alanı bulunamadı.",
			})
		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Expertise area deleted successfully",
		})
	}
}

func GetAllExpertises(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Orm := utilities.Orm

		GetExpertises := Orm.Select([]string{"uzid", "name"})
		GetExpertises.Table("uzmanliklar")
		GetExpertises.Where("is_active", "=", true)
		GetExpertises.OrderBy("name", "ASC")
		GetExpertises.Finish()
		err = GetExpertises.Execute()

		if err != nil {
			log.Printf("Cannot get expertises: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetExpertises.Rows()
		if err != nil {
			log.Printf("Cannot get expertises: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ExpertisesArray := []models.Uzmanliklar{}
		for _, row := range rows {
			ExpertisesArray = append(ExpertisesArray, models.Uzmanliklar{
				Uzid: lib.String(row["uzid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Expertise fetched successfully",
			"data":    ExpertisesArray,
		})
	}
}

func AddHomepageContent(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=only_admins_can_add_homepage_content")
		}

		inputs := models.HomepageContents{}
		c.BodyParser(&inputs)

		Orm := utilities.Orm

		// Required fields validation
		if inputs.Name == "" {
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=name_required")
		}

		if inputs.ContentType == "" {
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=content_type_required")
		}

		if inputs.ContentType == "popup" {
			CheckIfThereIsAPopup := Orm.Count("homepage_contents")
			CheckIfThereIsAPopup.Where("content_type", "=", "popup")
			CheckIfThereIsAPopup.Finish()

			err = CheckIfThereIsAPopup.Execute()

			if err != nil {
				log.Printf("%v\n", err)
			}

			if CheckIfThereIsAPopup.Length() > 0 {
				return c.Redirect("/panel/anasayfa-icerik-ekle?error=only_one_popup_content_allowed")
			}
		}

		if inputs.SortOrder == 0 {
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=sort_order_required")
		}

		columns := []string{"name", "content_type", "sort_order", "is_active"}
		values := []interface{}{inputs.Name, inputs.ContentType, inputs.SortOrder, inputs.IsActive}

		// Add optional fields if provided
		if inputs.UrlName != "" {
			columns = append(columns, "url_name")
			values = append(values, inputs.UrlName)
		}
		if inputs.Description != "" {
			columns = append(columns, "description")
			values = append(values, inputs.Description)
		}
		if inputs.LaterThanWhichContent != 0 {
			columns = append(columns, "later_than_which_content")
			values = append(values, inputs.LaterThanWhichContent)
		}
		if inputs.ContentHtml != "" {
			columns = append(columns, "content_html")
			values = append(values, inputs.ContentHtml)
		}
		if inputs.ContentCss != "" {
			columns = append(columns, "content_css")
			values = append(values, inputs.ContentCss)
		}
		if inputs.ContentJavascript != "" {
			columns = append(columns, "content_javascript")
			values = append(values, inputs.ContentJavascript)
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
		}

		insertHomepageContent := Orm.Insert(columns, values)
		insertHomepageContent.Table("homepage_contents")
		insertHomepageContent.Returning("hcid")
		insertHomepageContent.Finish()

		err = insertHomepageContent.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert homepage content: %v\n", err)
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
		}

		hcid, err := insertHomepageContent.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
		}

		handleSorting := Orm.SelectFunction("get_shift_for_insert_for_homepage_contents", inputs.SortOrder, hcid, inputs.ContentType)
		handleSorting.Finish()

		err = handleSorting.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot execute handle_homepage_content_sorting function: %v\n", err)
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
		}

		rows, err := handleSorting.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get handle_homepage_content_sorting function rows: %v\n", err)
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
		}

		if len(rows) == 0 {
			log.Printf("Cannot insert homepage content, issue is no rows returned : %v\n", len(rows) == 0)
			Orm.Rollback()
			return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
		}

		if rows[0]["our_hcid"] == nil {
			FuckingCleanup := Orm.Update()
			FuckingCleanup.Table("homepage_contents")
			FuckingCleanup.Set("sort_order", rows[0]["new_sort_order"])
			FuckingCleanup.Where("hcid", "=", hcid)
			FuckingCleanup.Finish()

			err = FuckingCleanup.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute fucking cleanup: %v\n", err)
				return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
			}
		} else {
			FuckingCleanup := Orm.Update()
			FuckingCleanup.Table("homepage_contents")
			FuckingCleanup.SetExpr("sort_order", "sort_order + 1")
			FuckingCleanup.Where("sort_order", ">=", inputs.SortOrder)
			FuckingCleanup.And("hcid", "!=", hcid)
			FuckingCleanup.Finish()

			err = FuckingCleanup.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute fucking cleanup: %v\n", err)
				return c.Redirect("/panel/anasayfa-icerik-ekle?error=internal_server_error")
			}
		}

		Orm.Commit()

		return c.Redirect("/panel/anasayfa-icerikleri/" + hcid)
	}
}

func EditHomepageContent(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Hcid := c.Params("hcid")
		Orm := utilities.Orm

		inputs := models.HomepageContentsEdit{}
		c.BodyParser(&inputs)
		inputs.Hcid = Hcid

		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "İçerik adı zorunludur.",
			})
		}
		if inputs.ContentType == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "İçerik tipi zorunludur.",
			})
		}

		if inputs.ContentType == "popup" {
			CheckIfThereIsAPopup := Orm.Count("homepage_contents")
			CheckIfThereIsAPopup.Where("content_type", "=", "popup")
			CheckIfThereIsAPopup.And("hcid", "!=", Hcid)
			CheckIfThereIsAPopup.Finish()

			err = CheckIfThereIsAPopup.Execute()

			if err != nil {
				log.Printf("%v\n", err)
			}

			if CheckIfThereIsAPopup.Length() > 0 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Birden fazla popup içeriği olamaz.",
				})
			}
		}

		// Check for duplicate URL name if changed
		if inputs.UrlName != inputs.OldUrlName && inputs.UrlName != "" {
			checkUrlName := Orm.Select([]string{"hcid"})
			checkUrlName.Table("homepage_contents")
			checkUrlName.Where("url_name", "=", inputs.UrlName)
			checkUrlName.And("hcid", "!=", Hcid)
			checkUrlName.Finish()
			err = checkUrlName.Execute()
			if err != nil {
				log.Printf("%v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
			rows, err := checkUrlName.Rows()
			if err != nil {
				log.Printf("%v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
			if len(rows) > 0 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Bu URL adı zaten kullanılıyor.",
				})
			}
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		orderingChanged := (inputs.SortOrder != inputs.OldSortOrder) || (inputs.ContentType != inputs.OldContentType)
		if orderingChanged {
			ReorderHomepageContents := Orm.SelectFunction("get_shift_for_update_for_homepage_contents", inputs.SortOrder, inputs.OldSortOrder, inputs.Hcid, inputs.ContentType)
			ReorderHomepageContents.Finish()
			err = ReorderHomepageContents.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute update_homepage_content_sort function: %v\n", err)
			}

			rows, err := ReorderHomepageContents.Rows()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get update_homepage_content_sort function rows: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			fmt.Printf("rows: %v\n", rows)

			if len(rows) != 0 {
				HcidsToChange := []any{}

				for _, row := range rows {
					HcidsToChange = append(HcidsToChange, row["our_hcid"])
				}

				Expression := ""

				if rows[0]["direction"] == "down" {
					Expression = "sort_order - 1"
				} else {
					Expression = "sort_order + 1"
				}

				UpdateSortings := Orm.Update()
				UpdateSortings.Table("homepage_contents")
				UpdateSortings.SetExpr("sort_order", Expression)
				UpdateSortings.In("WHERE", "hcid", HcidsToChange)
				UpdateSortings.Finish()

				fmt.Printf("UpdateSortings: %v\n", UpdateSortings.Query)

				err = UpdateSortings.Execute()

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update homepage content sortings: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				UpdateThisHomepageContent := Orm.Update()
				UpdateThisHomepageContent.Table("homepage_contents")
				UpdateThisHomepageContent.Set("sort_order", rows[0]["new_sort_order"])
				UpdateThisHomepageContent.Where("hcid", "=", inputs.Hcid)
				UpdateThisHomepageContent.Finish()
				err = UpdateThisHomepageContent.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update this homepage content sort order: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}
			} else {
				if inputs.ContentType != inputs.OldContentType {
					handleSorting := Orm.SelectFunction("get_shift_for_insert_for_homepage_contents", inputs.SortOrder, inputs.Hcid, inputs.ContentType)
					handleSorting.Finish()

					err = handleSorting.Execute()

					if err != nil {
						Orm.Rollback()
						log.Printf("Cannot execute handle_homepage_content_sorting function: %v\n", err)
						return c.JSON(fiber.Map{
							"status":  500,
							"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
						})
					}

					rows, err := handleSorting.Rows()
					if err != nil {
						Orm.Rollback()
						log.Printf("Cannot get handle_homepage_content_sorting function rows: %v\n", err)
						return c.JSON(fiber.Map{
							"status":  500,
							"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
						})
					}

					if len(rows) == 0 {
						log.Printf("Cannot insert homepage content, issue is no rows returned : %v\n", len(rows) == 0)
						Orm.Rollback()
						return c.JSON(fiber.Map{
							"status":  500,
							"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
						})
					}

					if rows[0]["our_hcid"] == nil {
						FuckingCleanup := Orm.Update()
						FuckingCleanup.Table("homepage_contents")
						FuckingCleanup.Set("sort_order", rows[0]["new_sort_order"])
						FuckingCleanup.Where("hcid", "=", inputs.Hcid)
						FuckingCleanup.Finish()

						err = FuckingCleanup.Execute()

						if err != nil {
							Orm.Rollback()
							log.Printf("Cannot execute fucking cleanup: %v\n", err)
							return c.JSON(fiber.Map{
								"status":  500,
								"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
							})
						}
					} else {
						FuckingCleanup := Orm.Update()
						FuckingCleanup.Table("homepage_contents")
						FuckingCleanup.SetExpr("sort_order", "sort_order + 1")
						FuckingCleanup.Where("sort_order", ">=", inputs.SortOrder)
						FuckingCleanup.And("hcid", "!=", inputs.Hcid)
						FuckingCleanup.Finish()

						err = FuckingCleanup.Execute()

						if err != nil {
							Orm.Rollback()
							log.Printf("Cannot execute fucking cleanup: %v\n", err)
							return c.JSON(fiber.Map{
								"status":  500,
								"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
							})
						}
					}
				}
			}
		}

		updateHomepageContent := Orm.Update()
		updateHomepageContent.Table("homepage_contents")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			updateHomepageContent.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateHomepageContent.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.ContentType != inputs.OldContentType {
			updateHomepageContent.Set("content_type", inputs.ContentType)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateHomepageContent.Set("description", inputs.Description)
			SomethingSet = true
		}

		if inputs.LaterThanWhichContent != inputs.OldLaterThanWhichContent {
			updateHomepageContent.Set("later_than_which_content", inputs.LaterThanWhichContent)
			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateHomepageContent.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		// Update HTML content if provided
		if inputs.ContentHtml != inputs.OldContentHtml {
			updateHomepageContent.Set("content_html", inputs.ContentHtml)
			SomethingSet = true
		}

		// Update CSS content if provided
		if inputs.ContentCss != inputs.OldContentCss {
			updateHomepageContent.Set("content_css", inputs.ContentCss)
			SomethingSet = true
		}

		// Update JavaScript content if provided
		if inputs.ContentJavascript != inputs.OldContentJavascript {
			updateHomepageContent.Set("content_javascript", inputs.ContentJavascript)
			SomethingSet = true
		}

		if SomethingSet {
			updateHomepageContent.Set("updated_at", "NOW()")
			updateHomepageContent.Where("hcid", "=", Hcid)
			updateHomepageContent.Finish()
			err = updateHomepageContent.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update homepage content: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Anasayfa içeriği başarıyla güncellendi.",
		})
	}
}

func DeleteHomepageContent(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		HomepageContentId := c.Params("hcid")
		if HomepageContentId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Homepage content ID is required",
			})
		}

		Orm := utilities.Orm

		// Check if homepage content exists
		GetHomepageContent := Orm.Select([]string{"hcid", "sort_order", "content_type"})
		GetHomepageContent.Table("homepage_contents")
		GetHomepageContent.Where("hcid", "=", HomepageContentId)
		GetHomepageContent.Finish()
		err = GetHomepageContent.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		homepageContentRows, err := GetHomepageContent.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(homepageContentRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Homepage content not found",
			})
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete the homepage content
		DeleteHomepageContent := Orm.Delete()
		DeleteHomepageContent.Table("homepage_contents")
		DeleteHomepageContent.Where("hcid", "=", HomepageContentId)
		DeleteHomepageContent.Finish()
		err = DeleteHomepageContent.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteHomepageContent.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Anasayfa içeriği bulunamadı.",
			})
		}

		handleSorting := Orm.SelectFunction("get_shift_for_delete_for_homepage_contents", homepageContentRows[0]["sort_order"], HomepageContentId, homepageContentRows[0]["content_type"])
		handleSorting.Finish()
		err = handleSorting.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot execute handle_homepage_content_sorting function: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := handleSorting.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get handle_homepage_content_sorting function rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if len(rows) != 0 {
			UpdateRows := Orm.Update()
			UpdateRows.Table("homepage_contents")
			UpdateRows.SetExpr("sort_order", "sort_order - 1")

			Ins := []any{}
			for _, row := range rows {
				if row["old_content_type"] != homepageContentRows[0]["content_type"] {
					continue
				}

				Ins = append(Ins, lib.String(row["our_hcid"]))
			}
			UpdateRows.In("WHERE", "hcid", Ins)
			UpdateRows.And("content_type", "=", homepageContentRows[0]["content_type"])
			UpdateRows.Finish()

			err = UpdateRows.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update homepage content sortings: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Anasayfa içeriği başarıyla silindi.",
		})
	}
}

func AddCustomContent(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			return c.Redirect("/panel/ozel-icerik-ekle?error=only_admins_can_add_custom_content")
		}

		inputs := models.CustomContents{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.Name == "" {
			return c.Redirect("/panel/ozel-icerik-ekle?error=name_required")
		}
		if inputs.ContentType == "" {
			return c.Redirect("/panel/ozel-icerik-ekle?error=content_type_required")
		}
		if inputs.SortOrder == 0 {
			return c.Redirect("/panel/ozel-icerik-ekle?error=sort_order_required")
		}

		columns := []string{"name", "content_type", "sort_order", "is_active"}
		values := []interface{}{inputs.Name, inputs.ContentType, inputs.SortOrder, inputs.IsActive}

		// Add optional fields if provided
		if inputs.UrlName != "" {
			columns = append(columns, "url_name")
			values = append(values, inputs.UrlName)
		}
		if inputs.Description != "" {
			columns = append(columns, "description")
			values = append(values, inputs.Description)
		}
		if inputs.Route != "" {
			columns = append(columns, "route")
			values = append(values, inputs.Route)
		}
		if inputs.ContentHtml != "" {
			columns = append(columns, "content_html")
			values = append(values, inputs.ContentHtml)
		}
		if inputs.ContentCss != "" {
			columns = append(columns, "content_css")
			values = append(values, inputs.ContentCss)
		}
		if inputs.ContentJavascript != "" {
			columns = append(columns, "content_javascript")
			values = append(values, inputs.ContentJavascript)
		}

		Orm := utilities.Orm

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
		}

		insertCustomContent := Orm.Insert(columns, values)
		insertCustomContent.Table("custom_contents")
		insertCustomContent.Returning("ccid")
		insertCustomContent.Finish()

		err = insertCustomContent.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert custom content: %v\n", err)
			return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
		}

		ccid, err := insertCustomContent.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
		}

		handleSorting := Orm.SelectFunction("get_shift_for_insert_for_custom_contents", inputs.SortOrder, ccid, inputs.ContentType)
		handleSorting.Finish()

		err = handleSorting.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot execute handle_button_sorting function: %v\n", err)
			return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
		}

		rows, err := handleSorting.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get handle_button_sorting function rows: %v\n", err)
			return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
		}

		if len(rows) == 0 {
			log.Printf("Cannot delete header button, issue is no rows returned : %v\n", len(rows) == 0)
			Orm.Rollback()
			return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
		}

		if rows[0]["our_ccid"] == nil {
			FuckingCleanup := Orm.Update()
			FuckingCleanup.Table("custom_contents")
			FuckingCleanup.Set("sort_order", rows[0]["new_sort_order"])
			FuckingCleanup.Where("ccid", "=", ccid)
			FuckingCleanup.Finish()

			err = FuckingCleanup.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute fucking cleanup: %v\n", err)
				return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
			}
		} else {
			FuckingCleanup := Orm.Update()
			FuckingCleanup.Table("custom_contents")
			FuckingCleanup.SetExpr("sort_order", "sort_order + 1")
			FuckingCleanup.Where("sort_order", ">=", inputs.SortOrder)
			FuckingCleanup.And("ccid", "!=", ccid)
			FuckingCleanup.Finish()

			err = FuckingCleanup.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute fucking cleanup: %v\n", err)
				return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
			}
		}

		Orm.Commit()

		return c.Redirect("/panel/ozel-icerikler/" + ccid)
	}
}

func EditCustomContent(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Ccid := c.Params("ccid")
		Orm := utilities.Orm

		inputs := models.CustomContentsEdit{}
		c.BodyParser(&inputs)
		inputs.Ccid = Ccid

		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "İçerik adı zorunludur.",
			})
		}
		if inputs.UrlName == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "URL adı zorunludur.",
			})
		}
		if inputs.ContentType == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "İçerik tipi zorunludur.",
			})
		}

		// Check for duplicate URL name if changed
		if inputs.UrlName != inputs.OldUrlName {
			checkUrlName := Orm.Select([]string{"ccid"})
			checkUrlName.Table("custom_contents")
			checkUrlName.Where("url_name", "=", inputs.UrlName)
			checkUrlName.And("ccid", "!=", Ccid)
			checkUrlName.Finish()
			err = checkUrlName.Execute()
			if err != nil {
				log.Printf("%v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
			rows, err := checkUrlName.Rows()
			if err != nil {
				log.Printf("%v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
			if len(rows) > 0 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Bu URL adı zaten kullanılıyor.",
				})
			}
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		orderingChanged := (inputs.SortOrder != inputs.OldSortOrder) || (inputs.ContentType != inputs.OldContentType)
		if orderingChanged {
			fmt.Printf("sort_order: %v\n", inputs.SortOrder)
			fmt.Printf("old_sort_order: %v\n", inputs.OldSortOrder)
			fmt.Printf("content_type: %v\n", inputs.ContentType)
			fmt.Printf("old_content_type: %v\n", inputs.OldContentType)
			fmt.Printf("ccid: %v\n", inputs.Ccid)
			fmt.Printf("orderingChanged: %v\n", orderingChanged)
			ReorderCustomContents := Orm.SelectFunction("get_shift_for_update_for_custom_contents", inputs.SortOrder, inputs.OldSortOrder, inputs.Ccid, inputs.ContentType)
			ReorderCustomContents.Finish()
			err = ReorderCustomContents.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute update_custom_content_sort function: %v\n", err)
			}

			rows, err := ReorderCustomContents.Rows()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get update_custom_content_sort function rows: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			fmt.Printf("rows: %v\n", rows)

			if len(rows) != 0 {
				CcidsToChange := []any{}

				for _, row := range rows {
					CcidsToChange = append(CcidsToChange, row["our_ccid"])
				}

				Expression := ""

				if rows[0]["direction"] == "down" {
					Expression = "sort_order - 1"
				} else {
					Expression = "sort_order + 1"
				}

				UpdateSortings := Orm.Update()
				UpdateSortings.Table("custom_contents")
				UpdateSortings.SetExpr("sort_order", Expression)
				UpdateSortings.In("WHERE", "ccid", CcidsToChange)
				UpdateSortings.Finish()

				fmt.Printf("UpdateSortings: %v\n", UpdateSortings.Query)

				err = UpdateSortings.Execute()

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update custom content sortings: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				UpdateThisCustomContent := Orm.Update()
				UpdateThisCustomContent.Table("custom_contents")
				UpdateThisCustomContent.Set("sort_order", rows[0]["new_sort_order"])
				UpdateThisCustomContent.Set("content_type", inputs.ContentType)
				UpdateThisCustomContent.Where("ccid", "=", inputs.Ccid)
				UpdateThisCustomContent.Finish()
				err = UpdateThisCustomContent.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update this custom content sort order: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}
			} else {
				if inputs.ContentType != inputs.OldContentType {
					handleSorting := Orm.SelectFunction("get_shift_for_insert_for_custom_contents", inputs.SortOrder, inputs.Ccid, inputs.ContentType)
					handleSorting.Finish()

					err = handleSorting.Execute()

					if err != nil {
						Orm.Rollback()
						log.Printf("Cannot execute handle_button_sorting function: %v\n", err)
						return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
					}

					rows, err := handleSorting.Rows()
					if err != nil {
						Orm.Rollback()
						log.Printf("Cannot get handle_button_sorting function rows: %v\n", err)
						return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
					}

					if len(rows) == 0 {
						log.Printf("Cannot delete header button, issue is no rows returned : %v\n", len(rows) == 0)
						Orm.Rollback()
						return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
					}

					if rows[0]["our_ccid"] == nil {
						FuckingCleanup := Orm.Update()
						FuckingCleanup.Table("custom_contents")
						FuckingCleanup.Set("sort_order", rows[0]["new_sort_order"])
						FuckingCleanup.Where("ccid", "=", inputs.Ccid)
						FuckingCleanup.Finish()

						err = FuckingCleanup.Execute()

						if err != nil {
							Orm.Rollback()
							log.Printf("Cannot execute fucking cleanup: %v\n", err)
							return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
						}
					} else {
						FuckingCleanup := Orm.Update()
						FuckingCleanup.Table("custom_contents")
						FuckingCleanup.SetExpr("sort_order", "sort_order + 1")
						FuckingCleanup.Where("sort_order", ">=", inputs.SortOrder)
						FuckingCleanup.And("ccid", "!=", inputs.Ccid)
						FuckingCleanup.Finish()

						err = FuckingCleanup.Execute()

						if err != nil {
							Orm.Rollback()
							log.Printf("Cannot execute fucking cleanup: %v\n", err)
							return c.Redirect("/panel/ozel-icerik-ekle?error=internal_server_error")
						}
					}
				}
			}
		}

		updateCustomContent := Orm.Update()
		updateCustomContent.Table("custom_contents")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			updateCustomContent.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateCustomContent.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.ContentType != inputs.OldContentType {
			updateCustomContent.Set("content_type", inputs.ContentType)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateCustomContent.Set("description", inputs.Description)
			SomethingSet = true
		}

		if inputs.Route != inputs.OldRoute {
			updateCustomContent.Set("route", inputs.Route)
			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateCustomContent.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		// Update HTML content if provided
		if inputs.ContentHtml != inputs.OldContentHtml {
			updateCustomContent.Set("content_html", inputs.ContentHtml)
			SomethingSet = true
		}

		fmt.Printf("inputs.ContentCss: %s\n", inputs.ContentCss)
		fmt.Printf("inputs.OldContentCss: %s\n", inputs.OldContentCss)
		// Update CSS content if provided
		if inputs.ContentCss != inputs.OldContentCss {
			updateCustomContent.Set("content_css", inputs.ContentCss)
			SomethingSet = true
		}

		fmt.Printf("inputs.ContentJavascript: %s\n", inputs.ContentJavascript)
		fmt.Printf("inputs.OldContentJavascript: %s\n", inputs.OldContentJavascript)
		// Update JavaScript content if provided
		if inputs.ContentJavascript != inputs.OldContentJavascript {
			updateCustomContent.Set("content_javascript", inputs.ContentJavascript)
			SomethingSet = true
		}

		if SomethingSet {
			updateCustomContent.Set("updated_at", "NOW()")
			updateCustomContent.Where("ccid", "=", Ccid)
			updateCustomContent.Finish()
			err = updateCustomContent.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update custom content: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Özel içerik başarıyla güncellendi.",
		})
	}
}

func DeleteCustomContent(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		CustomContentId := c.Params("ccid")
		if CustomContentId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Custom content ID is required",
			})
		}

		Orm := utilities.Orm

		// Check if custom content exists
		GetCustomContent := Orm.Select([]string{"ccid", "sort_order", "content_type"})
		GetCustomContent.Table("custom_contents")
		GetCustomContent.Where("ccid", "=", CustomContentId)
		GetCustomContent.Finish()
		err = GetCustomContent.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		customContentRows, err := GetCustomContent.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(customContentRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Custom content not found",
			})
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete the custom content
		DeleteCustomContent := Orm.Delete()
		DeleteCustomContent.Table("custom_contents")
		DeleteCustomContent.Where("ccid", "=", CustomContentId)
		DeleteCustomContent.Finish()
		err = DeleteCustomContent.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteCustomContent.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Özel içerik bulunamadı.",
			})
		}

		handleSorting := Orm.SelectFunction("get_shift_for_delete_for_custom_contents", customContentRows[0]["sort_order"], CustomContentId, customContentRows[0]["content_type"])
		handleSorting.Finish()
		err = handleSorting.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot execute handle_button_sorting function: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := handleSorting.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get handle_button_sorting function rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if len(rows) != 0 {
			UpdateRows := Orm.Update()
			UpdateRows.Table("custom_contents")
			UpdateRows.SetExpr("sort_order", "sort_order - 1")

			Ins := []any{}
			for _, row := range rows {
				if row["old_content_type"] != customContentRows[0]["content_type"] {
					continue
				}

				Ins = append(Ins, lib.String(row["our_ccid"]))
			}
			UpdateRows.In("WHERE", "ccid", Ins)
			UpdateRows.And("content_type", "=", customContentRows[0]["content_type"])
			UpdateRows.Finish()

			err = UpdateRows.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update custom content sortings: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Özel içerik başarıyla silindi.",
		})
	}
}

func AddContactRequest(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		inputs := models.ContactRequests{}
		c.BodyParser(&inputs)

		// Validate required fields
		if inputs.FirstName == "" || inputs.Email == "" || inputs.Message == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Ad, e-posta ve mesaj alanları zorunludur.",
			})
		}

		// Validate email format
		emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		matched, _ := regexp.MatchString(emailRegex, inputs.Email)
		if !matched {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Geçerli bir e-posta adresi girin.",
			})
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err := GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size", "o.smtp_host", "o.smtp_port",
			"o.smtp_username", "o.smtp_password", "o.primary_color", "o.secondary_color", "o.google_recaptcha_site_key", "o.google_recaptcha_secret_key",
			"o.site_name", "o.site_description", "o.contact_email", "o.contact_phone", "o.facebook_url",
			"o.twitter_url", "o.instagram_url", "o.linkedin_url", "m.file_path as logo_path"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		CheckIfItsRepeating := Orm.Count("contact_requests")
		CheckIfItsRepeating.Where("email", "=", inputs.Email)
		CheckIfItsRepeating.And("subject", "=", inputs.Subject)
		CheckIfItsRepeating.AndExpr("created_at", ">=", "NOW() - INTERVAL '1 month'")
		CheckIfItsRepeating.Finish()

		err = CheckIfItsRepeating.Execute()
		if err != nil {
			log.Printf("Cannot check if its repeating: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if CheckIfItsRepeating.Length() > 0 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Bu e-posta adresi ile aynı konuda zaten bir mesaj gönderilmiştir. Lütfen daha sonra tekrar deneyin.",
			})
		}

		if GetOptions.Options.RecaptchaSiteKey != "" && GetOptions.Options.RecaptchaSecretKey != "" {
			if !lib.VerifyRecaptcha(inputs.RecaptchaToken, GetOptions.Options.RecaptchaSecretKey) {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "reCAPTCHA doğrulama hatası. Lütfen tekrar deneyin.",
				})
			}
		}

		inputs.Status = "yeni"
		inputs.Priority = "normal"
		inputs.Source = "website"
		inputs.IpAddress = c.IP()
		inputs.UserAgent = c.Get("User-Agent")
		inputs.CreatedAt = time.Now()
		inputs.UpdatedAt = time.Now()

		columns := []string{"first_name", "last_name", "email", "message", "ip_address", "user_agent"}
		values := []interface{}{inputs.FirstName, inputs.LastName, inputs.Email, inputs.Message, inputs.IpAddress, inputs.UserAgent}

		if inputs.Phone != "" {
			columns = append(columns, "phone")
			values = append(values, inputs.Phone)
		}
		if inputs.Subject != "" {
			columns = append(columns, "subject")
			values = append(values, inputs.Subject)
		}

		InsertContactRequest := Orm.Insert(columns, values)
		InsertContactRequest.Table("contact_requests")
		InsertContactRequest.Returning("crid")
		InsertContactRequest.Finish()
		err = InsertContactRequest.Execute()
		if err != nil {
			log.Printf("Cannot insert contact request: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		crid, err := InsertContactRequest.LastInsertId()
		if err != nil {
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if crid == "" {
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if GetOptions.Options.SMTPHost != "" && GetOptions.Options.SMTPPort != 0 && GetOptions.Options.SMTPUsername != "" && GetOptions.Options.SMTPPassword != "" && inputs.Email != "" {
			GetLogo := ""

			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				log.Printf("ROOT_DIRECTORY not set")
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if (*GetOptions.Medias)[0].FilePath != "" {
				GetLogo = filepath.Join(RootDir, "static", (*GetOptions.Medias)[0].FilePath)
			} else {
				GetLogo = filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")
			}

			LogoName := filepath.Base(GetLogo)

			Html := ""
			// Create professional HTML email template
			{
				Html = `<!DOCTYPE html>
				<html lang="tr">
				<head>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<title>İş Başvurunuz Alındı</title>
					<style>
						body {
							margin: 0;
							padding: 0;
							font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
							background-color: #f4f7fa;
							color: #333333;
						}
						.email-container {
							max-width: 600px;
							margin: 40px auto;
							background-color: #ffffff;
							border-radius: 12px;
							overflow: hidden;
							box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
						}
						.header {
							padding: 40px 30px;
							text-align: center;
							color: #ffffff;
						}
						.header img {
							max-width: 180px;
							height: auto;
							margin-bottom: 20px;
							filter: brightness(0) invert(1);
						}
						.header h1 {
							margin: 0;
							font-size: 28px;
							font-weight: 600;
							letter-spacing: -0.5px;
							color: #252525 !important;
						}
						.content {
							padding: 40px 30px;
						}
						.greeting {
							font-size: 18px;
							color: #283b6a;
							margin-bottom: 20px;
							font-weight: 600;
						}
						.message {
							font-size: 16px;
							line-height: 1.8;
							color: #555555;
							margin-bottom: 25px;
						}
						.info-box {
							background-color: #f8f9fc;
							border-left: 4px solid ` + GetOptions.Options.AccentColor + `;
							padding: 20px 25px;
							margin: 25px 0;
							border-radius: 6px;
						}
						.info-box h3 {
							margin: 0 0 15px 0;
							color: #283b6a;
							font-size: 18px;
							font-weight: 600;
						}
						.info-row {
							display: flex;
							justify-content: space-between;
							padding: 10px 0;
							border-bottom: 1px solid #e1e8ed;
							flex-wrap: wrap;
						}
						.info-row:last-child {
							border-bottom: none;
						}
						.info-label {
							font-weight: 600;
							color: #283b6a;
							margin-right: 15px;
						}
						.info-value {
							color: #555555;
							text-align: right;
							flex: 1;
						}
						.status-badge {
							display: inline-block;
							background-color: #4CAF50;
							color: #ffffff;
							padding: 10px 20px;
							border-radius: 25px;
							font-size: 14px;
							font-weight: 600;
							margin: 20px 0;
						}
						.next-steps {
							background-color: #fff9e6;
							border-radius: 8px;
							padding: 20px;
							margin: 25px 0;
						}
						.next-steps h3 {
							color: #283b6a;
							margin-top: 0;
							font-size: 18px;
						}
						.next-steps ul {
							margin: 10px 0;
							padding-left: 20px;
						}
						.next-steps li {
							color: #555555;
							margin: 8px 0;
							line-height: 1.6;
						}
						.contact-info {
							background-color: #f0f4ff;
							border-radius: 8px;
							padding: 20px;
							margin: 25px 0;
							text-align: center;
						}
						.contact-info h3 {
							color: #283b6a;
							margin-top: 0;
							font-size: 18px;
						}
						.contact-info p {
							margin: 5px 0;
							color: #555555;
						}
						.contact-info a {
							color: ` + GetOptions.Options.PrimaryColor + `;
							text-decoration: none;
							font-weight: 600;
						}
						.footer {
							background-color: #f8f9fc;
							padding: 30px;
							text-align: center;
							font-size: 13px;
							color: #888888;
							border-top: 1px solid #e1e8ed;
						}
						.footer p {
							margin: 5px 0;
						}
						.footer a {
							color: ` + GetOptions.Options.PrimaryColor + `;
							text-decoration: none;
						}
						.social-links {
							margin: 20px 0 10px;
						}
						.social-links a {
							display: inline-block;
							margin: 0 8px;
							color: #888888;
							text-decoration: none;
							font-size: 12px;
						}
						@media only screen and (max-width: 600px) {
							.email-container {
								margin: 0;
								border-radius: 0;
							}
							.header, .content, .footer {
								padding: 25px 20px;
							}
							.header h1 {
								font-size: 24px;
							}
							.info-row {
								flex-direction: column;
							}
							.info-value {
								text-align: left;
								margin-top: 5px;
							}
						}
					</style>
				</head>
				<body>
					<div class="email-container">
						<div class="header">
							<img src="cid:` + LogoName + `" alt="` + GetOptions.Options.SiteName + `" />
							<h1>İletişim Talebiniz Alındı</h1>
						</div>
						
						<div class="content">
							<div class="greeting">
								Sayın ` + inputs.FirstName + ` ` + inputs.LastName + `,
							</div>
							
							<div class="message">
								<strong>` + GetOptions.Options.SiteName + `</strong>'ye gösterdiğiniz ilgi için teşekkür ederiz. 
								İletişim talebiniz başarıyla tarafımıza ulaşmıştır ve en kısa sürede size dönüş yapacağız.
							</div>
							
							<div class="status-badge">
								✓ Talep Alındı
							</div>
							
							<div class="info-box">
								<h3>📋 İletişim Bilgileriniz</h3>
								<div class="info-row">
									<span class="info-label">Ad Soyad:</span>
									<span class="info-value">` + inputs.FirstName + ` ` + inputs.LastName + `</span>
								</div>
								<div class="info-row">
									<span class="info-label">E-posta:</span>
									<span class="info-value">` + inputs.Email + `</span>
								</div>`

				if inputs.Phone != "" {
					Html += `
								<div class="info-row">
									<span class="info-label">Telefon:</span>
									<span class="info-value">` + inputs.Phone + `</span>
								</div>`
				}

				Html += `
								<div class="info-row">
									<span class="info-label">Konu:</span>
									<span class="info-value">` + inputs.Subject + `</span>
								</div>
							</div>
							
							<div class="contact-info">
								<h3>📞 Bize Ulaşın</h3>
								<p><strong>Telefon:</strong> <a href="tel:` + GetOptions.Options.ContactPhone + `">` + GetOptions.Options.ContactPhone + `</a></p>
								<p><strong>E-posta:</strong> <a href="mailto:` + GetOptions.Options.ContactEmail + `">` + GetOptions.Options.ContactEmail + `</a></p>
							</div>
							
							<div class="message" style="margin-top: 30px; font-size: 15px; color: #666;">
								Bu e-posta otomatik olarak oluşturulmuştur. Lütfen bu e-postaya cevap vermeyiniz. 
								Sorularınız için yukarıdaki iletişim bilgilerini kullanabilirsiniz.
							</div>
						</div>
						
						<div class="footer">
							<p><strong>` + GetOptions.Options.SiteName + `</strong></p>
							<p>` + GetOptions.Options.SiteDescription + `</p>
							
							<div class="social-links">`

				if GetOptions.Options.FacebookUrl != "" && GetOptions.Options.FacebookUrl != "#" {
					Html += `
								<a href="` + GetOptions.Options.FacebookUrl + `">Facebook</a> |`
				}
				if GetOptions.Options.TwitterUrl != "" && GetOptions.Options.TwitterUrl != "#" {
					Html += `
								<a href="` + GetOptions.Options.TwitterUrl + `">Twitter</a> |`
				}
				if GetOptions.Options.InstagramUrl != "" && GetOptions.Options.InstagramUrl != "#" {
					Html += `
								<a href="` + GetOptions.Options.InstagramUrl + `">Instagram</a> |`
				}
				if GetOptions.Options.LinkedinUrl != "" && GetOptions.Options.LinkedinUrl != "#" {
					Html += `
								<a href="` + GetOptions.Options.LinkedinUrl + `">LinkedIn</a>`
				}

				Html += `
							</div>
							
							<p style="margin-top: 20px;">&copy; 2025 ` + GetOptions.Options.SiteName + `. Tüm hakları saklıdır.</p>
						</div>
					</div>
				</body>
				</html>`
			}

			CreateEmailInfos := models.EmailInfos{
				From:        GetOptions.Options.SiteName,
				To:          []string{inputs.Email},
				Username:    GetOptions.Options.SMTPUsername,
				Password:    GetOptions.Options.SMTPPassword,
				Host:        GetOptions.Options.SMTPHost,
				Port:        lib.Int64(GetOptions.Options.SMTPPort),
				Subject:     "İletişim Talebiniz Alındı - " + GetOptions.Options.SiteName,
				PlainText:   "Sayın " + inputs.FirstName + " " + inputs.LastName + ", iletişim talebiniz başarıyla alınmıştır. En kısa sürede sizinle iletişime geçeceğiz.",
				Body:        Html,
				Attachments: []string{GetLogo},
			}

			err = lib.SendEmail(&CreateEmailInfos)

			if err != nil {
				log.Printf("Cannot send email: %v\n", err)
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Mesajınız başarıyla gönderildi. En kısa sürede size dönüş yapacağız.",
			"crid":    crid,
		})
	}
}

func DeleteContactRequest(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		ContactRequestId := c.Params("crid")
		if ContactRequestId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Contact request ID is required",
			})
		}

		Orm := utilities.Orm

		// Check if contact request exists
		GetContactRequest := Orm.Select([]string{"crid"})
		GetContactRequest.Table("contact_requests")
		GetContactRequest.Where("crid", "=", ContactRequestId)
		GetContactRequest.Finish()
		err = GetContactRequest.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		contactRequestRows, err := GetContactRequest.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(contactRequestRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Contact request not found",
			})
		}

		// Delete the contact request
		DeleteContactRequest := Orm.Delete()
		DeleteContactRequest.Table("contact_requests")
		DeleteContactRequest.Where("crid", "=", ContactRequestId)
		DeleteContactRequest.Finish()
		err = DeleteContactRequest.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteContactRequest.RowsAffected()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "İletişim talebi bulunamadı.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "İletişim talebi başarıyla silindi.",
		})
	}
}

func SetAsReadAContactRequest(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		ContactRequestId := c.Params("crid")
		if ContactRequestId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Contact request ID is required",
			})
		}

		// Parse request body to get is_read value
		var requestData struct {
			IsRead bool `json:"is_read"`
		}

		if err := c.BodyParser(&requestData); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		Orm := utilities.Orm

		// Update the is_read status
		UpdateContactRequest := Orm.Update()
		UpdateContactRequest.Table("contact_requests")
		UpdateContactRequest.SetExpr("is_read", "NOT is_read")
		UpdateContactRequest.Set("updated_at", "NOW()")
		UpdateContactRequest.Where("crid", "=", ContactRequestId)
		UpdateContactRequest.Finish()
		err = UpdateContactRequest.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := UpdateContactRequest.RowsAffected()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "İletişim talebi bulunamadı.",
			})
		}

		statusMessage := "okunmamış olarak işaretlendi"
		if requestData.IsRead {
			statusMessage = "okundu olarak işaretlendi"
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": fmt.Sprintf("İletişim talebi %s.", statusMessage),
		})
	}
}

func RespondToContactRequest(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			log.Printf("%v\n", err)
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		// Get contact request ID from route parameter
		ContactRequestId := c.Params("crid")
		if ContactRequestId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Contact request ID is required",
			})
		}

		// Parse request body
		var inputs struct {
			Title         string `json:"title"`
			ResponderName string `json:"responder_name"`
			ResponseText  string `json:"response_text"`
		}

		if err := c.BodyParser(&inputs); err != nil {
			log.Printf("Cannot parse request body: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request data",
			})
		}

		// Validate required fields
		if inputs.Title == "" || inputs.ResponseText == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Title and response text are required",
			})
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size", "o.smtp_host", "o.smtp_port",
			"o.smtp_username", "o.smtp_password", "o.primary_color", "o.secondary_color",
			"o.site_name", "o.site_description", "o.contact_email", "o.contact_phone", "o.facebook_url",
			"o.twitter_url", "o.instagram_url", "o.linkedin_url", "m.file_path as logo_path"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if GetOptions.Options.SMTPHost == "" || GetOptions.Options.SMTPPort == 0 || GetOptions.Options.SMTPUsername == "" || GetOptions.Options.SMTPPassword == "" || GetOptions.Options.SiteName == "" {
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "E-posta bilgileriniz girilmemişse e-posta gönderemezsiniz.",
			})
		}

		// 1. Check if contact request exists
		CheckContactRequest := Orm.Select([]string{"crid", "first_name", "last_name", "email"})
		CheckContactRequest.Table("contact_requests")
		CheckContactRequest.Where("crid", "=", ContactRequestId)
		CheckContactRequest.Finish()

		err = CheckContactRequest.Execute()
		if err != nil {
			log.Printf("Cannot check contact request: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := CheckContactRequest.Rows()
		if err != nil {
			log.Printf("Cannot get contact request rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(rows) == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Contact request not found",
			})
		}

		row := rows[0]
		ContactRequestData := struct {
			Crid      string
			FirstName string
			LastName  string
			Email     string
		}{
			Crid:      lib.String(row["crid"]),
			FirstName: lib.String(row["first_name"]),
			LastName:  lib.String(row["last_name"]),
			Email:     lib.String(row["email"]),
		}

		GetLogo := ""

		RootDir := os.Getenv("ROOT_DIRECTORY")
		if RootDir == "" {
			log.Printf("ROOT_DIRECTORY not set")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if (*GetOptions.Medias)[0].FilePath != "" {
			GetLogo = filepath.Join(RootDir, "static", (*GetOptions.Medias)[0].FilePath)
		} else {
			GetLogo = filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")
		}

		LogoName := filepath.Base(GetLogo)

		// 3. Send email with exact same steps
		Html := ""
		// Create professional HTML email template for job application response
		{
			Html = `<!DOCTYPE html>
<html lang="tr">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
						<title>` + inputs.Title + `</title>
	<style>
		body {
			margin: 0;
			padding: 0;
			font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
			background-color: #f4f7fa;
			color: #333333;
		}
		.email-container {
			max-width: 600px;
			margin: 40px auto;
			background-color: #ffffff;
			border-radius: 12px;
			overflow: hidden;
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
		}
		.header {
			padding: 40px 30px;
			text-align: center;
			color: #ffffff;
		}
		.header img {
			max-width: 180px;
			height: auto;
			margin-bottom: 20px;
			filter: brightness(0) invert(1);
		}
		.header h1 {
			margin: 0;
			font-size: 28px;
			font-weight: 600;
			letter-spacing: -0.5px;
			color: #252525 !important;
		}
		.content {
			padding: 40px 30px;
		}
		.greeting {
			font-size: 18px;
			color: #283b6a;
			margin-bottom: 20px;
			font-weight: 600;
		}
		.message {
			font-size: 16px;
			line-height: 1.8;
			color: #555555;
			margin-bottom: 25px;
		}
							.response-box {
			background-color: #f8f9fc;
								border-left: 4px solid ` + GetOptions.Options.PrimaryColor + `;
			padding: 20px 25px;
			margin: 25px 0;
			border-radius: 6px;
		}
							.response-box h3 {
			margin: 0 0 15px 0;
			color: #283b6a;
			font-size: 18px;
			font-weight: 600;
		}
							.response-text {
								background-color: #ffffff;
								padding: 20px;
								border-radius: 6px;
								border: 1px solid #e1e8ed;
								line-height: 1.6;
								color: #333333;
							}
							.responder-info {
								background-color: #f0f4ff;
			border-radius: 8px;
			padding: 20px;
			margin: 25px 0;
								text-align: center;
		}
							.responder-info h3 {
			color: #283b6a;
			margin-top: 0;
			font-size: 18px;
		}
							.responder-info p {
								margin: 5px 0;
			color: #555555;
		}
		.contact-info {
			background-color: #f0f4ff;
			border-radius: 8px;
			padding: 20px;
			margin: 25px 0;
			text-align: center;
		}
		.contact-info h3 {
			color: #283b6a;
			margin-top: 0;
			font-size: 18px;
		}
		.contact-info p {
			margin: 5px 0;
			color: #555555;
		}
		.contact-info a {
			color: ` + GetOptions.Options.PrimaryColor + `;
			text-decoration: none;
			font-weight: 600;
		}
		.footer {
			background-color: #f8f9fc;
			padding: 30px;
			text-align: center;
			font-size: 13px;
			color: #888888;
			border-top: 1px solid #e1e8ed;
		}
		.footer p {
			margin: 5px 0;
		}
		.footer a {
			color: ` + GetOptions.Options.PrimaryColor + `;
			text-decoration: none;
		}
		.social-links {
			margin: 20px 0 10px;
		}
		.social-links a {
			display: inline-block;
			margin: 0 8px;
			color: #888888;
			text-decoration: none;
			font-size: 12px;
		}
		@media only screen and (max-width: 600px) {
			.email-container {
				margin: 0;
				border-radius: 0;
			}
			.header, .content, .footer {
				padding: 25px 20px;
			}
			.header h1 {
				font-size: 24px;
			}
		}
	</style>
</head>
<body>
	<div class="email-container">
		<div class="header">
			<img src="cid:` + LogoName + `" alt="` + GetOptions.Options.SiteName + `" />
								<h1>` + inputs.Title + `</h1>
		</div>
		
		<div class="content">
			<div class="greeting">
									Sayın ` + ContactRequestData.FirstName + ` ` + ContactRequestData.LastName + `,
			</div>
			
			<div class="message">
									<strong>` + GetOptions.Options.SiteName + `</strong> İletişim ekibimiz tarafından size aşağıdaki gibi cevap verilmiştir.
			</div>
			
								<div class="response-box">
									<h3>📧 Cevabımız</h3>
									<div class="response-text">
										` + inputs.ResponseText + `
			</div>
				</div>
								
								<div class="responder-info">
									<h3>👤 Cevap Veren</h3>
									<p><strong>` + inputs.ResponderName + `</strong></p>
									<p>` + GetOptions.Options.SiteName + ` İletişim Ekibi</p>
			</div>
			
			<div class="contact-info">
				<h3>📞 Bize Ulaşın</h3>
				<p><strong>Telefon:</strong> <a href="tel:` + GetOptions.Options.ContactPhone + `">` + GetOptions.Options.ContactPhone + `</a></p>
				<p><strong>E-posta:</strong> <a href="mailto:` + GetOptions.Options.ContactEmail + `">` + GetOptions.Options.ContactEmail + `</a></p>
			</div>
			
			<div class="message" style="margin-top: 30px; font-size: 15px; color: #666;">
									Bu e-posta ` + GetOptions.Options.SiteName + ` İletişim ekibi tarafından gönderilmiştir. 
				Sorularınız için yukarıdaki iletişim bilgilerini kullanabilirsiniz.
			</div>
		</div>
		
		<div class="footer">
			<p><strong>` + GetOptions.Options.SiteName + `</strong></p>
			<p>` + GetOptions.Options.SiteDescription + `</p>
			
			<div class="social-links">`

			if GetOptions.Options.FacebookUrl != "" && GetOptions.Options.FacebookUrl != "#" {
				Html += `
				<a href="` + GetOptions.Options.FacebookUrl + `">Facebook</a> |`
			}
			if GetOptions.Options.TwitterUrl != "" && GetOptions.Options.TwitterUrl != "#" {
				Html += `
				<a href="` + GetOptions.Options.TwitterUrl + `">Twitter</a> |`
			}
			if GetOptions.Options.InstagramUrl != "" && GetOptions.Options.InstagramUrl != "#" {
				Html += `
				<a href="` + GetOptions.Options.InstagramUrl + `">Instagram</a> |`
			}
			if GetOptions.Options.LinkedinUrl != "" && GetOptions.Options.LinkedinUrl != "#" {
				Html += `
				<a href="` + GetOptions.Options.LinkedinUrl + `">LinkedIn</a>`
			}

			Html += `
			</div>
			
								<p style="margin-top: 20px;">&copy; 2025 ` + GetOptions.Options.SiteName + `. Tüm hakları saklıdır.</p>
		</div>
	</div>
</body>
</html>`
		}

		CreateEmailInfos := models.EmailInfos{
			From:        GetOptions.Options.SiteName,
			To:          []string{ContactRequestData.Email},
			Username:    GetOptions.Options.SMTPUsername,
			Password:    GetOptions.Options.SMTPPassword,
			Host:        GetOptions.Options.SMTPHost,
			Port:        lib.Int64(GetOptions.Options.SMTPPort),
			Subject:     inputs.Title + " - " + GetOptions.Options.SiteName,
			PlainText:   "Sayın " + ContactRequestData.FirstName + " " + ContactRequestData.LastName + ", " + inputs.ResponderName + " tarafından hazırlanan cevabımız: " + inputs.ResponseText,
			Body:        Html,
			Attachments: []string{GetLogo},
		}

		err = lib.SendEmail(&CreateEmailInfos)

		if err != nil {
			log.Printf("Cannot send email: %v\n", err)
		}

		// 4. Update the "is_replied" column
		UpdateContactRequest := Orm.Update()
		UpdateContactRequest.Table("contact_requests")
		UpdateContactRequest.Set("is_replied", true)
		UpdateContactRequest.Set("response_date", "NOW()")
		UpdateContactRequest.Set("updated_at", "NOW()")
		UpdateContactRequest.Where("crid", "=", ContactRequestId)
		UpdateContactRequest.Finish()

		err = UpdateContactRequest.Execute()
		if err != nil {
			log.Printf("Cannot update contact request: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Cevap başarıyla gönderildi.",
		})
	}
}

func AddJobApplication(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		inputs := models.JobApplications{}
		if err := c.BodyParser(&inputs); err != nil {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Geçersiz istek gövdesi",
			})
		}

		// Required validation
		if inputs.FirstName == "" || inputs.LastName == "" || inputs.Email == "" || inputs.City == "" || inputs.CoverLetter == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Ad, soyad, e-posta, şehir ve ön yazı zorunludur.",
			})
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err := GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size", "o.smtp_host", "o.smtp_port",
			"o.smtp_username", "o.smtp_password", "o.primary_color", "o.secondary_color", "o.google_recaptcha_site_key", "o.google_recaptcha_secret_key",
			"o.site_name", "o.site_description", "o.contact_email", "o.contact_phone", "o.facebook_url",
			"o.twitter_url", "o.instagram_url", "o.linkedin_url", "m.file_path as logo_path"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		CheckIfItsRepeating := Orm.Count("job_applications")
		CheckIfItsRepeating.Where("email", "=", inputs.Email)
		CheckIfItsRepeating.AndExpr("created_at", ">=", "NOW() - INTERVAL '1 month'")
		CheckIfItsRepeating.Finish()

		err = CheckIfItsRepeating.Execute()
		if err != nil {
			log.Printf("Cannot check if its repeating: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if CheckIfItsRepeating.Length() > 0 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Yakın zamanda zaten iş başvurusu iletmişsiniz. Lütfen başvurunuzu daha sonra tekrar deneyin.",
			})
		}

		if GetOptions.Options.RecaptchaSiteKey != "" && GetOptions.Options.RecaptchaSecretKey != "" {
			if !lib.VerifyRecaptcha(inputs.RecaptchaToken, GetOptions.Options.RecaptchaSecretKey) {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "reCAPTCHA doğrulama hatası. Lütfen tekrar deneyin.",
				})
			}
		}

		columns := []string{"first_name", "last_name", "email", "city", "cover_letter"}
		values := []interface{}{inputs.FirstName, inputs.LastName, inputs.Email, inputs.City, inputs.CoverLetter}

		// Optional fields
		if inputs.Phone != "" {
			columns = append(columns, "phone")
			values = append(values, inputs.Phone)
		}
		if inputs.TcKimlik != "" {
			columns = append(columns, "tc_kimlik")
			values = append(values, inputs.TcKimlik)
		}
		if !inputs.BirthDate.IsZero() {
			columns = append(columns, "birth_date")
			values = append(values, inputs.BirthDate)
		}
		if inputs.Gender != "" {
			columns = append(columns, "gender")
			values = append(values, inputs.Gender)
		}
		if inputs.Address != "" {
			columns = append(columns, "address")
			values = append(values, inputs.Address)
		}
		if inputs.EducationLevel != "" {
			columns = append(columns, "education_level")
			values = append(values, inputs.EducationLevel)
		}
		if inputs.University != "" {
			columns = append(columns, "university")
			values = append(values, inputs.University)
		}
		if inputs.Department != "" {
			columns = append(columns, "department")
			values = append(values, inputs.Department)
		}
		if inputs.GraduationYear > 0 {
			columns = append(columns, "graduation_year")
			values = append(values, inputs.GraduationYear)
		}
		if inputs.ExperienceYears > 0 {
			columns = append(columns, "experience_years")
			values = append(values, inputs.ExperienceYears)
		}
		if inputs.PositionApplied != "" {
			columns = append(columns, "position_applied")
			values = append(values, inputs.PositionApplied)
		}
		if inputs.DepartmentApplied != "" {
			columns = append(columns, "department_applied")
			values = append(values, inputs.DepartmentApplied)
		}
		if inputs.SalaryExpectation > 0 {
			columns = append(columns, "salary_expectation")
			values = append(values, inputs.SalaryExpectation)
		}
		if !inputs.AvailableStartDate.IsZero() {
			columns = append(columns, "available_start_date")
			values = append(values, inputs.AvailableStartDate)
		}
		if inputs.Languages != "" {
			columns = append(columns, "languages")
			values = append(values, inputs.Languages)
		}
		if inputs.Skills != "" {
			columns = append(columns, "skills")
			values = append(values, inputs.Skills)
		}
		if inputs.DiplomaFileMid > 0 {
			columns = append(columns, "diploma_file_mid")
			values = append(values, inputs.DiplomaFileMid)
		}
		if inputs.WorkReferences != "" {
			columns = append(columns, "work_references")
			values = append(values, inputs.WorkReferences)
		}

		insertReq := Orm.Insert(columns, values)
		insertReq.Table("job_applications")
		insertReq.Returning("jaid")
		insertReq.Finish()
		if err := insertReq.Execute(); err != nil {
			log.Printf("Cannot insert job application: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		lid, err := insertReq.LastInsertId()

		if err != nil {
			log.Printf("Cannot get last insert id for job application: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if lid == "" {
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		RootDir := os.Getenv("ROOT_DIRECTORY")
		if RootDir == "" {
			Orm.Rollback()
			log.Printf("ROOT_DIRECTORY not set")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		cvInput, err := c.FormFile("cv_file")
		if err == nil {
			// File size validation
			if cvInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "job-applications", lid)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + cvInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".pdf", ".doc", ".docx":
				break
			default:
				Orm.Rollback()
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=invalid_file_type")
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/job-applications/" + lid + "/" + UniqueFilePath.BaseName,
				FileSize: cvInput.Size,
				MimeType: cvInput.Header.Get("Content-Type"),
				FileType: "cv",
				Uid:      "",
				TargetId: lid,
			}

			optionals := models.MediaOptionals{
				AltText: "",
				Title:   "",
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			CvMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert CV media: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			// Update doctor with CV media ID
			updateDoctor := Orm.Update()
			updateDoctor.Table("job_applications")
			updateDoctor.Set("cv_file_mid", CvMediaMid)
			updateDoctor.Where("jaid", "=", lid)
			updateDoctor.Finish()
			err = updateDoctor.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor with CV: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			// Save CV file
			err = lib.SaveFileWithBuffering(estimatedPath, *cvInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save CV file: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}
		}

		if GetOptions.Options.SMTPHost != "" && GetOptions.Options.SMTPPort != 0 && GetOptions.Options.SMTPUsername != "" && GetOptions.Options.SMTPPassword != "" && inputs.Email != "" {
			GetLogo := ""

			if (*GetOptions.Medias)[0].FilePath != "" {
				GetLogo = filepath.Join(RootDir, "static", (*GetOptions.Medias)[0].FilePath)
			} else {
				GetLogo = filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")
			}

			LogoName := filepath.Base(GetLogo)

			Html := ""
			// Create professional HTML email template
			{
				Html = `<!DOCTYPE html>
<html lang="tr">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>İş Başvurunuz Alındı</title>
	<style>
		body {
			margin: 0;
			padding: 0;
			font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
			background-color: #f4f7fa;
			color: #333333;
		}
		.email-container {
			max-width: 600px;
			margin: 40px auto;
			background-color: #ffffff;
			border-radius: 12px;
			overflow: hidden;
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
		}
		.header {
			padding: 40px 30px;
			text-align: center;
			color: #ffffff;
		}
		.header img {
			max-width: 180px;
			height: auto;
			margin-bottom: 20px;
			filter: brightness(0) invert(1);
		}
		.header h1 {
			margin: 0;
			font-size: 28px;
			font-weight: 600;
			letter-spacing: -0.5px;
			color: #252525 !important;
		}
		.content {
			padding: 40px 30px;
		}
		.greeting {
			font-size: 18px;
			color: #283b6a;
			margin-bottom: 20px;
			font-weight: 600;
		}
		.message {
			font-size: 16px;
			line-height: 1.8;
			color: #555555;
			margin-bottom: 25px;
		}
		.info-box {
			background-color: #f8f9fc;
			border-left: 4px solid ` + GetOptions.Options.AccentColor + `;
			padding: 20px 25px;
			margin: 25px 0;
			border-radius: 6px;
		}
		.info-box h3 {
			margin: 0 0 15px 0;
			color: #283b6a;
			font-size: 18px;
			font-weight: 600;
		}
		.info-row {
			display: flex;
			justify-content: space-between;
			padding: 10px 0;
			border-bottom: 1px solid #e1e8ed;
			flex-wrap: wrap;
		}
		.info-row:last-child {
			border-bottom: none;
		}
		.info-label {
			font-weight: 600;
			color: #283b6a;
			margin-right: 15px;
		}
		.info-value {
			color: #555555;
			text-align: right;
			flex: 1;
		}
		.status-badge {
			display: inline-block;
			background-color: #4CAF50;
			color: #ffffff;
			padding: 10px 20px;
			border-radius: 25px;
			font-size: 14px;
			font-weight: 600;
			margin: 20px 0;
		}
		.next-steps {
			background-color: #fff9e6;
			border-radius: 8px;
			padding: 20px;
			margin: 25px 0;
		}
		.next-steps h3 {
			color: #283b6a;
			margin-top: 0;
			font-size: 18px;
		}
		.next-steps ul {
			margin: 10px 0;
			padding-left: 20px;
		}
		.next-steps li {
			color: #555555;
			margin: 8px 0;
			line-height: 1.6;
		}
		.contact-info {
			background-color: #f0f4ff;
			border-radius: 8px;
			padding: 20px;
			margin: 25px 0;
			text-align: center;
		}
		.contact-info h3 {
			color: #283b6a;
			margin-top: 0;
			font-size: 18px;
		}
		.contact-info p {
			margin: 5px 0;
			color: #555555;
		}
		.contact-info a {
			color: ` + GetOptions.Options.PrimaryColor + `;
			text-decoration: none;
			font-weight: 600;
		}
		.footer {
			background-color: #f8f9fc;
			padding: 30px;
			text-align: center;
			font-size: 13px;
			color: #888888;
			border-top: 1px solid #e1e8ed;
		}
		.footer p {
			margin: 5px 0;
		}
		.footer a {
			color: ` + GetOptions.Options.PrimaryColor + `;
			text-decoration: none;
		}
		.social-links {
			margin: 20px 0 10px;
		}
		.social-links a {
			display: inline-block;
			margin: 0 8px;
			color: #888888;
			text-decoration: none;
			font-size: 12px;
		}
		@media only screen and (max-width: 600px) {
			.email-container {
				margin: 0;
				border-radius: 0;
			}
			.header, .content, .footer {
				padding: 25px 20px;
			}
			.header h1 {
				font-size: 24px;
			}
			.info-row {
				flex-direction: column;
			}
			.info-value {
				text-align: left;
				margin-top: 5px;
			}
		}
	</style>
</head>
<body>
	<div class="email-container">
		<div class="header">
			<img src="cid:` + LogoName + `" alt="` + GetOptions.Options.SiteName + `" />
			<h1>İş Başvurunuz Alındı</h1>
		</div>
		
		<div class="content">
			<div class="greeting">
				Sayın ` + inputs.FirstName + ` ` + inputs.LastName + `,
			</div>
			
			<div class="message">
				<strong>` + GetOptions.Options.SiteName + `</strong> ailesine gösterdiğiniz ilgi için teşekkür ederiz. 
				İş başvurunuz başarıyla tarafımıza ulaşmıştır ve insan kaynakları departmanımız tarafından 
				titizlikle değerlendirilecektir.
			</div>
			
			<div class="status-badge">
				✓ Başvuru Alındı
			</div>
			
			<div class="info-box">
				<h3>📋 Başvuru Bilgileriniz</h3>
				<div class="info-row">
					<span class="info-label">Ad Soyad:</span>
					<span class="info-value">` + inputs.FirstName + ` ` + inputs.LastName + `</span>
				</div>
				<div class="info-row">
					<span class="info-label">E-posta:</span>
					<span class="info-value">` + inputs.Email + `</span>
				</div>`

				if inputs.Phone != "" {
					Html += `
				<div class="info-row">
					<span class="info-label">Telefon:</span>
					<span class="info-value">` + inputs.Phone + `</span>
				</div>`
				}

				Html += `
				<div class="info-row">
					<span class="info-label">Şehir:</span>
					<span class="info-value">` + inputs.City + `</span>
				</div>`

				if inputs.PositionApplied != "" {
					Html += `
				<div class="info-row">
					<span class="info-label">Başvurulan Pozisyon:</span>
					<span class="info-value">` + inputs.PositionApplied + `</span>
				</div>`
				}

				if inputs.University != "" {
					Html += `
				<div class="info-row">
					<span class="info-label">Üniversite:</span>
					<span class="info-value">` + inputs.University + `</span>
				</div>`
				}

				if inputs.Languages != "" {
					Html += `
				<div class="info-row">
					<span class="info-label">Diller:</span>
					<span class="info-value">` + inputs.Languages + `</span>
				</div>`
				}

				Html += `
			</div>
			
			<div class="next-steps">
				<h3>🎯 Sonraki Adımlar</h3>
				<ul>
					<li>Başvurunuz insan kaynakları departmanımız tarafından değerlendirilecektir</li>
					<li>Uygun görülmeniz halinde tarafınıza dönüş yapılacaktır</li>
					<li>Mülakat süreciyle ilgili detaylı bilgi e-posta veya telefon ile paylaşılacaktır</li>
				</ul>
			</div>
			
			<div class="contact-info">
				<h3>📞 İletişim</h3>
				<p><strong>Telefon:</strong> <a href="tel:` + GetOptions.Options.ContactPhone + `">` + GetOptions.Options.ContactPhone + `</a></p>
				<p><strong>E-posta:</strong> <a href="mailto:` + GetOptions.Options.ContactEmail + `">` + GetOptions.Options.ContactEmail + `</a></p>
			</div>
			
			<div class="message" style="margin-top: 30px; font-size: 15px; color: #666;">
				Bu e-posta otomatik olarak oluşturulmuştur. Lütfen bu e-postaya cevap vermeyiniz. 
				Sorularınız için yukarıdaki iletişim bilgilerini kullanabilirsiniz.
			</div>
		</div>
		
		<div class="footer">
			<p><strong>` + GetOptions.Options.SiteName + `</strong></p>
			<p>` + GetOptions.Options.SiteDescription + `</p>
			
			<div class="social-links">`

				if GetOptions.Options.FacebookUrl != "" && GetOptions.Options.FacebookUrl != "#" {
					Html += `
				<a href="` + GetOptions.Options.FacebookUrl + `">Facebook</a> |`
				}
				if GetOptions.Options.TwitterUrl != "" && GetOptions.Options.TwitterUrl != "#" {
					Html += `
				<a href="` + GetOptions.Options.TwitterUrl + `">Twitter</a> |`
				}
				if GetOptions.Options.InstagramUrl != "" && GetOptions.Options.InstagramUrl != "#" {
					Html += `
				<a href="` + GetOptions.Options.InstagramUrl + `">Instagram</a> |`
				}
				if GetOptions.Options.LinkedinUrl != "" && GetOptions.Options.LinkedinUrl != "#" {
					Html += `
				<a href="` + GetOptions.Options.LinkedinUrl + `">LinkedIn</a>`
				}

				Html += `
			</div>
			
									<p style="margin-top: 20px;">&copy; 2025 ` + GetOptions.Options.SiteName + `. Tüm hakları saklıdır.</p>
		</div>
	</div>
</body>
</html>`
			}

			CreateEmailInfos := models.EmailInfos{
				From:        GetOptions.Options.SiteName,
				To:          []string{inputs.Email},
				Username:    GetOptions.Options.SMTPUsername,
				Password:    GetOptions.Options.SMTPPassword,
				Host:        GetOptions.Options.SMTPHost,
				Port:        lib.Int64(GetOptions.Options.SMTPPort),
				Subject:     "İş Başvurunuz Alındı - " + GetOptions.Options.SiteName,
				PlainText:   "Sayın " + inputs.FirstName + " " + inputs.LastName + ", iş başvurunuz başarıyla alınmıştır. En kısa sürede sizinle iletişime geçeceğiz.",
				Body:        Html,
				Attachments: []string{GetLogo},
			}

			err = lib.SendEmail(&CreateEmailInfos)

			if err != nil {
				log.Printf("Cannot send email: %v\n", err)
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "İş başvurusu başarıyla gönderildi",
			"jaid":    lid,
		})
	}
}

func DeleteJobApplication(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			log.Printf("%v\n", err)
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		// Get job application ID from route parameter
		JobApplicationId := c.Params("jaid")
		if JobApplicationId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Job application ID is required",
			})
		}

		Orm := utilities.Orm

		// Check if job application exists
		CheckJobApplication := Orm.Select([]string{"ja.jaid", "m.file_path as cv_file_path", "m2.file_path as diploma_file_path"})
		CheckJobApplication.Table("job_applications ja")
		CheckJobApplication.LeftJoin("medias m", "ja.cv_file_mid", "=", "m.mid")
		CheckJobApplication.LeftJoin("medias m2", "ja.diploma_file_mid", "=", "m2.mid")
		CheckJobApplication.Where("ja.jaid", "=", JobApplicationId)
		CheckJobApplication.Finish()

		err = CheckJobApplication.Execute()
		if err != nil {
			log.Printf("Cannot check job application: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := CheckJobApplication.Rows()
		if err != nil {
			log.Printf("Cannot get job application rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(rows) == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Job application not found",
			})
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		fmt.Printf("cv_file_path: %v\n", rows[0]["cv_file_path"])

		if rows[0]["cv_file_path"] != nil && rows[0]["cv_file_path"] != "" {
			DeleteMedia := Orm.Delete()
			DeleteMedia.Table("medias")
			DeleteMedia.Where("file_path", "=", lib.String(rows[0]["cv_file_path"]))
			DeleteMedia.Finish()
			err = DeleteMedia.Execute()

			if err != nil {
				log.Printf("Cannot delete media record: %v\n", err)
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ra, err := DeleteMedia.RowsAffected()
			if err != nil {
				log.Printf("Cannot get rows affected: %v\n", err)
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "CV file not found",
				})
			}

			filePath := lib.String(rows[0]["cv_file_path"])
			if filePath != "" {
				// Delete physical file
				fullPath := "static/" + filePath
				err = lib.DeleteFile(fullPath)
				if err != nil {
					log.Printf("Error deleting file %s: %v\n", fullPath, err)
				}
			}
		}

		fmt.Printf("diploma_file_path: %v\n", rows[0]["diploma_file_path"])
		if rows[0]["diploma_file_path"] != nil && rows[0]["diploma_file_path"] != "" {
			DeleteMedia := Orm.Delete()
			DeleteMedia.Table("medias")
			DeleteMedia.Where("file_path", "=", lib.String(rows[0]["diploma_file_path"]))
			DeleteMedia.Finish()
			err = DeleteMedia.Execute()

			if err != nil {
				log.Printf("Cannot delete media record: %v\n", err)
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ra, err := DeleteMedia.RowsAffected()
			if err != nil {
				log.Printf("Cannot get rows affected: %v\n", err)
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "Diploma file not found",
				})
			}

			filePath := lib.String(rows[0]["diploma_file_path"])
			if filePath != "" {
				// Delete physical file
				fullPath := "static/" + filePath
				err = lib.DeleteFile(fullPath)
				if err != nil {
					log.Printf("Error deleting file %s: %v\n", fullPath, err)
				}
			}
		}
		// Delete job application
		DeleteJobApplication := Orm.Delete()
		DeleteJobApplication.Table("job_applications")
		DeleteJobApplication.Where("jaid", "=", JobApplicationId)
		DeleteJobApplication.Finish()

		err = DeleteJobApplication.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete job application: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}
		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "İş başvurusu başarıyla silindi.",
		})
	}
}

func SetAsReadAJobApplication(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		JobApplicationId := c.Params("jaid")
		if JobApplicationId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Job application ID is required",
			})
		}

		// Parse request body to get is_read value
		var requestData struct {
			IsRead bool `json:"is_read"`
		}

		if err := c.BodyParser(&requestData); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		Orm := utilities.Orm

		// Update the is_read status
		UpdateContactRequest := Orm.Update()
		UpdateContactRequest.Table("job_applications")
		UpdateContactRequest.SetExpr("is_read", "NOT is_read")
		UpdateContactRequest.Set("updated_at", "NOW()")
		UpdateContactRequest.Where("jaid", "=", JobApplicationId)
		UpdateContactRequest.Finish()
		err = UpdateContactRequest.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := UpdateContactRequest.RowsAffected()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "İletişim talebi bulunamadı.",
			})
		}

		statusMessage := "okunmamış olarak işaretlendi"
		if requestData.IsRead {
			statusMessage = "okundu olarak işaretlendi"
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": fmt.Sprintf("İletişim talebi %s.", statusMessage),
		})
	}
}

func RespondToJobApplication(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			log.Printf("%v\n", err)
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		// Get job application ID from route parameter
		JobApplicationId := c.Params("jaid")
		if JobApplicationId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Job application ID is required",
			})
		}

		// Parse request body
		var inputs struct {
			Title         string `json:"title"`
			ResponderName string `json:"responder_name"`
			ResponseText  string `json:"response_text"`
		}

		if err := c.BodyParser(&inputs); err != nil {
			log.Printf("Cannot parse request body: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request data",
			})
		}

		// Validate required fields
		if inputs.Title == "" || inputs.ResponseText == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Title and response text are required",
			})
		}

		Orm := utilities.Orm

		// 1. Check if job application exists
		CheckJobApplication := Orm.Select([]string{"jaid", "first_name", "last_name", "email"})
		CheckJobApplication.Table("job_applications")
		CheckJobApplication.Where("jaid", "=", JobApplicationId)
		CheckJobApplication.Finish()

		err = CheckJobApplication.Execute()
		if err != nil {
			log.Printf("Cannot check job application: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := CheckJobApplication.Rows()
		if err != nil {
			log.Printf("Cannot get job application rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(rows) == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Job application not found",
			})
		}

		row := rows[0]
		JobApplicationData := struct {
			Jaid      string
			FirstName string
			LastName  string
			Email     string
		}{
			Jaid:      lib.String(row["jaid"]),
			FirstName: lib.String(row["first_name"]),
			LastName:  lib.String(row["last_name"]),
			Email:     lib.String(row["email"]),
		}

		// 2. Fetch options for backend
		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size", "o.smtp_host", "o.smtp_port",
			"o.smtp_username", "o.smtp_password", "o.primary_color", "o.secondary_color",
			"o.site_name", "o.site_description", "o.contact_email", "o.contact_phone", "o.facebook_url",
			"o.twitter_url", "o.instagram_url", "o.linkedin_url", "m.file_path as logo_path"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if GetOptions.Options.SMTPHost == "" || GetOptions.Options.SMTPPort == 0 || GetOptions.Options.SMTPUsername == "" || GetOptions.Options.SMTPPassword == "" || GetOptions.Options.SiteName == "" {
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "E-posta bilgileriniz girilmemişse e-posta gönderemezsiniz.",
			})
		}

		GetLogo := ""

		RootDir := os.Getenv("ROOT_DIRECTORY")
		if RootDir == "" {
			log.Printf("ROOT_DIRECTORY not set")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if (*GetOptions.Medias)[0].FilePath != "" {
			GetLogo = filepath.Join(RootDir, "static", (*GetOptions.Medias)[0].FilePath)
		} else {
			GetLogo = filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")
		}

		LogoName := filepath.Base(GetLogo)

		// 3. Send email with exact same steps
		Html := ""
		// Create professional HTML email template for job application response
		{
			Html = `<!DOCTYPE html>
					<html lang="tr">
					<head>
						<meta charset="UTF-8">
						<meta name="viewport" content="width=device-width, initial-scale=1.0">
						<title>` + inputs.Title + `</title>
						<style>
							body {
								margin: 0;
								padding: 0;
								font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
								background-color: #f4f7fa;
								color: #333333;
							}
							.email-container {
								max-width: 600px;
								margin: 40px auto;
								background-color: #ffffff;
								border-radius: 12px;
								overflow: hidden;
								box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
							}
							.header {
								padding: 40px 30px;
								text-align: center;
								color: #ffffff;
							}
							.header img {
								max-width: 180px;
								height: auto;
								margin-bottom: 20px;
								filter: brightness(0) invert(1);
							}
							.header h1 {
								margin: 0;
								font-size: 28px;
								font-weight: 600;
								letter-spacing: -0.5px;
								color: #252525 !important;
							}
							.content {
								padding: 40px 30px;
							}
							.greeting {
								font-size: 18px;
								color: #283b6a;
								margin-bottom: 20px;
								font-weight: 600;
							}
							.message {
								font-size: 16px;
								line-height: 1.8;
								color: #555555;
								margin-bottom: 25px;
							}
							.response-box {
								background-color: #f8f9fc;
								border-left: 4px solid ` + GetOptions.Options.PrimaryColor + `;
								padding: 20px 25px;
								margin: 25px 0;
								border-radius: 6px;
							}
							.response-box h3 {
								margin: 0 0 15px 0;
								color: #283b6a;
								font-size: 18px;
								font-weight: 600;
							}
							.response-text {
								background-color: #ffffff;
								padding: 20px;
								border-radius: 6px;
								border: 1px solid #e1e8ed;
								line-height: 1.6;
								color: #333333;
							}
							.responder-info {
								background-color: #f0f4ff;
								border-radius: 8px;
								padding: 20px;
								margin: 25px 0;
								text-align: center;
							}
							.responder-info h3 {
								color: #283b6a;
								margin-top: 0;
								font-size: 18px;
							}
							.responder-info p {
								margin: 5px 0;
								color: #555555;
							}
							.contact-info {
								background-color: #f0f4ff;
								border-radius: 8px;
								padding: 20px;
								margin: 25px 0;
								text-align: center;
							}
							.contact-info h3 {
								color: #283b6a;
								margin-top: 0;
								font-size: 18px;
							}
							.contact-info p {
								margin: 5px 0;
								color: #555555;
							}
							.contact-info a {
								color: ` + GetOptions.Options.PrimaryColor + `;
								text-decoration: none;
								font-weight: 600;
							}
							.footer {
								background-color: #f8f9fc;
								padding: 30px;
								text-align: center;
								font-size: 13px;
								color: #888888;
								border-top: 1px solid #e1e8ed;
							}
							.footer p {
								margin: 5px 0;
							}
							.footer a {
								color: ` + GetOptions.Options.PrimaryColor + `;
								text-decoration: none;
							}
							.social-links {
								margin: 20px 0 10px;
							}
							.social-links a {
								display: inline-block;
								margin: 0 8px;
								color: #888888;
								text-decoration: none;
								font-size: 12px;
							}
							@media only screen and (max-width: 600px) {
								.email-container {
									margin: 0;
									border-radius: 0;
								}
								.header, .content, .footer {
									padding: 25px 20px;
								}
								.header h1 {
									font-size: 24px;
								}
							}
						</style>
					</head>
					<body>
						<div class="email-container">
							<div class="header">
								<img src="cid:` + LogoName + `" alt="` + GetOptions.Options.SiteName + `" />
								<h1>` + inputs.Title + `</h1>
							</div>
							
							<div class="content">
								<div class="greeting">
									Sayın ` + JobApplicationData.FirstName + ` ` + JobApplicationData.LastName + `,
								</div>
								
								<div class="message">
									<strong>` + GetOptions.Options.SiteName + `</strong> İletişim ekibimiz tarafından size aşağıdaki gibi cevap verilmiştir.
								</div>
								
								<div class="response-box">
									<h3>📧 Cevabımız</h3>
									<div class="response-text">
										` + inputs.ResponseText + `
									</div>
								</div>
								
								<div class="responder-info">
									<h3>👤 Cevap Veren</h3>
									<p><strong>` + inputs.ResponderName + `</strong></p>
									<p>` + GetOptions.Options.SiteName + ` İnsan Kaynakları Ekibi</p>
								</div>
								
								<div class="contact-info">
									<h3>📞 Bize Ulaşın</h3>
									<p><strong>Telefon:</strong> <a href="tel:` + GetOptions.Options.ContactPhone + `">` + GetOptions.Options.ContactPhone + `</a></p>
									<p><strong>E-posta:</strong> <a href="mailto:` + GetOptions.Options.ContactEmail + `">` + GetOptions.Options.ContactEmail + `</a></p>
								</div>
								
								<div class="message" style="margin-top: 30px; font-size: 15px; color: #666;">
									Bu e-posta ` + GetOptions.Options.SiteName + ` İnsan Kaynakları ekibi tarafından gönderilmiştir. 
									Sorularınız için yukarıdaki iletişim bilgilerini kullanabilirsiniz.
								</div>
							</div>
							
							<div class="footer">
								<p><strong>` + GetOptions.Options.SiteName + `</strong></p>
								<p>` + GetOptions.Options.SiteDescription + `</p>
								
								<div class="social-links">`

			if GetOptions.Options.FacebookUrl != "" && GetOptions.Options.FacebookUrl != "#" {
				Html += `
									<a href="` + GetOptions.Options.FacebookUrl + `">Facebook</a> |`
			}
			if GetOptions.Options.TwitterUrl != "" && GetOptions.Options.TwitterUrl != "#" {
				Html += `
									<a href="` + GetOptions.Options.TwitterUrl + `">Twitter</a> |`
			}
			if GetOptions.Options.InstagramUrl != "" && GetOptions.Options.InstagramUrl != "#" {
				Html += `
									<a href="` + GetOptions.Options.InstagramUrl + `">Instagram</a> |`
			}
			if GetOptions.Options.LinkedinUrl != "" && GetOptions.Options.LinkedinUrl != "#" {
				Html += `
									<a href="` + GetOptions.Options.LinkedinUrl + `">LinkedIn</a>`
			}

			Html += `
								</div>
								
								<p style="margin-top: 20px;">&copy; 2025 ` + GetOptions.Options.SiteName + `. Tüm hakları saklıdır.</p>
							</div>
						</div>
					</body>
					</html>`
		}

		CreateEmailInfos := models.EmailInfos{
			From:        GetOptions.Options.SiteName,
			To:          []string{JobApplicationData.Email},
			Username:    GetOptions.Options.SMTPUsername,
			Password:    GetOptions.Options.SMTPPassword,
			Host:        GetOptions.Options.SMTPHost,
			Port:        lib.Int64(GetOptions.Options.SMTPPort),
			Subject:     inputs.Title + " - " + GetOptions.Options.SiteName,
			PlainText:   "Sayın " + JobApplicationData.FirstName + " " + JobApplicationData.LastName + ", " + inputs.ResponderName + " tarafından hazırlanan cevabımız: " + inputs.ResponseText,
			Body:        Html,
			Attachments: []string{GetLogo},
		}

		err = lib.SendEmail(&CreateEmailInfos)

		if err != nil {
			log.Printf("Cannot send email: %v\n", err)
		}

		// 4. Update the job application with response
		UpdateJobApplication := Orm.Update()
		UpdateJobApplication.Table("job_applications")
		UpdateJobApplication.Set("is_replied", true)
		UpdateJobApplication.Set("response_date", "NOW()")
		UpdateJobApplication.Set("updated_at", "NOW()")
		UpdateJobApplication.Where("jaid", "=", JobApplicationId)
		UpdateJobApplication.Finish()

		err = UpdateJobApplication.Execute()
		if err != nil {
			log.Printf("Cannot update job application: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Cevap başarıyla gönderildi.",
		})
	}
}

func ModifyCss(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Printf("%s", string(c.Body()))

		return c.Redirect("/panel/css-duzenle")
	}
}

func ModifyJs(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Printf("%s", string(c.Body()))

		return c.Redirect("/panel/js-duzenle")
	}
}

func NotificationWebsocket(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	Domain := os.Getenv("DOMAIN")

	if Domain == "" {
		Port := os.Getenv("PORT")

		if Port == "" {
			Port = "2000"
		}

		Domain = "http://localhost:" + Port
	}

	return websocket.New(func(c *websocket.Conn) {
		ourUser, err := lib.CheckAuth(c)

		broadcaster := states.Broadcaster

		id := ""

		if err != nil {
			id = string(lib.GenerateRandomString(10))
		} else {
			id = ourUser.Uid
		}

		protocol := lib.String(c.Locals("protocol"))

		broadcaster.Handle("notifications")

		room := broadcaster.RoomById("notifications")

		room.Handle(c, id, protocol)

		Orm := utilities.Orm

		var (
			mt   int
			msg  []byte
			err2 error
		)

		// mesaj tipleri şöyle: 1: text, 2: binary, 8: close, 9: ping, 10: pong
		//err2 = c.WriteMessage(1, []byte("Merhaba"))

	Loop:
		for {
			mt, msg, err2 = c.ReadMessage()

			if err2 != nil {
				room.RemoveById(id)

				broadcaster.RemoveIf(func(r *wsb.Room) bool {
					return r.IsRoomEmpty()
				})

				break Loop
			}

			switch mt {
			case 1:
				// text message
				WebsocketMessage := models.WebsocketMessage{}
				json.Unmarshal(msg, &WebsocketMessage)

				// herşeyden önce bildirim link ve metni oluşturulacak.
				if protocol == "randevu" && (id != WebsocketMessage.Uid) {
					RandevuTalebi := models.RandevuRequests{}
					json.Unmarshal([]byte(WebsocketMessage.Message), &RandevuTalebi)

					bildirimLink := "/panel/randevu-talepleri/" + RandevuTalebi.Rrid + "?notification=true"
					bildirimMetni := RandevuTalebi.PatientFirstName + " " + RandevuTalebi.PatientLastName + " tarafından"

					if RandevuTalebi.PatientEmail != "" {
						if !RandevuTalebi.PreferredDate.IsZero() {
							bildirimMetni += ", " + RandevuTalebi.PreferredDate.Format("02.01.2006") + " tarihinde "
						}

						if !RandevuTalebi.PreferredTime.IsZero() {
							bildirimMetni += ", " + RandevuTalebi.PreferredTime.Format("15:04") + " saatinde "
						}

						if RandevuTalebi.PreferredDate.IsZero() && RandevuTalebi.PreferredTime.IsZero() {
							bildirimMetni += " randevu talebi gönderildi."
						} else {
							bildirimMetni += "gerçekleşmek üzere randevu talebi gönderildi."
						}
					} else {
						bildirimMetni += " hızlı randevu formuyla randevu talebi gönderildi."
					}

					NewWebsocketMessage := models.WebsocketMessage{
						Uid: "",
						//InsertForm:  "randevu",
						Message:     bildirimMetni,
						RequestLink: bildirimLink,
					}

					Columns := []string{"message", "notification_type", "notification_level", "link"}
					Values := []interface{}{bildirimMetni, "info", "santral", bildirimLink}

					if RandevuTalebi.Sid != "" {
						Columns = append(Columns, "sid")
						Values = append(Values, RandevuTalebi.Sid)
					}

					InsertNotification := Orm.Insert(
						Columns,
						Values,
					)

					InsertNotification.Table("notifications")
					InsertNotification.Returning("nid")
					InsertNotification.Finish()

					err = InsertNotification.Execute()

					if err != nil {
						log.Println("cannot insert notification:", err)
					}

					lid, err := InsertNotification.LastInsertId()

					if err != nil {
						log.Println("cannot get last insert id:", err)
					}

					if lid == "" {
						log.Println("cannot get last insert id:", err)
					}

					NewWebsocketMessageBytes, err := json.Marshal(NewWebsocketMessage)

					if err != nil {
						log.Println("json marshal:", err)
					}

					room.BroadcastIf(NewWebsocketMessageBytes, nil, func(c *wsb.Connection) bool {
						if lib.String(c.Data) != "kullanici" {
							return false
						}

						GetUserRole := Orm.Select([]string{"role", "sid"})
						GetUserRole.Table("users")
						GetUserRole.Where("uid", "=", lib.String(c.Id))
						GetUserRole.Finish()

						err = GetUserRole.Execute()

						// eğer bu kullanıcı randevuyu oluşturan ziyaretçiyse, buradaki hata'dan çıkış
						// yapılıyor.
						if err != nil {
							return false
						}

						rows, err := GetUserRole.Rows()
						if err != nil {
							return false
						}

						if len(rows) == 0 {
							return false
						}

						Role := lib.String(rows[0]["role"])
						Sid := lib.String(rows[0]["sid"])

						if Role == "admin" || Role == "moderator" {
							return true
						}

						if Role == "santral" && Sid == RandevuTalebi.Sid {
							return true
						}

						return false
					})
				}

				if protocol == "is-basvurusu" && (id != WebsocketMessage.Uid) {
					JobApplication := models.JobApplications{}
					json.Unmarshal([]byte(WebsocketMessage.Message), &JobApplication)

					bildirimLink := "/panel/is-basvurulari/" + JobApplication.Jaid + "?notification=true"
					bildirimMetni := JobApplication.FirstName + " " + JobApplication.LastName + " tarafından bir iş başvurusu gönderildi."

					NewWebsocketMessage := models.WebsocketMessage{
						Uid: "",
						//InsertForm:  "is-basvurusu",
						Message:     bildirimMetni,
						RequestLink: bildirimLink,
					}

					InsertNotification := Orm.Insert(
						[]string{"message", "notification_type", "notification_level", "link"},
						[]interface{}{bildirimMetni, "info", "ik", bildirimLink},
					)

					InsertNotification.Table("notifications")
					InsertNotification.Returning("nid")
					InsertNotification.Finish()

					err = InsertNotification.Execute()

					if err != nil {
						log.Println("cannot insert notification:", err)
					}

					lid, err := InsertNotification.LastInsertId()

					if err != nil {
						log.Println("cannot get last insert id:", err)
					}

					if lid == "" {
						log.Println("cannot get last insert id:", err)
					}

					NewWebsocketMessageBytes, err := json.Marshal(NewWebsocketMessage)

					if err != nil {
						log.Println("json marshal:", err)
					}

					room.BroadcastIf(NewWebsocketMessageBytes, nil, func(c *wsb.Connection) bool {
						if lib.String(c.Data) != "kullanici" {
							return false
						}

						CheckIfUserIsAdminOrIk := Orm.Count("users")
						CheckIfUserIsAdminOrIk.Where("uid", "=", lib.String(c.Id))
						CheckIfUserIsAdminOrIk.OpenParenthesis("AND")
						CheckIfUserIsAdminOrIk.And("role", "=", "admin")
						CheckIfUserIsAdminOrIk.Or("role", "=", "moderator")
						CheckIfUserIsAdminOrIk.Or("role", "=", "ik")
						CheckIfUserIsAdminOrIk.CloseParenthesis()
						CheckIfUserIsAdminOrIk.Finish()

						err = CheckIfUserIsAdminOrIk.Execute()

						if err != nil {
							return false
						}

						if CheckIfUserIsAdminOrIk.Length() == 0 {
							return false
						}

						return true
					})
				}

				if protocol == "iletisim" && WebsocketMessage.Uid == "" {
					ContactRequest := models.ContactRequests{}
					json.Unmarshal([]byte(WebsocketMessage.Message), &ContactRequest)

					bildirimLink := "/panel/iletisim-istekleri/" + ContactRequest.Crid + "?notification=true"
					bildirimMetni := ContactRequest.FirstName + " " + ContactRequest.LastName + " tarafından bir iletişim talebi gönderildi."

					NewWebsocketMessage := models.WebsocketMessage{
						Uid: "",
						//InsertForm:  "is-basvurusu",
						Message:     bildirimMetni,
						RequestLink: bildirimLink,
					}

					InsertNotification := Orm.Insert(
						[]string{"message", "notification_type", "notification_level", "link"},
						[]interface{}{bildirimMetni, "info", "all", bildirimLink},
					)

					InsertNotification.Table("notifications")
					InsertNotification.Returning("nid")
					InsertNotification.Finish()

					err = InsertNotification.Execute()

					if err != nil {
						log.Println("cannot insert notification:", err)
					}

					lid, err := InsertNotification.LastInsertId()

					if err != nil {
						log.Println("cannot get last insert id:", err)
					}

					if lid == "" {
						log.Println("cannot get last insert id:", err)
					}

					NewWebsocketMessageBytes, err := json.Marshal(NewWebsocketMessage)

					if err != nil {
						log.Println("json marshal:", err)
					}

					room.BroadcastIf(NewWebsocketMessageBytes, nil, func(c *wsb.Connection) bool {
						return lib.String(c.Data) == "kullanici"
					})
				}

				break Loop
			case 2:
				// binary message
			case 8:
				fmt.Printf("Close message received\n")

				room.RemoveById(id)

				broadcaster.RemoveIf(func(r *wsb.Room) bool {
					return r.IsRoomEmpty()
				})

				break Loop
			case 9:
				// ping message
			case 10:
				// pong message
			}
		}
	}, websocket.Config{
		Subprotocols: []string{"kullanici", "randevu", "is-basvurusu", "iletisim"},
		Origins:      []string{Domain},
	})
}
