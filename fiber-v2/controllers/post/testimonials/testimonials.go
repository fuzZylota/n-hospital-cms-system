package testimonials

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

func AddTestimonial(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		// Admin mi değil mi kontrol ediyoruz.
		if OurUser.Role != "admin" {
			checkIfUserIsAdmin := Orm.Count("users")
			checkIfUserIsAdmin.Where("uid", "=", OurUser.Uid)
			checkIfUserIsAdmin.And("role", "=", "admin")
			checkIfUserIsAdmin.Finish()

			err = checkIfUserIsAdmin.Execute()

			if err != nil {
				log.Printf("Cannot check if user is admin: %v\n", err)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
			}

			if checkIfUserIsAdmin.Length() == 0 {
				log.Printf("User is not admin")
				return c.Redirect("/panel/musteri-yorumu-ekle?error=only_admins_can_add_testimonials")
			}
		}

		BackendOptions := database.Options{}
		uploadPolicy, err := uploadpolicy.Read(c.UserContext(), utilities.UploadPolicyReader)

		if err != nil {
			log.Print("Cannot get options")
			return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
		}

		inputs := models.Testimonials{}
		err = c.BodyParser(&inputs)

		if err != nil {
			log.Printf("Cannot parse body: %v\n", err)
			return c.Redirect("/panel/musteri-yorumu-ekle?error=invalid_inputs")
		}

		// Validate required fields
		if inputs.FirstName == "" {
			log.Printf("First name is required")
			return c.Redirect("/panel/musteri-yorumu-ekle?error=first_name_required")
		}

		if inputs.LastName == "" {
			log.Printf("Last name is required")
			return c.Redirect("/panel/musteri-yorumu-ekle?error=last_name_required")
		}

		if inputs.Occupation == "" {
			log.Printf("Occupation is required")
			return c.Redirect("/panel/musteri-yorumu-ekle?error=occupation_required")
		}

		if inputs.Content == "" {
			log.Printf("Content is required")
			return c.Redirect("/panel/musteri-yorumu-ekle?error=content_required")
		}

		if inputs.Rating < 0 || inputs.Rating > 5 {
			log.Printf("Rating must be between 0 and 5")
			return c.Redirect("/panel/musteri-yorumu-ekle?error=invalid_rating")
		}

		// Content length validation
		if len(inputs.Content) > 1000 {
			log.Printf("Content is too long")
			return c.Redirect("/panel/musteri-yorumu-ekle?error=content_too_long")
		}

		columns := []string{"first_name", "last_name", "occupation", "content", "rating", "is_active"}
		values := []interface{}{inputs.FirstName, inputs.LastName, inputs.Occupation, inputs.Content, inputs.Rating, inputs.IsActive}

		err = Orm.Begin()

		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
		}

		insertTestimonial := Orm.Insert(columns, values)
		insertTestimonial.Table("testimonials")
		insertTestimonial.Returning("tid")
		insertTestimonial.Finish()
		err = insertTestimonial.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert testimonial: %v\n", err)
			return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
		}

		tid, err := insertTestimonial.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
		}

		log.Printf("Inserted testimonial with tid: %v\n", tid)

		// Handle customer picture upload if provided
		var CustomerPictureMid string = ""
		customerPictureInput, err := c.FormFile("customer_picture_mid")
		if err == nil {
			// File uploaded and accessible
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				Orm.Rollback()
				log.Printf("Cannot get root directory")
				return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
			}

			if customerPictureInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				log.Printf("File size is too large: %v bytes\n", customerPictureInput.Size)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "testimonials", tid)

			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + customerPictureInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
			}

			// File type validation
			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", UniqueFilePath.Extension)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=invalid_file_type")
			}

			Media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/testimonials/" + tid + "/" + UniqueFilePath.BaseName,
				FileSize: customerPictureInput.Size,
				MimeType: customerPictureInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: tid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.CustomerPictureAltText,
				Title:   inputs.CustomerPictureTitle,
				Width:   0,
				Height:  0,
			}

			// Insert media record using the same pattern as options
			MediaMid, err := BackendOptions.InsertMedia(Orm, Media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
			}

			if MediaMid == "" {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
			}

			CustomerPictureMid = MediaMid
			// Save the actual file
			err = lib.SaveFileWithBuffering(estimatedPath, *customerPictureInput)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
			}

			// Update testimonial with customer picture mid
			updateTestimonial := Orm.Update()
			updateTestimonial.Table("testimonials")
			updateTestimonial.Set("customer_picture_mid", CustomerPictureMid)
			updateTestimonial.Where("tid", "=", tid)
			updateTestimonial.Finish()
			err = updateTestimonial.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update testimonial with picture: %v\n", err)
				return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
			}

			log.Printf("Updated testimonial with customer picture mid: %v\n", CustomerPictureMid)
		}

		err = Orm.Commit()

		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.Redirect("/panel/musteri-yorumu-ekle?error=internal_server_error")
		}

		return c.Redirect("/panel/musteri-yorumlari")
	}
}

func EditTestimonial(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Tid := c.Params("tid")

		Orm := utilities.Orm

		// Admin mi değil mi kontrol ediyoruz.
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
					"message": "Only admins can edit testimonials",
				})
			}

			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can edit testimonials",
			})
		}

		inputs := models.TestimonialsEdit{}
		c.BodyParser(&inputs)
		inputs.Tid = lib.String(Tid)

		Orm.Begin()

		updateTestimonial := Orm.Update()
		updateTestimonial.Table("testimonials")

		SomethingSet := false

		{
			if inputs.FirstName != inputs.OldFirstName && inputs.FirstName != "" {
				updateTestimonial.Set("first_name", inputs.FirstName)
				SomethingSet = true
			}

			if inputs.LastName != inputs.OldLastName && inputs.LastName != "" {
				updateTestimonial.Set("last_name", inputs.LastName)
				SomethingSet = true
			}

			if inputs.Occupation != inputs.OldOccupation && inputs.Occupation != "" {
				updateTestimonial.Set("occupation", inputs.Occupation)
				SomethingSet = true
			}

			if inputs.Content != inputs.OldContent && inputs.Content != "" {
				updateTestimonial.Set("content", inputs.Content)
				SomethingSet = true
			}

			if inputs.Rating != inputs.OldRating {
				updateTestimonial.Set("rating", inputs.Rating)
				SomethingSet = true
			}

			if inputs.IsActive != inputs.OldIsActive {
				updateTestimonial.Set("is_active", inputs.IsActive)
				SomethingSet = true
			}
		}

		if !SomethingSet {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Nothing changed",
			})
		}

		updateTestimonial.Where("tid", "=", Tid)
		updateTestimonial.Finish()

		err = updateTestimonial.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update testimonial: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Testimonial edited successfully",
		})
	}
}

func UpdateTestimonialPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		customerPictureAltText := c.FormValue("customer_picture_alt_text")
		customerPictureTitle := c.FormValue("customer_picture_title")
		oldCustomerPictureAltText := c.FormValue("old_customer_picture_alt_text")
		oldCustomerPictureTitle := c.FormValue("old_customer_picture_title")

		Tid := c.Params("tid")

		BackendOptions := database.Options{}
		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("Cannot fetch root directory\n")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		Orm := utilities.Orm

		uploadPolicy, err := uploadpolicy.Read(c.UserContext(), utilities.UploadPolicyReader)

		if err != nil {
			log.Print("Cannot get options")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Get current testimonial data
		getCurrentTestimonial := Orm.Select([]string{"customer_picture_mid"})
		getCurrentTestimonial.Table("testimonials")
		getCurrentTestimonial.Where("tid", "=", Tid)
		getCurrentTestimonial.Finish()
		err = getCurrentTestimonial.Execute()

		if err != nil {
			log.Printf("Cannot get current testimonial: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := getCurrentTestimonial.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get testimonial rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Testimonial not found",
			})
		}

		currentCustomerPictureMid := lib.Int64(rows[0]["customer_picture_mid"])

		customerPictureInput, err := c.FormFile("customer_picture_path")
		if err == nil {
			Orm.Begin()

			if customerPictureInput.Size > uploadPolicy.MaxBytes {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "File size is too large",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "testimonials", Tid)

			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + customerPictureInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Invalid file type",
				})
			}

			Media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/testimonials/" + Tid + "/" + UniqueFilePath.BaseName,
				FileSize: customerPictureInput.Size,
				MimeType: customerPictureInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: Tid,
			}

			optionals := models.MediaOptionals{
				AltText: customerPictureAltText,
				Title:   customerPictureTitle,
				Width:   0,
				Height:  0,
			}

			MediaMid, err := BackendOptions.InsertMedia(Orm, Media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if MediaMid == "" {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			// Delete old files if they exist
			if currentCustomerPictureMid != 0 {
				entries, err := lib.ReadDirectory(estimatedPath)
				if err != nil && err != os.ErrNotExist {
					log.Printf("Cannot read directory: %v\n", err)
				} else if err == nil {
					FilePaths := []any{}

					for _, entry := range entries {
						FilePath := filepath.Join("files/testimonials/"+Tid, entry.Name())
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

					// Delete actual files
					for _, filePath := range FilePaths {
						GetAbsolutePath := filepath.Join(RootDir, "static", filePath.(string))
						err := lib.DeleteFile(GetAbsolutePath)
						if err != nil {
							log.Printf("Cannot remove file: %v\n", err)
						}
					}
				}
			}

			UpdateTestimonial := Orm.Update()
			UpdateTestimonial.Table("testimonials")
			UpdateTestimonial.Set("customer_picture_mid", MediaMid)
			UpdateTestimonial.Where("tid", "=", Tid)
			UpdateTestimonial.Finish()
			err = UpdateTestimonial.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update testimonial: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			ra, err := UpdateTestimonial.RowsAffected()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows affected: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				log.Printf("Cannot update testimonial: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *customerPictureInput)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			Orm.Commit()
		} else {
			if currentCustomerPictureMid != 0 && (customerPictureAltText != oldCustomerPictureAltText || customerPictureTitle != oldCustomerPictureTitle) {
				UpdateMedia := Orm.Update()
				UpdateMedia.Table("medias")

				if customerPictureAltText != oldCustomerPictureAltText {
					if customerPictureAltText != "" {
						UpdateMedia.Set("alt_text", customerPictureAltText)
					} else {
						UpdateMedia.Set("alt_text", nil)
					}
				}

				if customerPictureTitle != oldCustomerPictureTitle {
					if customerPictureTitle != "" {
						UpdateMedia.Set("title", customerPictureTitle)
					} else {
						UpdateMedia.Set("title", nil)
					}
				}

				UpdateMedia.Where("mid", "=", currentCustomerPictureMid)
				UpdateMedia.And("target_id", "=", Tid)
				UpdateMedia.Finish()

				err = UpdateMedia.Execute()

				if err != nil {
					log.Printf("Cannot update media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Internal server error",
					})
				}
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Testimonial picture updated successfully",
		})
	}
}

func DeleteTestimonialPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Tid := c.Params("tid")
		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("Cannot fetch root directory\n")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		Orm := utilities.Orm

		// Get current testimonial data
		getCurrentTestimonial := Orm.Select([]string{"customer_picture_mid"})
		getCurrentTestimonial.Table("testimonials")
		getCurrentTestimonial.Where("tid", "=", Tid)
		getCurrentTestimonial.Finish()
		err = getCurrentTestimonial.Execute()

		if err != nil {
			log.Printf("Cannot get current testimonial: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := getCurrentTestimonial.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get testimonial rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Testimonial not found",
			})
		}

		currentCustomerPictureMid := lib.Int64(rows[0]["customer_picture_mid"])

		if currentCustomerPictureMid == 0 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "No picture to delete",
			})
		}

		Orm.Begin()

		// Delete media record
		deleteMedia := Orm.Delete()
		deleteMedia.Table("medias")
		deleteMedia.Where("mid", "=", currentCustomerPictureMid)
		deleteMedia.And("target_id", "=", Tid)
		deleteMedia.Finish()
		err = deleteMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete media record: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Update testimonial to remove picture reference
		updateTestimonial := Orm.Update()
		updateTestimonial.Table("testimonials")
		updateTestimonial.Set("customer_picture_mid", nil)
		updateTestimonial.Where("tid", "=", Tid)
		updateTestimonial.Finish()
		err = updateTestimonial.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update testimonial: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Delete actual files
		estimatedPath := filepath.Join(RootDir, "static", "files", "testimonials", Tid)
		entries, err := lib.ReadDirectory(estimatedPath)
		if err == nil {
			for _, entry := range entries {
				GetAbsolutePath := filepath.Join(estimatedPath, entry.Name())
				err := lib.DeleteFile(GetAbsolutePath)
				if err != nil {
					log.Printf("Cannot remove file: %v\n", err)
				}
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Testimonial picture deleted successfully",
		})
	}
}

func DeleteTestimonial(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Tid := c.Params("tid")

		Orm := utilities.Orm

		// admin mi değil mi kontrol ediyoruz.
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
					"message": "Only admins can delete testimonials",
				})
			}
		}

		// Get testimonial data before deletion for file cleanup
		Testimonials := Orm.Select([]string{"t.tid", "m.file_path as customer_picture_path"})
		Testimonials.Table("testimonials t")
		Testimonials.LeftJoin("medias m", "t.customer_picture_mid", "=", "m.mid")
		Testimonials.Where("t.tid", "=", Tid)
		Testimonials.Finish()

		err = Testimonials.Execute()

		if err != nil {
			log.Printf("Cannot fetch testimonial: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := Testimonials.Rows()
		if err != nil {
			log.Printf("Cannot get rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if len(rows) == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Testimonial not found",
			})
		}

		Orm.Begin()

		// Delete associated media if exists
		if len(rows) > 0 && rows[0]["customer_picture_path"] != nil && rows[0]["customer_picture_path"] != "" {
			DeleteMedia := Orm.Delete()
			DeleteMedia.Table("medias")
			DeleteMedia.Where("file_path", "=", lib.String(rows[0]["customer_picture_path"]))
			DeleteMedia.Finish()

			err = DeleteMedia.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot delete media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		// Delete testimonial
		DeleteTestimonial := Orm.Delete()
		DeleteTestimonial.Table("testimonials")
		DeleteTestimonial.Where("tid", "=", Tid)
		DeleteTestimonial.Finish()
		err = DeleteTestimonial.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete testimonial: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		ra, err := DeleteTestimonial.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get rows affected: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Testimonial not found",
			})
		}

		// Delete physical file if exists
		if len(rows) > 0 && rows[0]["customer_picture_path"] != nil && rows[0]["customer_picture_path"] != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			Path := filepath.Join(RootDir, "static", lib.String(rows[0]["customer_picture_path"]))

			err = lib.DeleteFile(Path)

			if err != nil {
				log.Printf("Cannot delete file: %v\n", err)
				// Don't rollback for file deletion errors, just log
				// The database operation was successful
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Testimonial deleted successfully",
		})
	}
}
