package main

import (
	"context"
	"database/sql"
	"embed"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gizwiz/domain_config/database"
	"github.com/gizwiz/domain_config/handlers"
	"github.com/gizwiz/domain_config/views"
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
	err := mainWithErrors()
	if err != nil {
		log.Fatalf("error: %+v", err)
	}
}

//go:embed static/css/tailwind.css
var tailwindCSS embed.FS

func handlePage(tabName string, db *sql.DB, c echo.Context) error {
	keyFilter := c.QueryParam("keyFilter")
	modifiedOnly := c.QueryParam("modifiedOnly")
	modifiedOnlyB := false
	var err error
	if modifiedOnly != "" {
		modifiedOnlyB, err = strconv.ParseBool(modifiedOnly)
		if err != nil {
			return errors.Wrap(err, "can not convert modifiedOnly to bool")
		}
	}
	allTags, err := database.FetchTags(db)
	if err != nil {
		return errors.Wrapf(err, "can not fetch properties")
	}
	selectedTags := c.QueryParams()["selectedTags"]
	props, err := database.FetchProperties(db, keyFilter, modifiedOnlyB, selectedTags)
	if err != nil {
		return errors.Wrapf(err, "can not fetch properties")
	}

	// Use the properties templ to render the HTML table
	//return c.Render(http.StatusOK, "", views.PropertiesPage(props, keyFilter, modifiedOnlyB, allTags, selectedTags))
	return c.Render(http.StatusOK, "", views.PropertiesPage(tabName, props, keyFilter, modifiedOnlyB, allTags, func(tagID string) bool {
		for _, selectedTag := range selectedTags {
			if selectedTag == tagID {
				return true
			}
		}
		return false
	}))
}

func mainWithErrors() error {

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return errors.Wrapf(err, "Error opening database: %s", dbName)
	}
	defer db.Close()

	err = database.ApplyLatestDBMigrations(db)
	if err != nil {
		return errors.Wrapf(err, "can not apply latest DB migrations")
	}

	// Create a new instance of Echo
	e := echo.New()
	e.HideBanner = true
	e.Debug = true

	e.Renderer = &TemplRender{}

	e.GET("/static/css/tailwind.css", func(c echo.Context) error {
		file, err := tailwindCSS.Open("static/css/tailwind.css")
		if err != nil {
			return err // Properly handle the error
		}
		defer file.Close()
		return c.Stream(http.StatusOK, "text/css", file)
	})

	// display the key-value table
	e.GET("/properties", func(c echo.Context) error {
		return handlePage("properties", db, c)
	})

	e.GET("/tables", func(c echo.Context) error {
		return handlePage("tables", db, c)
	})

	e.GET("/calculate", func(c echo.Context) error {
		return handlers.CalculateProperties(db, c)
	})

	e.GET("/property/:id", func(c echo.Context) error {
		return handlers.GetPropertyByID(db, c)
	})

	e.POST("/insert", func(c echo.Context) error {
		return handlers.InsertProperty(db, c)
	})

	e.POST("/update", func(c echo.Context) error {
		return handlers.UpdateProperty(db, c)
	})

	e.GET("/export", func(c echo.Context) error {
		return handlers.ExportTablesToJson(db, c)
	})
	// Start the server
	err = e.Start(":8080")
	if err != nil {
		return errors.Wrapf(err, "can not start echo server")
		//e.Logger.Fatal(e.Start(":8080"))
	}

	return nil
}
