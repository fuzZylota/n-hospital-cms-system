package tibbibirimler

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

func AddTibbiBirim(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Redirect("/giris")
		}

		if OurUser.Role != "admin" {
			return c.Redirect("/panel/tibbi-birim-ekle?error=only_admins_can_add_units")
		}

		inputs := models.TibbiBirimler{}
		c.BodyParser(&inputs)

		if inputs.Name == "" {
			return c.Redirect("/panel/tibbi-birim-ekle?error=name_required")
		}

		columns := []string{"name", "is_active"}
		values := []interface{}{inputs.Name, inputs.IsActive}

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
			return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
		}

		insertTB := Orm.Insert(columns, values)
		insertTB.Table("tibbi_birimler")
		insertTB.Returning("tbid")
		insertTB.Finish()
		err = insertTB.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert tibbi_birim: %v\n", err)
			return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
		}

		tbid, err := insertTB.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
		}

		// Handle cover upload if provided
		tibbiBirimCoverInput, err := c.FormFile("cover_mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			if tibbiBirimCoverInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				return c.Redirect("/panel/tibbi-birim-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "tibbi_birimler", tbid, "cover")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + tibbiBirimCoverInput.Filename)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg":
				break
			default:
				Orm.Rollback()
				return c.Redirect("/panel/tibbi-birim-ekle?error=invalid_file_type")
			}

			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/tibbi_birimler/" + tbid + "/cover/" + UniqueFilePath.BaseName,
				FileSize: tibbiBirimCoverInput.Size,
				MimeType: tibbiBirimCoverInput.Header.Get("Content-Type"),
				FileType: "tibbi_birim_cover",
				Uid:      OurUser.Uid,
				TargetId: tbid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.CoverAltText,
				Title:   inputs.CoverTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			CoverMid, err := OurOptions.InsertMedia(Orm, media, optionals)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *tibbiBirimCoverInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			if CoverMid != "" {
				updateTB := Orm.Update()
				updateTB.Table("tibbi_birimler")
				updateTB.Set("cover_mid", CoverMid)
				updateTB.Where("tbid", "=", tbid)
				updateTB.Finish()
				err = updateTB.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update tibbi_birim with media: %v\n", err)
					return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
				}
			}
		}

		// Handle cover upload if provided
		tibbiBirimVideoInput, err := c.FormFile("video_mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			if tibbiBirimVideoInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				return c.Redirect("/panel/tibbi-birim-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "tibbi_birimler", tbid, "video")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + tibbiBirimVideoInput.Filename)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".mp4", ".webm", ".ogg", ".mov", ".avi", ".mkv", ".flv":
				break
			default:
				Orm.Rollback()
				return c.Redirect("/panel/tibbi-birim-ekle?error=invalid_file_type")
			}

			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/tibbi_birimler/" + tbid + "/video/" + UniqueFilePath.BaseName,
				FileSize: tibbiBirimVideoInput.Size,
				MimeType: tibbiBirimVideoInput.Header.Get("Content-Type"),
				FileType: "tibbi_birim_video",
				Uid:      OurUser.Uid,
				TargetId: tbid,
			}

			optionals := models.MediaOptionals{
				AltText: "",
				Title:   "",
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			VideoMid, err := OurOptions.InsertMedia(Orm, media, optionals)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *tibbiBirimVideoInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
			}

			if VideoMid != "" {
				updateTB := Orm.Update()
				updateTB.Table("tibbi_birimler")
				updateTB.Set("video_mid", VideoMid)
				updateTB.Where("tbid", "=", tbid)
				updateTB.Finish()
				err = updateTB.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update tibbi_birim with media: %v\n", err)
					return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
				}
			}
		}

		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.Redirect("/panel/tibbi-birim-ekle?error=internal_server_error")
		}

		states.TibbiBirimlerLinks = []models.TibbiBirimLink{}

		return c.Redirect("/panel/tibbi-birimler/" + tbid)
	}
}

func EditTibbiBirim(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{"status": 401, "message": "Unauthorized"})
		}
		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{"status": 403, "message": "Forbidden: Admin access required"})
		}

		Tbid := c.Params("tbid")
		Orm := utilities.Orm

		inputs := models.TibbiBirimlerEdit{}
		c.BodyParser(&inputs)
		inputs.Tbid = Tbid

		// Required validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{"status": 400, "message": "Birim adı zorunludur."})
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}

		update := Orm.Update()
		update.Table("tibbi_birimler")

		somethingSet := false
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			update.Set("name", inputs.Name)
			somethingSet = true
		}
		if inputs.UrlName != inputs.OldUrlName {
			update.Set("url_name", inputs.UrlName)
			somethingSet = true
		}
		if inputs.Description != inputs.OldDescription {
			update.Set("description", inputs.Description)
			somethingSet = true
		}
		if inputs.IsActive != inputs.OldIsActive {
			update.Set("is_active", inputs.IsActive)
			somethingSet = true
		}
		if inputs.HtmlContent != inputs.OldHtmlContent {
			update.Set("html_content", inputs.HtmlContent)
			somethingSet = true
		}
		if inputs.JavascriptContent != inputs.OldJavascriptContent {
			update.Set("javascript_content", inputs.JavascriptContent)
			somethingSet = true
		}
		if inputs.CssContent != inputs.OldCssContent {
			update.Set("css_content", inputs.CssContent)
			somethingSet = true
		}

		if somethingSet {
			update.Set("updated_at", "NOW()")
			update.Where("tbid", "=", Tbid)
			update.Finish()
			err = update.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update tibbi_birim: %v\n", err)
				return c.JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
			}
		}

		Orm.Commit()
		states.TibbiBirimlerLinks = []models.TibbiBirimLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Tıbbi birim başarıyla güncellendi.",
		})
	}
}

func DeleteTibbiBirim(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		TibbiBirimId := c.Params("tbid")
		if TibbiBirimId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Branch ID is required",
			})
		}

		Orm := utilities.Orm

		// Fetch branch media information before deletion
		GetTibbiBirimMedia := Orm.Select([]string{"cover_mid", "video_mid"})
		GetTibbiBirimMedia.Table("tibbi_birimler")
		GetTibbiBirimMedia.Where("tbid", "=", TibbiBirimId)
		GetTibbiBirimMedia.Finish()
		err = GetTibbiBirimMedia.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		tibbiBirimRows, err := GetTibbiBirimMedia.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(tibbiBirimRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Tibbi birim not found",
			})
		}

		states.TibbiBirimlerLinks = []models.TibbiBirimLink{}

		var CoverMediaId int64
		if tibbiBirimRows[0]["cover_mid"] != nil {
			CoverMediaId = lib.Int64(tibbiBirimRows[0]["cover_mid"])
		}

		var VideoMediaId int64
		if tibbiBirimRows[0]["video_mid"] != nil {
			VideoMediaId = lib.Int64(tibbiBirimRows[0]["video_mid"])
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
		if CoverMediaId > 0 || VideoMediaId > 0 {
			// Get media file path
			GetMediaPath := Orm.Select([]string{"file_path"})
			GetMediaPath.Table("medias")
			GetMediaPath.Where("mid", "=", CoverMediaId)
			GetMediaPath.Or("mid", "=", VideoMediaId)
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

			MediaPaths := []any{}
			for _, mediaRow := range mediaRows {
				MediaPaths = append(MediaPaths, lib.String(mediaRow["file_path"]))
			}

			if len(MediaPaths) == 0 {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "Media not found",
				})
			}

			DeleteMedia := Orm.Delete()
			DeleteMedia.Table("medias")
			DeleteMedia.In("WHERE", "file_path", MediaPaths)
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

			for _, filePath := range MediaPaths {
				// Delete physical file
				fullPath := "static/" + lib.String(filePath)
				err = lib.DeleteFile(fullPath)
				if err != nil {
					log.Printf("Error deleting file %s: %v\n", fullPath, err)
				}
			}
			// Delete media record
		}

		// Delete the branch
		DeleteBranch := Orm.Delete()
		DeleteBranch.Table("tibbi_birimler")
		DeleteBranch.Where("tbid", "=", TibbiBirimId)
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

func UpdateTibbiBirimPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Tbid := c.Params("tbid")
		tibbiBirimCoverAltText := c.FormValue("cover_alt_text")
		tibbiBirimCoverTitle := c.FormValue("cover_title")
		oldTibbiBirimCoverAltText := c.FormValue("old_cover_alt_text")
		oldTibbiBirimCoverTitle := c.FormValue("old_cover_title")

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

		tibbiBirimCoverInput, err := c.FormFile("cover_mid")
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
			if tibbiBirimCoverInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "tibbi_birimler", Tbid, "cover")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + tibbiBirimCoverInput.Filename)

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
				FilePath: "files/tibbi_birimler/" + Tbid + "/cover/" + UniqueFilePath.BaseName,
				FileSize: tibbiBirimCoverInput.Size,
				MimeType: tibbiBirimCoverInput.Header.Get("Content-Type"),
				FileType: "tibbi_birim_cover",
				Uid:      OurUser.Uid,
				TargetId: Tbid,
			}

			optionals := models.MediaOptionals{
				AltText: tibbiBirimCoverAltText,
				Title:   tibbiBirimCoverTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			TibbiBirimCoverMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/tibbi_birimler/"+Tbid+"/cover", entry.Name())
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

			// Update tibbi_birim with new media ID
			UpdateTibbiBirim := Orm.Update()
			UpdateTibbiBirim.Table("tibbi_birimler")
			UpdateTibbiBirim.Set("cover_mid", TibbiBirimCoverMid)
			UpdateTibbiBirim.Where("tbid", "=", Tbid)
			UpdateTibbiBirim.Finish()
			err = UpdateTibbiBirim.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update tibbi_birim: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *tibbiBirimCoverInput)
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
			if tibbiBirimCoverAltText != oldTibbiBirimCoverAltText || tibbiBirimCoverTitle != oldTibbiBirimCoverTitle {
				// Get current media ID
				GetTibbiBirimMedia := Orm.Select([]string{"cover_mid"})
				GetTibbiBirimMedia.Table("tibbi_birimler")
				GetTibbiBirimMedia.Where("tbid", "=", Tbid)
				GetTibbiBirimMedia.Finish()
				err = GetTibbiBirimMedia.Execute()

				if err != nil {
					log.Printf("Cannot get tibbi_birim media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetTibbiBirimMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Tıbbi birim bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["cover_mid"])
				if mediaId > 0 {
					UpdateMedia := Orm.Update()
					UpdateMedia.Table("medias")

					if tibbiBirimCoverAltText != oldTibbiBirimCoverAltText {
						if tibbiBirimCoverAltText != "" {
							UpdateMedia.Set("alt_text", tibbiBirimCoverAltText)
						} else {
							UpdateMedia.Set("alt_text", nil)
						}
					}

					if tibbiBirimCoverTitle != oldTibbiBirimCoverTitle {
						if tibbiBirimCoverTitle != "" {
							UpdateMedia.Set("title", tibbiBirimCoverTitle)
						} else {
							UpdateMedia.Set("title", nil)
						}
					}

					UpdateMedia.Where("mid", "=", mediaId)
					UpdateMedia.And("target_id", "=", Tbid)
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
			"message": "Tıbbi birim görseli başarıyla güncellendi.",
		})
	}
}

func DeleteTibbiBirimPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Tbid := c.Params("tbid")
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
		GetTibbiBirimMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetTibbiBirimMedia.Table("tibbi_birimler tb")
		GetTibbiBirimMedia.LeftJoin("medias m", "tb.cover_mid", "=", "m.mid")
		GetTibbiBirimMedia.Where("tb.tbid", "=", Tbid)
		GetTibbiBirimMedia.Finish()
		err = GetTibbiBirimMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get tibbi_birim media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetTibbiBirimMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tıbbi birim bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tıbbi birim görseli bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", Tbid)
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
				"message": "Tıbbi birim görseli bulunamadı.",
			})
		}

		// Update tibbi_birim to remove media reference
		UpdateTibbiBirim := Orm.Update()
		UpdateTibbiBirim.Table("tibbi_birimler")
		UpdateTibbiBirim.Set("cover_mid", nil)
		UpdateTibbiBirim.Where("tbid", "=", Tbid)
		UpdateTibbiBirim.Finish()
		err = UpdateTibbiBirim.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update tibbi_birim: %v\n", err)
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
			"message": "Tıbbi birim görseli başarıyla silindi.",
		})
	}
}

func UpdateTibbiBirimVideo(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Tbid := c.Params("tbid")

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

		tibbiBirimVideoInput, err := c.FormFile("video_mid")
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
			if tibbiBirimVideoInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "tibbi_birimler", Tbid, "video")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + tibbiBirimVideoInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			switch UniqueFilePath.Extension {
			case ".mp4", ".webm", ".ogg", ".mov", ".avi", ".mkv", ".flv":
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
				FilePath: "files/tibbi_birimler/" + Tbid + "/video/" + UniqueFilePath.BaseName,
				FileSize: tibbiBirimVideoInput.Size,
				MimeType: tibbiBirimVideoInput.Header.Get("Content-Type"),
				FileType: "tibbi_birim_video",
				Uid:      OurUser.Uid,
				TargetId: Tbid,
			}

			optionals := models.MediaOptionals{
				AltText: "",
				Title:   "",
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			TibbiBirimVideoMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/tibbi_birimler/"+Tbid+"/video", entry.Name())
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

			// Update tibbi_birim with new media ID
			UpdateTibbiBirim := Orm.Update()
			UpdateTibbiBirim.Table("tibbi_birimler")
			UpdateTibbiBirim.Set("video_mid", TibbiBirimVideoMid)
			UpdateTibbiBirim.Where("tbid", "=", Tbid)
			UpdateTibbiBirim.Finish()
			err = UpdateTibbiBirim.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update tibbi_birim: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *tibbiBirimVideoInput)
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
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Tıbbi birim görseli başarıyla güncellendi.",
		})
	}
}

func DeleteTibbiBirimVideo(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Tbid := c.Params("tbid")
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
		GetTibbiBirimMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetTibbiBirimMedia.Table("tibbi_birimler tb")
		GetTibbiBirimMedia.LeftJoin("medias m", "tb.video_mid", "=", "m.mid")
		GetTibbiBirimMedia.Where("tb.tbid", "=", Tbid)
		GetTibbiBirimMedia.Finish()
		err = GetTibbiBirimMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get tibbi_birim media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetTibbiBirimMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tıbbi birim bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tıbbi birim görseli bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", Tbid)
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
				"message": "Tıbbi birim görseli bulunamadı.",
			})
		}

		// Update tibbi_birim to remove media reference
		UpdateTibbiBirim := Orm.Update()
		UpdateTibbiBirim.Table("tibbi_birimler")
		UpdateTibbiBirim.Set("video_mid", nil)
		UpdateTibbiBirim.Where("tbid", "=", Tbid)
		UpdateTibbiBirim.Finish()
		err = UpdateTibbiBirim.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update tibbi_birim: %v\n", err)
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
			"message": "Tıbbi birim görseli başarıyla silindi.",
		})
	}
}

func GetAllTibbiBirimler(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Orm := utilities.Orm
		GetAllTibbiBirimler := Orm.Select([]string{"tbid", "name"})
		GetAllTibbiBirimler.Table("tibbi_birimler")
		GetAllTibbiBirimler.Finish()

		err = GetAllTibbiBirimler.Execute()

		if err != nil {
			log.Printf("Cannot get all tibbi birimler: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetAllTibbiBirimler.Rows()
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Tıbbi birim bulunamadı.",
			})
		}

		TibbiBirimler := []models.TibbiBirimler{}
		for _, row := range rows {
			TibbiBirimler = append(TibbiBirimler, models.TibbiBirimler{
				Tbid: lib.String(row["tbid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Tıbbi birimler başarıyla getirildi.",
			"data":    TibbiBirimler,
		})
	}
}
