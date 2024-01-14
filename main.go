package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const dbName = "main.db"

// TemplRender is a custom renderer for Templ components in Echo
type TemplRender struct{}

// Render implements the Renderer interface for TemplRender
func (t *TemplRender) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if templData, ok := data.(templ.Component); ok {
		// Set the content type to HTML
		c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Response().WriteHeader(http.StatusOK)

		// Render the templ component
		return templData.Render(context.Background(), w)
	}
	return nil
}

func main() {
	// Create a new instance of Echo
	e := echo.New()

	e.Renderer = &TemplRender{}

	// display the key-value table
	e.GET("/properties", func(c echo.Context) error {
		keyFilter := c.QueryParam("keyFilter")
		props, err := fetchProperties(keyFilter)
		if err != nil {
			log.Println("Error fetching Properties:", err)
			return err
		}

		// Use the properties templ to render the HTML table
		return c.Render(http.StatusOK, "", properties_page(props, keyFilter))
	})

	e.GET("/property/:id", getPropertyByID)

	e.POST("/insert", insertProperty)
	e.POST("/update", updateProperty)

	// Start the server
	e.Logger.Fatal(e.Start(":8080"))
}

// PropertyValue represents a row in the property_values view (a view calculating the value on top of the properties table)
type PropertyValue struct {
	id          int
	key         string
	description sql.NullString
	value       sql.NullString
}

// Property represents a row in the properties table
type Property struct {
	ID            int            `json:"id"`
	Key           string         `json:"key"`
	Description   sql.NullString `json:"-"`
	DefaultValue  sql.NullString `json:"-"`
	ModifiedValue sql.NullString `json:"-"`
}

// PropertyJSON is a struct used for custom JSON marshaling.
type PropertyJSON struct {
	ID            int    `json:"id"`
	Key           string `json:"key"`
	Description   string `json:"description,omitempty"`
	DefaultValue  string `json:"default_value,omitempty"`
	ModifiedValue string `json:"modified_value,omitempty"`
}

// MarshalJSON customizes the JSON output.
func (p Property) MarshalJSON() ([]byte, error) {
	j := PropertyJSON{
		ID:  p.ID,
		Key: p.Key,
	}

	if p.Description.Valid {
		j.Description = p.Description.String
	}
	if p.DefaultValue.Valid {
		j.DefaultValue = p.DefaultValue.String
	}
	if p.ModifiedValue.Valid {
		j.ModifiedValue = p.ModifiedValue.String
	}

	return json.Marshal(j)
}

// Fetch all rows from the Property table
func fetchProperties(keyFilter string) ([]PropertyValue, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT id, key, description, value FROM property_values"
	if keyFilter != "" {
		query += " WHERE Key like ?"
	}
	rows, err := db.Query(query, keyFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var propertyValues []PropertyValue
	for rows.Next() {
		var pv PropertyValue
		if err := rows.Scan(&pv.id, &pv.key, &pv.description, &pv.value); err != nil {
			return nil, err
		}
		propertyValues = append(propertyValues, pv)
	}

	return propertyValues, nil
}

func insertProperty(c echo.Context) error {
	key := c.FormValue("key")
	description := c.FormValue("description")
	defaultValue := c.FormValue("defaultValue")
	modifiedValue := c.FormValue("modifiedValue")

	// Insert logic here
	err := insertIntoDB(key, description, defaultValue, modifiedValue)
	if err != nil {
		log.Println("Error inserting Properties:", err)
		return err
	}

	return c.Redirect(http.StatusFound, "/properties")
}

func insertIntoDB(key, description, defaultValue, modifiedValue string) error {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	// Prepare insert statement
	stmt, err := db.Prepare("INSERT INTO properties (key, description, default_value, modified_value) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(key, description, defaultValue, modifiedValue)
	return err
}

func updateProperty(c echo.Context) error {
	id, err := strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return errors.Wrapf(err, "can not convert %s into int", c.FormValue("id"))
	}
	key := c.FormValue("key")
	description := c.FormValue("description")
	defaultValue := c.FormValue("defaultValue")
	modifiedValue := c.FormValue("modifiedValue")

	// Update logic here
	err = updateDB(id, key, description, defaultValue, modifiedValue)
	if err != nil {
		log.Println("Error updating Properties:", err)
		return err
	}

	return c.Redirect(http.StatusFound, "/properties")
}

func updateDB(id int, key, description, defaultValue, modifiedValue string) error {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	// Prepare update statement
	stmt, err := db.Prepare("UPDATE properties SET key = ?, description = ?, default_value = ?, modified_value = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(key, description, defaultValue, modifiedValue, id)
	return err
}

func getPropertyByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.Wrapf(err, "cannot convert %s into int", c.Param("id"))
	}

	property, err := getRecordByID(id)
	if err != nil {
		log.Println("Error fetching property by ID:", err)
		return err
	}

	return c.JSON(http.StatusOK, property)
}

func getRecordByID(id int) (*Property, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var prop Property
	query := "SELECT id, key, description, default_value, modified_value FROM properties where id = ?"
	err = db.QueryRow(query, id).Scan(&prop.ID, &prop.Key, &prop.Description, &prop.DefaultValue, &prop.ModifiedValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &prop, nil

}
