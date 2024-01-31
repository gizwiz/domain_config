package handlers

import (
	"database/sql"
	"github.com/gizwiz/domain_config/models"
	"net/http"
	"strconv"

	"github.com/gizwiz/domain_config/database"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func InsertProperty(db *sql.DB, c echo.Context) error {
	key := c.FormValue("key")
	description := c.FormValue("description")
	defaultValue := c.FormValue("defaultValue")
	modifiedValue := c.FormValue("modifiedValue")

	form, err := c.FormParams()
	if err != nil {
		return errors.Wrapf(err, "can not get the map of all form parameters")
	}
	selectedTags := form["propertyTags"]

	// Insert logic here
	err = database.InsertProperty(db, key, description, defaultValue, modifiedValue, selectedTags)
	if err != nil {
		return errors.Wrapf(err, "can not inset property %s", key)
	}

	err = CalculateProperties(db, c)
	if err != nil {
		return errors.Wrapf(err, "can not calculate properties after inset property %s", key)
	}

	return c.Redirect(http.StatusFound, "/properties")
}

func UpdateProperty(db *sql.DB, c echo.Context) error {
	id, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return errors.Wrapf(err, "can not convert %s into int", c.FormValue("id"))
	}
	key := c.FormValue("key")
	description := c.FormValue("description")
	defaultValue := c.FormValue("defaultValue")
	modifiedValue := c.FormValue("modifiedValue")

	form, err := c.FormParams()
	if err != nil {
		return errors.Wrapf(err, "can not get the map of all form parameters")
	}
	propertyTags := form["propertyTags"]

	// Update logic here
	err = database.UpdateProperty(db, id, key, description, defaultValue, modifiedValue, propertyTags)
	if err != nil {
		return errors.Wrapf(err, "can not update property %s and tags", key)
	}

	err = CalculateProperties(db, c)
	if err != nil {
		return errors.Wrapf(err, "can not calculate properties after inset property %s", key)
	}
	return c.Redirect(http.StatusFound, "/properties")
}

func GetPropertyByID(db *sql.DB, c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.Wrapf(err, "cannot convert %s into int", c.Param("id"))
	}

	var propertyWithTagIDs models.PropertyWithTagIDs
	propertyWithTagIDs.Property, err = database.GetPropertyByID(db, id)
	if err != nil {
		return errors.Wrapf(err, "can not fetch property by id %d", id)
	}

	propertyWithTagIDs.TagIDs, err = database.FetchPropertyTagIDs(db, id)
	if err != nil {
		return errors.Wrapf(err, "can not fetch tags for propertyID %d", id)
	}

	return c.JSON(http.StatusOK, propertyWithTagIDs)
}
