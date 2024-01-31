package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gizwiz/domain_config/database"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func InsertProperty(dbName string, c echo.Context) error {
	key := c.FormValue("key")
	description := c.FormValue("description")
	defaultValue := c.FormValue("defaultValue")
	modifiedValue := c.FormValue("modifiedValue")

	// Insert logic here
	err := database.InsertProperty(dbName, key, description, defaultValue, modifiedValue)
	if err != nil {
		return errors.Wrapf(err, "can not inset property %s", key)
	}

	err = CalculateProperties(dbName, c)
	if err != nil {
		return errors.Wrapf(err, "can not calculate properties after inset property %s", key)
	}

	return c.Redirect(http.StatusFound, "/properties")
}

func UpdateProperty(dbName string, c echo.Context) error {
	id, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return errors.Wrapf(err, "can not convert %s into int", c.FormValue("id"))
	}
	key := c.FormValue("key")
	description := c.FormValue("description")
	defaultValue := c.FormValue("defaultValue")
	modifiedValue := c.FormValue("modifiedValue")

	// Update logic here
	err = database.UpdateProperty(dbName, id, key, description, defaultValue, modifiedValue)
	if err != nil {
		return errors.Wrapf(err, "can not updat property %s", key)
	}

	err = CalculateProperties(dbName, c)
	if err != nil {
		return errors.Wrapf(err, "can not calculate properties after inset property %s", key)
	}
	return c.Redirect(http.StatusFound, "/properties")
}

func GetPropertyByID(dbName string, c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.Wrapf(err, "cannot convert %s into int", c.Param("id"))
	}

	property, err := database.GetPropertyByID(dbName, id)
	if err != nil {
		log.Println("Error fetching property by ID:", err)
		return err
	}

	return c.JSON(http.StatusOK, property)
}
