package headerbuttons

import (
	"fmt"
	lib "lib"
	"log"
	"models"

	"github.com/gofiber/fiber/v2"
)

func AddHeaderButton(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		inputs := models.HeaderButton{}
		c.BodyParser(&inputs)

		Orm := utilities.Orm

		if (inputs.ButtonType == "subeler" || inputs.ButtonType == "branslar" || inputs.ButtonType == "doktorlar" || inputs.ButtonType == "tedkikler" || inputs.ButtonType == "tibbi_birimler") && inputs.ParentId != "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Girdiğiniz buton tipi özel bir buton tipidir, herhangi bir alt buton olamaz. Bu tiplerin alt butonları otomatik olarak oluşturulur.",
			})
		}

		// Validate required fields
		if inputs.Title == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Title is required",
			})
		}

		if inputs.Target == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Target is required",
			})
		}

		if inputs.SortOrder < 1 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Sort order must be at least 1",
			})
		}

		// Validate target value
		if inputs.Target != "_self" && inputs.Target != "_blank" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Target must be either _self or _blank",
			})
		}

		// Validate URL if provided
		if inputs.Url != "" {
			if len(inputs.Url) > 255 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "URL is too long",
				})
			}
		}

		// Validate icon if provided
		if inputs.Icon != "" {
			if len(inputs.Icon) > 50 {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Icon name is too long",
				})
			}
		}

		// Handle parent_id - convert empty string to nil for database
		var parentId interface{}
		if inputs.ParentId == "" || inputs.ParentId == "0" {
			parentId = nil
		} else {
			parentId = inputs.ParentId
		}

		fmt.Printf("parentId: %v\n", parentId)

		// Begin transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Insert header button
		columns := []string{"title", "url", "target", "icon", "sort_order", "is_active", "parent_id", "button_type"}
		values := []interface{}{inputs.Title, inputs.Url, inputs.Target, inputs.Icon, inputs.SortOrder, inputs.IsActive, parentId, inputs.ButtonType}

		InsertHeaderButton := Orm.Insert(columns, values)
		InsertHeaderButton.Table("header_buttons")
		InsertHeaderButton.Returning("hbid")
		InsertHeaderButton.Finish()
		err = InsertHeaderButton.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert header button: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Get the inserted button ID
		hbid, err := InsertHeaderButton.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Call handle_button_sorting function with INSERT action
		handleSorting := Orm.SelectFunction("get_shift_for_insert", inputs.SortOrder, hbid, parentId)
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

		if len(rows) == 0 {
			log.Printf("Cannot delete header button, issue is no rows returned : %v\n", len(rows) == 0)
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if rows[0]["our_hbid"] == nil {
			FuckingCleanup := Orm.Update()
			FuckingCleanup.Table("header_buttons")
			FuckingCleanup.Set("sort_order", rows[0]["new_sort_order"])
			FuckingCleanup.Where("hbid", "=", hbid)
			FuckingCleanup.Finish()

			err = FuckingCleanup.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute fucking cleanup: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		} else {
			FuckingCleanup := Orm.Update()
			FuckingCleanup.Table("header_buttons")
			FuckingCleanup.SetExpr("sort_order", "sort_order + 1")
			FuckingCleanup.Where("sort_order", ">=", inputs.SortOrder)
			FuckingCleanup.And("hbid", "!=", hbid)
			FuckingCleanup.Finish()

			err = FuckingCleanup.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute fucking cleanup: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		states.HeaderButtons = []models.HeaderButton{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Header button added successfully",
		})
	}
}

func EditHeaderButton(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		inputs := models.HeaderButtonEdit{}
		c.BodyParser(&inputs)

		Orm := utilities.Orm

		if (inputs.ButtonType == "subeler" || inputs.ButtonType == "branslar" || inputs.ButtonType == "doktorlar" || inputs.ButtonType == "tedkikler" || inputs.ButtonType == "tibbi_birimler") && inputs.ParentId != "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Girdiğiniz buton tipi özel bir buton tipidir, herhangi bir alt buton olamaz. Bu tiplerin alt butonları otomatik olarak oluşturulur.",
			})
		}

		// Basic validations
		if inputs.Title == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Title is required",
			})
		}

		if inputs.Target == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Target is required",
			})
		}

		if inputs.Target != "_self" && inputs.Target != "_blank" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Target must be either _self or _blank",
			})
		}

		if inputs.Url != "" && len(inputs.Url) > 255 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "URL is too long",
			})
		}

		if inputs.Icon != "" && len(inputs.Icon) > 50 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Icon name is too long",
			})
		}

		// oldParent not required explicitly; function uses current DB state

		// Begin transaction
		err = Orm.Begin()

		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		UpdateHB := Orm.Update()
		UpdateHB.Table("header_buttons")

		somethingSet := false

		{
			if inputs.Title != inputs.OldTitle {
				UpdateHB.Set("title", inputs.Title)
				somethingSet = true
			}
			if inputs.Url != inputs.OldUrl {
				UpdateHB.Set("url", inputs.Url)
				somethingSet = true
			}
			if inputs.Target != inputs.OldTarget {
				UpdateHB.Set("target", inputs.Target)
				somethingSet = true
			}
			if inputs.Icon != inputs.OldIcon {
				UpdateHB.Set("icon", inputs.Icon)
				somethingSet = true
			}
			if inputs.IsActive != inputs.OldIsActive {
				UpdateHB.Set("is_active", inputs.IsActive)
				somethingSet = true
			}
			if inputs.ButtonType != inputs.OldButtonType {
				UpdateHB.Set("button_type", inputs.ButtonType)
				somethingSet = true
			}
		}

		if somethingSet {
			UpdateHB.Where("hbid", "=", inputs.Hbid)
			UpdateHB.Finish()

			err = UpdateHB.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update header button: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		// Commit
		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		states.HeaderButtons = []models.HeaderButton{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Header button edited successfully",
		})
	}
}

func DeleteHeaderButton(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		HeaderButtonId := c.Params("hbid")

		Orm := utilities.Orm

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can delete header buttons",
			})
		}

		// Get header button info before deletion for sort order handling
		GetHeaderButton := Orm.Select([]string{"hbid", "sort_order", "parent_id"})
		GetHeaderButton.Table("header_buttons")
		GetHeaderButton.Where("hbid", "=", HeaderButtonId)
		GetHeaderButton.Finish()

		err = GetHeaderButton.Execute()
		if err != nil {
			log.Printf("Cannot get header button: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := GetHeaderButton.Rows()
		if err != nil || len(rows) == 0 {
			log.Printf("Header button not found: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Header button not found",
			})
		}

		// Extract values for function call
		sortOrder := lib.Int64(rows[0]["sort_order"])
		parentId := lib.Int64(rows[0]["parent_id"])

		fmt.Printf("sortOrder: %v\n", sortOrder)
		fmt.Printf("parentId: %v\n", parentId)

		// Begin transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Delete header button
		DeleteHeaderButton := Orm.Delete()
		DeleteHeaderButton.Table("header_buttons")
		DeleteHeaderButton.Where("hbid", "=", HeaderButtonId)
		DeleteHeaderButton.Finish()

		err = DeleteHeaderButton.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete header button: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		/*UpdateButtonSortings := Orm.Update()
		UpdateButtonSortings.Table("header_buttons")
		UpdateButtonSortings.SetExpr("sort_order", "sort_order - 1")
		UpdateButtonSortings.Where("sort_order", ">", sortOrder)
		UpdateButtonSortings.And("parent_id", "=", parentIdInt)
		UpdateButtonSortings.Finish()

		fmt.Printf("UpdateButtonSortings: %v\n", UpdateButtonSortings.Query)
		err = UpdateButtonSortings.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update button sortings: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}*/

		// Call handle_button_sorting function with DELETE action
		handleSorting := Orm.SelectFunction("get_shift_for_delete", sortOrder, HeaderButtonId, parentId)
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

		rows, err = handleSorting.Rows()
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
			UpdateRows.Table("header_buttons")
			UpdateRows.SetExpr("sort_order", "sort_order - 1")

			Ins := []any{}
			for _, row := range rows {
				if row["old_parent_id"] != parentId {
					continue
				}

				Ins = append(Ins, lib.String(row["our_hbid"]))
			}
			UpdateRows.In("WHERE", "hbid", Ins)
			UpdateRows.And("parent_id", "=", parentId)
			UpdateRows.Finish()

			err = UpdateRows.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update button sortings: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		states.HeaderButtons = []models.HeaderButton{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Header button deleted successfully",
		})
	}
}

// bunu yapay zeka yazdı, test et.
func GetMainHeaderButtons(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden",
			})
		}

		Orm := utilities.Orm

		// Fetch top-level header buttons (parents)
		SelectParents := Orm.Select([]string{"hbid", "title", "parent_id"})
		SelectParents.Table("header_buttons")
		SelectParents.Where("parent_id", "IS", nil)
		SelectParents.Finish()

		err = SelectParents.Execute()
		if err != nil {
			log.Printf("Cannot fetch header buttons: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := SelectParents.Rows()
		if err != nil {
			log.Printf("Cannot get rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		HeaderButtons := []models.HeaderButton{}
		for _, row := range rows {
			HeaderButtons = append(HeaderButtons, models.HeaderButton{
				Hbid:     lib.String(row["hbid"]),
				Title:    lib.String(row["title"]),
				ParentId: lib.String(row["parent_id"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":         200,
			"message":        "Header buttons fetched successfully",
			"header_buttons": HeaderButtons,
		})
	}
}

// bunu yapay zeka yazdı, test et.
func ChangeHeaderButtonOrder(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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
				"message": "Only admins can change header button order",
			})
		}

		inputs := models.ChangeOrderInputs{}
		err = c.BodyParser(&inputs)
		if err != nil {
			log.Printf("Cannot parse body: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Bad request",
			})
		}

		// Basic validations
		if inputs.NewSortOrder < 1 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Sort order must be at least 1",
			})
		}

		if inputs.NewSortOrder == inputs.OldSortOrder {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Sort order must be different from old sort order",
			})
		}

		// Normalize parent_id
		var newParent interface{}
		if inputs.ParentId == "" || inputs.ParentId == "0" {
			newParent = nil
		} else {
			newParent = inputs.ParentId
		}

		Orm := utilities.Orm
		HeaderButtonId := c.Params("hbid")

		// Begin transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		fmt.Printf("inputs.NewSortOrder: %v\n", inputs.NewSortOrder)
		fmt.Printf("inputs.OldSortOrder: %v\n", inputs.OldSortOrder)
		fmt.Printf("inputs.ParentId: %v\n", inputs.ParentId)
		fmt.Printf("inputs.OldParentId: %v\n", inputs.OldParentId)

		// First adjust other rows according to the new order/parent
		ReorderButtons := Orm.SelectFunction("get_shift_for_update", inputs.NewSortOrder, inputs.OldSortOrder, HeaderButtonId, newParent)
		ReorderButtons.Finish()
		err = ReorderButtons.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot execute get_shift_for_update function: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := ReorderButtons.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get get_shift_for_update function rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if len(rows) == 0 {
			Orm.Rollback()
			log.Printf("get_shift_for_update returned no rows\n")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		HbidsToChange := []any{}
		for _, row := range rows {
			HbidsToChange = append(HbidsToChange, row["our_hbid"])
		}

		Expression := ""
		if rows[0]["direction"] == "down" {
			Expression = "sort_order - 1"
		} else {
			Expression = "sort_order + 1"
		}

		UpdateSortings := Orm.Update()
		UpdateSortings.Table("header_buttons")
		UpdateSortings.SetExpr("sort_order", Expression)
		UpdateSortings.In("WHERE", "hbid", HbidsToChange)
		UpdateSortings.Finish()
		err = UpdateSortings.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update neighbor button sortings: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		UpdateHB := Orm.Update()
		UpdateHB.Table("header_buttons")
		UpdateHB.Set("parent_id", newParent)
		UpdateHB.Set("sort_order", rows[0]["new_sort_order"]) // set to clamped/normalized new order
		UpdateHB.Where("hbid", "=", HeaderButtonId)
		UpdateHB.Finish()
		err = UpdateHB.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update header button new order: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		// Commit
		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		states.HeaderButtons = []models.HeaderButton{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Header button order changed successfully",
		})
	}
}
