package tedkikler

import (
	"database"
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"
	"post/uploadpolicy"

	"github.com/gofiber/fiber/v2"
)

func AddTedkik(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			return c.Redirect("/panel/tedkikler/tedkik-ekle?error=only_admins_can_add_examinations")
		}

		inputs := models.Tedkikler{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.Name == "" {
			return c.Redirect("/panel/tedkikler/tedkik-ekle?error=name_required")
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
		if inputs.HtmlContent != "" {
			columns = append(columns, "html_content")
			values = append(values, inputs.HtmlContent)
		}
		if inputs.JavascriptContent != "" {
			columns = append(columns, "javascript_content")
			values = append(values, inputs.JavascriptContent)
		}
		if inputs.CssContent != "" {
			columns = append(columns, "css_content")
			values = append(values, inputs.CssContent)
		}

		Orm := utilities.Orm

		uploadPolicy, err := uploadpolicy.Read(c.UserContext(), utilities.UploadPolicyReader)
		if err != nil {
			log.Print("Cannot get options")
			return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
		}

		insertTedkik := Orm.Insert(columns, values)
		insertTedkik.Table("tedkikler")
		insertTedkik.Returning("tid")
		insertTedkik.Finish()
		err = insertTedkik.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert tedkik: %v\n", err)
			return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
		}

		tid, err := insertTedkik.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
		}

		// Handle file upload if provided
		var TedkikMediaMid string = ""
		tedkikMediaInput, err := c.FormFile("cover_mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
			}

			// File size validation (5MB max)
			if tedkikMediaInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", tedkikMediaInput.Size)
				return c.Redirect("/panel/tedkikler/tedkik-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "tedkikler", tid)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + tedkikMediaInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", UniqueFilePath.Extension)
				return c.Redirect("/panel/tedkikler/tedkik-ekle?error=invalid_file_type")
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/tedkikler/" + tid + "/" + UniqueFilePath.BaseName,
				FileSize: tedkikMediaInput.Size,
				MimeType: tedkikMediaInput.Header.Get("Content-Type"),
				FileType: "tedkik_cover",
				Uid:      OurUser.Uid,
				TargetId: tid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.CoverAltText,
				Title:   inputs.CoverTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			TedkikMediaMid, err = OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *tedkikMediaInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
			}

			// Update tedkik with media ID
			if TedkikMediaMid != "" {
				updateTedkik := Orm.Update()
				updateTedkik.Table("tedkikler")
				updateTedkik.Set("cover_mid", TedkikMediaMid)
				updateTedkik.Where("tid", "=", tid)
				updateTedkik.Finish()
				err = updateTedkik.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update tedkik with media: %v\n", err)
					return c.Redirect("/panel/tedkikler/tedkik-ekle?error=internal_server_error")
				}
			}
		}

		Orm.Commit()

		states.TedkiklerLinks = []models.TedkikLink{}

		return c.Redirect("/panel/tedkikler/" + tid)
	}
}

func EditTedkik(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		TedkikId := c.Params("tid")
		Orm := utilities.Orm

		inputs := models.TedkiklerEdit{}
		c.BodyParser(&inputs)
		inputs.Tid = TedkikId

		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Tedkik adı zorunludur.",
			})
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		updateTedkik := Orm.Update()
		updateTedkik.Table("tedkikler")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			updateTedkik.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateTedkik.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateTedkik.Set("description", inputs.Description)
			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateTedkik.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		if inputs.HtmlContent != inputs.OldHtmlContent {
			updateTedkik.Set("html_content", inputs.HtmlContent)
			SomethingSet = true
		}
		if inputs.JavascriptContent != inputs.OldJavascriptContent {
			updateTedkik.Set("javascript_content", inputs.JavascriptContent)
			SomethingSet = true
		}
		if inputs.CssContent != inputs.OldCssContent {
			updateTedkik.Set("css_content", inputs.CssContent)
			SomethingSet = true
		}

		if SomethingSet {
			updateTedkik.Set("updated_at", "NOW()")
			updateTedkik.Where("tid", "=", TedkikId)
			updateTedkik.Finish()
			err = updateTedkik.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update tedkik: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		Orm.Commit()

		states.TedkiklerLinks = []models.TedkikLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Tedkik başarıyla güncellendi.",
		})
	}
}

func DeleteTedkik(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		TedkikId := c.Params("tid")
		Orm := utilities.Orm

		if TedkikId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Tedkik ID is required",
			})
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// First, get the tedkik to find the cover_mid
		GetTedkik := Orm.Select([]string{"tid", "name", "cover_mid"})
		GetTedkik.Table("tedkikler")
		GetTedkik.Where("tid", "=", TedkikId)
		GetTedkik.Finish()
		err = GetTedkik.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get tedkik: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetTedkik.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			log.Printf("Tedkik not found: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tedkik bulunamadı",
			})
		}

		coverMid := lib.Int64(rows[0]["cover_mid"])

		// Delete the tedkik
		DeleteTedkikQuery := Orm.Delete()
		DeleteTedkikQuery.Table("tedkikler")
		DeleteTedkikQuery.Where("tid", "=", TedkikId)
		DeleteTedkikQuery.Finish()
		err = DeleteTedkikQuery.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete tedkik: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// If there's a cover image, delete it and its file
		if coverMid > 0 {
			// Get media file path
			GetMedia := Orm.Select([]string{"mid", "file_path"})
			GetMedia.Table("medias")
			GetMedia.Where("mid", "=", coverMid)
			GetMedia.Finish()
			err = GetMedia.Execute()

			if err == nil {
				mediaRows, err := GetMedia.Rows()
				if err == nil && len(mediaRows) > 0 {
					filePath := lib.String(mediaRows[0]["file_path"])

					// Delete the physical file
					if filePath != "" {
						RootDir := os.Getenv("ROOT_DIRECTORY")
						if RootDir != "" {
							fullPath := RootDir + "/static/" + filePath
							err = lib.DeleteFile(fullPath)
							if err != nil {
								log.Printf("Cannot delete file %s: %v\n", fullPath, err)
							}
						}
					}

					// Delete media record
					DeleteMedia := Orm.Delete()
					DeleteMedia.Table("medias")
					DeleteMedia.Where("mid", "=", coverMid)
					DeleteMedia.Finish()
					err = DeleteMedia.Execute()

					if err != nil {
						log.Printf("Cannot delete media record: %v\n", err)
					}
				}
			}
		}

		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		states.TedkiklerLinks = []models.TedkikLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Tedkik başarıyla silindi",
		})
	}
}

func UpdateTedkikPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		TedkikId := c.Params("tid")

		inputs := models.TedkiklerEdit{}
		if err := c.BodyParser(&inputs); err != nil {
			log.Printf("error: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		Orm := utilities.Orm
		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("ROOT_DIRECTORY is empty")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		uploadPolicy, err := uploadpolicy.Read(c.UserContext(), utilities.UploadPolicyReader)
		if err != nil {
			log.Print("Cannot get options")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		coverInput, err := c.FormFile("cover_mid")
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
			if coverInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "tedkikler", TedkikId)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + coverInput.Filename)

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
				FilePath: "files/tedkikler/" + TedkikId + "/" + UniqueFilePath.BaseName,
				FileSize: coverInput.Size,
				MimeType: coverInput.Header.Get("Content-Type"),
				FileType: "tedkik_cover",
				Uid:      ourUser.Uid,
				TargetId: TedkikId,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.CoverAltText,
				Title:   inputs.CoverTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			TedkikCoverMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/tedkikler/"+TedkikId, entry.Name())
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

			// Update tedkik with new media ID
			UpdateTedkik := Orm.Update()
			UpdateTedkik.Table("tedkikler")
			UpdateTedkik.Set("cover_mid", TedkikCoverMid)
			UpdateTedkik.Where("tid", "=", TedkikId)
			UpdateTedkik.Finish()
			err = UpdateTedkik.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update tedkik: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *coverInput)
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
			if inputs.CoverAltText != inputs.OldCoverAltText || inputs.CoverTitle != inputs.OldCoverTitle {
				// Get current media ID
				GetTedkikMedia := Orm.Select([]string{"cover_mid"})
				GetTedkikMedia.Table("tedkikler")
				GetTedkikMedia.Where("tid", "=", TedkikId)
				GetTedkikMedia.Finish()
				err = GetTedkikMedia.Execute()

				if err != nil {
					log.Printf("Cannot get tedkik media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetTedkikMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Tedkik bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["cover_mid"])
				if mediaId == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Tedkik görseli bulunamadı.",
					})
				}

				// Update media metadata
				UpdateMedia := Orm.Update()
				UpdateMedia.Table("medias")
				UpdateMedia.Set("alt_text", inputs.CoverAltText)
				UpdateMedia.Set("title", inputs.CoverTitle)
				UpdateMedia.Where("mid", "=", mediaId)
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

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Tedkik görseli başarıyla güncellendi.",
		})
	}
}

func DeleteTedkikPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		TedkikId := c.Params("tid")
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
		GetTedkikMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetTedkikMedia.Table("tedkikler t")
		GetTedkikMedia.LeftJoin("medias m", "t.cover_mid", "=", "m.mid")
		GetTedkikMedia.Where("t.tid", "=", TedkikId)
		GetTedkikMedia.Finish()
		err = GetTedkikMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get tedkik media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetTedkikMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tedkik bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tedkik görseli bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", TedkikId)
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
				"message": "Tedkik görseli bulunamadı.",
			})
		}

		// Update tedkik to remove media reference
		UpdateTedkik := Orm.Update()
		UpdateTedkik.Table("tedkikler")
		UpdateTedkik.Set("cover_mid", nil)
		UpdateTedkik.Where("tid", "=", TedkikId)
		UpdateTedkik.Finish()
		err = UpdateTedkik.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update tedkik: %v\n", err)
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
			"message": "Tedkik görseli başarıyla silindi.",
		})
	}
}

func GetAllTedkikler(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Orm := utilities.Orm
		GetAllTedkikler := Orm.Select([]string{"tid", "name"})
		GetAllTedkikler.Table("tedkikler")
		GetAllTedkikler.Finish()

		err = GetAllTedkikler.Execute()

		if err != nil {
			log.Printf("Cannot get all tedkikler: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetAllTedkikler.Rows()
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tedkikler bulunamadı.",
			})
		}

		Tedkikler := []models.Tedkikler{}
		for _, row := range rows {
			Tedkikler = append(Tedkikler, models.Tedkikler{
				Tid:  lib.String(row["tid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Tedkikler başarıyla getirildi.",
			"data":    Tedkikler,
		})
	}
}
