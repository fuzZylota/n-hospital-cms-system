package panel

import (
	"database"
	"fmt"
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func AddFilePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/dosya-ekle", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Dosya Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func ListFilesPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Get files from uploads directory
		RootDir := os.Getenv("ROOT_DIRECTORY")
		if RootDir == "" {
			// Try to get current working directory as fallback
			wd, err := os.Getwd()
			if err != nil {
				log.Printf("Cannot get current working directory: %v", err)
				return c.Render("views/panel/dosyalar", fiber.Map{
					"PathOnStart": "../",
					"PageTitle":   "N-Hospital | Dosyalar",
					"User":        ourUser,
					"Files":       []models.File{},
				}, "layouts/panel/panel")
			}
			RootDir = wd
			log.Printf("Using current working directory as ROOT_DIRECTORY: %s", RootDir)
		}

		// Read uploads directory
		uploadsDir := filepath.Join(RootDir, "static/uploads")
		entries, err := lib.ReadDirectory(uploadsDir)
		if err != nil {
			log.Printf("Cannot read uploads directory: %v\n", err)
			return c.Render("views/panel/dosyalar", fiber.Map{
				"PathOnStart": "../",
				"PageTitle":   "N-Hospital | Dosyalar",
				"User":        ourUser,
				"Files":       []models.File{},
			}, "layouts/panel/panel")
		}

		var Files []models.File
		for _, entry := range entries {
			if !entry.IsDir() {
				if entry.Name() == ".gitkeep" {
					continue
				}

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
					fileType = "document"
				case ".doc", ".docx", ".txt", ".rtf":
					fileType = "document"
				case ".xls", ".xlsx":
					fileType = "spreadsheet"
				case ".ppt", ".pptx":
					fileType = "presentation"
				case ".zip", ".rar", ".7z", ".tar", ".gz":
					fileType = "archive"
				}

				// Determine icon based on file type and extension
				Icon := "fas fa-file"
				switch fileType {
				case "image":
					Icon = "fas fa-image"
				case "video":
					Icon = "fas fa-video"
				case "audio":
					Icon = "fas fa-music"
				case "document":
					if ext == ".pdf" {
						Icon = "fas fa-file-pdf"
					} else {
						Icon = "fas fa-file-word"
					}
				case "archive":
					Icon = "fas fa-file-archive"
				case "spreadsheet":
					Icon = "fas fa-file-excel"
				case "presentation":
					Icon = "fas fa-file-powerpoint"
				}

				Files = append(Files, models.File{
					Name: entry.Name(),
					Url:  "/uploads/" + entry.Name(),
					Size: fileInfo.Size(),
					Type: fileType,
					Icon: Icon,
				})
			}
		}

		return c.Render("views/panel/dosyalar", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Dosyalar",
			"User":        ourUser,
			"Files":       Files,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func PanelPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch all statistics in a single query
		GetStatistics := Orm.CustomSelectQuery(`
		SELECT
			-- Users statistics
			(SELECT COUNT(*) FROM users) as total_users,
			(SELECT COUNT(*) FROM users WHERE is_active = true) as active_users,
			(SELECT COUNT(*) FROM users WHERE is_active = false) as inactive_users,
			(SELECT COUNT(*) FROM users WHERE role = 'admin') as admin_users,
			(SELECT COUNT(*) FROM users WHERE role = 'moderator') as moderator_users,
			
			-- Branches statistics
			(SELECT COUNT(*) FROM subeler) as total_branches,
			(SELECT COUNT(*) FROM subeler WHERE is_active = true) as active_branches,
			(SELECT COUNT(*) FROM subeler WHERE is_active = false) as inactive_branches,
			(SELECT COUNT(*) FROM subeler WHERE is_main = true) as main_branch,
			
			-- Doctors statistics
			(SELECT COUNT(*) FROM doktorlar) as total_doctors,
			(SELECT COUNT(*) FROM doktorlar WHERE is_active = true) as active_doctors,
			(SELECT COUNT(*) FROM doktorlar WHERE is_active = false) as inactive_doctors,
			(SELECT COUNT(*) FROM doktorlar WHERE online_appointment = true) as online_appointment_doctors,
			
			-- Departments statistics
			(SELECT COUNT(*) FROM branslar) as total_departments,
			(SELECT COUNT(*) FROM branslar WHERE is_active = true) as active_departments,
			(SELECT COUNT(*) FROM branslar WHERE is_active = false) as inactive_departments,
			(SELECT COUNT(*) FROM branslar WHERE head_drid IS NOT NULL) as departments_with_head,
			
			-- Appointments statistics
			(SELECT COUNT(*) FROM randevular) as total_appointments,
			(SELECT COUNT(*) FROM randevular WHERE status = 'beklemede') as pending_appointments,
			(SELECT COUNT(*) FROM randevular WHERE status = 'onaylandi') as confirmed_appointments,
			(SELECT COUNT(*) FROM randevular WHERE status = 'tamamlandi') as completed_appointments,
			(SELECT COUNT(*) FROM randevular WHERE status = 'iptal') as cancelled_appointments,
			(SELECT COUNT(*) FROM randevular WHERE appointment_date = CURRENT_DATE) as today_appointments,
			(SELECT COUNT(*) FROM randevular WHERE appointment_date >= CURRENT_DATE - INTERVAL '7 days' AND appointment_date <= CURRENT_DATE) as week_appointments,
			(SELECT COUNT(*) FROM randevular WHERE EXTRACT(MONTH FROM appointment_date) = EXTRACT(MONTH FROM CURRENT_DATE) AND EXTRACT(YEAR FROM appointment_date) = EXTRACT(YEAR FROM CURRENT_DATE)) as month_appointments,
			
			-- Appointment Requests statistics
			(SELECT COUNT(*) FROM randevu_talepleri) as total_appointment_requests,
			(SELECT COUNT(*) FROM randevu_talepleri WHERE DATE(created_at) = CURRENT_DATE) as today_appointment_requests,
			(SELECT COUNT(*) FROM randevu_talepleri WHERE created_at >= CURRENT_DATE - INTERVAL '7 days') as week_appointment_requests,
			(SELECT COUNT(*) FROM randevu_talepleri WHERE EXTRACT(MONTH FROM created_at) = EXTRACT(MONTH FROM CURRENT_DATE) AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE)) as month_appointment_requests,
			
			-- Contact Requests statistics
			(SELECT COUNT(*) FROM contact_requests) as total_contact_requests,
			(SELECT COUNT(*) FROM contact_requests WHERE is_read = false) as unread_contact_requests,
			(SELECT COUNT(*) FROM contact_requests WHERE is_replied = true) as replied_contact_requests,
			
			-- Job Applications statistics
			(SELECT COUNT(*) FROM job_applications) as total_job_applications,
			(SELECT COUNT(*) FROM job_applications WHERE is_read = true) as read_job_applications,
			(SELECT COUNT(*) FROM job_applications WHERE is_read = false) as unread_job_applications,
			
			-- News statistics
			(SELECT COUNT(*) FROM haberler) as total_news,
			(SELECT COUNT(*) FROM haberler WHERE is_published = true) as published_news,
			(SELECT COUNT(*) FROM haberler WHERE is_published = false) as draft_news,
			(SELECT COUNT(*) FROM haberler WHERE is_featured = true) as featured_news,
			(SELECT COALESCE(SUM(views_count), 0) FROM haberler) as total_news_views,
			
			-- Contracted Institutions statistics
			(SELECT COUNT(*) FROM anlasmali_kurumlar) as total_contracted_institutions,
			(SELECT COUNT(*) FROM anlasmali_kurumlar WHERE is_active = true) as active_contracted_institutions,
			(SELECT COUNT(*) FROM anlasmali_kurumlar WHERE is_active = false) as inactive_contracted_institutions,
			(SELECT COUNT(*) FROM anlasmali_kurumlar WHERE type = 'sigorta') as insurance_institutions,
			(SELECT COUNT(*) FROM anlasmali_kurumlar WHERE type = 'kurumsal') as corporate_institutions,
			(SELECT COUNT(*) FROM anlasmali_kurumlar WHERE contract_end_date IS NOT NULL AND contract_end_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days') as expiring_soon_contracts,
			
			-- Medical Units statistics
			(SELECT COUNT(*) FROM tibbi_birimler) as total_medical_units,
			(SELECT COUNT(*) FROM tibbi_birimler WHERE is_active = true) as active_medical_units,
			(SELECT COUNT(*) FROM tibbi_birimler WHERE is_active = false) as inactive_medical_units,
			
			-- Tests statistics
			(SELECT COUNT(*) FROM tedkikler) as total_tests,
			(SELECT COUNT(*) FROM tedkikler WHERE is_active = true) as active_tests,
			(SELECT COUNT(*) FROM tedkikler WHERE is_active = false) as inactive_tests,
			
			-- Specializations statistics
			(SELECT COUNT(*) FROM uzmanliklar) as total_specializations,
			(SELECT COUNT(*) FROM uzmanliklar WHERE is_active = true) as active_specializations,
			(SELECT COUNT(*) FROM uzmanliklar WHERE is_active = false) as inactive_specializations,
			
			-- Doctor Experiences statistics
			(SELECT COUNT(*) FROM doctor_experiences) as total_doctor_experiences,
			(SELECT COUNT(*) FROM doctor_experiences WHERE is_active = true) as active_doctor_experiences,
			(SELECT COUNT(*) FROM doctor_experiences WHERE is_active = false) as inactive_doctor_experiences,
			
			-- Doctor Expertises statistics
			(SELECT COUNT(*) FROM doctor_expertises) as total_doctor_expertises,
			(SELECT COUNT(*) FROM doctor_expertises WHERE is_primary = true) as primary_doctor_expertises,
			
			-- Header Buttons statistics
			(SELECT COUNT(*) FROM header_buttons) as total_header_buttons,
			(SELECT COUNT(*) FROM header_buttons WHERE is_active = true) as active_header_buttons,
			(SELECT COUNT(*) FROM header_buttons WHERE is_active = false) as inactive_header_buttons,
			(SELECT COUNT(*) FROM header_buttons WHERE parent_id IS NULL) as parent_header_buttons,
			(SELECT COUNT(*) FROM header_buttons WHERE parent_id IS NOT NULL) as child_header_buttons,
			
			-- Testimonials statistics
			(SELECT COUNT(*) FROM testimonials) as total_testimonials,
			(SELECT COUNT(*) FROM testimonials WHERE is_active = true) as active_testimonials,
			(SELECT COUNT(*) FROM testimonials WHERE is_active = false) as inactive_testimonials,
			
			-- Media Files statistics
			(SELECT COUNT(*) FROM medias) as total_media_files,
			(SELECT COUNT(*) FROM medias WHERE file_type = 'image') as image_files,
			(SELECT COUNT(*) FROM medias WHERE file_type = 'video') as video_files,
			(SELECT COUNT(*) FROM medias WHERE file_type = 'document') as document_files,
			(SELECT COALESCE(SUM(file_size), 0) FROM medias) as total_media_size,
			
			-- Homepage Contents statistics
			(SELECT COUNT(*) FROM homepage_contents) as total_homepage_contents,
			(SELECT COUNT(*) FROM homepage_contents WHERE is_active = true) as active_homepage_contents,
			(SELECT COUNT(*) FROM homepage_contents WHERE is_active = false) as inactive_homepage_contents,
			(SELECT COUNT(*) FROM homepage_contents WHERE content_type = 'popup') as popup_homepage_contents,
			
			-- Custom Contents statistics
			(SELECT COUNT(*) FROM custom_contents) as total_custom_contents,
			(SELECT COUNT(*) FROM custom_contents WHERE is_active = true) as active_custom_contents,
			(SELECT COUNT(*) FROM custom_contents WHERE is_active = false) as inactive_custom_contents,
			(SELECT COUNT(*) FROM custom_contents WHERE content_type = 'main-page') as main_page_custom_contents,
			(SELECT COUNT(*) FROM custom_contents WHERE content_type = 'history') as history_custom_contents,
			(SELECT COUNT(*) FROM custom_contents WHERE content_type = 'about-us') as about_us_custom_contents,
			
			-- Options statistics
			(SELECT COUNT(*) FROM options) as total_option_sets,
			(SELECT COUNT(*) FROM options WHERE option_set_is_active = true) as active_option_sets,
			(SELECT COUNT(*) FROM options WHERE option_set_is_active = false) as inactive_option_sets,
			(SELECT COUNT(*) FROM options WHERE option_set_is_testing_now = true) as testing_option_sets,
			
			-- Notifications statistics
			(SELECT COUNT(*) FROM notifications) as total_notifications,
			(SELECT COUNT(*) FROM notifications WHERE is_read = false) as unread_notifications,
			(SELECT COUNT(*) FROM notifications WHERE is_read = true) as read_notifications,
			
			-- Financial statistics
			(SELECT COALESCE(SUM(price), 0) FROM randevular WHERE payment_status = 'odendi') as total_revenue,
			(SELECT COALESCE(SUM(price), 0) FROM randevular WHERE payment_status = 'odendi' AND EXTRACT(MONTH FROM created_at) = EXTRACT(MONTH FROM CURRENT_DATE) AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE)) as month_revenue,
			(SELECT COALESCE(SUM(price), 0) FROM randevular WHERE payment_status = 'odendi' AND DATE(created_at) = CURRENT_DATE) as today_revenue,
			(SELECT COALESCE(AVG(price), 0) FROM randevular WHERE price > 0) as average_appointment_price
	`)

		GetStatistics.Execute()

		rows, err := GetStatistics.Rows()
		if err != nil {
			log.Printf("Error getting statistics rows: %v\n", err)
			return c.Redirect("/giris")
		}

		PanelStatistics := models.PanelStatistics{}

		if len(rows) > 0 {
			row := rows[0]

			// Users statistics
			PanelStatistics.TotalUsers = int(lib.Int64(row["total_users"]))
			PanelStatistics.ActiveUsers = int(lib.Int64(row["active_users"]))
			PanelStatistics.InactiveUsers = int(lib.Int64(row["inactive_users"]))
			PanelStatistics.AdminUsers = int(lib.Int64(row["admin_users"]))
			PanelStatistics.ModeratorUsers = int(lib.Int64(row["moderator_users"]))

			// Branches statistics
			PanelStatistics.TotalBranches = int(lib.Int64(row["total_branches"]))
			PanelStatistics.ActiveBranches = int(lib.Int64(row["active_branches"]))
			PanelStatistics.InactiveBranches = int(lib.Int64(row["inactive_branches"]))
			PanelStatistics.MainBranch = int(lib.Int64(row["main_branch"]))

			// Doctors statistics
			PanelStatistics.TotalDoctors = int(lib.Int64(row["total_doctors"]))
			PanelStatistics.ActiveDoctors = int(lib.Int64(row["active_doctors"]))
			PanelStatistics.InactiveDoctors = int(lib.Int64(row["inactive_doctors"]))
			PanelStatistics.OnlineAppointmentDoctors = int(lib.Int64(row["online_appointment_doctors"]))

			// Departments statistics
			PanelStatistics.TotalDepartments = int(lib.Int64(row["total_departments"]))
			PanelStatistics.ActiveDepartments = int(lib.Int64(row["active_departments"]))
			PanelStatistics.InactiveDepartments = int(lib.Int64(row["inactive_departments"]))
			PanelStatistics.DepartmentsWithHead = int(lib.Int64(row["departments_with_head"]))

			// Appointments statistics
			PanelStatistics.TotalAppointments = int(lib.Int64(row["total_appointments"]))
			PanelStatistics.PendingAppointments = int(lib.Int64(row["pending_appointments"]))
			PanelStatistics.ConfirmedAppointments = int(lib.Int64(row["confirmed_appointments"]))
			PanelStatistics.CompletedAppointments = int(lib.Int64(row["completed_appointments"]))
			PanelStatistics.CancelledAppointments = int(lib.Int64(row["cancelled_appointments"]))
			PanelStatistics.TodayAppointments = int(lib.Int64(row["today_appointments"]))
			PanelStatistics.WeekAppointments = int(lib.Int64(row["week_appointments"]))
			PanelStatistics.MonthAppointments = int(lib.Int64(row["month_appointments"]))

			// Appointment Requests statistics
			PanelStatistics.TotalAppointmentRequests = int(lib.Int64(row["total_appointment_requests"]))
			PanelStatistics.TodayAppointmentRequests = int(lib.Int64(row["today_appointment_requests"]))
			PanelStatistics.WeekAppointmentRequests = int(lib.Int64(row["week_appointment_requests"]))
			PanelStatistics.MonthAppointmentRequests = int(lib.Int64(row["month_appointment_requests"]))

			// Contact Requests statistics
			PanelStatistics.TotalContactRequests = int(lib.Int64(row["total_contact_requests"]))
			PanelStatistics.UnreadContactRequests = int(lib.Int64(row["unread_contact_requests"]))
			PanelStatistics.RepliedContactRequests = int(lib.Int64(row["replied_contact_requests"]))

			// Job Applications statistics
			PanelStatistics.TotalJobApplications = int(lib.Int64(row["total_job_applications"]))
			PanelStatistics.ReadJobApplications = int(lib.Int64(row["read_job_applications"]))
			PanelStatistics.UnreadJobApplications = int(lib.Int64(row["unread_job_applications"]))

			// News statistics
			PanelStatistics.TotalNews = int(lib.Int64(row["total_news"]))
			PanelStatistics.PublishedNews = int(lib.Int64(row["published_news"]))
			PanelStatistics.DraftNews = int(lib.Int64(row["draft_news"]))
			PanelStatistics.FeaturedNews = int(lib.Int64(row["featured_news"]))
			PanelStatistics.TotalNewsViews = int(lib.Int64(row["total_news_views"]))

			// Contracted Institutions statistics
			PanelStatistics.TotalContractedInstitutions = int(lib.Int64(row["total_contracted_institutions"]))
			PanelStatistics.ActiveContractedInstitutions = int(lib.Int64(row["active_contracted_institutions"]))
			PanelStatistics.InactiveContractedInstitutions = int(lib.Int64(row["inactive_contracted_institutions"]))
			PanelStatistics.InsuranceInstitutions = int(lib.Int64(row["insurance_institutions"]))
			PanelStatistics.CorporateInstitutions = int(lib.Int64(row["corporate_institutions"]))
			PanelStatistics.ExpiringSoonContracts = int(lib.Int64(row["expiring_soon_contracts"]))

			// Medical Units statistics
			PanelStatistics.TotalMedicalUnits = int(lib.Int64(row["total_medical_units"]))
			PanelStatistics.ActiveMedicalUnits = int(lib.Int64(row["active_medical_units"]))
			PanelStatistics.InactiveMedicalUnits = int(lib.Int64(row["inactive_medical_units"]))

			// Tests statistics
			PanelStatistics.TotalTests = int(lib.Int64(row["total_tests"]))
			PanelStatistics.ActiveTests = int(lib.Int64(row["active_tests"]))
			PanelStatistics.InactiveTests = int(lib.Int64(row["inactive_tests"]))

			// Specializations statistics
			PanelStatistics.TotalSpecializations = int(lib.Int64(row["total_specializations"]))
			PanelStatistics.ActiveSpecializations = int(lib.Int64(row["active_specializations"]))
			PanelStatistics.InactiveSpecializations = int(lib.Int64(row["inactive_specializations"]))

			// Doctor Experiences statistics
			PanelStatistics.TotalDoctorExperiences = int(lib.Int64(row["total_doctor_experiences"]))
			PanelStatistics.ActiveDoctorExperiences = int(lib.Int64(row["active_doctor_experiences"]))
			PanelStatistics.InactiveDoctorExperiences = int(lib.Int64(row["inactive_doctor_experiences"]))

			// Doctor Expertises statistics
			PanelStatistics.TotalDoctorExpertises = int(lib.Int64(row["total_doctor_expertises"]))
			PanelStatistics.PrimaryDoctorExpertises = int(lib.Int64(row["primary_doctor_expertises"]))

			// Header Buttons statistics
			PanelStatistics.TotalHeaderButtons = int(lib.Int64(row["total_header_buttons"]))
			PanelStatistics.ActiveHeaderButtons = int(lib.Int64(row["active_header_buttons"]))
			PanelStatistics.InactiveHeaderButtons = int(lib.Int64(row["inactive_header_buttons"]))
			PanelStatistics.ParentHeaderButtons = int(lib.Int64(row["parent_header_buttons"]))
			PanelStatistics.ChildHeaderButtons = int(lib.Int64(row["child_header_buttons"]))

			// Testimonials statistics
			PanelStatistics.TotalTestimonials = int(lib.Int64(row["total_testimonials"]))
			PanelStatistics.ActiveTestimonials = int(lib.Int64(row["active_testimonials"]))
			PanelStatistics.InactiveTestimonials = int(lib.Int64(row["inactive_testimonials"]))

			// Media Files statistics
			PanelStatistics.TotalMediaFiles = int(lib.Int64(row["total_media_files"]))
			PanelStatistics.ImageFiles = int(lib.Int64(row["image_files"]))
			PanelStatistics.VideoFiles = int(lib.Int64(row["video_files"]))
			PanelStatistics.DocumentFiles = int(lib.Int64(row["document_files"]))
			PanelStatistics.TotalMediaSize = lib.Int64(row["total_media_size"])

			// Homepage Contents statistics
			PanelStatistics.TotalHomepageContents = int(lib.Int64(row["total_homepage_contents"]))
			PanelStatistics.ActiveHomepageContents = int(lib.Int64(row["active_homepage_contents"]))
			PanelStatistics.InactiveHomepageContents = int(lib.Int64(row["inactive_homepage_contents"]))
			PanelStatistics.PopupHomepageContents = int(lib.Int64(row["popup_homepage_contents"]))

			// Custom Contents statistics
			PanelStatistics.TotalCustomContents = int(lib.Int64(row["total_custom_contents"]))
			PanelStatistics.ActiveCustomContents = int(lib.Int64(row["active_custom_contents"]))
			PanelStatistics.InactiveCustomContents = int(lib.Int64(row["inactive_custom_contents"]))
			PanelStatistics.MainPageCustomContents = int(lib.Int64(row["main_page_custom_contents"]))
			PanelStatistics.HistoryCustomContents = int(lib.Int64(row["history_custom_contents"]))
			PanelStatistics.AboutUsCustomContents = int(lib.Int64(row["about_us_custom_contents"]))

			// Options statistics
			PanelStatistics.TotalOptionSets = int(lib.Int64(row["total_option_sets"]))
			PanelStatistics.ActiveOptionSets = int(lib.Int64(row["active_option_sets"]))
			PanelStatistics.InactiveOptionSets = int(lib.Int64(row["inactive_option_sets"]))
			PanelStatistics.TestingOptionSets = int(lib.Int64(row["testing_option_sets"]))

			// Notifications statistics
			PanelStatistics.TotalNotifications = int(lib.Int64(row["total_notifications"]))
			PanelStatistics.UnreadNotifications = int(lib.Int64(row["unread_notifications"]))
			PanelStatistics.ReadNotifications = int(lib.Int64(row["read_notifications"]))

			// Financial statistics
			PanelStatistics.TotalRevenue = lib.Float64(row["total_revenue"])
			PanelStatistics.MonthRevenue = lib.Float64(row["month_revenue"])
			PanelStatistics.TodayRevenue = lib.Float64(row["today_revenue"])
			PanelStatistics.AverageAppointmentPrice = lib.Float64(row["average_appointment_price"])
		}

		// Fetch doctors by title separately as it returns multiple rows
		GetDoctorsByTitle := Orm.CustomSelectQuery(`
		SELECT title, COUNT(*) as count
		FROM doktorlar
		GROUP BY title
	`)
		GetDoctorsByTitle.Execute()
		doctorTitleRows, err := GetDoctorsByTitle.Rows()
		if err == nil && len(doctorTitleRows) > 0 {
			PanelStatistics.DoctorsByTitle = make(map[string]int)
			for _, titleRow := range doctorTitleRows {
				title := lib.String(titleRow["title"])
				count := int(lib.Int64(titleRow["count"]))
				PanelStatistics.DoctorsByTitle[title] = count
			}
		}

		return c.Render("views/panel/panel", fiber.Map{
			"PathOnStart": "",
			"PageTitle":   "N-Hospital | Yönetim Paneli",
			"User":        ourUser,
			"Options":     GetOptions,
			"Statistics":  PanelStatistics,
		}, "layouts/panel/panel")
	}
}

func SeceneklerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Query := c.Query("query", "")
		Status := c.Query("status", "all")
		SortBy := c.Query("sort_by", "option_set_updated_at")
		SortOrder := c.Query("sort_order", "DESC")

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		Options := Orm.Select([]string{"oid", "option_set_name", "option_set_description", "option_set_is_active", "option_set_updated_at", "option_set_created_at", "site_name", "site_description", "contact_email", "contact_phone"})
		Options.Table("options")

		if Query != "" {
			Options.OpenParenthesis("WHERE")
			Options.Like("WHERE", "option_set_name", Query, "contains")
			Options.Like("OR", "option_set_description", Query, "contains")
			Options.Like("OR", "site_name", Query, "contains")
			Options.Like("OR", "site_description", Query, "contains")
			Options.Like("OR", "contact_email", Query, "contains")
			Options.CloseParenthesis()

			if Status != "all" {
				Options.And("option_set_is_active", "=", Status == "active")
			}
		}

		if Status != "all" {
			if strings.Contains(Options.Query, "WHERE") {
				Options.And("option_set_is_active", "=", Status == "active")
			} else {
				Options.Where("option_set_is_active", "=", Status == "active")
			}
		}

		Options.OrderBy(SortBy, SortOrder)
		Options.Limit(int(itemsPerPage))
		Options.Offset(offset)
		Options.Finish()

		err = Options.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Options.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		OptionsArray := []models.Options{}
		for _, row := range rows {
			OptionsArray = append(OptionsArray, models.Options{
				Oid:                  lib.String(row["oid"]),
				OptionSetName:        lib.String(row["option_set_name"]),
				OptionSetDescription: lib.String(row["option_set_description"]),
				OptionSetIsActive:    row["option_set_is_active"].(bool),
				OptionSetUpdatedAt:   row["option_set_updated_at"].(time.Time),
				OptionSetCreatedAt:   row["option_set_created_at"].(time.Time),
				SiteName:             lib.String(row["site_name"]),
				SiteDescription:      lib.String(row["site_description"]),
				ContactEmail:         row["contact_email"].(string),
				ContactPhone:         lib.String(row["contact_phone"]),
			})
		}

		return c.Render("views/panel/secenek-sayfalari/secenekler", fiber.Map{
			"PathOnStart":  "../",
			"PageTitle":    "N-Hospital | Seçenekler",
			"Page":         c.Query("page"),
			"OptionsToGet": OptionsArray,
			"Count":        len(OptionsArray),
			"User":         ourUser,
			"Options":      GetOptions,
		}, "layouts/panel/panel")
	}
}

func SecenekPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		log.Printf("ourUser Log'u: %v\n", ourUser)

		if err != nil {
			log.Printf("that error occured on redirect: %v\n", err)
			return c.Redirect("/giris")
		}

		Oid := c.Params("secenek")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Options := Orm.Select([]string{"o.*", "m.file_path as logo_path", "m.alt_text as logo_alt_text", "m.title as logo_title", "m2.file_path as favicon_path", "m3.file_path as default_page_media_path", "m3.alt_text as default_page_media_alt_text", "m3.title as default_page_media_title"})
		Options.Table("options o")
		Options.LeftJoin("medias m", "o.site_logo_mid", "=", "m.mid")
		Options.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
		Options.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
		Options.Where("o.oid", "=", Oid)
		Options.Finish()
		err = Options.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Options.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Option := models.Options{
			Oid:                           lib.String(rows[0]["oid"]),
			OptionSetName:                 lib.String(rows[0]["option_set_name"]),
			OptionSetDescription:          lib.String(rows[0]["option_set_description"]),
			OptionSetIsActive:             rows[0]["option_set_is_active"].(bool),
			SiteName:                      lib.String(rows[0]["site_name"]),
			SiteDescription:               lib.String(rows[0]["site_description"]),
			ContactEmail:                  lib.String(rows[0]["contact_email"]),
			ContactPhone:                  lib.String(rows[0]["contact_phone"]),
			SiteLogoMid:                   lib.Int64(rows[0]["site_logo_mid"]),
			FaviconMid:                    lib.Int64(rows[0]["site_favicon_mid"]),
			DefaultPageMid:                lib.Int64(rows[0]["default_page_mid"]),
			MaintenanceMode:               rows[0]["maintenance_mode"].(bool),
			SMTPHost:                      lib.String(rows[0]["smtp_host"]),
			SMTPPort:                      lib.Int64(rows[0]["smtp_port"]),
			SMTPUsername:                  lib.String(rows[0]["smtp_username"]),
			SMTPPassword:                  lib.String(rows[0]["smtp_password"]),
			SMTPEncryption:                lib.String(rows[0]["smtp_encryption"]),
			FacebookUrl:                   lib.String(rows[0]["facebook_url"]),
			TwitterUrl:                    lib.String(rows[0]["twitter_url"]),
			InstagramUrl:                  lib.String(rows[0]["instagram_url"]),
			LinkedinUrl:                   lib.String(rows[0]["linkedin_url"]),
			MainPageMetaTitle:             lib.String(rows[0]["main_page_meta_title"]),
			MainPageMetaDescription:       lib.String(rows[0]["main_page_meta_description"]),
			GoogleAnalytics:               lib.String(rows[0]["google_analytics"]),
			PrimaryColor:                  lib.String(rows[0]["primary_color"]),
			SecondaryColor:                lib.String(rows[0]["secondary_color"]),
			AccentColor:                   lib.String(rows[0]["accent_color"]),
			BackgroundColor:               lib.String(rows[0]["background_color"]),
			FontColor:                     lib.String(rows[0]["font_color"]),
			FontFamily:                    lib.String(rows[0]["font_family"]),
			RequireStrongPassword:         rows[0]["require_strong_password"].(bool),
			ItemsPerPage:                  lib.Int64(rows[0]["items_per_page"]),
			ShowDoctorsOnSameCity:         rows[0]["show_doctors_on_same_city"].(bool),
			ShowDoctorsOnSameCountry:      rows[0]["show_doctors_on_same_country"].(bool),
			AutoRemovePartnersWhenExpired: rows[0]["auto_remove_partners_when_expired"].(bool),
			MaxUploadSize:                 lib.Int64(rows[0]["max_upload_size"]),
			Timezone:                      lib.String(rows[0]["timezone"]),
			Language:                      lib.String(rows[0]["language"]),
			EnableTestimonials:            rows[0]["enable_testimonials"].(bool),
			EnableOurHistory:              rows[0]["enable_our_history"].(bool),
			MaximumSublinksOnAMenuItem:    lib.Int64(rows[0]["maximum_sublinks_on_a_menu_item"]),
			OptionSetUpdatedAt:            rows[0]["option_set_updated_at"].(time.Time),
			OptionSetCreatedAt:            rows[0]["option_set_created_at"].(time.Time),
			SiteLogoPath:                  lib.String(rows[0]["logo_path"]),
			SiteLogoAltText:               lib.String(rows[0]["logo_alt_text"]),
			SiteLogoTitle:                 lib.String(rows[0]["logo_title"]),
			SiteFaviconPath:               lib.String(rows[0]["favicon_path"]),
			DefaultPageMediaPath:          lib.String(rows[0]["default_page_media_path"]),
			DefaultPageMediaAltText:       lib.String(rows[0]["default_page_media_alt_text"]),
			DefaultPageMediaTitle:         lib.String(rows[0]["default_page_media_title"]),
		}

		return c.Render("views/panel/secenek-sayfalari/secenek", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Seçenek",
			"User":        ourUser,
			"Option":      Option,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func SecenekEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/secenek-sayfalari/secenek-ekle", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Seçenek Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func SecenekDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Oid := c.Params("secenek")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Options := Orm.Select([]string{"o.*", "m.file_path as logo_path", "m.alt_text as logo_alt_text", "m.title as logo_title", "m2.file_path as favicon_path", "m3.file_path as default_page_media_path", "m3.alt_text as default_page_media_alt_text", "m3.title as default_page_media_title"})
		Options.Table("options o")
		Options.LeftJoin("medias m", "o.site_logo_mid", "=", "m.mid")
		Options.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
		Options.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
		Options.Where("o.oid", "=", Oid)
		Options.Finish()
		err = Options.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Options.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Option := models.Options{
			Oid:                           lib.String(rows[0]["oid"]),
			OptionSetName:                 lib.String(rows[0]["option_set_name"]),
			OptionSetDescription:          lib.String(rows[0]["option_set_description"]),
			OptionSetIsActive:             lib.Bool(rows[0]["option_set_is_active"]),
			OptionSetIsTestingNow:         lib.Bool(rows[0]["option_set_is_testing_now"]),
			SiteName:                      lib.String(rows[0]["site_name"]),
			SiteDescription:               lib.String(rows[0]["site_description"]),
			ContactEmail:                  lib.String(rows[0]["contact_email"]),
			ContactPhone:                  lib.String(rows[0]["contact_phone"]),
			SiteLogoMid:                   lib.Int64(rows[0]["site_logo_mid"]),
			FaviconMid:                    lib.Int64(rows[0]["site_favicon_mid"]),
			DefaultPageMid:                lib.Int64(rows[0]["default_page_mid"]),
			MaintenanceMode:               rows[0]["maintenance_mode"].(bool),
			SMTPHost:                      lib.String(rows[0]["smtp_host"]),
			SMTPPort:                      lib.Int64(rows[0]["smtp_port"]),
			SMTPUsername:                  lib.String(rows[0]["smtp_username"]),
			SMTPPassword:                  lib.String(rows[0]["smtp_password"]),
			SMTPEncryption:                lib.String(rows[0]["smtp_encryption"]),
			FacebookUrl:                   lib.String(rows[0]["facebook_url"]),
			TwitterUrl:                    lib.String(rows[0]["twitter_url"]),
			InstagramUrl:                  lib.String(rows[0]["instagram_url"]),
			LinkedinUrl:                   lib.String(rows[0]["linkedin_url"]),
			MainPageMetaTitle:             lib.String(rows[0]["main_page_meta_title"]),
			MainPageMetaDescription:       lib.String(rows[0]["main_page_meta_description"]),
			GoogleAnalytics:               lib.String(rows[0]["google_analytics"]),
			PrimaryColor:                  lib.String(rows[0]["primary_color"]),
			SecondaryColor:                lib.String(rows[0]["secondary_color"]),
			AccentColor:                   lib.String(rows[0]["accent_color"]),
			BackgroundColor:               lib.String(rows[0]["background_color"]),
			FontColor:                     lib.String(rows[0]["font_color"]),
			FontFamily:                    lib.String(rows[0]["font_family"]),
			RequireStrongPassword:         lib.Bool(rows[0]["require_strong_password"]),
			ItemsPerPage:                  lib.Int64(rows[0]["items_per_page"]),
			ShowDoctorsOnSameCity:         lib.Bool(rows[0]["show_doctors_on_same_city"]),
			ShowDoctorsOnSameCountry:      lib.Bool(rows[0]["show_doctors_on_same_country"]),
			AutoRemovePartnersWhenExpired: lib.Bool(rows[0]["auto_remove_partners_when_expired"]),
			MaxUploadSize:                 lib.Int64(rows[0]["max_upload_size"]),
			Timezone:                      lib.String(rows[0]["timezone"]),
			Language:                      lib.String(rows[0]["language"]),
			EnableTestimonials:            lib.Bool(rows[0]["enable_testimonials"]),
			EnableOurHistory:              lib.Bool(rows[0]["enable_our_history"]),
			MaximumSublinksOnAMenuItem:    lib.Int64(rows[0]["maximum_sublinks_on_a_menu_item"]),
			OptionSetUpdatedAt:            lib.Time(rows[0]["option_set_updated_at"]),
			OptionSetCreatedAt:            lib.Time(rows[0]["option_set_created_at"]),
			ShowDoctorSocialMedia:         lib.Bool(rows[0]["show_doctor_social_media"]),
			ShowDoctorAppointmentFee:      lib.Bool(rows[0]["show_doctor_appointment_fee"]),
			SiteLogoPath:                  lib.String(rows[0]["logo_path"]),
			SiteLogoAltText:               lib.String(rows[0]["logo_alt_text"]),
			SiteLogoTitle:                 lib.String(rows[0]["logo_title"]),
			SiteFaviconPath:               lib.String(rows[0]["favicon_path"]),
			DefaultPageMediaPath:          lib.String(rows[0]["default_page_media_path"]),
			DefaultPageMediaAltText:       lib.String(rows[0]["default_page_media_alt_text"]),
			DefaultPageMediaTitle:         lib.String(rows[0]["default_page_media_title"]),
		}

		return c.Render("views/panel/secenek-sayfalari/secenek-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | Seçenek Düzenle",
			"User":        ourUser,
			"Option":      Option,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func KullanicilarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1

		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))

			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		BackendOptions := database.Options{}

		BackendOptions, err = BackendOptions.FetchOptionsForPanel(utilities.Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		itemsPerPage := BackendOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		Users := utilities.Orm.Select([]string{"uid", "name", "surname", "email", "phone", "role", "is_active", "last_login"})
		Users.Table("users")
		Users.Limit(int(itemsPerPage))
		Users.Offset(offset)
		Users.Finish()
		err = Users.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Users.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		UsersArray := []models.Users{}
		for _, row := range rows {
			UsersArray = append(UsersArray, models.Users{
				Uid:       lib.String(row["uid"]),
				Email:     row["email"].(string),
				Phone:     row["phone"].(string),
				Name:      row["name"].(string),
				Surname:   row["surname"].(string),
				Role:      row["role"].(string),
				IsActive:  row["is_active"].(bool),
				LastLogin: row["last_login"].(time.Time),
			})
		}

		return c.Render("views/panel/kullanici-sayfalari/kullanicilar", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Kullanıcılar",
			"Page":        c.Query("page"),
			"Users":       UsersArray,
			"Count":       len(UsersArray),
			"User":        ourUser,
			"Options":     BackendOptions,
		}, "layouts/panel/panel")
	}
}

func KullaniciPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Oid := c.Params("kullanici")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Users := Orm.Select("*")
		Users.Table("users")
		Users.Where("uid", "=", Oid)
		Users.Finish()
		err = Users.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Users.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		User := models.Users{
			Uid:       lib.String(rows[0]["uid"]),
			Email:     lib.String(rows[0]["email"]),
			Password:  lib.String(rows[0]["password"]),
			Phone:     lib.String(rows[0]["phone"]),
			Name:      lib.String(rows[0]["name"]),
			Surname:   lib.String(rows[0]["surname"]),
			Role:      lib.String(rows[0]["role"]),
			Timezone:  lib.String(rows[0]["timezone"]),
			IsActive:  rows[0]["is_active"].(bool),
			LastLogin: rows[0]["last_login"].(time.Time),
		}

		return c.Render("views/panel/kullanici-sayfalari/kullanici", fiber.Map{
			"PathOnStart":    "../../",
			"PageTitle":      "N-Hospital | Kullanıcı",
			"User":           ourUser,
			"IndividualUser": User,
			"Options":        GetOptions,
		}, "layouts/panel/panel")
	}
}

func KullaniciEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/kullanici-sayfalari/kullanici-ekle", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Kullanıcı Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func KullaniciDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Oid := c.Params("kullanici")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Users := Orm.Select("*")
		Users.Table("users")
		Users.Where("uid", "=", Oid)
		Users.Finish()
		err = Users.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Users.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		User := models.Users{
			Uid:       lib.String(rows[0]["uid"]),
			Email:     lib.String(rows[0]["email"]),
			Phone:     lib.String(rows[0]["phone"]),
			Name:      lib.String(rows[0]["name"]),
			Surname:   lib.String(rows[0]["surname"]),
			Role:      lib.String(rows[0]["role"]),
			IsActive:  rows[0]["is_active"].(bool),
			LastLogin: rows[0]["last_login"].(time.Time),
		}

		return c.Render("views/panel/kullanici-sayfalari/kullanici-duzenle", fiber.Map{
			"PathOnStart":    "../../../",
			"PageTitle":      "N-Hospital | Kullanıcı Düzenle",
			"User":           ourUser,
			"IndividualUser": User,
			"Options":        GetOptions,
		}, "layouts/panel/panel")
	}
}

func HeaderTuslariPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1

		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))

			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		BackendOptions := database.Options{}

		BackendOptions, err = BackendOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		itemsPerPage := BackendOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		Users := Orm.Select([]string{"hbid", "title", "url", "target", "icon", "sort_order", "is_active", "parent_id"})
		Users.Table("header_buttons")
		Users.Limit(int(itemsPerPage))
		Users.Offset(offset)
		Users.Finish()
		err = Users.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Users.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		UsersArray := []models.HeaderButton{}
		for _, row := range rows {
			UsersArray = append(UsersArray, models.HeaderButton{
				Hbid:      lib.String(row["hbid"]),
				Title:     lib.String(row["title"]),
				Url:       lib.String(row["url"]),
				Target:    lib.String(row["target"]),
				Icon:      lib.String(row["icon"]),
				SortOrder: lib.Int64(row["sort_order"]),
				IsActive:  lib.Bool(row["is_active"]),
				ParentId:  lib.String(row["parent_id"]),
				CreatedAt: lib.Time(row["created_at"]),
				UpdatedAt: lib.Time(row["updated_at"]),
			})
		}

		GlobalCountOfHeaderButtons := Users.Count("header_buttons")
		GlobalCountOfHeaderButtons.Finish()
		err = GlobalCountOfHeaderButtons.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		CountOfHeaderButtons := GlobalCountOfHeaderButtons.Length()

		return c.Render("views/panel/header-tuslari-sayfalari/header-tuslari", fiber.Map{
			"PathOnStart":   "../",
			"PageTitle":     "N-Hospital | Header Tuşları",
			"User":          ourUser,
			"HeaderButtons": UsersArray,
			"Count":         CountOfHeaderButtons,
			"Page":          Page,
			"Options":       BackendOptions,
		}, "layouts/panel/panel")
	}
}

func HeaderTusPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Oid := c.Params("hbid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		HeaderButtons := Orm.Select([]string{"hbid", "title", "url", "target", "icon", "sort_order", "is_active", "parent_id", "button_type"})
		HeaderButtons.Table("header_buttons")
		HeaderButtons.Where("hbid", "=", Oid)
		HeaderButtons.Finish()
		err = HeaderButtons.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := HeaderButtons.Rows()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		HeaderButton := models.HeaderButton{
			Hbid:       lib.String(rows[0]["hbid"]),
			Title:      lib.String(rows[0]["title"]),
			Url:        lib.String(rows[0]["url"]),
			Target:     lib.String(rows[0]["target"]),
			Icon:       lib.String(rows[0]["icon"]),
			SortOrder:  lib.Int64(rows[0]["sort_order"]),
			IsActive:   lib.Bool(rows[0]["is_active"]),
			ButtonType: lib.String(rows[0]["button_type"]),
			ParentId:   lib.String(rows[0]["parent_id"]),
			CreatedAt:  lib.Time(rows[0]["created_at"]),
			UpdatedAt:  lib.Time(rows[0]["updated_at"]),
		}

		ParentButtons := Orm.Select([]string{"hbid", "title"})
		ParentButtons.Table("header_buttons")
		ParentButtons.Where("parent_id", "=", nil)
		ParentButtons.Finish()
		err = ParentButtons.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err = ParentButtons.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		ParentButtonsArray := []models.HeaderButton{}
		for _, row := range rows {
			ParentButtonsArray = append(ParentButtonsArray, models.HeaderButton{
				Hbid:  lib.String(row["hbid"]),
				Title: lib.String(row["title"]),
			})
		}

		return c.Render("views/panel/header-tuslari-sayfalari/header-tusu", fiber.Map{
			"PathOnStart":            "../../",
			"PageTitle":              "N-Hospital | Header Tuş",
			"User":                   ourUser,
			"IndividualHeaderButton": HeaderButton,
			"ParentButtons":          ParentButtonsArray,
			"Options":                GetOptions,
		}, "layouts/panel/panel")
	}
}

func HeaderTusEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		GetParentButtons := Orm.Select([]string{"hbid", "title"})
		GetParentButtons.Table("header_buttons")
		GetParentButtons.Where("parent_id", "=", nil)
		GetParentButtons.Finish()
		err = GetParentButtons.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := GetParentButtons.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		ParentButtons := []models.HeaderButton{}
		for _, row := range rows {
			ParentButtons = append(ParentButtons, models.HeaderButton{
				Hbid:  lib.String(row["hbid"]),
				Title: lib.String(row["title"]),
			})
		}

		return c.Render("views/panel/header-tuslari-sayfalari/header-tusu-ekle", fiber.Map{
			"PathOnStart":   "../",
			"PageTitle":     "N-Hospital | Header Tuş Ekle",
			"User":          ourUser,
			"ParentButtons": ParentButtons,
			"Options":       GetOptions,
		}, "layouts/panel/panel")
	}
}

func HeaderTusDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Oid := c.Params("hbid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		HeaderButtons := Orm.Select([]string{"hbid", "title", "url", "target", "icon", "sort_order", "is_active", "parent_id", "button_type"})
		HeaderButtons.Table("header_buttons")
		HeaderButtons.Where("hbid", "=", Oid)
		HeaderButtons.Finish()
		err = HeaderButtons.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := HeaderButtons.Rows()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		HeaderButton := models.HeaderButton{
			Hbid:       lib.String(rows[0]["hbid"]),
			Title:      lib.String(rows[0]["title"]),
			Url:        lib.String(rows[0]["url"]),
			Target:     lib.String(rows[0]["target"]),
			Icon:       lib.String(rows[0]["icon"]),
			SortOrder:  lib.Int64(rows[0]["sort_order"]),
			IsActive:   lib.Bool(rows[0]["is_active"]),
			ButtonType: lib.String(rows[0]["button_type"]),
			ParentId:   lib.String(rows[0]["parent_id"]),
			CreatedAt:  lib.Time(rows[0]["created_at"]),
			UpdatedAt:  lib.Time(rows[0]["updated_at"]),
		}

		ParentButtons := Orm.Select([]string{"hbid", "title"})
		ParentButtons.Table("header_buttons")
		ParentButtons.Where("parent_id", "=", nil)
		ParentButtons.Finish()
		err = ParentButtons.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err = ParentButtons.Rows()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		ParentButtonsArray := []models.HeaderButton{}
		for _, row := range rows {
			ParentButtonsArray = append(ParentButtonsArray, models.HeaderButton{
				Hbid:  lib.String(row["hbid"]),
				Title: lib.String(row["title"]),
			})
		}

		return c.Render("views/panel/header-tuslari-sayfalari/header-tusu-duzenle", fiber.Map{
			"PathOnStart":            "../../../",
			"PageTitle":              "N-Hospital | Header Tuş Düzenle",
			"User":                   ourUser,
			"IndividualHeaderButton": HeaderButton,
			"ParentButtons":          ParentButtonsArray,
			"Options":                GetOptions,
		}, "layouts/panel/panel")
	}
}

func TestimonialsPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		BackendOptions := database.Options{}
		BackendOptions, err = BackendOptions.FetchOptionsForPanel(utilities.Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		itemsPerPage := BackendOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		TestimonialsSel := utilities.Orm.Select([]string{"t.tid", "t.first_name", "t.last_name", "t.occupation", "t.content", "t.is_active", "t.created_at", "m.file_path as customer_picture_path", "m.alt_text as customer_picture_alt_text", "m.title as customer_picture_title"})
		TestimonialsSel.Table("testimonials t")
		TestimonialsSel.LeftJoin("medias m", "t.customer_picture_mid", "=", "m.mid")
		TestimonialsSel.Limit(int(itemsPerPage))
		TestimonialsSel.Offset(offset)
		TestimonialsSel.Finish()
		err = TestimonialsSel.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := TestimonialsSel.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		TestimonialsArray := []models.Testimonials{}
		for _, row := range rows {
			content := lib.String(row["content"])
			if len(content) > 120 {
				content = content[:117] + "..."
			}
			TestimonialsArray = append(TestimonialsArray, models.Testimonials{
				Tid:                    lib.String(row["tid"]),
				FirstName:              lib.String(row["first_name"]),
				LastName:               lib.String(row["last_name"]),
				Occupation:             lib.String(row["occupation"]),
				Content:                content,
				Rating:                 lib.Float64(row["rating"]),
				IsActive:               lib.Bool(row["is_active"]),
				CustomerPictureMid:     lib.Int64(row["customer_picture_mid"]),
				CustomerPicturePath:    lib.String(row["customer_picture_path"]),
				CustomerPictureAltText: lib.String(row["customer_picture_alt_text"]),
				CustomerPictureTitle:   lib.String(row["customer_picture_title"]),
				CreatedAt:              lib.Time(row["created_at"]),
				UpdatedAt:              lib.Time(row["updated_at"]),
			})
		}

		return c.Render("views/panel/testimonials/musteri-yorumlari", fiber.Map{
			"PathOnStart":  "../",
			"PageTitle":    "N-Hospital | Müşteri Yorumları",
			"Page":         c.Query("page"),
			"Testimonials": TestimonialsArray,
			"Count":        len(TestimonialsArray),
			"User":         ourUser,
			"Options":      BackendOptions,
		}, "layouts/panel/panel")
	}
}

func TestimonialsEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/testimonials/musteri-yorumu-ekle", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Müşteri Yorumları",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TestimonialPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Tid := c.Params("tid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Testimonials := Orm.Select([]string{"t.*", "m.file_path as customer_picture_path", "m.alt_text as customer_picture_alt_text", "m.title as customer_picture_title"})
		Testimonials.Table("testimonials t")
		Testimonials.LeftJoin("medias m", "t.customer_picture_mid", "=", "m.mid")
		Testimonials.Where("t.tid", "=", Tid)
		Testimonials.Finish()
		err = Testimonials.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Testimonials.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/musteri-yorumlari")
		}

		Testimonial := models.Testimonials{
			Tid:                    lib.String(rows[0]["tid"]),
			FirstName:              lib.String(rows[0]["first_name"]),
			LastName:               lib.String(rows[0]["last_name"]),
			Occupation:             lib.String(rows[0]["occupation"]),
			Content:                lib.String(rows[0]["content"]),
			Rating:                 lib.Float64(rows[0]["rating"]),
			IsActive:               rows[0]["is_active"].(bool),
			CustomerPictureMid:     lib.Int64(rows[0]["customer_picture_mid"]),
			CreatedAt:              rows[0]["created_at"].(time.Time),
			UpdatedAt:              rows[0]["updated_at"].(time.Time),
			CustomerPicturePath:    lib.String(rows[0]["customer_picture_path"]),
			CustomerPictureAltText: lib.String(rows[0]["customer_picture_alt_text"]),
			CustomerPictureTitle:   lib.String(rows[0]["customer_picture_title"]),
		}

		return c.Render("views/panel/testimonials/musteri-yorumu", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Müşteri Yorumu",
			"User":        ourUser,
			"Options":     GetOptions,
			"Testimonial": Testimonial,
		}, "layouts/panel/panel")
	}
}

func TestimonialsDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Tid := c.Params("tid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Testimonials := Orm.Select([]string{"t.*", "m.file_path as customer_picture_path", "m.alt_text as customer_picture_alt_text", "m.title as customer_picture_title"})
		Testimonials.Table("testimonials t")
		Testimonials.LeftJoin("medias m", "t.customer_picture_mid", "=", "m.mid")
		Testimonials.Where("t.tid", "=", Tid)
		Testimonials.Finish()
		err = Testimonials.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Testimonials.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/musteri-yorumlari")
		}

		Testimonial := models.Testimonials{
			Tid:                    lib.String(rows[0]["tid"]),
			FirstName:              lib.String(rows[0]["first_name"]),
			LastName:               lib.String(rows[0]["last_name"]),
			Occupation:             lib.String(rows[0]["occupation"]),
			Content:                lib.String(rows[0]["content"]),
			Rating:                 lib.Float64(rows[0]["rating"]),
			IsActive:               rows[0]["is_active"].(bool),
			CustomerPictureMid:     lib.Int64(rows[0]["customer_picture_mid"]),
			CreatedAt:              rows[0]["created_at"].(time.Time),
			UpdatedAt:              rows[0]["updated_at"].(time.Time),
			CustomerPicturePath:    lib.String(rows[0]["customer_picture_path"]),
			CustomerPictureAltText: lib.String(rows[0]["customer_picture_alt_text"]),
			CustomerPictureTitle:   lib.String(rows[0]["customer_picture_title"]),
		}

		return c.Render("views/panel/testimonials/musteri-yorumu-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | Müşteri Yorumu Düzenle",
			"User":        ourUser,
			"Options":     GetOptions,
			"Testimonial": Testimonial,
		}, "layouts/panel/panel")
	}
}

func SubelerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		Subeler := Orm.Select([]string{"sid", "name", "url_name", "description", "address", "city", "district", "postal_code", "phone", "fax", "email", "website", "latitude", "longitude", "working_hours", "mid", "is_main", "is_active", "created_at", "updated_at"})
		Subeler.Table("subeler")
		Subeler.OrderBy("is_main", "DESC")
		Subeler.OrderBy("sid", "DESC")
		Subeler.Limit(int(itemsPerPage))
		Subeler.Offset(offset)
		Subeler.Finish()

		err = Subeler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := Subeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		SubelerArray := []models.Subeler{}
		for _, row := range rows {
			SubelerArray = append(SubelerArray, models.Subeler{
				Sid:          lib.String(row["sid"]),
				Name:         lib.String(row["name"]),
				UrlName:      lib.String(row["url_name"]),
				Description:  lib.String(row["description"]),
				Address:      lib.String(row["address"]),
				City:         lib.String(row["city"]),
				District:     lib.String(row["district"]),
				PostalCode:   lib.String(row["postal_code"]),
				Phone:        lib.String(row["phone"]),
				Fax:          lib.String(row["fax"]),
				Email:        lib.String(row["email"]),
				Website:      lib.String(row["website"]),
				Latitude:     lib.Float64(row["latitude"]),
				Longitude:    lib.Float64(row["longitude"]),
				WorkingHours: lib.String(row["working_hours"]),
				Mid:          lib.Int64(row["mid"]),
				IsMain:       row["is_main"].(bool),
				IsActive:     row["is_active"].(bool),
				CreatedAt:    row["created_at"].(time.Time),
				UpdatedAt:    row["updated_at"].(time.Time),
			})
		}

		return c.Render("views/panel/subeler/subeler", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Şubeler",
			"Page":        c.Query("page"),
			"Subeler":     SubelerArray,
			"Count":       len(SubelerArray),
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func SubelerEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/subeler/sube-ekle", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Şube Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func SubePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		SubeId := c.Params("sid")

		if SubeId == "" {
			return c.Redirect("/panel/subeler")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the branch with media information
		Sube := Orm.Select([]string{"s.*", "m.file_path as sube_media_path", "m.alt_text as sube_media_alt_text", "m.title as sube_media_title"})
		Sube.Table("subeler s")
		Sube.LeftJoin("medias m", "s.mid", "=", "m.mid")
		Sube.Where("s.sid", "=", SubeId)
		Sube.Finish()
		err = Sube.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/subeler")
		}

		rows, err := Sube.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/subeler")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/subeler")
		}

		SubeData := models.Subeler{
			Sid:              lib.String(rows[0]["sid"]),
			Name:             lib.String(rows[0]["name"]),
			UrlName:          lib.String(rows[0]["url_name"]),
			Description:      lib.String(rows[0]["description"]),
			Address:          lib.String(rows[0]["address"]),
			City:             lib.String(rows[0]["city"]),
			District:         lib.String(rows[0]["district"]),
			PostalCode:       lib.String(rows[0]["postal_code"]),
			Phone:            lib.String(rows[0]["phone"]),
			Fax:              lib.String(rows[0]["fax"]),
			DocumentMids:     lib.StringArray(rows[0]["document_mids"]),
			Email:            lib.String(rows[0]["email"]),
			Website:          lib.String(rows[0]["website"]),
			Latitude:         lib.Float64(rows[0]["latitude"]),
			Longitude:        lib.Float64(rows[0]["longitude"]),
			WorkingHours:     lib.String(rows[0]["working_hours"]),
			Mid:              lib.Int64(rows[0]["mid"]),
			IsMain:           rows[0]["is_main"].(bool),
			IsActive:         rows[0]["is_active"].(bool),
			CreatedAt:        rows[0]["created_at"].(time.Time),
			UpdatedAt:        rows[0]["updated_at"].(time.Time),
			SubeMediaPath:    lib.String(rows[0]["sube_media_path"]),
			SubeMediaAltText: lib.String(rows[0]["sube_media_alt_text"]),
			SubeMediaTitle:   lib.String(rows[0]["sube_media_title"]),
		}

		SubeMids := []any{}
		if len(SubeData.DocumentMids) > 0 {
			for _, mid := range SubeData.DocumentMids {
				SubeMids = append(SubeMids, mid)
			}
		}

		SubeDocumentsArray := []models.Medias{}
		if len(SubeMids) > 0 {
			SubeDocuments := Orm.Select([]string{"mid", "file_path", "file_name", "file_size", "mime_type", "data"})
			SubeDocuments.Table("medias")
			SubeDocuments.In("WHERE", "mid", SubeMids)
			SubeDocuments.Finish()
			err = SubeDocuments.Execute()
			if err != nil {
				log.Printf("Cannot get sube documents: %v\n", err)
			}

			subeDocumentsRows, err := SubeDocuments.Rows()
			if err != nil {
				log.Printf("Cannot get sube documents rows: %v\n", err)
			}

			for _, row := range subeDocumentsRows {
				SubeDocumentsArray = append(SubeDocumentsArray, models.Medias{
					Mid:      lib.Int64(row["mid"]),
					FilePath: lib.String(row["file_path"]),
					FileName: lib.String(row["file_name"]),
					FileSize: lib.Int64(row["file_size"]),
					MimeType: lib.String(row["mime_type"]),
					Data:     lib.String(row["data"]),
				})
			}
		}

		AnlasmaliKurumlar := Orm.Select([]string{"ak.akid", "ak.name", "ak.url_name", "ak.type", "ak.contact_person", "ak.phone", "ak.email", "ak.sid", "ak.is_active", "ak.created_at", "ak.updated_at"})
		AnlasmaliKurumlar.Table("anlasmali_kurumlar ak")
		AnlasmaliKurumlar.Where("ak.sid", "=", SubeId)
		AnlasmaliKurumlar.OrderBy("ak.name", "ASC")
		AnlasmaliKurumlar.Finish()
		err = AnlasmaliKurumlar.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		anlasmaliKurumlarRows, err := AnlasmaliKurumlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		AnlasmaliKurumlarArray := []models.AnlasmaliKurumlar{}
		for _, row := range anlasmaliKurumlarRows {
			AnlasmaliKurumlarArray = append(AnlasmaliKurumlarArray, models.AnlasmaliKurumlar{
				Akid:              lib.String(row["akid"]),
				Name:              lib.String(row["name"]),
				UrlName:           lib.String(row["url_name"]),
				Type:              lib.String(row["type"]),
				ContactPerson:     lib.String(row["contact_person"]),
				Phone:             lib.String(row["phone"]),
				Email:             lib.String(row["email"]),
				Address:           lib.String(row["address"]),
				ContractStartDate: lib.Time(row["contract_start_date"]),
				ContractEndDate:   lib.Time(row["contract_end_date"]),
				DiscountRate:      lib.Float64(row["discount_rate"]),
				PaymentTerms:      lib.String(row["payment_terms"]),
				Notes:             lib.String(row["notes"]),
				LogoMid:           lib.Int64(row["logo_mid"]),
				LogoPath:          lib.String(row["logo_path"]),
				LogoAltText:       lib.String(row["logo_alt_text"]),
				LogoTitle:         lib.String(row["logo_title"]),
				IsActive:          row["is_active"].(bool),
				CreatedAt:         row["created_at"].(time.Time),
				UpdatedAt:         row["updated_at"].(time.Time),
			})
		}

		// Fetch doctors in this branch
		Doktorlar := Orm.Select([]string{"d.drid", "d.title", "d.first_name", "d.last_name", "d.phone", "d.email", "d.is_active"})
		Doktorlar.Table("doktorlar d")
		Doktorlar.Where("d.sid", "=", SubeId)
		Doktorlar.And("d.is_active", "=", true)
		Doktorlar.OrderBy("d.first_name", "ASC")
		Doktorlar.Finish()

		err = Doktorlar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		doktorlarRows, err := Doktorlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoktorlarArray := []models.Doktorlar{}
		for _, row := range doktorlarRows {
			DoktorlarArray = append(DoktorlarArray, models.Doktorlar{
				Drid:      lib.String(row["drid"]),
				Title:     lib.String(row["title"]),
				FirstName: lib.String(row["first_name"]),
				LastName:  lib.String(row["last_name"]),
				Phone:     lib.String(row["phone"]),
				Email:     lib.String(row["email"]),
				IsActive:  row["is_active"].(bool),
			})
		}

		// Fetch branches in this branch
		Branslar := Orm.Select([]string{"br.brid", "br.name", "br.phone", "br.email", "br.is_active"})
		Branslar.Table("branslar br")
		Branslar.Where("br.sid", "=", SubeId)
		Branslar.And("br.is_active", "=", true)
		Branslar.OrderBy("br.name", "ASC")
		Branslar.Finish()
		err = Branslar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		branslarRows, err := Branslar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		BranslarArray := []models.Branslar{}
		for _, row := range branslarRows {
			BranslarArray = append(BranslarArray, models.Branslar{
				Brid:     lib.String(row["brid"]),
				Name:     lib.String(row["name"]),
				Phone:    lib.String(row["phone"]),
				Email:    lib.String(row["email"]),
				IsActive: row["is_active"].(bool),
			})
		}

		// Fetch recent appointments for this branch
		Randevular := Orm.Select([]string{"rid", "patient_first_name", "patient_last_name", "patient_phone", "appointment_date", "appointment_time", "status"})
		Randevular.Table("randevular")
		Randevular.Where("sid", "=", SubeId)
		Randevular.OrderBy("appointment_date", "DESC")
		Randevular.Limit(10)
		Randevular.Finish()
		err = Randevular.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		randevularRows, err := Randevular.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		RandevularArray := []models.Randevular{}
		for _, row := range randevularRows {
			RandevularArray = append(RandevularArray, models.Randevular{
				Rid:              lib.String(row["rid"]),
				PatientFirstName: lib.String(row["patient_first_name"]),
				PatientLastName:  lib.String(row["patient_last_name"]),
				PatientPhone:     lib.String(row["patient_phone"]),
				AppointmentDate:  row["appointment_date"].(time.Time),
				AppointmentTime:  row["appointment_time"].(time.Time),
				Status:           lib.String(row["status"]),
			})
		}

		return c.Render("views/panel/subeler/sube", fiber.Map{
			"PathOnStart":       "../../",
			"PageTitle":         "N-Hospital | " + SubeData.Name,
			"Options":           GetOptions,
			"User":              ourUser,
			"Sube":              SubeData,
			"Doktorlar":         DoktorlarArray,
			"Branslar":          BranslarArray,
			"Randevular":        RandevularArray,
			"AnlasmaliKurumlar": AnlasmaliKurumlarArray,
			"SubeDocuments":     SubeDocumentsArray,
		}, "layouts/panel/panel")
	}
}

func SubeDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Sid := c.Params("sid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		SubeQuery := Orm.Select([]string{"s.*", "m.file_path as sube_media_path", "m.alt_text as sube_media_alt_text", "m.title as sube_media_title"})
		SubeQuery.Table("subeler s")
		SubeQuery.LeftJoin("medias m", "s.mid", "=", "m.mid")
		SubeQuery.LeftJoin("medias md", "md.mid", "=", "ANY(s.document_mids)")
		SubeQuery.Where("s.sid", "=", Sid)
		SubeQuery.Finish()
		err = SubeQuery.Execute()

		if err != nil {
			log.Printf("Cannot get sube: %v\n", err)
			return c.Redirect("/panel/subeler")
		}

		rows, err := SubeQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get sube rows: %v\n", err)
			return c.Redirect("/panel/subeler")
		}

		Sube := models.Subeler{
			Sid:              lib.String(rows[0]["sid"]),
			Name:             lib.String(rows[0]["name"]),
			UrlName:          lib.String(rows[0]["url_name"]),
			Description:      lib.String(rows[0]["description"]),
			Address:          lib.String(rows[0]["address"]),
			City:             lib.String(rows[0]["city"]),
			District:         lib.String(rows[0]["district"]),
			PostalCode:       lib.String(rows[0]["postal_code"]),
			Phone:            lib.String(rows[0]["phone"]),
			Fax:              lib.String(rows[0]["fax"]),
			Email:            lib.String(rows[0]["email"]),
			Website:          lib.String(rows[0]["website"]),
			Latitude:         lib.Float64(rows[0]["latitude"]),
			Longitude:        lib.Float64(rows[0]["longitude"]),
			WorkingHours:     lib.String(rows[0]["working_hours"]),
			Mid:              lib.Int64(rows[0]["mid"]),
			DocumentMids:     lib.StringArray(rows[0]["document_mids"]),
			IsMain:           rows[0]["is_main"].(bool),
			IsActive:         rows[0]["is_active"].(bool),
			CreatedAt:        rows[0]["created_at"].(time.Time),
			UpdatedAt:        rows[0]["updated_at"].(time.Time),
			SubeMediaPath:    lib.String(rows[0]["sube_media_path"]),
			SubeMediaAltText: lib.String(rows[0]["sube_media_alt_text"]),
			SubeMediaTitle:   lib.String(rows[0]["sube_media_title"]),
		}

		SubeMids := []any{}
		if len(Sube.DocumentMids) > 0 {
			for _, mid := range Sube.DocumentMids {
				SubeMids = append(SubeMids, mid)
			}
		}

		SubeDocumentsArray := []models.Medias{}
		if len(SubeMids) > 0 {
			SubeDocuments := Orm.Select([]string{"mid", "file_path", "file_name", "file_size", "mime_type", "data"})
			SubeDocuments.Table("medias")
			SubeDocuments.In("WHERE", "mid", SubeMids)
			SubeDocuments.Finish()
			err = SubeDocuments.Execute()
			if err != nil {
				log.Printf("Cannot get sube documents: %v\n", err)
			}

			subeDocumentsRows, err := SubeDocuments.Rows()
			if err != nil {
				log.Printf("Cannot get sube documents rows: %v\n", err)
			}

			for _, row := range subeDocumentsRows {
				SubeDocumentsArray = append(SubeDocumentsArray, models.Medias{
					Mid:      lib.Int64(row["mid"]),
					FilePath: lib.String(row["file_path"]),
					FileName: lib.String(row["file_name"]),
					FileSize: lib.Int64(row["file_size"]),
					MimeType: lib.String(row["mime_type"]),
					Data:     lib.String(row["data"]),
				})
			}
		}

		return c.Render("views/panel/subeler/sube-duzenle", fiber.Map{
			"PathOnStart":   "../../../",
			"PageTitle":     "N-Hospital | Şube Düzenle",
			"User":          ourUser,
			"Sube":          Sube,
			"Options":       GetOptions,
			"SubeDocuments": SubeDocumentsArray,
		}, "layouts/panel/panel")
	}
}

func AnlasmaliKurumlarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		AnlasmaliKurumlar := Orm.Select([]string{"ak.akid", "ak.name", "ak.url_name", "ak.type", "ak.contact_person", "ak.phone", "ak.email", "ak.is_active", "ak.created_at", "ak.updated_at", "s.name as sube_name", "s.sid as sube_sid"})
		AnlasmaliKurumlar.Table("anlasmali_kurumlar ak")
		AnlasmaliKurumlar.InnerJoin("subeler s", "ak.sid", "=", "s.sid")
		AnlasmaliKurumlar.OrderBy("akid", "DESC")
		AnlasmaliKurumlar.Limit(int(itemsPerPage))
		AnlasmaliKurumlar.Offset(offset)
		AnlasmaliKurumlar.Finish()

		err = AnlasmaliKurumlar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := AnlasmaliKurumlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		AnlasmaliKurumlarArray := []models.AnlasmaliKurumlar{}
		for _, row := range rows {
			AnlasmaliKurumlarArray = append(AnlasmaliKurumlarArray, models.AnlasmaliKurumlar{
				Akid:              lib.String(row["akid"]),
				Name:              lib.String(row["name"]),
				UrlName:           lib.String(row["url_name"]),
				Description:       lib.String(row["description"]),
				Type:              lib.String(row["type"]),
				Sid:               lib.String(row["sube_sid"]),
				SubeName:          lib.String(row["sube_name"]),
				ContactPerson:     lib.String(row["contact_person"]),
				Phone:             lib.String(row["phone"]),
				Email:             lib.String(row["email"]),
				Address:           lib.String(row["address"]),
				ContractStartDate: lib.Time(row["contract_start_date"]),
				ContractEndDate:   lib.Time(row["contract_end_date"]),
				DiscountRate:      lib.Float64(row["discount_rate"]),
				PaymentTerms:      lib.String(row["payment_terms"]),
				Notes:             lib.String(row["notes"]),
				LogoMid:           lib.Int64(row["logo_mid"]),
				IsActive:          lib.Bool(row["is_active"]),
				CreatedAt:         lib.Time(row["created_at"]),
				UpdatedAt:         lib.Time(row["updated_at"]),
			})
		}

		return c.Render("views/panel/anlasmali-kurumlar-sayfalari/anlasmali-kurumlar", fiber.Map{
			"PathOnStart":       "../",
			"PageTitle":         "N-Hospital | Anlaşmalı Kurumlar",
			"Page":              c.Query("page"),
			"AnlasmaliKurumlar": AnlasmaliKurumlarArray,
			"Count":             len(AnlasmaliKurumlarArray),
			"User":              ourUser,
			"Options":           GetOptions,
		}, "layouts/panel/panel")
	}
}

func AnlasmaliKurumlarEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Subeler := Orm.Select([]string{"sid", "name"})
		Subeler.Table("subeler")
		Subeler.Finish()
		err = Subeler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}
		subelerRows, err := Subeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}
		SubelerArray := []models.Subeler{}
		for _, row := range subelerRows {
			SubelerArray = append(SubelerArray, models.Subeler{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.Render("views/panel/anlasmali-kurumlar-sayfalari/anlasmali-kurum-ekle", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Anlaşmalı Kurumlar Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
			"Subeler":     SubelerArray,
		}, "layouts/panel/panel")
	}
}

func AnlasmaliKurumPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		AnlasmaliKurumId := c.Params("akid")

		if AnlasmaliKurumId == "" {
			return c.Redirect("/panel/anlasmali-kurumlar")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the contract partner with media information
		AnlasmaliKurum := Orm.Select([]string{"ak.*", "s.name as sube_name", "s.sid as sube_sid", "m.file_path as logo_path", "m.alt_text as logo_alt_text", "m.title as logo_title", "md.file_path as sube_logo_path", "md.alt_text as sube_logo_alt_text", "md.title as sube_logo_title"})
		AnlasmaliKurum.Table("anlasmali_kurumlar ak")
		AnlasmaliKurum.InnerJoin("subeler s", "ak.sid", "=", "s.sid")
		AnlasmaliKurum.LeftJoin("medias m", "ak.logo_mid", "=", "m.mid")
		AnlasmaliKurum.LeftJoin("medias md", "md.mid", "=", "s.mid")
		AnlasmaliKurum.Where("ak.akid", "=", AnlasmaliKurumId)
		AnlasmaliKurum.Finish()
		err = AnlasmaliKurum.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar")
		}

		rows, err := AnlasmaliKurum.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/anlasmali-kurumlar")
		}

		AnlasmaliKurumData := models.AnlasmaliKurumlar{
			Akid:              lib.String(rows[0]["akid"]),
			Name:              lib.String(rows[0]["name"]),
			UrlName:           lib.String(rows[0]["url_name"]),
			Description:       lib.String(rows[0]["description"]),
			Type:              lib.String(rows[0]["type"]),
			ContactPerson:     lib.String(rows[0]["contact_person"]),
			Phone:             lib.String(rows[0]["phone"]),
			Email:             lib.String(rows[0]["email"]),
			Address:           lib.String(rows[0]["address"]),
			ContractStartDate: lib.Time(rows[0]["contract_start_date"]),
			ContractEndDate:   lib.Time(rows[0]["contract_end_date"]),
			DiscountRate:      lib.Float64(rows[0]["discount_rate"]),
			PaymentTerms:      lib.String(rows[0]["payment_terms"]),
			Notes:             lib.String(rows[0]["notes"]),
			LogoMid:           lib.Int64(rows[0]["logo_mid"]),
			LogoAltText:       lib.String(rows[0]["logo_alt_text"]),
			LogoTitle:         lib.String(rows[0]["logo_title"]),
			IsActive:          lib.Bool(rows[0]["is_active"]),
			CreatedAt:         lib.Time(rows[0]["created_at"]),
			UpdatedAt:         lib.Time(rows[0]["updated_at"]),
			LogoPath:          lib.String(rows[0]["logo_path"]),
		}

		SubeData := models.Subeler{
			Sid:              lib.String(rows[0]["sube_sid"]),
			Name:             lib.String(rows[0]["sube_name"]),
			SubeMediaPath:    lib.String(rows[0]["sube_logo_path"]),
			SubeMediaAltText: lib.String(rows[0]["sube_logo_alt_text"]),
			SubeMediaTitle:   lib.String(rows[0]["sube_logo_title"]),
		}

		return c.Render("views/panel/anlasmali-kurumlar-sayfalari/anlasmali-kurum", fiber.Map{
			"PathOnStart":    "../../",
			"PageTitle":      "N-Hospital | " + AnlasmaliKurumData.Name,
			"User":           ourUser,
			"AnlasmaliKurum": AnlasmaliKurumData,
			"Options":        GetOptions,
			"SubeData":       SubeData,
		}, "layouts/panel/panel")
	}
}

func AnlasmaliKurumlarDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Akid := c.Params("akid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		AnlasmaliKurumQuery := Orm.Select([]string{"ak.*", "m.file_path as logo_path", "m.alt_text as logo_alt_text", "m.title as logo_title"})
		AnlasmaliKurumQuery.Table("anlasmali_kurumlar ak")
		AnlasmaliKurumQuery.LeftJoin("medias m", "ak.logo_mid", "=", "m.mid")
		AnlasmaliKurumQuery.Where("ak.akid", "=", Akid)
		AnlasmaliKurumQuery.Finish()
		err = AnlasmaliKurumQuery.Execute()

		if err != nil {
			log.Printf("Cannot get anlasmali kurum: %v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar")
		}

		rows, err := AnlasmaliKurumQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get anlasmali kurum rows: %v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar")
		}

		AnlasmaliKurum := models.AnlasmaliKurumlar{
			Akid:              lib.String(rows[0]["akid"]),
			Name:              lib.String(rows[0]["name"]),
			UrlName:           lib.String(rows[0]["url_name"]),
			Description:       lib.String(rows[0]["description"]),
			Type:              lib.String(rows[0]["type"]),
			ContactPerson:     lib.String(rows[0]["contact_person"]),
			Sid:               lib.String(rows[0]["sid"]),
			Phone:             lib.String(rows[0]["phone"]),
			Email:             lib.String(rows[0]["email"]),
			Address:           lib.String(rows[0]["address"]),
			ContractStartDate: lib.Time(rows[0]["contract_start_date"]),
			ContractEndDate:   lib.Time(rows[0]["contract_end_date"]),
			DiscountRate:      lib.Float64(rows[0]["discount_rate"]),
			PaymentTerms:      lib.String(rows[0]["payment_terms"]),
			Notes:             lib.String(rows[0]["notes"]),
			LogoMid:           lib.Int64(rows[0]["logo_mid"]),
			LogoAltText:       lib.String(rows[0]["logo_alt_text"]),
			LogoTitle:         lib.String(rows[0]["logo_title"]),
			IsActive:          lib.Bool(rows[0]["is_active"]),
			CreatedAt:         lib.Time(rows[0]["created_at"]),
			UpdatedAt:         lib.Time(rows[0]["updated_at"]),
			LogoPath:          lib.String(rows[0]["logo_path"]),
		}

		Subeler := Orm.Select([]string{"sid", "name"})
		Subeler.Table("subeler")
		Subeler.Finish()
		err = Subeler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		subelerRows, err := Subeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		SubelerArray := []models.Subeler{}
		for _, row := range subelerRows {
			SubelerArray = append(SubelerArray, models.Subeler{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.Render("views/panel/anlasmali-kurumlar-sayfalari/anlasmali-kurum-duzenle", fiber.Map{
			"PathOnStart":    "../../../",
			"PageTitle":      "N-Hospital | Anlaşmalı Kurum Düzenle",
			"User":           ourUser,
			"AnlasmaliKurum": AnlasmaliKurum,
			"Options":        GetOptions,
			"Subeler":        SubelerArray,
		}, "layouts/panel/panel")
	}
}

func UzmanliklarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		Uzmanliklar := Orm.Select([]string{"uzid", "name", "url_name", "description", "icon", "is_active", "created_at", "updated_at"})
		Uzmanliklar.Table("uzmanliklar")
		Uzmanliklar.OrderBy("uzid", "ASC")
		Uzmanliklar.Limit(int(itemsPerPage))
		Uzmanliklar.Offset(offset)
		Uzmanliklar.Finish()

		err = Uzmanliklar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := Uzmanliklar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		UzmanliklarArray := []models.Uzmanliklar{}
		for _, row := range rows {
			UzmanliklarArray = append(UzmanliklarArray, models.Uzmanliklar{
				Uzid:        lib.String(row["uzid"]),
				Name:        lib.String(row["name"]),
				UrlName:     lib.String(row["url_name"]),
				Description: lib.String(row["description"]),
				Icon:        lib.String(row["icon"]),
				IsActive:    row["is_active"].(bool),
				CreatedAt:   row["created_at"].(time.Time),
				UpdatedAt:   row["updated_at"].(time.Time),
			})
		}

		return c.Render("views/panel/uzmanliklar-sayfalari/uzmanliklar", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Uzmanlıklar",
			"Page":        c.Query("page"),
			"Uzmanliklar": UzmanliklarArray,
			"Count":       len(UzmanliklarArray),
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func UzmanlikPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		UzmanlikId := c.Params("uzid")

		if UzmanlikId == "" {
			return c.Redirect("/panel/uzmanliklar")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the expertise with basic information
		Uzmanlik := Orm.Select([]string{"*"})
		Uzmanlik.Table("uzmanliklar")
		Uzmanlik.Where("uzid", "=", UzmanlikId)
		Uzmanlik.Finish()
		err = Uzmanlik.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/uzmanliklar")
		}

		rows, err := Uzmanlik.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/uzmanliklar")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/uzmanliklar")
		}

		UzmanlikData := models.Uzmanliklar{
			Uzid:        lib.String(rows[0]["uzid"]),
			Name:        lib.String(rows[0]["name"]),
			UrlName:     lib.String(rows[0]["url_name"]),
			Description: lib.String(rows[0]["description"]),
			Icon:        lib.String(rows[0]["icon"]),
			IsActive:    rows[0]["is_active"].(bool),
			CreatedAt:   rows[0]["created_at"].(time.Time),
			UpdatedAt:   rows[0]["updated_at"].(time.Time),
		}

		// Fetch doctors with this expertise
		Doktorlar := Orm.Select([]string{"d.drid", "d.title", "d.first_name", "d.last_name", "d.phone", "d.email", "d.is_active", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		Doktorlar.Table("doktorlar d")
		Doktorlar.LeftJoin("doctor_expertises de", "d.drid", "=", "de.drid")
		Doktorlar.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		Doktorlar.Where("de.uzid", "=", UzmanlikId)
		Doktorlar.And("d.is_active", "=", true)
		Doktorlar.OrderBy("d.first_name", "ASC")
		Doktorlar.Finish()
		err = Doktorlar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		doktorlarRows, err := Doktorlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoktorlarArray := []models.Doktorlar{}
		for _, row := range doktorlarRows {
			DoktorlarArray = append(DoktorlarArray, models.Doktorlar{
				Drid:         lib.String(row["drid"]),
				Title:        lib.String(row["title"]),
				FirstName:    lib.String(row["first_name"]),
				LastName:     lib.String(row["last_name"]),
				Phone:        lib.String(row["phone"]),
				Email:        lib.String(row["email"]),
				PhotoPath:    lib.String(row["photo_path"]),
				PhotoAltText: lib.String(row["photo_alt_text"]),
				PhotoTitle:   lib.String(row["photo_title"]),
				IsActive:     lib.Bool(row["is_active"]),
			})
		}

		// Fetch branches with this expertise
		Branslar := Orm.Select([]string{"br.brid", "br.name", "br.phone", "br.email", "br.is_active", "m.file_path as brans_media_path", "m.alt_text as brans_media_alt_text", "m.title as brans_media_title"})
		Branslar.Table("branslar br")
		Branslar.LeftJoin("medias m", "br.mid", "=", "m.mid")
		Branslar.Where("br.is_active", "=", true)
		Branslar.OrderBy("br.name", "ASC")
		Branslar.Finish()
		err = Branslar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		branslarRows, err := Branslar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		BranslarArray := []models.Branslar{}
		for _, row := range branslarRows {
			BranslarArray = append(BranslarArray, models.Branslar{
				Brid:              lib.String(row["brid"]),
				Name:              lib.String(row["name"]),
				Phone:             lib.String(row["phone"]),
				Email:             lib.String(row["email"]),
				BransMediaPath:    lib.String(row["brans_media_path"]),
				BransMediaAltText: lib.String(row["brans_media_alt_text"]),
				BransMediaTitle:   lib.String(row["brans_media_title"]),
				IsActive:          row["is_active"].(bool),
			})
		}

		// Fetch recent appointments for this expertise
		Randevular := Orm.Select([]string{"r.rid", "r.patient_first_name", "r.patient_last_name", "r.patient_phone", "r.appointment_date", "r.appointment_time", "r.status"})
		Randevular.Table("randevular r")
		Randevular.LeftJoin("doktorlar d", "r.drid", "=", "d.drid")
		Randevular.LeftJoin("doctor_expertises de", "d.drid", "=", "de.drid")
		Randevular.Where("de.uzid", "=", UzmanlikId)
		Randevular.OrderBy("r.appointment_date", "DESC")
		Randevular.Limit(10)
		Randevular.Finish()
		err = Randevular.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		randevularRows, err := Randevular.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		RandevularArray := []models.Randevular{}
		for _, row := range randevularRows {
			RandevularArray = append(RandevularArray, models.Randevular{
				Rid:              lib.String(row["rid"]),
				PatientFirstName: lib.String(row["patient_first_name"]),
				PatientLastName:  lib.String(row["patient_last_name"]),
				PatientPhone:     lib.String(row["patient_phone"]),
				AppointmentDate:  lib.Time(row["appointment_date"]),
				AppointmentTime:  lib.Time(row["appointment_time"]),
				Status:           lib.String(row["status"]),
			})
		}

		return c.Render("views/panel/uzmanliklar-sayfalari/uzmanlik", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | " + UzmanlikData.Name,
			"User":        ourUser,
			"Uzmanlik":    UzmanlikData,
			"Doktorlar":   DoktorlarArray,
			"Branslar":    BranslarArray,
			"Randevular":  RandevularArray,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func UzmanlikEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/uzmanliklar-sayfalari/uzmanlik-ekle", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Uzmanlik Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func UzmanlikDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		UzmanlikId := c.Params("uzid")

		if UzmanlikId == "" {
			return c.Redirect("/panel/uzmanliklar")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the expertise with basic information
		Uzmanlik := Orm.Select([]string{"*"})
		Uzmanlik.Table("uzmanliklar")
		Uzmanlik.Where("uzid", "=", UzmanlikId)
		Uzmanlik.Finish()
		err = Uzmanlik.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/uzmanliklar")
		}

		rows, err := Uzmanlik.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/uzmanliklar")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/uzmanliklar")
		}

		UzmanlikData := models.Uzmanliklar{
			Uzid:        lib.String(rows[0]["uzid"]),
			Name:        lib.String(rows[0]["name"]),
			UrlName:     lib.String(rows[0]["url_name"]),
			Description: lib.String(rows[0]["description"]),
			Icon:        lib.String(rows[0]["icon"]),
			IsActive:    rows[0]["is_active"].(bool),
			CreatedAt:   rows[0]["created_at"].(time.Time),
			UpdatedAt:   rows[0]["updated_at"].(time.Time),
		}

		return c.Render("views/panel/uzmanliklar-sayfalari/uzmanlik-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | " + UzmanlikData.Name + " Düzenle",
			"User":        ourUser,
			"Uzmanlik":    UzmanlikData,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func BranslarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		Branslar := Orm.Select([]string{
			"br.brid", "br.name", "br.url_name", "br.description", "br.short_description",
			"br.services", "br.mid", "br.icon", "br.sid", "br.head_drid", "br.phone",
			"br.email", "br.is_active", "br.created_at", "br.updated_at",
			// Head doctor info
			"d.title as head_doctor_title", "d.first_name as head_doctor_first_name",
			"d.last_name as head_doctor_last_name", "d.phone as head_doctor_phone",
			"m.file_path as head_doctor_photo_path", "m.alt_text as head_doctor_photo_alt_text",
			// Branch info
			"s.name as branch_name", "s.city as branch_city",
		})
		Branslar.Table("branslar br")
		Branslar.LeftJoin("doktorlar d", "br.head_drid", "=", "d.drid")
		Branslar.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		Branslar.LeftJoin("subeler s", "br.sid", "=", "s.sid")
		Branslar.OrderBy("br.created_at", "DESC")
		Branslar.Limit(int(itemsPerPage))
		Branslar.Offset(offset)
		Branslar.Finish()

		err = Branslar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := Branslar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		BranslarArray := []models.Branslar{}
		for _, row := range rows {
			// Combine head doctor name
			headDoctorName := ""
			headDoctorTitle := lib.String(row["head_doctor_title"])
			headDoctorFirstName := lib.String(row["head_doctor_first_name"])
			headDoctorLastName := lib.String(row["head_doctor_last_name"])
			if headDoctorFirstName != "" && headDoctorLastName != "" {
				headDoctorName = headDoctorFirstName + " " + headDoctorLastName
			}

			BranslarArray = append(BranslarArray, models.Branslar{
				Brid:             lib.String(row["brid"]),
				Name:             lib.String(row["name"]),
				UrlName:          lib.String(row["url_name"]),
				Description:      lib.String(row["description"]),
				ShortDescription: lib.String(row["short_description"]),
				Services:         lib.StringArray(row["services"]),
				Mid:              lib.Int64(row["mid"]),
				Icon:             lib.String(row["icon"]),
				Sid:              lib.String(row["sid"]),
				HeadDrid:         lib.String(row["head_drid"]),
				Phone:            lib.String(row["phone"]),
				Email:            lib.String(row["email"]),
				IsActive:         lib.Bool(row["is_active"]),
				CreatedAt:        lib.Time(row["created_at"]),
				UpdatedAt:        lib.Time(row["updated_at"]),
				// Head doctor info
				HeadDoctorName:         headDoctorName,
				HeadDoctorTitle:        headDoctorTitle,
				HeadDoctorPhone:        lib.String(row["head_doctor_phone"]),
				HeadDoctorPhotoPath:    lib.String(row["head_doctor_photo_path"]),
				HeadDoctorPhotoAltText: lib.String(row["head_doctor_photo_alt_text"]),
				// Branch info
				BranchName: lib.String(row["branch_name"]),
				BranchCity: lib.String(row["branch_city"]),
			})
		}

		return c.Render("views/panel/branslar-sayfalari/branslar", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Bölümler",
			"Page":        c.Query("page"),
			"Branslar":    BranslarArray,
			"Count":       len(BranslarArray),
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func BransPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		BransId := c.Params("brid")

		if BransId == "" {
			return c.Redirect("/panel/branslar")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the branch with media information and related data
		Brans := Orm.Select([]string{"br.*", "m.file_path as brans_media_path", "m.alt_text as brans_media_alt_text", "m.title as brans_media_title",
			"d.title as head_doctor_title", "d.first_name as head_doctor_first_name", "d.last_name as head_doctor_last_name",
			"d.phone as head_doctor_phone", "dm.file_path as head_doctor_photo_path", "dm.alt_text as head_doctor_photo_alt_text",
			"s.name as branch_name", "s.city as branch_city"})
		Brans.Table("branslar br")
		Brans.LeftJoin("medias m", "br.mid", "=", "m.mid")
		Brans.LeftJoin("doktorlar d", "br.head_drid", "=", "d.drid")
		Brans.LeftJoin("medias dm", "d.photo_mid", "=", "dm.mid")
		Brans.LeftJoin("subeler s", "br.sid", "=", "s.sid")
		Brans.Where("br.brid", "=", BransId)
		Brans.Finish()
		err = Brans.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/branslar")
		}

		rows, err := Brans.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/branslar")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/branslar")
		}

		BransData := models.Branslar{
			Brid:              lib.String(rows[0]["brid"]),
			Name:              lib.String(rows[0]["name"]),
			UrlName:           lib.String(rows[0]["url_name"]),
			Description:       lib.String(rows[0]["description"]),
			ShortDescription:  lib.String(rows[0]["short_description"]),
			Services:          lib.StringArray(rows[0]["services"]),
			Mid:               lib.Int64(rows[0]["mid"]),
			Icon:              lib.String(rows[0]["icon"]),
			Sid:               lib.String(rows[0]["sid"]),
			HeadDrid:          lib.String(rows[0]["head_drid"]),
			Phone:             lib.String(rows[0]["phone"]),
			Email:             lib.String(rows[0]["email"]),
			IsActive:          lib.Bool(rows[0]["is_active"]),
			CreatedAt:         lib.Time(rows[0]["created_at"]),
			UpdatedAt:         lib.Time(rows[0]["updated_at"]),
			BransMediaPath:    lib.String(rows[0]["brans_media_path"]),
			BransMediaAltText: lib.String(rows[0]["brans_media_alt_text"]),
			BransMediaTitle:   lib.String(rows[0]["brans_media_title"]),
			// Head doctor info
			HeadDoctorName:         lib.String(rows[0]["head_doctor_first_name"]) + " " + lib.String(rows[0]["head_doctor_last_name"]),
			HeadDoctorTitle:        lib.String(rows[0]["head_doctor_title"]),
			HeadDoctorPhone:        lib.String(rows[0]["head_doctor_phone"]),
			HeadDoctorPhotoPath:    lib.String(rows[0]["head_doctor_photo_path"]),
			HeadDoctorPhotoAltText: lib.String(rows[0]["head_doctor_photo_alt_text"]),
			// Branch info
			BranchName: lib.String(rows[0]["branch_name"]),
			BranchCity: lib.String(rows[0]["branch_city"]),
		}

		// Fetch doctors in this branch
		Doktorlar := Orm.Select([]string{"d.drid", "d.title", "d.first_name", "d.last_name", "d.phone", "d.email", "d.is_active", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		Doktorlar.Table("doktorlar d")
		Doktorlar.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		Doktorlar.Where("d.brid", "=", BransId)
		Doktorlar.OrderBy("d.first_name", "ASC")
		Doktorlar.Finish()
		err = Doktorlar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		doktorlarRows, err := Doktorlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoktorlarArray := []models.Doktorlar{}
		for _, row := range doktorlarRows {
			DoktorlarArray = append(DoktorlarArray, models.Doktorlar{
				Drid:         lib.String(row["drid"]),
				Title:        lib.String(row["title"]),
				FirstName:    lib.String(row["first_name"]),
				LastName:     lib.String(row["last_name"]),
				Phone:        lib.String(row["phone"]),
				Email:        lib.String(row["email"]),
				PhotoPath:    lib.String(row["photo_path"]),
				PhotoAltText: lib.String(row["photo_alt_text"]),
				PhotoTitle:   lib.String(row["photo_title"]),
				IsActive:     lib.Bool(row["is_active"]),
			})
		}

		// Fetch recent appointments for this branch
		Randevular := Orm.Select([]string{"rid", "patient_first_name", "patient_last_name", "patient_phone", "appointment_date", "appointment_time", "status"})
		Randevular.Table("randevular")
		Randevular.Where("brid", "=", BransId)
		Randevular.OrderBy("appointment_date", "DESC")
		Randevular.Limit(10)
		Randevular.Finish()
		err = Randevular.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		randevularRows, err := Randevular.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		RandevularArray := []models.Randevular{}
		for _, row := range randevularRows {
			RandevularArray = append(RandevularArray, models.Randevular{
				Rid:              lib.String(row["rid"]),
				PatientFirstName: lib.String(row["patient_first_name"]),
				PatientLastName:  lib.String(row["patient_last_name"]),
				PatientPhone:     lib.String(row["patient_phone"]),
				AppointmentDate:  lib.Time(row["appointment_date"]),
				AppointmentTime:  lib.Time(row["appointment_time"]),
				Status:           lib.String(row["status"]),
			})
		}

		return c.Render("views/panel/branslar-sayfalari/brans", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | " + BransData.Name,
			"User":        ourUser,
			"Brans":       BransData,
			"Doktorlar":   DoktorlarArray,
			"Randevular":  RandevularArray,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func BransEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch head doctor candidates
		HeadDoctors := Orm.Select([]string{"drid", "title", "first_name", "last_name"})
		HeadDoctors.Table("doktorlar")
		HeadDoctors.Where("is_active", "=", true)
		HeadDoctors.OrderBy("first_name", "ASC")
		HeadDoctors.Finish()
		err = HeadDoctors.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := HeadDoctors.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		HeadDoctorsArray := []models.Doktorlar{}
		for _, row := range rows {
			HeadDoctorsArray = append(HeadDoctorsArray, models.Doktorlar{
				Drid:      lib.String(row["drid"]),
				Title:     lib.String(row["title"]),
				FirstName: lib.String(row["first_name"]),
				LastName:  lib.String(row["last_name"]),
			})
		}

		GetSubeler := Orm.Select([]string{"sid", "name"})
		GetSubeler.Table("subeler")
		GetSubeler.Where("is_active", "=", true)
		GetSubeler.OrderBy("name", "ASC")
		GetSubeler.Finish()
		err = GetSubeler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		subelerRows, err := GetSubeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		SubelerArray := []models.Subeler{}
		for _, row := range subelerRows {
			SubelerArray = append(SubelerArray, models.Subeler{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.Render("views/panel/branslar-sayfalari/brans-ekle", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Bölüm Ekle",
			"User":        ourUser,
			"HeadDoctors": HeadDoctorsArray,
			"Subeler":     SubelerArray,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func BransDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Brid := c.Params("brid")

		if Brid == "" {
			return c.Redirect("/panel/branslar")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the branch with media information
		BransQuery := Orm.Select([]string{"br.*", "m.file_path as brans_media_path", "m.alt_text as brans_media_alt_text", "m.title as brans_media_title"})
		BransQuery.Table("branslar br")
		BransQuery.LeftJoin("medias m", "br.mid", "=", "m.mid")
		BransQuery.Where("br.brid", "=", Brid)
		BransQuery.Finish()
		err = BransQuery.Execute()

		if err != nil {
			log.Printf("Cannot get brans: %v\n", err)
			return c.Redirect("/panel/branslar")
		}

		rows, err := BransQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get brans rows: %v\n", err)
			return c.Redirect("/panel/branslar")
		}

		Brans := models.Branslar{
			Brid:              lib.String(rows[0]["brid"]),
			Name:              lib.String(rows[0]["name"]),
			UrlName:           lib.String(rows[0]["url_name"]),
			Description:       lib.String(rows[0]["description"]),
			ShortDescription:  lib.String(rows[0]["short_description"]),
			Services:          lib.StringArray(rows[0]["services"]),
			Mid:               lib.Int64(rows[0]["mid"]),
			Icon:              lib.String(rows[0]["icon"]),
			Sid:               lib.String(rows[0]["sid"]),
			HeadDrid:          lib.String(rows[0]["head_drid"]),
			Phone:             lib.String(rows[0]["phone"]),
			Email:             lib.String(rows[0]["email"]),
			IsActive:          lib.Bool(rows[0]["is_active"]),
			CreatedAt:         lib.Time(rows[0]["created_at"]),
			UpdatedAt:         lib.Time(rows[0]["updated_at"]),
			BransMediaPath:    lib.String(rows[0]["brans_media_path"]),
			BransMediaAltText: lib.String(rows[0]["brans_media_alt_text"]),
			BransMediaTitle:   lib.String(rows[0]["brans_media_title"]),
		}

		// Fetch all branches for the select box
		Subeler := Orm.Select([]string{"sid", "name", "city"})
		Subeler.Table("subeler")
		Subeler.Where("is_active", "=", true)
		Subeler.OrderBy("name", "ASC")
		Subeler.Finish()
		err = Subeler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		subelerRows, err := Subeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		SubelerArray := []models.Subeler{}
		for _, row := range subelerRows {
			SubelerArray = append(SubelerArray, models.Subeler{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
				City: lib.String(row["city"]),
			})
		}

		return c.Render("views/panel/branslar-sayfalari/brans-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | " + Brans.Name + " Düzenle",
			"User":        ourUser,
			"Brans":       Brans,
			"Subeler":     SubelerArray,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func DoktorlarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		// Fetch doctors with joined data from branslar, subeler, and medias tables
		Doktorlar := Orm.Select([]string{
			"d.drid", "d.title", "d.first_name", "d.last_name", "d.url_name",
			"d.phone", "d.email", "d.biography", "d.experience_years",
			"d.online_appointment", "d.is_active", "d.created_at", "d.updated_at",
			"b.name as branch_name", "s.name as sube_name", "s.city as sube_city",
			"m1.file_path as photo_path", "m1.alt_text as photo_alt_text",
		})
		Doktorlar.Table("doktorlar d")
		Doktorlar.LeftJoin("branslar b", "d.brid", "=", "b.brid")
		Doktorlar.LeftJoin("subeler s", "d.sid", "=", "s.sid")
		Doktorlar.LeftJoin("medias m1", "d.photo_mid", "=", "m1.mid")
		Doktorlar.OrderBy("d.is_active", "DESC")
		Doktorlar.OrderBy("d.drid", "DESC")
		Doktorlar.Limit(int(itemsPerPage))
		Doktorlar.Offset(offset)
		Doktorlar.Finish()

		err = Doktorlar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := Doktorlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		DoktorlarArray := []models.Doktorlar{}
		for _, row := range rows {
			DoktorlarArray = append(DoktorlarArray, models.Doktorlar{
				Drid:              lib.String(row["drid"]),
				Title:             lib.String(row["title"]),
				FirstName:         lib.String(row["first_name"]),
				LastName:          lib.String(row["last_name"]),
				UrlName:           lib.String(row["url_name"]),
				Phone:             lib.String(row["phone"]),
				Email:             lib.String(row["email"]),
				Biography:         lib.String(row["biography"]),
				ExperienceYears:   lib.Int64(row["experience_years"]),
				OnlineAppointment: lib.Bool(row["online_appointment"]),
				IsActive:          lib.Bool(row["is_active"]),
				CreatedAt:         lib.Time(row["created_at"]),
				UpdatedAt:         lib.Time(row["updated_at"]),
				BranchName:        lib.String(row["branch_name"]),
				SubeName:          lib.String(row["sube_name"]),
				SubeCity:          lib.String(row["sube_city"]),
				PhotoPath:         lib.String(row["photo_path"]),
				PhotoAltText:      lib.String(row["photo_alt_text"]),
			})
		}

		return c.Render("views/panel/doktorlar-sayfalari/doktorlar", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Doktorlar",
			"Page":        c.Query("page"),
			"Doktorlar":   DoktorlarArray,
			"Count":       len(DoktorlarArray),
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func DoktorPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		DoktorId := c.Params("doktor")

		if DoktorId == "" {
			return c.Redirect("/panel/doktorlar")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the doctor with media information and related data
		Doktor := Orm.Select([]string{
			"d.*",
			"m1.file_path as photo_path", "m1.alt_text as photo_alt_text", "m1.title as photo_title",
			"m2.file_path as cv_file_path",
			"b.name as branch_name", "s.name as sube_name", "s.city as sube_city",
		})
		Doktor.Table("doktorlar d")
		Doktor.LeftJoin("medias m1", "d.photo_mid", "=", "m1.mid")
		Doktor.LeftJoin("medias m2", "d.cv_file_mid", "=", "m2.mid")
		Doktor.LeftJoin("branslar b", "d.brid", "=", "b.brid")
		Doktor.LeftJoin("subeler s", "d.sid", "=", "s.sid")
		Doktor.Where("d.drid", "=", DoktorId)
		Doktor.Finish()
		err = Doktor.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		rows, err := Doktor.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/doktorlar")
		}

		DoktorData := models.Doktorlar{
			Drid:                lib.String(rows[0]["drid"]),
			Title:               lib.String(rows[0]["title"]),
			FirstName:           lib.String(rows[0]["first_name"]),
			LastName:            lib.String(rows[0]["last_name"]),
			UrlName:             lib.String(rows[0]["url_name"]),
			TcKimlik:            lib.String(rows[0]["tc_kimlik"]),
			DiplomaNo:           lib.String(rows[0]["diploma_no"]),
			Phone:               lib.String(rows[0]["phone"]),
			Email:               lib.String(rows[0]["email"]),
			Biography:           lib.String(rows[0]["biography"]),
			Education:           lib.String(rows[0]["education"]),
			ExperienceYears:     lib.Int64(rows[0]["experience_years"]),
			Languages:           lib.String(rows[0]["languages"]),
			BirthDate:           lib.Time(rows[0]["birth_date"]),
			Gender:              lib.String(rows[0]["gender"]),
			PhotoMid:            lib.Int64(rows[0]["photo_mid"]),
			CvFileMid:           lib.Int64(rows[0]["cv_file_mid"]),
			PhotoPath:           lib.String(rows[0]["photo_path"]),
			PhotoAltText:        lib.String(rows[0]["photo_alt_text"]),
			PhotoTitle:          lib.String(rows[0]["photo_title"]),
			CvFilePath:          lib.String(rows[0]["cv_file_path"]),
			Brid:                lib.String(rows[0]["brid"]),
			Sid:                 lib.String(rows[0]["sid"]),
			RoomNumber:          lib.String(rows[0]["room_number"]),
			AppointmentDuration: lib.Int64(rows[0]["appointment_duration"]),
			AppointmentFee:      lib.Float64(rows[0]["appointment_fee"]),
			OnlineAppointment:   lib.Bool(rows[0]["online_appointment"]),
			WorkingHours:        lib.String(rows[0]["working_hours"]),
			VacationDates:       lib.String(rows[0]["vacation_dates"]),
			FacebookUrl:         lib.String(rows[0]["facebook_url"]),
			LinkedinUrl:         lib.String(rows[0]["linkedin_url"]),
			InstagramUrl:        lib.String(rows[0]["instagram_url"]),
			XUrl:                lib.String(rows[0]["x_url"]),
			PersonalUrl:         lib.String(rows[0]["personal_url"]),
			IsActive:            lib.Bool(rows[0]["is_active"]),
			CreatedAt:           lib.Time(rows[0]["created_at"]),
			UpdatedAt:           lib.Time(rows[0]["updated_at"]),
			BranchName:          lib.String(rows[0]["branch_name"]),
			SubeName:            lib.String(rows[0]["sube_name"]),
			SubeCity:            lib.String(rows[0]["sube_city"]),
		}

		// Fetch doctor expertises
		DoctorExpertises := Orm.Select([]string{
			"de.duid", "de.drid", "de.uzid", "de.certification_date", "de.certification_institution", "de.is_primary",
			"de.created_at", "de.updated_at", "u.name as uzmanlik_name", "u.description as uzmanlik_description",
		})
		DoctorExpertises.Table("doctor_expertises de")
		DoctorExpertises.LeftJoin("uzmanliklar u", "de.uzid", "=", "u.uzid")
		DoctorExpertises.Where("de.drid", "=", DoktorId)
		DoctorExpertises.OrderBy("de.is_primary", "DESC")
		DoctorExpertises.OrderBy("de.created_at", "ASC")
		DoctorExpertises.Finish()
		err = DoctorExpertises.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		expertiseRows, err := DoctorExpertises.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoctorExpertisesArray := []models.DoctorExpertises{}
		for _, row := range expertiseRows {
			DoctorExpertisesArray = append(DoctorExpertisesArray, models.DoctorExpertises{
				Duid:                     lib.String(row["duid"]),
				Drid:                     lib.String(row["drid"]),
				Uzid:                     lib.String(row["uzid"]),
				UzmanlikName:             lib.String(row["uzmanlik_name"]),
				UzmanlikDescription:      lib.String(row["uzmanlik_description"]),
				CertificationDate:        lib.Time(row["certification_date"]),
				CertificationInstitution: lib.String(row["certification_institution"]),
				IsPrimary:                lib.Bool(row["is_primary"]),
				CreatedAt:                lib.Time(row["created_at"]),
				UpdatedAt:                lib.Time(row["updated_at"]),
			})
		}

		// Fetch doctor experiences
		DoctorExperiences := Orm.Select([]string{
			"de.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title",
		})
		DoctorExperiences.Table("doctor_experiences de")
		DoctorExperiences.LeftJoin("medias m", "de.cover_mid", "=", "m.mid")
		DoctorExperiences.Where("de.drid", "=", DoktorId)
		DoctorExperiences.OrderBy("de.start_date", "DESC")
		DoctorExperiences.Finish()
		err = DoctorExperiences.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		experienceRows, err := DoctorExperiences.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoctorExperiencesArray := []models.DoctorExperiences{}
		for _, row := range experienceRows {
			DoctorExperiencesArray = append(DoctorExperiencesArray, models.DoctorExperiences{
				Dtid:         lib.String(row["dtid"]),
				Drid:         lib.String(row["drid"]),
				Name:         lib.String(row["name"]),
				StartDate:    lib.Time(row["start_date"]),
				EndDate:      lib.Time(row["end_date"]),
				CoverMid:     lib.Int64(row["cover_mid"]),
				Description:  lib.String(row["description"]),
				IsActive:     lib.Bool(row["is_active"]),
				CreatedAt:    lib.Time(row["created_at"]),
				UpdatedAt:    lib.Time(row["updated_at"]),
				CoverPath:    lib.String(row["cover_path"]),
				CoverAltText: lib.String(row["cover_alt_text"]),
				CoverTitle:   lib.String(row["cover_title"]),
			})
		}

		return c.Render("views/panel/doktorlar-sayfalari/doktor", fiber.Map{
			"PathOnStart":       "../../",
			"PageTitle":         "N-Hospital | " + DoktorData.Title + " " + DoktorData.FirstName + " " + DoktorData.LastName,
			"User":              ourUser,
			"Doktor":            DoktorData,
			"DoctorExpertises":  DoctorExpertisesArray,
			"DoctorExperiences": DoctorExperiencesArray,
			"Options":           GetOptions,
		}, "layouts/panel/panel")
	}
}

func DoktorEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch active branches for the dropdown
		Branches := Orm.Select([]string{"brid", "name"})
		Branches.Table("branslar")
		Branches.Where("is_active", "=", true)
		Branches.And("sid", "=", nil)
		Branches.OrderBy("name", "ASC")
		Branches.Finish()

		err = Branches.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := Branches.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		BranchesArray := []models.Branslar{}
		for _, row := range rows {
			BranchesArray = append(BranchesArray, models.Branslar{
				Brid: lib.String(row["brid"]),
				Name: lib.String(row["name"]),
			})
		}

		GetAllSubeler := Orm.Select([]string{"sid", "name"})
		GetAllSubeler.Table("subeler")
		GetAllSubeler.Where("is_active", "=", true)
		GetAllSubeler.OrderBy("name", "ASC")
		GetAllSubeler.Finish()
		err = GetAllSubeler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		subelerRows, err := GetAllSubeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		SubelerArray := []models.Subeler{}
		for _, row := range subelerRows {
			SubelerArray = append(SubelerArray, models.Subeler{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.Render("views/panel/doktorlar-sayfalari/doktor-ekle", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Doktor Ekle",
			"User":        ourUser,
			"Branches":    BranchesArray,
			"Subeler":     SubelerArray,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func DoktorDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Drid := c.Params("doktor")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch doctor with joined data from medias table
		DoktorQuery := Orm.Select([]string{
			"d.*",
			"m1.file_path as photo_path", "m1.alt_text as photo_alt_text", "m1.title as photo_title",
			"m2.file_path as cv_file_path", "m2.alt_text as cv_alt_text", "m2.title as cv_title",
		})
		DoktorQuery.Table("doktorlar d")
		DoktorQuery.LeftJoin("medias m1", "d.photo_mid", "=", "m1.mid")
		DoktorQuery.LeftJoin("medias m2", "d.cv_file_mid", "=", "m2.mid")
		DoktorQuery.Where("d.drid", "=", Drid)
		DoktorQuery.Finish()
		err = DoktorQuery.Execute()

		if err != nil {
			log.Printf("Cannot get doktor: %v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		rows, err := DoktorQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get doktor rows: %v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		Doktor := models.Doktorlar{
			Drid:                lib.String(rows[0]["drid"]),
			Title:               lib.String(rows[0]["title"]),
			FirstName:           lib.String(rows[0]["first_name"]),
			LastName:            lib.String(rows[0]["last_name"]),
			UrlName:             lib.String(rows[0]["url_name"]),
			TcKimlik:            lib.String(rows[0]["tc_kimlik"]),
			DiplomaNo:           lib.String(rows[0]["diploma_no"]),
			Phone:               lib.String(rows[0]["phone"]),
			Email:               lib.String(rows[0]["email"]),
			Biography:           lib.String(rows[0]["biography"]),
			Education:           lib.String(rows[0]["education"]),
			ExperienceYears:     lib.Int64(rows[0]["experience_years"]),
			Languages:           lib.String(rows[0]["languages"]),
			BirthDate:           lib.Time(rows[0]["birth_date"]),
			Gender:              lib.String(rows[0]["gender"]),
			PhotoMid:            lib.Int64(rows[0]["photo_mid"]),
			CvFileMid:           lib.Int64(rows[0]["cv_file_mid"]),
			PhotoPath:           lib.String(rows[0]["photo_path"]),
			PhotoAltText:        lib.String(rows[0]["photo_alt_text"]),
			PhotoTitle:          lib.String(rows[0]["photo_title"]),
			CvFilePath:          lib.String(rows[0]["cv_file_path"]),
			Brid:                lib.String(rows[0]["brid"]),
			Sid:                 lib.String(rows[0]["sid"]),
			HeadDrid:            lib.String(rows[0]["head_drid"]),
			RoomNumber:          lib.String(rows[0]["room_number"]),
			AppointmentDuration: lib.Int64(rows[0]["appointment_duration"]),
			AppointmentFee:      lib.Float64(rows[0]["appointment_fee"]),
			OnlineAppointment:   lib.Bool(rows[0]["online_appointment"]),
			WorkingHours:        lib.String(rows[0]["working_hours"]),
			VacationDates:       lib.String(rows[0]["vacation_dates"]),
			FacebookUrl:         lib.String(rows[0]["facebook_url"]),
			LinkedinUrl:         lib.String(rows[0]["linkedin_url"]),
			InstagramUrl:        lib.String(rows[0]["instagram_url"]),
			XUrl:                lib.String(rows[0]["x_url"]),
			PersonalUrl:         lib.String(rows[0]["personal_url"]),
			IsActive:            lib.Bool(rows[0]["is_active"]),
			CreatedAt:           lib.Time(rows[0]["created_at"]),
			UpdatedAt:           lib.Time(rows[0]["updated_at"]),
		}

		// Fetch active branches for the dropdown
		Branches := Orm.Select([]string{"brid", "name"})
		Branches.Table("branslar")
		Branches.OpenParenthesis("WHERE")
		Branches.And("sid", "=", nil)

		if Doktor.Sid != "" {
			Branches.Or("sid", "=", Doktor.Sid)
		}

		if Doktor.Brid != "" {
			Branches.Or("brid", "=", Doktor.Brid)
		}

		Branches.CloseParenthesis()
		Branches.And("is_active", "=", true)
		Branches.OrderBy("name", "ASC")
		Branches.Finish()

		err = Branches.Execute()

		if err != nil {
			log.Printf("Cannot get branches: %v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		branchRows, err := Branches.Rows()
		if err != nil {
			log.Printf("Cannot get branch rows: %v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		BranchesArray := []models.Branslar{}
		for _, row := range branchRows {
			BranchesArray = append(BranchesArray, models.Branslar{
				Brid: lib.String(row["brid"]),
				Name: lib.String(row["name"]),
			})
		}

		GetSubeler := Orm.Select([]string{"sid", "name"})
		GetSubeler.Table("subeler")
		GetSubeler.Where("is_active", "=", true)
		GetSubeler.OrderBy("is_main", "DESC")
		GetSubeler.OrderBy("name", "ASC")
		GetSubeler.Finish()
		err = GetSubeler.Execute()

		if err != nil {
			log.Printf("Cannot get subeler: %v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		subelerRows, err := GetSubeler.Rows()
		if err != nil {
			log.Printf("Cannot get subeler rows: %v\n", err)
			return c.Redirect("/panel/doktorlar")
		}

		SubelerArray := []models.Subeler{}
		for _, row := range subelerRows {
			SubelerArray = append(SubelerArray, models.Subeler{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.Render("views/panel/doktorlar-sayfalari/doktor-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | Doktor Düzenle",
			"User":        ourUser,
			"Doktor":      Doktor,
			"Branches":    BranchesArray,
			"Subeler":     SubelerArray,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TibbiBirimlerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		TibbiBirimler := Orm.Select([]string{"tbid", "name", "url_name", "description", "cover_mid", "is_active", "created_at", "updated_at"})
		TibbiBirimler.Table("tibbi_birimler")
		TibbiBirimler.OrderBy("is_active", "DESC")
		TibbiBirimler.OrderBy("tbid", "DESC")
		TibbiBirimler.Limit(int(itemsPerPage))
		TibbiBirimler.Offset(offset)
		TibbiBirimler.Finish()

		err = TibbiBirimler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := TibbiBirimler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		TibbiBirimlerArray := []models.TibbiBirimler{}
		for _, row := range rows {
			// Get cover media information
			var CoverPath, CoverAltText, CoverTitle string
			if coverMid := lib.Int64(row["cover_mid"]); coverMid > 0 {
				mediaQuery := Orm.Select([]string{"file_path", "alt_text", "title"})
				mediaQuery.Table("medias")
				mediaQuery.Where("mid", "=", coverMid)
				mediaQuery.Finish()

				err = mediaQuery.Execute()
				if err == nil {
					mediaRows, _ := mediaQuery.Rows()
					if len(mediaRows) > 0 {
						CoverPath = lib.String(mediaRows[0]["file_path"])
						CoverAltText = lib.String(mediaRows[0]["alt_text"])
						CoverTitle = lib.String(mediaRows[0]["title"])
					}
				}
			}

			TibbiBirimlerArray = append(TibbiBirimlerArray, models.TibbiBirimler{
				Tbid:         lib.String(row["tbid"]),
				Name:         lib.String(row["name"]),
				UrlName:      lib.String(row["url_name"]),
				Description:  lib.String(row["description"]),
				CoverMid:     lib.Int64(row["cover_mid"]),
				CoverPath:    CoverPath,
				CoverAltText: CoverAltText,
				CoverTitle:   CoverTitle,
				IsActive:     lib.Bool(row["is_active"]),
				CreatedAt:    lib.Time(row["created_at"]),
				UpdatedAt:    lib.Time(row["updated_at"]),
			})
		}

		return c.Render("views/panel/tibbi-birimler-sayfalari/tibbi-birimler", fiber.Map{
			"PathOnStart":   "../",
			"PageTitle":     "N-Hospital | Tıbbi Birimler",
			"Page":          c.Query("page"),
			"TibbiBirimler": TibbiBirimlerArray,
			"Count":         len(TibbiBirimlerArray),
			"User":          ourUser,
			"Options":       GetOptions,
		}, "layouts/panel/panel")
	}
}

func TibbiBirimPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Redirect("/giris")
		}

		TibbiBirimId := c.Params("tbid")

		if TibbiBirimId == "" {
			return c.Redirect("/panel/tibbi-birimler")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the tibbi_birim with media information
		TibbiBirim := Orm.Select([]string{"tb.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title", "m2.file_path as video_path"})
		TibbiBirim.Table("tibbi_birimler tb")
		TibbiBirim.LeftJoin("medias m", "tb.cover_mid", "=", "m.mid")
		TibbiBirim.LeftJoin("medias m2", "tb.video_mid", "=", "m2.mid")
		TibbiBirim.Where("tb.tbid", "=", TibbiBirimId)
		TibbiBirim.Finish()
		err = TibbiBirim.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/tibbi-birimler")
		}

		rows, err := TibbiBirim.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/tibbi-birimler")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/tibbi-birimler")
		}

		TibbiBirimData := models.TibbiBirimler{
			Tbid:         lib.String(rows[0]["tbid"]),
			Name:         lib.String(rows[0]["name"]),
			UrlName:      lib.String(rows[0]["url_name"]),
			Description:  lib.String(rows[0]["description"]),
			CoverMid:     lib.Int64(rows[0]["cover_mid"]),
			CoverPath:    lib.String(rows[0]["cover_path"]),
			CoverAltText: lib.String(rows[0]["cover_alt_text"]),
			CoverTitle:   lib.String(rows[0]["cover_title"]),
			VideoMid:     lib.Int64(rows[0]["video_mid"]),
			VideoPath:    lib.String(rows[0]["video_path"]),
			IsActive:     lib.Bool(rows[0]["is_active"]),
			CreatedAt:    lib.Time(rows[0]["created_at"]),
			UpdatedAt:    lib.Time(rows[0]["updated_at"]),
		}

		// Fetch doctors in this tibbi_birim
		Doktorlar := Orm.Select([]string{"d.drid", "d.title", "d.first_name", "d.last_name", "d.phone", "d.email", "d.is_active", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		Doktorlar.Table("doktorlar d")
		Doktorlar.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		Doktorlar.Where("d.tbid", "=", TibbiBirimId)
		Doktorlar.And("d.is_active", "=", true)
		Doktorlar.OrderBy("d.first_name", "ASC")
		Doktorlar.Finish()

		err = Doktorlar.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		doktorlarRows, err := Doktorlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoktorlarArray := []models.Doktorlar{}
		for _, row := range doktorlarRows {
			DoktorlarArray = append(DoktorlarArray, models.Doktorlar{
				Drid:         lib.String(row["drid"]),
				Title:        lib.String(row["title"]),
				FirstName:    lib.String(row["first_name"]),
				LastName:     lib.String(row["last_name"]),
				Phone:        lib.String(row["phone"]),
				Email:        lib.String(row["email"]),
				PhotoPath:    lib.String(row["photo_path"]),
				PhotoAltText: lib.String(row["photo_alt_text"]),
				PhotoTitle:   lib.String(row["photo_title"]),
				IsActive:     row["is_active"].(bool),
			})
		}

		// Fetch recent appointments for this tibbi_birim
		Randevular := Orm.Select([]string{"rid", "patient_first_name", "patient_last_name", "patient_phone", "appointment_date", "appointment_time", "status"})
		Randevular.Table("randevular")
		Randevular.Where("tbid", "=", TibbiBirimId)
		Randevular.OrderBy("appointment_date", "DESC")
		Randevular.Limit(10)
		Randevular.Finish()
		err = Randevular.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		randevularRows, err := Randevular.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		RandevularArray := []models.Randevular{}
		for _, row := range randevularRows {
			RandevularArray = append(RandevularArray, models.Randevular{
				Rid:              lib.String(row["rid"]),
				PatientFirstName: lib.String(row["patient_first_name"]),
				PatientLastName:  lib.String(row["patient_last_name"]),
				PatientPhone:     lib.String(row["patient_phone"]),
				AppointmentDate:  row["appointment_date"].(time.Time),
				AppointmentTime:  row["appointment_time"].(time.Time),
				Status:           lib.String(row["status"]),
			})
		}

		return c.Render("views/panel/tibbi-birimler-sayfalari/tibbi-birim", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | " + TibbiBirimData.Name,
			"User":        ourUser,
			"TibbiBirim":  TibbiBirimData,
			"Doktorlar":   DoktorlarArray,
			"Randevular":  RandevularArray,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TibbiBirimEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/tibbi-birimler-sayfalari/tibbi-birim-ekle", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Tıbbi Birim Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TibbiBirimDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Tbid := c.Params("tbid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		TibbiBirimQuery := Orm.Select([]string{"tb.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title", "m2.file_path as video_path"})
		TibbiBirimQuery.Table("tibbi_birimler tb")
		TibbiBirimQuery.LeftJoin("medias m", "tb.cover_mid", "=", "m.mid")
		TibbiBirimQuery.LeftJoin("medias m2", "tb.video_mid", "=", "m2.mid")
		TibbiBirimQuery.Where("tb.tbid", "=", Tbid)
		TibbiBirimQuery.Finish()
		err = TibbiBirimQuery.Execute()

		if err != nil {
			log.Printf("Cannot get tibbi_birim: %v\n", err)
			return c.Redirect("/panel/tibbi-birimler")
		}

		rows, err := TibbiBirimQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get tibbi_birim rows: %v\n", err)
			return c.Redirect("/panel/tibbi-birimler")
		}

		TibbiBirim := models.TibbiBirimler{
			Tbid:         lib.String(rows[0]["tbid"]),
			Name:         lib.String(rows[0]["name"]),
			UrlName:      lib.String(rows[0]["url_name"]),
			Description:  lib.String(rows[0]["description"]),
			CoverMid:     lib.Int64(rows[0]["cover_mid"]),
			CoverPath:    lib.String(rows[0]["cover_path"]),
			CoverAltText: lib.String(rows[0]["cover_alt_text"]),
			CoverTitle:   lib.String(rows[0]["cover_title"]),
			VideoMid:     lib.Int64(rows[0]["video_mid"]),
			VideoPath:    lib.String(rows[0]["video_path"]),
			IsActive:     lib.Bool(rows[0]["is_active"]),
			CreatedAt:    lib.Time(rows[0]["created_at"]),
			UpdatedAt:    lib.Time(rows[0]["updated_at"]),
		}

		fmt.Printf("video path: %+v\n", TibbiBirim.VideoPath)

		return c.Render("views/panel/tibbi-birimler-sayfalari/tibbi-birim-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | Tıbbi Birim Düzenle",
			"User":        ourUser,
			"TibbiBirim":  TibbiBirim,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func HomepageContentsPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		HomepageContents := Orm.Select([]string{"hcid", "name", "content_type", "sort_order", "url_name", "content_html", "content_javascript", "content_css", "description", "later_than_which_content", "is_active", "created_at", "updated_at"})
		HomepageContents.Table("homepage_contents")
		HomepageContents.OrderBy("sort_order", "ASC")
		HomepageContents.OrderBy("hcid", "DESC")
		HomepageContents.Limit(int(itemsPerPage))
		HomepageContents.Offset(offset)
		HomepageContents.Finish()

		err = HomepageContents.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := HomepageContents.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		HomepageContentsArray := []models.HomepageContents{}
		for _, row := range rows {
			HomepageContentsArray = append(HomepageContentsArray, models.HomepageContents{
				Hcid:              lib.String(row["hcid"]),
				Name:              lib.String(row["name"]),
				ContentType:       lib.String(row["content_type"]),
				SortOrder:         lib.Int64(row["sort_order"]),
				UrlName:           lib.String(row["url_name"]),
				ContentHtml:       lib.String(row["content_html"]),
				ContentJavascript: lib.String(row["content_javascript"]),
				ContentCss:        lib.String(row["content_css"]),
				Description:       lib.String(row["description"]),
				IsActive:          row["is_active"].(bool),
				CreatedAt:         row["created_at"].(time.Time),
				UpdatedAt:         row["updated_at"].(time.Time),
			})
		}

		return c.Render("views/panel/homepage-contents-sayfalari/homepage-contents", fiber.Map{
			"PathOnStart":      "../",
			"PageTitle":        "N-Hospital | Anasayfa İçerikleri",
			"Page":             c.Query("page"),
			"HomepageContents": HomepageContentsArray,
			"Count":            len(HomepageContentsArray),
			"User":             ourUser,
			"Options":          GetOptions,
		}, "layouts/panel/panel")
	}
}

func HomepageContentPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		HomepageContentId := c.Params("hcid")

		if HomepageContentId == "" {
			return c.Redirect("/panel/anasayfa-icerikleri")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the homepage content
		HomepageContent := Orm.Select([]string{"*"})
		HomepageContent.Table("homepage_contents")
		HomepageContent.Where("hcid", "=", HomepageContentId)
		HomepageContent.Finish()
		err = HomepageContent.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/anasayfa-icerikleri")
		}

		rows, err := HomepageContent.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/anasayfa-icerikleri")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/anasayfa-icerikleri")
		}

		HomepageContentData := models.HomepageContents{
			Hcid:                  lib.String(rows[0]["hcid"]),
			Name:                  lib.String(rows[0]["name"]),
			ContentType:           lib.String(rows[0]["content_type"]),
			SortOrder:             lib.Int64(rows[0]["sort_order"]),
			UrlName:               lib.String(rows[0]["url_name"]),
			ContentHtml:           lib.String(rows[0]["content_html"]),
			ContentJavascript:     lib.String(rows[0]["content_javascript"]),
			ContentCss:            lib.String(rows[0]["content_css"]),
			LaterThanWhichContent: lib.Int64(rows[0]["later_than_which_content"]),
			Description:           lib.String(rows[0]["description"]),
			IsActive:              rows[0]["is_active"].(bool),
			CreatedAt:             rows[0]["created_at"].(time.Time),
			UpdatedAt:             rows[0]["updated_at"].(time.Time),
		}

		return c.Render("views/panel/homepage-contents-sayfalari/homepage-content", fiber.Map{
			"PathOnStart":     "../../",
			"PageTitle":       "N-Hospital | " + HomepageContentData.Name,
			"User":            ourUser,
			"HomepageContent": HomepageContentData,
			"Options":         GetOptions,
		}, "layouts/panel/panel")
	}
}

func HomepageContentEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/homepage-contents-sayfalari/homepage-content-ekle", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Anasayfa İçeriği Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func HomepageContentDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Hcid := c.Params("hcid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		HomepageContentQuery := Orm.Select([]string{"*"})
		HomepageContentQuery.Table("homepage_contents")
		HomepageContentQuery.Where("hcid", "=", Hcid)
		HomepageContentQuery.Finish()
		err = HomepageContentQuery.Execute()

		if err != nil {
			log.Printf("Cannot get homepage content: %v\n", err)
			return c.Redirect("/panel/anasayfa-icerikleri")
		}

		rows, err := HomepageContentQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get homepage content rows: %v\n", err)
			return c.Redirect("/panel/anasayfa-icerikleri")
		}

		HomepageContent := models.HomepageContents{
			Hcid:                  lib.String(rows[0]["hcid"]),
			Name:                  lib.String(rows[0]["name"]),
			ContentType:           lib.String(rows[0]["content_type"]),
			SortOrder:             lib.Int64(rows[0]["sort_order"]),
			UrlName:               lib.String(rows[0]["url_name"]),
			ContentHtml:           lib.String(rows[0]["content_html"]),
			ContentJavascript:     lib.String(rows[0]["content_javascript"]),
			ContentCss:            lib.String(rows[0]["content_css"]),
			LaterThanWhichContent: lib.Int64(rows[0]["later_than_which_content"]),
			Description:           lib.String(rows[0]["description"]),
			IsActive:              rows[0]["is_active"].(bool),
			CreatedAt:             rows[0]["created_at"].(time.Time),
			UpdatedAt:             rows[0]["updated_at"].(time.Time),
		}

		return c.Render("views/panel/homepage-contents-sayfalari/homepage-content-duzenle", fiber.Map{
			"PathOnStart":     "../../../",
			"PageTitle":       "N-Hospital | Anasayfa İçeriği Düzenle",
			"User":            ourUser,
			"HomepageContent": HomepageContent,
			"Options":         GetOptions,
		}, "layouts/panel/panel")
	}
}

func CustomContentsPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		CustomContents := Orm.Select([]string{"ccid", "name", "content_type", "sort_order", "url_name", "content_html", "content_javascript", "content_css", "description", "route", "is_active", "created_at", "updated_at"})
		CustomContents.Table("custom_contents")
		CustomContents.OrderBy("sort_order", "ASC")
		CustomContents.OrderBy("ccid", "DESC")
		CustomContents.Limit(int(itemsPerPage))
		CustomContents.Offset(offset)
		CustomContents.Finish()

		err = CustomContents.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := CustomContents.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		CustomContentsArray := []models.CustomContents{}
		for _, row := range rows {
			CustomContentsArray = append(CustomContentsArray, models.CustomContents{
				Ccid:              lib.String(row["ccid"]),
				Name:              lib.String(row["name"]),
				ContentType:       lib.String(row["content_type"]),
				SortOrder:         lib.Int64(row["sort_order"]),
				UrlName:           lib.String(row["url_name"]),
				ContentHtml:       lib.String(row["content_html"]),
				ContentJavascript: lib.String(row["content_javascript"]),
				ContentCss:        lib.String(row["content_css"]),
				Description:       lib.String(row["description"]),
				IsActive:          row["is_active"].(bool),
				CreatedAt:         row["created_at"].(time.Time),
				UpdatedAt:         row["updated_at"].(time.Time),
			})
		}

		return c.Render("views/panel/custom-contents-sayfalari/custom-contents", fiber.Map{
			"PathOnStart":    "../",
			"PageTitle":      "N-Hospital | Özel İçerikler",
			"Page":           c.Query("page"),
			"CustomContents": CustomContentsArray,
			"Count":          len(CustomContentsArray),
			"User":           ourUser,
			"Options":        GetOptions,
		}, "layouts/panel/panel")
	}
}

func CustomContentPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		CustomContentId := c.Params("ccid")

		if CustomContentId == "" {
			return c.Redirect("/panel/ozel-icerikler")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the custom content
		CustomContent := Orm.Select([]string{"*"})
		CustomContent.Table("custom_contents")
		CustomContent.Where("ccid", "=", CustomContentId)
		CustomContent.Finish()
		err = CustomContent.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/ozel-icerikler")
		}

		rows, err := CustomContent.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/ozel-icerikler")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/ozel-icerikler")
		}

		CustomContentData := models.CustomContents{
			Ccid:              lib.String(rows[0]["ccid"]),
			Name:              lib.String(rows[0]["name"]),
			ContentType:       lib.String(rows[0]["content_type"]),
			SortOrder:         lib.Int64(rows[0]["sort_order"]),
			UrlName:           lib.String(rows[0]["url_name"]),
			ContentHtml:       lib.String(rows[0]["content_html"]),
			ContentJavascript: lib.String(rows[0]["content_javascript"]),
			ContentCss:        lib.String(rows[0]["content_css"]),
			Description:       lib.String(rows[0]["description"]),
			Route:             lib.String(rows[0]["route"]),
			IsActive:          rows[0]["is_active"].(bool),
			CreatedAt:         rows[0]["created_at"].(time.Time),
			UpdatedAt:         rows[0]["updated_at"].(time.Time),
		}

		return c.Render("views/panel/custom-contents-sayfalari/custom-content", fiber.Map{
			"PathOnStart":   "../../",
			"PageTitle":     "N-Hospital | " + CustomContentData.Name,
			"User":          ourUser,
			"CustomContent": CustomContentData,
			"Options":       GetOptions,
		}, "layouts/panel/panel")
	}
}

func CustomContentEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/custom-contents-sayfalari/custom-content-ekle", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Özel İçerik Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func CustomContentDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Ccid := c.Params("ccid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		CustomContentQuery := Orm.Select([]string{"*"})
		CustomContentQuery.Table("custom_contents")
		CustomContentQuery.Where("ccid", "=", Ccid)
		CustomContentQuery.Finish()
		err = CustomContentQuery.Execute()

		if err != nil {
			log.Printf("Cannot get custom content: %v\n", err)
			return c.Redirect("/panel/ozel-icerikler")
		}

		rows, err := CustomContentQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get custom content rows: %v\n", err)
			return c.Redirect("/panel/ozel-icerikler")
		}

		CustomContent := models.CustomContents{
			Ccid:              lib.String(rows[0]["ccid"]),
			Name:              lib.String(rows[0]["name"]),
			ContentType:       lib.String(rows[0]["content_type"]),
			SortOrder:         lib.Int64(rows[0]["sort_order"]),
			UrlName:           lib.String(rows[0]["url_name"]),
			ContentHtml:       lib.String(rows[0]["content_html"]),
			ContentJavascript: lib.String(rows[0]["content_javascript"]),
			ContentCss:        lib.String(rows[0]["content_css"]),
			Description:       lib.String(rows[0]["description"]),
			Route:             lib.String(rows[0]["route"]),
			IsActive:          rows[0]["is_active"].(bool),
			CreatedAt:         rows[0]["created_at"].(time.Time),
			UpdatedAt:         rows[0]["updated_at"].(time.Time),
		}

		return c.Render("views/panel/custom-contents-sayfalari/custom-content-duzenle", fiber.Map{
			"PathOnStart":   "../../../",
			"PageTitle":     "N-Hospital | Özel İçerik Düzenle",
			"User":          ourUser,
			"CustomContent": CustomContent,
			"Options":       GetOptions,
		}, "layouts/panel/panel")
	}
}

func HaberlerListPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		Haberler := Orm.Select([]string{"hid", "title", "url_name", "summary", "content", "cover_mid", "category", "tags", "author", "publish_date", "is_featured", "is_published", "views_count", "seo_title", "seo_description", "seo_keywords", "created_at", "updated_at"})
		Haberler.Table("haberler")
		Haberler.Where("is_published", "=", true)
		Haberler.And("publish_date", "<=", time.Now())
		Haberler.OrderBy("is_featured", "DESC")
		Haberler.OrderBy("publish_date", "DESC")
		Haberler.Limit(int(itemsPerPage))
		Haberler.Offset(offset)
		Haberler.Finish()

		err = Haberler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := Haberler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		HaberlerArray := []models.Haberler{}
		for _, row := range rows {
			HaberlerArray = append(HaberlerArray, models.Haberler{
				Hid:            lib.String(row["hid"]),
				Title:          lib.String(row["title"]),
				UrlName:        lib.String(row["url_name"]),
				Summary:        lib.String(row["summary"]),
				Content:        lib.String(row["content"]),
				CoverMid:       lib.Int64(row["cover_mid"]),
				Category:       lib.String(row["category"]),
				Tags:           lib.StringArray(row["tags"]),
				Author:         lib.String(row["author"]),
				PublishDate:    row["publish_date"].(time.Time),
				IsFeatured:     row["is_featured"].(bool),
				IsPublished:    row["is_published"].(bool),
				ViewsCount:     lib.Int64(row["views_count"]),
				SeoTitle:       lib.String(row["seo_title"]),
				SeoDescription: lib.String(row["seo_description"]),
				SeoKeywords:    lib.String(row["seo_keywords"]),
				CreatedAt:      row["created_at"].(time.Time),
				UpdatedAt:      row["updated_at"].(time.Time),
			})
		}

		return c.Render("views/panel/haberler-sayfalari/haberler", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Haberler",
			"Page":        c.Query("page"),
			"Haberler":    HaberlerArray,
			"Count":       len(HaberlerArray),
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func HaberPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		HaberId := c.Params("haber")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		if HaberId == "" {
			return c.Redirect("/panel/haberler")
		}

		// Get haber details
		GetHaber := Orm.Select([]string{"h.hid", "h.title", "h.url_name", "h.summary", "h.content", "h.cover_mid", "h.category", "h.tags", "h.author", "h.publish_date", "h.is_featured", "h.is_published", "h.views_count", "h.seo_title", "h.seo_description", "h.seo_keywords", "h.created_at", "h.updated_at", "m.file_path", "m.alt_text", "m.title as media_title"})
		GetHaber.Table("haberler h")
		GetHaber.LeftJoin("medias m", "h.cover_mid", "=", "m.mid")
		GetHaber.Where("h.hid", "=", HaberId)
		GetHaber.And("h.is_published", "=", true)
		GetHaber.And("h.publish_date", "<=", time.Now())
		GetHaber.Finish()

		err = GetHaber.Execute()

		if err != nil {
			log.Printf("Cannot get haber: %v\n", err)
			return c.Redirect("/panel/haberler")
		}

		rows, err := GetHaber.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Haber not found: %v\n", err)
			return c.Redirect("/panel/haberler")
		}

		row := rows[0]

		// Parse tags array
		var tags []string
		if row["tags"] != nil {
			tags = lib.StringArray(row["tags"])
		}

		haber := models.Haberler{
			Hid:            lib.String(row["hid"]),
			Title:          lib.String(row["title"]),
			UrlName:        lib.String(row["url_name"]),
			Summary:        lib.String(row["summary"]),
			Content:        lib.String(row["content"]),
			CoverMid:       lib.Int64(row["cover_mid"]),
			Category:       lib.String(row["category"]),
			Tags:           tags,
			Author:         lib.String(row["author"]),
			PublishDate:    lib.Time(row["publish_date"]),
			IsFeatured:     lib.Bool(row["is_featured"]),
			IsPublished:    lib.Bool(row["is_published"]),
			ViewsCount:     lib.Int64(row["views_count"]),
			SeoTitle:       lib.String(row["seo_title"]),
			SeoDescription: lib.String(row["seo_description"]),
			SeoKeywords:    lib.String(row["seo_keywords"]),
			CreatedAt:      lib.Time(row["created_at"]),
			UpdatedAt:      lib.Time(row["updated_at"]),
		}

		// Add cover image details if exists
		if haber.CoverMid > 0 {
			haber.CoverPath = lib.String(row["file_path"])
			haber.CoverAltText = lib.String(row["alt_text"])
			haber.CoverTitle = lib.String(row["media_title"])
		}

		return c.Render("views/panel/haberler-sayfalari/haber", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | " + haber.Title,
			"Haber":       haber,
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func HaberEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/haberler-sayfalari/haber-ekle", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Haber Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func HaberDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Hid := c.Params("haber")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		HaberQuery := Orm.Select([]string{"h.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		HaberQuery.Table("haberler h")
		HaberQuery.LeftJoin("medias m", "h.cover_mid", "=", "m.mid")
		HaberQuery.Where("h.hid", "=", Hid)
		HaberQuery.And("h.is_published", "=", true)
		HaberQuery.And("h.publish_date", "<=", time.Now())
		HaberQuery.Finish()
		err = HaberQuery.Execute()

		if err != nil {
			log.Printf("Cannot get haber: %v\n", err)
			return c.Redirect("/panel/haberler")
		}

		rows, err := HaberQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get haber rows: %v\n", err)
			return c.Redirect("/panel/haberler")
		}

		row := rows[0]

		// Parse tags array
		var tags []string
		if row["tags"] != nil {
			tags = lib.StringArray(row["tags"])
		}

		Haber := models.Haberler{
			Hid:            lib.String(row["hid"]),
			Title:          lib.String(row["title"]),
			UrlName:        lib.String(row["url_name"]),
			Summary:        lib.String(row["summary"]),
			Content:        lib.String(row["content"]),
			CoverMid:       lib.Int64(row["cover_mid"]),
			Category:       lib.String(row["category"]),
			Tags:           tags,
			Author:         lib.String(row["author"]),
			PublishDate:    lib.Time(row["publish_date"]),
			IsFeatured:     lib.Bool(row["is_featured"]),
			IsPublished:    lib.Bool(row["is_published"]),
			ViewsCount:     lib.Int64(row["views_count"]),
			SeoTitle:       lib.String(row["seo_title"]),
			SeoDescription: lib.String(row["seo_description"]),
			SeoKeywords:    lib.String(row["seo_keywords"]),
			CreatedAt:      lib.Time(row["created_at"]),
			UpdatedAt:      lib.Time(row["updated_at"]),
		}

		// Add cover image details if exists
		if Haber.CoverMid > 0 {
			Haber.CoverPath = lib.String(row["cover_path"])
			Haber.CoverAltText = lib.String(row["cover_alt_text"])
			Haber.CoverTitle = lib.String(row["cover_title"])
		}

		return c.Render("views/panel/haberler-sayfalari/haber-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | Haber Düzenle",
			"User":        ourUser,
			"Haber":       Haber,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TedkiklerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(utilities.Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		// Build query with filters
		Tedkikler := utilities.Orm.Select([]string{"t.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		Tedkikler.Table("tedkikler t")
		Tedkikler.LeftJoin("medias m", "t.cover_mid", "=", "m.mid")

		// Apply search filter
		if c.Query("search") != "" {
			searchTerm := c.Query("search")
			Tedkikler.Where("t.name", "ILIKE", "%"+searchTerm+"%")
			Tedkikler.Or("t.description", "ILIKE", "%"+searchTerm+"%")
		}

		// Apply status filter
		if c.Query("status") != "" {
			status := c.Query("status")
			if status == "active" {
				Tedkikler.And("t.is_active", "=", true)
			} else if status == "inactive" {
				Tedkikler.And("t.is_active", "=", false)
			}
		}

		// Apply sorting
		sortBy := c.Query("sort_by")
		if sortBy == "" {
			sortBy = "updated_at"
		}
		sortOrder := c.Query("sort_order")
		if sortOrder == "" {
			sortOrder = "DESC"
		}
		Tedkikler.OrderBy("t."+sortBy, sortOrder)

		Tedkikler.Limit(int(itemsPerPage))
		Tedkikler.Offset(offset)
		Tedkikler.Finish()
		err = Tedkikler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := Tedkikler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		TedkiklerArray := []models.Tedkikler{}
		for _, row := range rows {
			CoverPath := lib.String(row["cover_path"])
			CoverAltText := lib.String(row["cover_alt_text"])
			CoverTitle := lib.String(row["cover_title"])

			TedkiklerArray = append(TedkiklerArray, models.Tedkikler{
				Tid:          lib.String(row["tid"]),
				Name:         lib.String(row["name"]),
				UrlName:      lib.String(row["url_name"]),
				Description:  lib.String(row["description"]),
				CoverMid:     lib.Int64(row["cover_mid"]),
				CoverPath:    CoverPath,
				CoverAltText: CoverAltText,
				CoverTitle:   CoverTitle,
				IsActive:     lib.Bool(row["is_active"]),
				CreatedAt:    lib.Time(row["created_at"]),
				UpdatedAt:    lib.Time(row["updated_at"]),
			})
		}

		return c.Render("views/panel/tedkikler-sayfalari/tedkikler", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Tedkikler",
			"Page":        c.Query("page"),
			"Tedkikler":   TedkiklerArray,
			"Count":       len(TedkiklerArray),
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TedkikPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		TedkikId := c.Params("tid")

		if TedkikId == "" {
			return c.Redirect("/panel/tedkikler")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the tedkik with media information
		Tedkik := Orm.Select([]string{"t.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		Tedkik.Table("tedkikler t")
		Tedkik.LeftJoin("medias m", "t.cover_mid", "=", "m.mid")
		Tedkik.Where("t.tid", "=", TedkikId)
		Tedkik.Finish()
		err = Tedkik.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/tedkikler")
		}

		rows, err := Tedkik.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/tedkikler")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/tedkikler")
		}

		TedkikData := models.Tedkikler{
			Tid:          lib.String(rows[0]["tid"]),
			Name:         lib.String(rows[0]["name"]),
			UrlName:      lib.String(rows[0]["url_name"]),
			Description:  lib.String(rows[0]["description"]),
			CoverMid:     lib.Int64(rows[0]["cover_mid"]),
			CoverPath:    lib.String(rows[0]["cover_path"]),
			CoverAltText: lib.String(rows[0]["cover_alt_text"]),
			CoverTitle:   lib.String(rows[0]["cover_title"]),
			IsActive:     lib.Bool(rows[0]["is_active"]),
			CreatedAt:    lib.Time(rows[0]["created_at"]),
			UpdatedAt:    lib.Time(rows[0]["updated_at"]),
		}

		return c.Render("views/panel/tedkikler-sayfalari/tedkik", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | " + TedkikData.Name,
			"User":        ourUser,
			"Tedkik":      TedkikData,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TedkikEklePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/tedkikler-sayfalari/tedkik-ekle", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Tedkik Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func TedkikDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Tid := c.Params("tid")

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		TedkikQuery := Orm.Select([]string{"t.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		TedkikQuery.Table("tedkikler t")
		TedkikQuery.LeftJoin("medias m", "t.cover_mid", "=", "m.mid")
		TedkikQuery.Where("t.tid", "=", Tid)
		TedkikQuery.Finish()
		err = TedkikQuery.Execute()

		if err != nil {
			log.Printf("Cannot get tedkik: %v\n", err)
			return c.Redirect("/panel/tedkikler")
		}

		rows, err := TedkikQuery.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Cannot get tedkik rows: %v\n", err)
			return c.Redirect("/panel/tedkikler")
		}

		Tedkik := models.Tedkikler{
			Tid:          lib.String(rows[0]["tid"]),
			Name:         lib.String(rows[0]["name"]),
			UrlName:      lib.String(rows[0]["url_name"]),
			Description:  lib.String(rows[0]["description"]),
			IsActive:     rows[0]["is_active"].(bool),
			CoverMid:     lib.Int64(rows[0]["cover_mid"]),
			CoverPath:    lib.String(rows[0]["cover_path"]),
			CoverAltText: lib.String(rows[0]["cover_alt_text"]),
			CoverTitle:   lib.String(rows[0]["cover_title"]),
			CreatedAt:    rows[0]["created_at"].(time.Time),
			UpdatedAt:    rows[0]["updated_at"].(time.Time),
		}

		return c.Render("views/panel/tedkikler-sayfalari/tedkik-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | Tedkik Düzenle",
			"User":        ourUser,
			"Tedkik":      Tedkik,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func RandevuTalebiPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Rrid := c.Params("rrid")

		if Rrid == "" {
			return c.Redirect("/panel/randevu-talepleri")
		}

		Orm := utilities.Orm

		if c.Query("notification") == "true" {
			GetOriginalUrl := c.OriginalURL()
			fmt.Printf("GetOriginalUrl: %s\n", GetOriginalUrl)

			UpdateNotification := Orm.Update()
			UpdateNotification.Table("notifications")
			UpdateNotification.Set("is_read", true)
			UpdateNotification.Where("link", "=", GetOriginalUrl)

			UpdateNotification.Finish()

			err = UpdateNotification.Execute()

			if err != nil {
				log.Printf("%v\n", err)
			}

			ra, err := UpdateNotification.RowsAffected()
			if err != nil {
				log.Printf("%v\n", err)
			}

			if ra == 0 {
				log.Printf("Notification not found: %s\n", GetOriginalUrl)
			}
		}

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the randevu talebi
		RandevuTalebi := Orm.Select([]string{"rt.*", "d.title as doctor_title", "d.first_name as doctor_first_name", "d.last_name as doctor_last_name", "s.name as sube_name", "s.city as sube_city", "r.rid as related_appointment_rid", "r.appointment_date as related_appointment_date", "r.appointment_time as related_appointment_time", "r.status as related_appointment_status"})
		RandevuTalebi.Table("randevu_talepleri rt")
		RandevuTalebi.LeftJoin("doktorlar d", "rt.drid", "=", "d.drid")
		RandevuTalebi.LeftJoin("subeler s", "rt.sid", "=", "s.sid")
		RandevuTalebi.LeftJoin("randevular r", "rt.rrid", "=", "r.rrid")
		RandevuTalebi.Where("rt.rrid", "=", Rrid)
		RandevuTalebi.Finish()
		err = RandevuTalebi.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/randevu-talepleri")
		}

		rows, err := RandevuTalebi.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/randevu-talepleri")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/randevu-talepleri")
		}

		RandevuTalebiData := models.RandevuRequests{
			Rrid:             lib.String(rows[0]["rrid"]),
			PatientFirstName: lib.String(rows[0]["patient_first_name"]),
			PatientLastName:  lib.String(rows[0]["patient_last_name"]),
			PatientPhone:     lib.String(rows[0]["patient_phone"]),
			PatientEmail:     lib.String(rows[0]["patient_email"]),
			PreferredDate:    lib.Time(rows[0]["preferred_date"]),
			PreferredTime:    lib.Time(rows[0]["preferred_time"]),
			Message:          lib.String(rows[0]["message"]),
			Drid:             lib.String(rows[0]["drid"]),
			Sid:              lib.String(rows[0]["sid"]),
			CreatedAt:        lib.Time(rows[0]["created_at"]),
			UpdatedAt:        lib.Time(rows[0]["updated_at"]),
		}

		// Fetch doctor info if drid is set
		DoktorInfo := models.Doktorlar{
			Drid:      lib.String(rows[0]["drid"]),
			Title:     lib.String(rows[0]["doctor_title"]),
			FirstName: lib.String(rows[0]["doctor_first_name"]),
			LastName:  lib.String(rows[0]["doctor_last_name"]),
		}

		// Fetch sube info if sid is set
		SubeInfo := models.Subeler{
			Sid:  lib.String(rows[0]["sid"]),
			Name: lib.String(rows[0]["sube_name"]),
			City: lib.String(rows[0]["sube_city"]),
		}

		RelatedAppointment := models.Randevular{
			Rid:             lib.String(rows[0]["related_appointment_rid"]),
			AppointmentDate: lib.Time(rows[0]["related_appointment_date"]),
			AppointmentTime: lib.Time(rows[0]["related_appointment_time"]),
			Status:          lib.String(rows[0]["related_appointment_status"]),
		}

		return c.Render("views/panel/randevular-sayfalari/randevu-talebi", fiber.Map{
			"PathOnStart":           "../../",
			"PageTitle":             "N-Hospital | Randevu Talebi #" + RandevuTalebiData.Rrid,
			"User":                  ourUser,
			"RandevuTalebi":         RandevuTalebiData,
			"DoktorInfo":            DoktorInfo,
			"SubeInfo":              SubeInfo,
			"HasRelatedAppointment": lib.String(rows[0]["related_appointment_rid"]) != "",
			"RelatedAppointment":    RelatedAppointment,
			"Options":               GetOptions,
		}, "layouts/panel/panel")
	}
}

func RandevuTalepleriPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		// Build query with filters
		RandevuTalepleri := Orm.Select([]string{"rrid", "patient_first_name", "patient_last_name", "patient_phone", "patient_email", "preferred_date", "preferred_time", "message", "drid", "sid", "created_at", "updated_at"})
		RandevuTalepleri.Table("randevu_talepleri")

		// Apply search filter
		if c.Query("search") != "" {
			searchTerm := c.Query("search")
			RandevuTalepleri.Like("WHERE", "patient_first_name", searchTerm, "contains")
			RandevuTalepleri.Like("OR", "patient_last_name", searchTerm, "contains")
			RandevuTalepleri.Like("OR", "patient_phone", searchTerm, "contains")
			RandevuTalepleri.Like("OR", "patient_email", searchTerm, "contains")
		}

		// Apply date filter
		if c.Query("date") != "" {
			dateFilter := c.Query("date")
			now := time.Now()
			switch dateFilter {
			case "today":
				today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				tomorrow := today.AddDate(0, 0, 1)
				RandevuTalepleri.Where("created_at", ">=", today)
				RandevuTalepleri.Where("created_at", "<", tomorrow)
			case "week":
				weekStart := now.AddDate(0, 0, -int(now.Weekday()))
				weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
				RandevuTalepleri.Where("created_at", ">=", weekStart)
			case "month":
				monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
				RandevuTalepleri.Where("created_at", ">=", monthStart)
			}
		}

		// Apply sorting
		sortBy := c.Query("sort_by")
		if sortBy == "" {
			sortBy = "created_at"
		}
		sortOrder := c.Query("sort_order")
		if sortOrder == "" {
			sortOrder = "DESC"
		}
		RandevuTalepleri.OrderBy(sortBy, sortOrder)

		RandevuTalepleri.Limit(int(itemsPerPage))
		RandevuTalepleri.Offset(offset)
		RandevuTalepleri.Finish()

		err = RandevuTalepleri.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := RandevuTalepleri.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		RandevuTalepleriArray := []models.RandevuRequests{}
		for _, row := range rows {
			RandevuTalepleriArray = append(RandevuTalepleriArray, models.RandevuRequests{
				Rrid:             lib.String(row["rrid"]),
				PatientFirstName: lib.String(row["patient_first_name"]),
				PatientLastName:  lib.String(row["patient_last_name"]),
				PatientPhone:     lib.String(row["patient_phone"]),
				PatientEmail:     lib.String(row["patient_email"]),
				PreferredDate:    lib.Time(row["preferred_date"]),
				PreferredTime:    lib.Time(row["preferred_time"]),
				Message:          lib.String(row["message"]),
				Drid:             lib.String(row["drid"]),
				Sid:              lib.String(row["sid"]),
				CreatedAt:        row["created_at"].(time.Time),
				UpdatedAt:        row["updated_at"].(time.Time),
			})
		}

		return c.Render("views/panel/randevular-sayfalari/randevu-talepleri", fiber.Map{
			"PathOnStart":      "../",
			"PageTitle":        "N-Hospital | Randevu Talepleri",
			"Page":             c.Query("page"),
			"RandevuTalepleri": RandevuTalepleriArray,
			"Count":            len(RandevuTalepleriArray),
			"User":             ourUser,
			"Options":          GetOptions,
		}, "layouts/panel/panel")
	}
}

func RandevularPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Get pagination parameters
		page := c.QueryInt("page", 1)
		perPage := 10

		if page < 1 {
			page = 1
		}

		offset := (page - 1) * perPage

		// Get randevular data
		RandevularQuery := Orm.Select([]string{
			"r.rid", "r.patient_first_name", "r.patient_last_name", "r.patient_phone",
			"r.patient_email", "r.appointment_date", "r.appointment_time", "r.duration",
			"r.status", "r.payment_status", "r.price", "r.created_at", "r.updated_at",
			"d.title", "d.first_name as doctor_first_name", "d.last_name as doctor_last_name",
			"s.name as sube_name", "s.city as sube_city",
			"b.name as branch_name",
		})
		RandevularQuery.Table("randevular r")
		RandevularQuery.LeftJoin("doktorlar d", "r.drid", "=", "d.drid")
		RandevularQuery.LeftJoin("subeler s", "r.sid", "=", "s.sid")
		RandevularQuery.LeftJoin("branslar b", "r.brid", "=", "b.brid")
		RandevularQuery.OrderBy("r.appointment_date", "desc")
		RandevularQuery.Limit(perPage)
		RandevularQuery.Offset(offset)
		RandevularQuery.Finish()

		err = RandevularQuery.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := RandevularQuery.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Get total count
		CountQuery := Orm.Select([]string{"COUNT(*) as count"})
		CountQuery.Table("randevular")
		CountQuery.Finish()
		err = CountQuery.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		countRows, err := CountQuery.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Count := 0
		if len(countRows) > 0 {
			Count = int(lib.Int64(countRows[0]["count"]))
		}

		// Map rows to Randevular structs
		RandevularArray := []models.Randevular{}
		for _, row := range rows {
			randevu := models.Randevular{
				Rid:              lib.String(row["rid"]),
				PatientFirstName: lib.String(row["patient_first_name"]),
				PatientLastName:  lib.String(row["patient_last_name"]),
				PatientPhone:     lib.String(row["patient_phone"]),
				PatientEmail:     lib.String(row["patient_email"]),
				AppointmentDate:  lib.Time(row["appointment_date"]),
				AppointmentTime:  lib.Time(row["appointment_time"]),
				Duration:         lib.Int64(row["duration"]),
				Status:           lib.String(row["status"]),
				PaymentStatus:    lib.String(row["payment_status"]),
				Price:            lib.Float64(row["price"]),
				CreatedAt:        lib.Time(row["created_at"]),
				UpdatedAt:        lib.Time(row["updated_at"]),
			}

			// Add doctor name if available
			if lib.String(row["doctor_first_name"]) != "" {
				randevu.DoctorName = lib.String(row["title"]) + " " + lib.String(row["doctor_first_name"]) + " " + lib.String(row["doctor_last_name"])
			}

			// Add sube name if available
			if lib.String(row["sube_name"]) != "" {
				randevu.SubeName = lib.String(row["sube_name"])
				randevu.SubeCity = lib.String(row["sube_city"])
			}

			// Add branch name if available
			if lib.String(row["branch_name"]) != "" {
				randevu.BranchName = lib.String(row["branch_name"])
			}

			RandevularArray = append(RandevularArray, randevu)
		}

		return c.Render("views/panel/randevular-sayfalari/randevular", fiber.Map{
			"PathOnStart": "../../",
			"PageTitle":   "N-Hospital | Randevular",
			"User":        ourUser,
			"Randevular":  RandevularArray,
			"Count":       Count,
			"Page":        page,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func RandevuPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Rid := c.Params("rid")

		if Rid == "" {
			fmt.Printf("Randevu not found: %s\n", Rid)
			return c.Redirect("/panel/randevular")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the randevu
		RandevuQuery := Orm.Select([]string{"r.*", "d.title as doctor_title", "d.first_name as doctor_first_name",
			"d.last_name as doctor_last_name", "s.name as sube_name", "s.city as sube_city", "b.name as branch_name",
			"rt.rrid as related_appointment_rid", "rt.patient_first_name as related_appointment_patient_first_name",
			"rt.patient_last_name as related_appointment_patient_last_name", "rt.patient_phone as related_appointment_patient_phone",
			"rt.patient_email as related_appointment_patient_email", "rt.preferred_date as related_appointment_preferred_date",
			"rt.preferred_time as related_appointment_preferred_time", "rt.message as related_appointment_message",
			"ak.name as related_appointment_anlasmali_kurum_name", "tb.name as related_appointment_tibbi_birim_name",
			"t.name as related_appointment_tedkik_name"})
		RandevuQuery.Table("randevular r")
		RandevuQuery.LeftJoin("doktorlar d", "r.drid", "=", "d.drid")
		RandevuQuery.LeftJoin("subeler s", "r.sid", "=", "s.sid")
		RandevuQuery.LeftJoin("branslar b", "r.brid", "=", "b.brid")
		RandevuQuery.LeftJoin("randevu_talepleri rt", "r.rrid", "=", "rt.rrid")
		RandevuQuery.LeftJoin("anlasmali_kurumlar ak", "r.akid", "=", "ak.akid")
		RandevuQuery.LeftJoin("tibbi_birimler tb", "r.tbid", "=", "tb.tbid")
		RandevuQuery.LeftJoin("tedkikler t", "r.tid", "=", "t.tid")
		RandevuQuery.Where("r.rid", "=", Rid)
		RandevuQuery.Finish()

		fmt.Printf("RandevuQuery: %v\n", RandevuQuery.Query)

		err = RandevuQuery.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/randevular")
		}

		rows, err := RandevuQuery.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/randevular")
		}

		if len(rows) == 0 {
			fmt.Printf("Randevu not found: %s\n", Rid)
			return c.Redirect("/panel/randevular")
		}

		RandevuData := models.Randevular{
			Rid:              lib.String(rows[0]["rid"]),
			PatientFirstName: lib.String(rows[0]["patient_first_name"]),
			PatientLastName:  lib.String(rows[0]["patient_last_name"]),
			PatientPhone:     lib.String(rows[0]["patient_phone"]),
			PatientEmail:     lib.String(rows[0]["patient_email"]),
			PatientTcKimlik:  lib.String(rows[0]["patient_tc_kimlik"]),
			PatientBirthDate: lib.Time(rows[0]["patient_birth_date"]),
			PatientGender:    lib.String(rows[0]["patient_gender"]),
			Drid:             lib.String(rows[0]["drid"]),
			Brid:             lib.String(rows[0]["brid"]),
			Sid:              lib.String(rows[0]["sid"]),
			Akid:             lib.String(rows[0]["akid"]),
			Tid:              lib.String(rows[0]["tid"]),
			Tbid:             lib.String(rows[0]["tbid"]),
			Rrid:             lib.String(rows[0]["rrid"]),
			AppointmentDate:  lib.Time(rows[0]["appointment_date"]),
			AppointmentTime:  lib.Time(rows[0]["appointment_time"]),
			Duration:         lib.Int64(rows[0]["duration"]),
			Status:           lib.String(rows[0]["status"]),
			Notes:            lib.String(rows[0]["notes"]),
			Complaint:        lib.String(rows[0]["complaint"]),
			CancelReason:     lib.String(rows[0]["cancel_reason"]),
			ReminderSent:     lib.Bool(rows[0]["reminder_sent"]),
			ConfirmationCode: lib.String(rows[0]["confirmation_code"]),
			Price:            lib.Float64(rows[0]["price"]),
			PaymentStatus:    lib.String(rows[0]["payment_status"]),
			CreatedAt:        lib.Time(rows[0]["created_at"]),
			UpdatedAt:        lib.Time(rows[0]["updated_at"]),
		}

		// Fetch doctor info if drid is set
		DoktorInfo := models.Doktorlar{
			Drid:       lib.String(rows[0]["drid"]),
			Title:      lib.String(rows[0]["doctor_title"]),
			FirstName:  lib.String(rows[0]["doctor_first_name"]),
			LastName:   lib.String(rows[0]["doctor_last_name"]),
			Brid:       lib.String(rows[0]["brid"]),
			BranchName: lib.String(rows[0]["branch_name"]),
		}

		// Fetch sube info if sid is set
		SubeInfo := models.Subeler{
			Sid:  lib.String(rows[0]["sid"]),
			Name: lib.String(rows[0]["sube_name"]),
			City: lib.String(rows[0]["sube_city"]),
		}

		// Fetch tibbi birim info if tbid is set
		TibbiBirimInfo := models.TibbiBirimler{
			Tbid:        lib.String(rows[0]["tbid"]),
			Name:        lib.String(rows[0]["tibbi_birim_name"]),
			Description: lib.String(rows[0]["tibbi_birim_description"]),
		}

		// Fetch tedkik info if tid is set
		TedkikInfo := models.Tedkikler{
			Tid:         lib.String(rows[0]["tid"]),
			Name:        lib.String(rows[0]["tedkik_name"]),
			Description: lib.String(rows[0]["tedkik_description"]),
		}

		// Fetch anlasmali kurum info if akid is set
		AnlasmaliKurumInfo := models.AnlasmaliKurumlar{
			Akid:          lib.String(rows[0]["akid"]),
			Name:          lib.String(rows[0]["anlasmali_kurum_name"]),
			ContactPerson: lib.String(rows[0]["anlasmali_kurum_contact_person"]),
			Phone:         lib.String(rows[0]["anlasmali_kurum_phone"]),
		}

		// Fetch related randevu talebi if rrid is set
		RandevuTalebiInfo := models.RandevuRequests{
			Rrid:             lib.String(rows[0]["related_appointment_rid"]),
			PatientFirstName: lib.String(rows[0]["related_appointment_patient_first_name"]),
			PatientLastName:  lib.String(rows[0]["related_appointment_patient_last_name"]),
			PatientPhone:     lib.String(rows[0]["related_appointment_patient_phone"]),
			PatientEmail:     lib.String(rows[0]["related_appointment_patient_email"]),
			PreferredDate:    lib.Time(rows[0]["related_appointment_preferred_date"]),
			PreferredTime:    lib.Time(rows[0]["related_appointment_preferred_time"]),
			Message:          lib.String(rows[0]["related_appointment_message"]),
		}

		return c.Render("views/panel/randevular-sayfalari/randevu", fiber.Map{
			"PathOnStart":        "../../",
			"PageTitle":          "N-Hospital | Randevu #" + RandevuData.Rid,
			"User":               ourUser,
			"Randevu":            RandevuData,
			"DoktorInfo":         DoktorInfo,
			"SubeInfo":           SubeInfo,
			"TibbiBirimInfo":     TibbiBirimInfo,
			"TedkikInfo":         TedkikInfo,
			"AnlasmaliKurumInfo": AnlasmaliKurumInfo,
			"RandevuTalebiInfo":  RandevuTalebiInfo,
			"Options":            GetOptions,
		}, "layouts/panel/panel")
	}
}

func RandevuDuzenlePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Rid := c.Params("rid")

		if Rid == "" {
			return c.Redirect("/panel/randevular")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Fetch the randevu
		RandevuQuery := Orm.Select([]string{
			"rid", "patient_first_name", "patient_last_name", "patient_phone", "patient_email",
			"patient_tc_kimlik", "patient_birth_date", "patient_gender", "drid", "brid", "sid",
			"akid", "tid", "tbid", "rrid", "appointment_date", "appointment_time", "duration",
			"status", "notes", "complaint", "cancel_reason", "reminder_sent", "confirmation_code",
			"price", "payment_status", "created_at", "updated_at",
		})
		RandevuQuery.Table("randevular")
		RandevuQuery.Where("rid", "=", Rid)
		RandevuQuery.Finish()
		err = RandevuQuery.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/randevular")
		}

		rows, err := RandevuQuery.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/randevular")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/randevular")
		}

		RandevuData := models.Randevular{
			Rid:              lib.String(rows[0]["rid"]),
			PatientFirstName: lib.String(rows[0]["patient_first_name"]),
			PatientLastName:  lib.String(rows[0]["patient_last_name"]),
			PatientPhone:     lib.String(rows[0]["patient_phone"]),
			PatientEmail:     lib.String(rows[0]["patient_email"]),
			PatientTcKimlik:  lib.String(rows[0]["patient_tc_kimlik"]),
			PatientBirthDate: lib.Time(rows[0]["patient_birth_date"]),
			PatientGender:    lib.String(rows[0]["patient_gender"]),
			Drid:             lib.String(rows[0]["drid"]),
			Brid:             lib.String(rows[0]["brid"]),
			Sid:              lib.String(rows[0]["sid"]),
			Akid:             lib.String(rows[0]["akid"]),
			Tid:              lib.String(rows[0]["tid"]),
			Tbid:             lib.String(rows[0]["tbid"]),
			Rrid:             lib.String(rows[0]["rrid"]),
			AppointmentDate:  lib.Time(rows[0]["appointment_date"]),
			AppointmentTime:  lib.Time(rows[0]["appointment_time"]),
			Duration:         lib.Int64(rows[0]["duration"]),
			Status:           lib.String(rows[0]["status"]),
			Notes:            lib.String(rows[0]["notes"]),
			Complaint:        lib.String(rows[0]["complaint"]),
			CancelReason:     lib.String(rows[0]["cancel_reason"]),
			ReminderSent:     lib.Bool(rows[0]["reminder_sent"]),
			ConfirmationCode: lib.String(rows[0]["confirmation_code"]),
			Price:            lib.Float64(rows[0]["price"]),
			PaymentStatus:    lib.String(rows[0]["payment_status"]),
			CreatedAt:        lib.Time(rows[0]["created_at"]),
			UpdatedAt:        lib.Time(rows[0]["updated_at"]),
		}

		return c.Render("views/panel/randevular-sayfalari/randevu-duzenle", fiber.Map{
			"PathOnStart": "../../../",
			"PageTitle":   "N-Hospital | Randevu Düzenle #" + RandevuData.Rid,
			"User":        ourUser,
			"Randevu":     RandevuData,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}

func ContactRequestsPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		itemsPerPage := GetOptions.Options.ItemsPerPage
		var offset int = (Page - 1) * int(itemsPerPage)

		ContactRequests := Orm.Select([]string{"crid", "first_name", "last_name", "email", "phone", "subject", "message", "department", "priority", "status", "assigned_to", "response", "response_date", "ip_address", "user_agent", "source", "is_read", "created_at", "updated_at"})
		ContactRequests.Table("contact_requests")
		ContactRequests.OrderBy("is_read", "ASC")
		ContactRequests.OrderBy("created_at", "DESC")
		ContactRequests.Limit(int(itemsPerPage))
		ContactRequests.Offset(offset)
		ContactRequests.Finish()

		err = ContactRequests.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		rows, err := ContactRequests.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel")
		}

		ContactRequestsArray := []models.ContactRequests{}
		for _, row := range rows {
			ContactRequestsArray = append(ContactRequestsArray, models.ContactRequests{
				Crid:         lib.String(row["crid"]),
				FirstName:    lib.String(row["first_name"]),
				LastName:     lib.String(row["last_name"]),
				Email:        lib.String(row["email"]),
				Phone:        lib.String(row["phone"]),
				Subject:      lib.String(row["subject"]),
				Message:      lib.String(row["message"]),
				Department:   lib.String(row["department"]),
				Priority:     lib.String(row["priority"]),
				Status:       lib.String(row["status"]),
				AssignedTo:   lib.String(row["assigned_to"]),
				Response:     lib.String(row["response"]),
				ResponseDate: lib.Time(row["response_date"]),
				IpAddress:    lib.String(row["ip_address"]),
				UserAgent:    lib.String(row["user_agent"]),
				Source:       lib.String(row["source"]),
				IsRead:       row["is_read"].(bool),
				CreatedAt:    row["created_at"].(time.Time),
				UpdatedAt:    row["updated_at"].(time.Time),
			})
		}

		return c.Render("views/panel/contact-requests-sayfalari/contact-requests", fiber.Map{
			"PathOnStart":     "../",
			"PageTitle":       "N-Hospital | İletişim Talepleri",
			"Page":            c.Query("page"),
			"ContactRequests": ContactRequestsArray,
			"Count":           len(ContactRequestsArray),
			"User":            ourUser,
			"Options":         GetOptions,
		}, "layouts/panel/panel")
	}
}

func ContactRequestPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Get contact request ID from route parameter
		ContactRequestId := c.Params("crid")
		if ContactRequestId == "" {
			return c.Redirect("/panel/contact-requests")
		}

		// Fetch the specific contact request
		ContactRequest := Orm.Select([]string{"crid", "first_name", "last_name", "email", "phone", "subject", "message", "department", "priority", "status", "assigned_to", "response", "response_date", "ip_address", "user_agent", "source", "is_read", "created_at", "updated_at"})
		ContactRequest.Table("contact_requests")
		ContactRequest.Where("crid", "=", ContactRequestId)
		ContactRequest.Finish()

		err = ContactRequest.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/contact-requests")
		}

		rows, err := ContactRequest.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/contact-requests")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/contact-requests")
		}

		row := rows[0]
		ContactRequestData := models.ContactRequests{
			Crid:         lib.String(row["crid"]),
			FirstName:    lib.String(row["first_name"]),
			LastName:     lib.String(row["last_name"]),
			Email:        lib.String(row["email"]),
			Phone:        lib.String(row["phone"]),
			Subject:      lib.String(row["subject"]),
			Message:      lib.String(row["message"]),
			Department:   lib.String(row["department"]),
			Priority:     lib.String(row["priority"]),
			Status:       lib.String(row["status"]),
			AssignedTo:   lib.String(row["assigned_to"]),
			Response:     lib.String(row["response"]),
			ResponseDate: lib.Time(row["response_date"]),
			IpAddress:    lib.String(row["ip_address"]),
			UserAgent:    lib.String(row["user_agent"]),
			Source:       lib.String(row["source"]),
			IsRead:       row["is_read"].(bool),
			CreatedAt:    row["created_at"].(time.Time),
			UpdatedAt:    row["updated_at"].(time.Time),
		}

		return c.Render("views/panel/contact-requests-sayfalari/contact-request", fiber.Map{
			"PathOnStart":    "../../",
			"PageTitle":      "N-Hospital | İletişim Talebi",
			"ContactRequest": ContactRequestData,
			"User":           ourUser,
			"Options":        GetOptions,
		}, "layouts/panel/panel")
	}
}

func RespondToContactRequestPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		ContactRequestId := c.Query("crid")

		// Fetch all contact requests for dropdown
		ContactRequests := Orm.Select([]string{"crid", "subject", "is_replied"})
		ContactRequests.Table("contact_requests")
		if ContactRequestId != "" {
			ContactRequests.Where("crid", "=", ContactRequestId)
		}
		ContactRequests.OrderBy("created_at", "DESC")
		ContactRequests.Finish()

		err = ContactRequests.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/contact-requests")
		}

		rows, err := ContactRequests.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/contact-requests")
		}

		ContactRequestsArray := []models.ContactRequests{}
		for _, row := range rows {
			ContactRequestsArray = append(ContactRequestsArray, models.ContactRequests{
				Crid:      lib.String(row["crid"]),
				Subject:   lib.String(row["subject"]),
				IsReplied: row["is_replied"].(bool),
			})
		}

		if len(ContactRequestsArray) == 0 {
			return c.Redirect("/panel/contact-requests")
		}

		return c.Render("views/panel/contact-requests-sayfalari/respond-to-contact-request", fiber.Map{
			"PathOnStart":     "../",
			"PageTitle":       "N-Hospital | İletişim Talebine Cevap Ver",
			"ContactRequests": ContactRequestsArray,
			"User":            ourUser,
			"Options":         GetOptions,
		}, "layouts/panel/panel")
	}
}

func JobApplicationsPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Get pagination parameters
		page := c.Query("page")

		if page == "" {
			page = "1"
		}

		perPage := c.Query("per_page", "10")
		//search := c.Query("search", "")
		status := c.Query("status", "")
		position := c.Query("position", "")
		sortBy := c.Query("sort_by", "created_at")
		sortOrder := c.Query("sort_order", "desc")

		// Convert to integers
		pageInt, _ := strconv.Atoi(page)
		perPageInt, _ := strconv.Atoi(perPage)

		// Build query
		JobApplications := Orm.Select([]string{"jaid", "first_name", "last_name", "email", "phone", "position_applied", "department", "experience_years", "university", "status", "interview_date", "is_read", "created_at", "updated_at"})
		JobApplications.Table("job_applications")

		// Apply filters
		/*if search != "" {
			JobApplications.Where("(first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ? OR position_applied ILIKE ?)", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
		}*/
		if status != "" {
			JobApplications.Where("status", "=", status)
		}
		if position != "" {
			JobApplications.And("position_applied", "=", position)
		}

		// Apply sorting
		JobApplications.OrderBy(sortBy, strings.ToUpper(sortOrder))
		JobApplications.Finish()

		err = JobApplications.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := JobApplications.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		// Calculate pagination
		totalCount := len(rows)
		totalPages := (totalCount + perPageInt - 1) / perPageInt
		startIndex := (pageInt - 1) * perPageInt
		endIndex := startIndex + perPageInt

		if startIndex >= totalCount {
			startIndex = 0
			endIndex = perPageInt
		}
		if endIndex > totalCount {
			endIndex = totalCount
		}

		// Get paginated results
		paginatedRows := rows[startIndex:endIndex]

		JobApplicationsArray := []models.JobApplications{}
		for _, row := range paginatedRows {
			JobApplicationsArray = append(JobApplicationsArray, models.JobApplications{
				Jaid:            lib.String(row["jaid"]),
				FirstName:       lib.String(row["first_name"]),
				LastName:        lib.String(row["last_name"]),
				Email:           lib.String(row["email"]),
				Phone:           lib.String(row["phone"]),
				PositionApplied: lib.String(row["position_applied"]),
				Department:      lib.String(row["department"]),
				ExperienceYears: lib.Int64(row["experience_years"]),
				University:      lib.String(row["university"]),
				Status:          lib.String(row["status"]),
				InterviewDate:   lib.Time(row["interview_date"]),
				IsRead:          lib.Bool(row["is_read"]),
				CreatedAt:       lib.Time(row["created_at"]),
				UpdatedAt:       lib.Time(row["updated_at"]),
			})
		}

		return c.Render("views/panel/job-applications-sayfalari/job-applications", fiber.Map{
			"PathOnStart":     "../",
			"PageTitle":       "N-Hospital | İş Başvuruları",
			"JobApplications": JobApplicationsArray,
			"Count":           totalCount,
			"Page":            pageInt,
			"PerPage":         perPageInt,
			"TotalPages":      totalPages,
			"User":            ourUser,
			"Options":         GetOptions,
		}, "layouts/panel/panel")
	}
}

func JobApplicationPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Notification := c.Query("notification")

		if Notification == "true" {
			UpdateNotification := Orm.Update()
			UpdateNotification.Table("notifications")
			UpdateNotification.Set("is_read", true)
			UpdateNotification.Where("link", "=", c.OriginalURL())
			UpdateNotification.Finish()

			err = UpdateNotification.Execute()
			if err != nil {
				log.Printf("%v\n", err)
			}
		}

		// Get job application ID from route parameter
		JobApplicationId := c.Params("jaid")
		if JobApplicationId == "" {
			return c.Redirect("/panel/job-applications")
		}

		// Fetch job application details
		JobApplication := Orm.Select([]string{
			"ja.*",
			"m.file_path as cv_file_path",
			"m.alt_text as cv_file_alt_text",
			"m.title as cv_file_title",
			"m2.file_path as diploma_file_path",
			"m2.alt_text as diploma_file_alt_text",
			"m2.title as diploma_file_title",
		})
		JobApplication.Table("job_applications ja")
		JobApplication.LeftJoin("medias m", "ja.cv_file_mid", "=", "m.mid")
		JobApplication.LeftJoin("medias m2", "ja.diploma_file_mid", "=", "m2.mid")
		JobApplication.Where("jaid", "=", JobApplicationId)
		JobApplication.Finish()

		err = JobApplication.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/job-applications")
		}

		rows, err := JobApplication.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/job-applications")
		}

		if len(rows) == 0 {
			return c.Redirect("/panel/job-applications")
		}

		row := rows[0]
		JobApplicationData := models.JobApplications{
			Jaid:               lib.String(row["jaid"]),
			FirstName:          lib.String(row["first_name"]),
			LastName:           lib.String(row["last_name"]),
			Email:              lib.String(row["email"]),
			Phone:              lib.String(row["phone"]),
			BirthDate:          lib.Time(row["birth_date"]),
			Gender:             lib.String(row["gender"]),
			City:               lib.String(row["city"]),
			University:         lib.String(row["university"]),
			Department:         lib.String(row["department"]),
			GraduationYear:     lib.Int64(row["graduation_year"]),
			ExperienceYears:    lib.Int64(row["experience_years"]),
			PositionApplied:    lib.String(row["position_applied"]),
			Sid:                lib.Int64(row["sid"]),
			SalaryExpectation:  lib.Float64(row["salary_expectation"]),
			AvailableStartDate: lib.Time(row["available_start_date"]),
			Languages:          lib.String(row["languages"]),
			Skills:             lib.String(row["skills"]),
			CoverLetter:        lib.String(row["cover_letter"]),
			CvFileMid:          lib.Int64(row["cv_file_mid"]),
			DiplomaFileMid:     lib.Int64(row["diploma_file_mid"]),
			WorkReferences:     lib.String(row["work_references"]),
			Status:             lib.String(row["status"]),
			Notes:              lib.String(row["notes"]),
			InterviewDate:      lib.Time(row["interview_date"]),
			InterviewNotes:     lib.String(row["interview_notes"]),
			RejectionReason:    lib.String(row["rejection_reason"]),
			IsRead:             lib.Bool(row["is_read"]),
			CreatedAt:          lib.Time(row["created_at"]),
			UpdatedAt:          lib.Time(row["updated_at"]),
			CvFilePath:         lib.String(row["cv_file_path"]),
			DiplomaFilePath:    lib.String(row["diploma_file_path"]),
		}

		return c.Render("views/panel/job-applications-sayfalari/job-application", fiber.Map{
			"PathOnStart":    "../../",
			"PageTitle":      "N-Hospital | İş Başvurusu",
			"JobApplication": JobApplicationData,
			"User":           ourUser,
			"Options":        GetOptions,
		}, "layouts/panel/panel")
	}
}

func RespondToJobApplicationPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		JobApplicationId := c.Query("jaid")

		// Fetch all job applications for dropdown
		JobApplications := Orm.Select([]string{"jaid", "first_name", "last_name", "position_applied", "status"})
		JobApplications.Table("job_applications")
		if JobApplicationId != "" {
			JobApplications.Where("jaid", "=", JobApplicationId)
		}
		JobApplications.OrderBy("created_at", "DESC")
		JobApplications.Finish()

		err = JobApplications.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/is-basvurulari")
		}

		rows, err := JobApplications.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/is-basvurulari")
		}

		JobApplicationsArray := []models.JobApplications{}
		for _, row := range rows {
			JobApplicationsArray = append(JobApplicationsArray, models.JobApplications{
				Jaid:            lib.String(row["jaid"]),
				FirstName:       lib.String(row["first_name"]),
				LastName:        lib.String(row["last_name"]),
				PositionApplied: lib.String(row["position_applied"]),
				Status:          lib.String(row["status"]),
			})
		}

		return c.Render("views/panel/job-applications-sayfalari/respond-to-job-application", fiber.Map{
			"PathOnStart":     "../../",
			"PageTitle":       "N-Hospital | İş Başvurusuna Cevap Ver",
			"JobApplications": JobApplicationsArray,
			"User":            ourUser,
			"Options":         GetOptions,
		}, "layouts/panel/panel")
	}
}

func DocumentationPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForPanel(Orm, []string{}, []string{}, ourUser)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		return c.Render("views/panel/dokumantasyon", fiber.Map{
			"PathOnStart": "../",
			"PageTitle":   "N-Hospital | Dosya Ekle",
			"User":        ourUser,
			"Options":     GetOptions,
		}, "layouts/panel/panel")
	}
}
