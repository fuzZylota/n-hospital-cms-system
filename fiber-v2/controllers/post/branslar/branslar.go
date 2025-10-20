package branslar

import (
	"database"
	"encoding/json"
	"fmt"
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

func AddBranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			return c.Redirect("/panel/branslar/brans-ekle?error=only_admins_can_add_branch")
		}

		inputs := models.Branslar{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.Name == "" {
			return c.Redirect("/panel/branslar/brans-ekle?error=name_required")
		}

		columns := []string{"name", "is_active", "sid"}
		values := []interface{}{inputs.Name, inputs.IsActive, inputs.Sid}

		// Add optional fields if provided
		if inputs.UrlName != "" {
			columns = append(columns, "url_name")
			values = append(values, inputs.UrlName)
		}
		if inputs.Description != "" {
			columns = append(columns, "description")
			values = append(values, inputs.Description)
		}
		if inputs.ShortDescription != "" {
			columns = append(columns, "short_description")
			values = append(values, inputs.ShortDescription)
		}
		if len(inputs.Services) > 0 {
			ServicesText := inputs.Services[0]
			Services := []string{}
			err = json.Unmarshal([]byte(ServicesText), &Services)
			if err != nil {
				log.Printf("Cannot unmarshal services: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			if len(Services) > 0 {
				columns = append(columns, "services")
				values = append(values, Services)
			}
		}
		if inputs.Icon != "" {
			columns = append(columns, "icon")
			values = append(values, inputs.Icon)
		}
		if inputs.Phone != "" {
			columns = append(columns, "phone")
			values = append(values, inputs.Phone)
		}
		if inputs.Email != "" {
			columns = append(columns, "email")
			values = append(values, inputs.Email)
		}

		Orm := utilities.Orm

		// Handle file upload if present
		if inputs.Mid != 0 {
			// File upload logic would go here
			// For now, just add the mid to columns
			columns = append(columns, "mid")
			values = append(values, inputs.Mid)
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
		}

		insertBrans := Orm.Insert(columns, values)
		insertBrans.Table("branslar")
		insertBrans.Returning("brid")
		insertBrans.Finish()

		fmt.Printf("insertBrans: %v\n", insertBrans.Query)

		err = insertBrans.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert branch: %v\n", err)
			return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
		}

		brid, err := insertBrans.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
		}

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
		}

		if inputs.HeadDrid != "" {
			UpdateHeadDrid := Orm.SelectFunction("update_head_drid", brid, inputs.HeadDrid)
			err = UpdateHeadDrid.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update head doctor: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			rows, err := UpdateHeadDrid.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			if len(rows) == 0 {
				Orm.Rollback()
				log.Printf("Cannot update head doctor: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			if rows[0]["result"] == "false" {
				Orm.Rollback()
				log.Printf("Cannot update head doctor: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

		}

		var BransMediaMid string = ""
		bransMediaInput, err := c.FormFile("mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			// File size validation (5MB max)
			if bransMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", bransMediaInput.Size)
				return c.Redirect("/panel/branslar/brans-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "branslar", brid)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + bransMediaInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", UniqueFilePath.Extension)
				return c.Redirect("/panel/branslar/brans-ekle?error=invalid_file_type")
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/branslar/" + brid + "/" + UniqueFilePath.BaseName,
				FileSize: bransMediaInput.Size,
				MimeType: bransMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: brid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.BransMediaAltText,
				Title:   inputs.BransMediaTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			BransMediaMid, err = OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *bransMediaInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
			}

			// Update branch with media ID
			if BransMediaMid != "" {
				updateBrans := Orm.Update()
				updateBrans.Table("branslar")
				updateBrans.Set("mid", BransMediaMid)
				updateBrans.Where("brid", "=", brid)
				updateBrans.Finish()
				err = updateBrans.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update branch with media: %v\n", err)
					return c.Redirect("/panel/branslar/brans-ekle?error=internal_server_error")
				}
			}
		}

		Orm.Commit()

		return c.Redirect("/panel/branslar/" + brid)
	}
}

func EditBranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		BransId := c.Params("brid")
		Orm := utilities.Orm

		inputs := models.BranslarEdit{}
		c.BodyParser(&inputs)
		inputs.Brid = BransId

		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Bölüm adı zorunludur.",
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
			CheckIfUrlNameExists := Orm.Count("branslar")
			CheckIfUrlNameExists.Where("url_name", "=", inputs.UrlName)
			CheckIfUrlNameExists.And("brid", "!=", BransId)
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

		updateBrans := Orm.Update()
		updateBrans.Table("branslar")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			updateBrans.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateBrans.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateBrans.Set("description", inputs.Description)
			SomethingSet = true
		}

		if inputs.ShortDescription != inputs.OldShortDescription {
			updateBrans.Set("short_description", inputs.ShortDescription)
			SomethingSet = true
		}

		if len(inputs.Services) > 0 || len(inputs.OldServices) > 0 {
			ThereIsDifference := false

			if len(inputs.Services) != len(inputs.OldServices) {
				ThereIsDifference = true
			}

			if !ThereIsDifference {
				for i, service := range inputs.Services {
					if service != inputs.OldServices[i] {
						ThereIsDifference = true
						break
					}
				}
			}

			if ThereIsDifference {
				updateBrans.Set("services", inputs.Services)
				SomethingSet = true
			}
		}

		if inputs.Icon != inputs.OldIcon {
			updateBrans.Set("icon", inputs.Icon)
			SomethingSet = true
		}

		if inputs.Sid != inputs.OldSid {
			if inputs.Sid == "" {
				updateBrans.Set("sid", nil)
			} else {
				updateBrans.Set("sid", inputs.Sid)
			}

			SomethingSet = true
		}

		if inputs.HeadDrid != inputs.OldHeadDrid {
			updateBrans.Set("head_drid", inputs.HeadDrid)
			SomethingSet = true
		}

		if inputs.Phone != inputs.OldPhone {
			updateBrans.Set("phone", inputs.Phone)
			SomethingSet = true
		}

		if inputs.Email != inputs.OldEmail {
			updateBrans.Set("email", inputs.Email)
			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateBrans.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		if SomethingSet {
			updateBrans.Set("updated_at", "NOW()")
			updateBrans.Where("brid", "=", BransId)
			updateBrans.Finish()
			err = updateBrans.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update branch: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		if inputs.HeadDrid != inputs.OldHeadDrid {
			UpdateHeadDrid := Orm.SelectFunction("update_head_drid", BransId, inputs.HeadDrid)
			err = UpdateHeadDrid.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update head doctor: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			rows, err := UpdateHeadDrid.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if len(rows) == 0 {
				Orm.Rollback()
				log.Printf("Cannot update head doctor: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if rows[0]["result"] == "false" {
				Orm.Rollback()
				log.Printf("Cannot update head doctor: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Bölüm başarıyla güncellendi.",
		})
	}
}

func UpdateBranchPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		BransId := c.Params("brid")
		bransMediaAltText := c.FormValue("brans_media_alt_text")
		bransMediaTitle := c.FormValue("brans_media_title")
		oldBransMediaAltText := c.FormValue("old_brans_media_alt_text")
		oldBransMediaTitle := c.FormValue("old_brans_media_title")

		Orm := utilities.Orm
		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("ROOT_DIRECTORY is empty")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		bransMediaInput, err := c.FormFile("brans_media_path")
		if err == nil {
			// File upload case
			err = Orm.Begin()
			if err != nil {
				log.Printf("Cannot begin transaction: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// File size validation (5MB max)
			if bransMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "branslar", BransId)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + bransMediaInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.",
				})
			}

			// Insert new media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/branslar/" + BransId + "/" + UniqueFilePath.BaseName,
				FileSize: bransMediaInput.Size,
				MimeType: bransMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: BransId,
			}

			optionals := models.MediaOptionals{
				AltText: bransMediaAltText,
				Title:   bransMediaTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			BransMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Delete old media files
			entries, err := lib.ReadDirectory(estimatedPath)
			if err != nil && err != os.ErrNotExist {
				log.Printf("Cannot read directory: %v\n", err)
			}

			FilePaths := []any{}
			for _, entry := range entries {
				FilePath := filepath.Join("files/branslar/"+BransId, entry.Name())
				FilePaths = append(FilePaths, FilePath)
			}

			if len(FilePaths) > 0 {
				DeleteMedias := Orm.Delete()
				DeleteMedias.Table("medias")
				DeleteMedias.In("WHERE", "file_path", FilePaths)
				DeleteMedias.Finish()
				err = DeleteMedias.Execute()
				if err != nil {
					log.Printf("Cannot delete old media records: %v\n", err)
				}
			}

			// Update branch with new media ID
			UpdateBrans := Orm.Update()
			UpdateBrans.Table("branslar")
			UpdateBrans.Set("mid", BransMediaMid)
			UpdateBrans.Where("brid", "=", BransId)
			UpdateBrans.Finish()
			err = UpdateBrans.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update branch: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *bransMediaInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Delete old physical files
			for _, filePath := range FilePaths {
				GetAbsolutePath := filepath.Join(RootDir, "static", lib.String(filePath))
				err := lib.DeleteFile(GetAbsolutePath)
				if err != nil {
					log.Printf("Cannot remove old file: %v\n", err)
				}
			}

			Orm.Commit()
		} else {
			// Metadata only update case
			if bransMediaAltText != oldBransMediaAltText || bransMediaTitle != oldBransMediaTitle {
				// Get current media ID
				GetBransMedia := Orm.Select([]string{"mid"})
				GetBransMedia.Table("branslar")
				GetBransMedia.Where("brid", "=", BransId)
				GetBransMedia.Finish()
				err = GetBransMedia.Execute()

				if err != nil {
					log.Printf("Cannot get branch media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetBransMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Bölüm bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["mid"])
				if mediaId > 0 {
					UpdateMedia := Orm.Update()
					UpdateMedia.Table("medias")

					if bransMediaAltText != oldBransMediaAltText {
						if bransMediaAltText != "" {
							UpdateMedia.Set("alt_text", bransMediaAltText)
						} else {
							UpdateMedia.Set("alt_text", nil)
						}
					}

					if bransMediaTitle != oldBransMediaTitle {
						if bransMediaTitle != "" {
							UpdateMedia.Set("title", bransMediaTitle)
						} else {
							UpdateMedia.Set("title", nil)
						}
					}

					UpdateMedia.Where("mid", "=", mediaId)
					UpdateMedia.And("target_id", "=", BransId)
					UpdateMedia.Finish()

					err = UpdateMedia.Execute()
					if err != nil {
						log.Printf("Cannot update media metadata: %v\n", err)
						return c.JSON(fiber.Map{
							"status":  500,
							"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
						})
					}
				}
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Bölüm görseli başarıyla güncellendi.",
		})
	}
}

func DeleteBranchPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		BransId := c.Params("brid")
		Orm := utilities.Orm

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Get current media information
		GetBransMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetBransMedia.Table("branslar b")
		GetBransMedia.LeftJoin("medias m", "b.mid", "=", "m.mid")
		GetBransMedia.Where("b.brid", "=", BransId)
		GetBransMedia.Finish()
		err = GetBransMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get branch media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetBransMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Bölüm bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Bölüm görseli bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", BransId)
		DeleteMedia.Finish()
		err = DeleteMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete media record: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteMedia.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get rows affected: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Bölüm görseli bulunamadı.",
			})
		}

		// Update branch to remove media reference
		UpdateBrans := Orm.Update()
		UpdateBrans.Table("branslar")
		UpdateBrans.Set("mid", nil)
		UpdateBrans.Where("brid", "=", BransId)
		UpdateBrans.Finish()
		err = UpdateBrans.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update branch: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete physical file
		if filePath != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			fullPath := filepath.Join(RootDir, "static", filePath)
			err = lib.DeleteFile(fullPath)
			if err != nil {
				log.Printf("Cannot delete physical file: %v\n", err)
				// Don't rollback for file deletion errors, just log
				// The database operation was successful
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Bölüm görseli başarıyla silindi.",
		})
	}
}

func DeleteBranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		BransId := c.Params("brid")
		if BransId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Branch ID is required",
			})
		}

		Orm := utilities.Orm

		// Fetch branch media information before deletion
		GetBranchMedia := Orm.Select([]string{"mid"})
		GetBranchMedia.Table("branslar")
		GetBranchMedia.Where("brid", "=", BransId)
		GetBranchMedia.Finish()
		err = GetBranchMedia.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		branchRows, err := GetBranchMedia.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(branchRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Branch not found",
			})
		}

		var mediaId int64
		if branchRows[0]["mid"] != nil {
			mediaId = lib.Int64(branchRows[0]["mid"])
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

		// Delete related media files if exists
		if mediaId > 0 {
			// Get media file path
			GetMediaPath := Orm.Select([]string{"file_path"})
			GetMediaPath.Table("medias")
			GetMediaPath.Where("mid", "=", mediaId)
			GetMediaPath.Finish()
			err = GetMediaPath.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			mediaRows, err := GetMediaPath.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if len(mediaRows) == 0 {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "Media not found",
				})
			}

			DeleteMedia := Orm.Delete()
			DeleteMedia.Table("medias")
			DeleteMedia.Where("mid", "=", mediaId)
			DeleteMedia.Finish()
			err = DeleteMedia.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ra, err := DeleteMedia.RowsAffected()
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
				log.Printf("%v\n", err)
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "Media not found",
				})
			}

			filePath := lib.String(mediaRows[0]["file_path"])
			if filePath != "" {
				// Delete physical file
				fullPath := "static/" + filePath
				err = lib.DeleteFile(fullPath)
				if err != nil {
					log.Printf("Error deleting file %s: %v\n", fullPath, err)
				}
			}
			// Delete media record
		}

		// Delete the branch
		DeleteBranch := Orm.Delete()
		DeleteBranch.Table("branslar")
		DeleteBranch.Where("brid", "=", BransId)
		DeleteBranch.Finish()
		err = DeleteBranch.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteBranch.RowsAffected()
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
				"message": "Bölüm bulunamadı.",
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
			"message": "Branch deleted successfully",
		})
	}
}

func ChangeHeadDoctorOfABranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		BransId := c.Params("brid")
		if BransId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Branch ID is required",
			})
		}

		inputs := models.BranslarEdit{}
		c.BodyParser(&inputs)

		if inputs.HeadDrid == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Head doctor ID is required",
			})
		}

		Orm := utilities.Orm

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Update doctor to be active and set as head doctor
		UpdateDoctor := Orm.Update()
		UpdateDoctor.Table("doktorlar")
		UpdateDoctor.Set("is_active", true)
		UpdateDoctor.Set("brid", BransId)
		UpdateDoctor.Where("drid", "=", inputs.HeadDrid)
		UpdateDoctor.Finish()
		err = UpdateDoctor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Update branch head doctor
		UpdateBranch := Orm.Update()
		UpdateBranch.Table("branslar")
		UpdateBranch.Set("head_drid", inputs.HeadDrid)
		UpdateBranch.Where("brid", "=", BransId)
		UpdateBranch.Finish()
		err = UpdateBranch.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
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
			"message": "Head doctor changed successfully",
		})
	}
}

func AddDoctorToABranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")
		Orm := utilities.Orm

		inputs := models.ChangeDoctorBranchInputs{}
		c.BodyParser(&inputs)

		if inputs.Brid == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Bölüm ID zorunludur.",
			})
		}

		// Check if branch exists
		CheckBrans := Orm.Select([]string{"brid"})
		CheckBrans.Table("branslar")
		CheckBrans.Where("brid", "=", inputs.Brid)
		CheckBrans.Finish()
		err = CheckBrans.Execute()

		if err != nil {
			log.Printf("Cannot check branch: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := CheckBrans.Rows()
		if err != nil || len(rows) == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Bölüm bulunamadı.",
			})
		}

		// Check if doctor exists
		CheckDoctor := Orm.Select([]string{"drid"})
		CheckDoctor.Table("doktorlar")
		CheckDoctor.Where("drid", "=", Drid)
		CheckDoctor.Finish()
		err = CheckDoctor.Execute()

		if err != nil {
			log.Printf("Cannot check doctor: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		doctorRows, err := CheckDoctor.Rows()
		if err != nil || len(doctorRows) == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Check if doctor is head of another branch and remove them
		CheckHeadDoctor := Orm.Select([]string{"brid"})
		CheckHeadDoctor.Table("branslar")
		CheckHeadDoctor.Where("head_drid", "=", Drid)
		CheckHeadDoctor.Finish()
		err = CheckHeadDoctor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot check head doctor: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		headDoctorRows, err := CheckHeadDoctor.Rows()
		if err == nil && len(headDoctorRows) > 0 {
			// Remove doctor as head from other branches
			UpdateOtherBranches := Orm.Update()
			UpdateOtherBranches.Table("branslar")
			UpdateOtherBranches.Set("head_drid", nil)
			UpdateOtherBranches.Where("head_drid", "=", Drid)
			UpdateOtherBranches.Finish()
			err = UpdateOtherBranches.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update other branches: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		// Update doctor's branch
		UpdateDoctor := Orm.Update()
		UpdateDoctor.Table("doktorlar")
		UpdateDoctor.Set("brid", inputs.Brid)
		UpdateDoctor.Where("drid", "=", Drid)
		UpdateDoctor.Finish()
		err = UpdateDoctor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update doctor: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Doktor başarıyla bölüme eklendi.",
		})
	}
}

func RemoveDoctorFromABranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")
		Orm := utilities.Orm

		// Check if doctor exists
		CheckDoctor := Orm.Select([]string{"drid", "brid"})
		CheckDoctor.Table("doktorlar")
		CheckDoctor.Where("drid", "=", Drid)
		CheckDoctor.Finish()
		err = CheckDoctor.Execute()

		if err != nil {
			log.Printf("Cannot check doctor: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		doctorRows, err := CheckDoctor.Rows()
		if err != nil || len(doctorRows) == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
		}

		currentBrid := lib.String(doctorRows[0]["brid"])
		if currentBrid == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Doktor zaten hiçbir bölümde değil.",
			})
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Check if doctor is head of current branch and remove them
		CheckHeadDoctor := Orm.Select([]string{"brid"})
		CheckHeadDoctor.Table("branslar")
		CheckHeadDoctor.Where("head_drid", "=", Drid)
		CheckHeadDoctor.And("brid", "=", currentBrid)
		CheckHeadDoctor.Finish()
		err = CheckHeadDoctor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot check head doctor: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		headDoctorRows, err := CheckHeadDoctor.Rows()
		if err == nil && len(headDoctorRows) > 0 {
			// Remove doctor as head from current branch
			UpdateBranch := Orm.Update()
			UpdateBranch.Table("branslar")
			UpdateBranch.Set("head_drid", nil)
			UpdateBranch.Where("brid", "=", currentBrid)
			UpdateBranch.Finish()
			err = UpdateBranch.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update branch: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		// Remove doctor from branch
		UpdateDoctor := Orm.Update()
		UpdateDoctor.Table("doktorlar")
		UpdateDoctor.Set("brid", nil)
		UpdateDoctor.Where("drid", "=", Drid)
		UpdateDoctor.Finish()
		err = UpdateDoctor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update doctor: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Doktor başarıyla bölümden çıkarıldı.",
		})
	}
}

func GetDoctorsForAddingBranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Brid := c.Params("brid")

		GetDoctors := Orm.Select([]string{"drid", "first_name", "last_name", "title"})
		GetDoctors.Table("doktorlar")
		GetDoctors.Where("brid", "!=", Brid)
		GetDoctors.Finish()
		err = GetDoctors.Execute()

		if err != nil {
			log.Printf("Cannot get doctors: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetDoctors.Rows()
		if err != nil {
			log.Printf("Cannot get doctors: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		DoctorsArray := []models.Doktorlar{}
		for _, row := range rows {
			DoctorsArray = append(DoctorsArray, models.Doktorlar{
				Drid:      lib.String(row["drid"]),
				FirstName: lib.String(row["first_name"]),
				LastName:  lib.String(row["last_name"]),
				Title:     lib.String(row["title"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Doctors fetched successfully",
			"data":    DoctorsArray,
		})
	}
}
