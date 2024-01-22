package main

import (
	"context"
	"io"
	"log"
	"net/http"

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

func mainWithErrors() error {
	{
		err := database.ApplyLatestDBMigrations(dbName)
		if err != nil {
			return errors.Wrapf(err, "can not apply latest DB migrations")
		}
	}

	// Create a new instance of Echo
	e := echo.New()

	e.Renderer = &TemplRender{}

	// display the key-value table
	e.GET("/properties", func(c echo.Context) error {
		keyFilter := c.QueryParam("keyFilter")
		props, err := handlers.FetchProperties(dbName, keyFilter)
		if err != nil {
			return errors.Wrapf(err, "can not fetch properties")
		}

		// Use the properties templ to render the HTML table
		return c.Render(http.StatusOK, "", views.PropertiesPage(props, keyFilter))
	})

	e.GET("/calculate", func(c echo.Context) error {
		return handlers.CalculateProperties(dbName, c)
	})

	e.GET("/property/:id", func(c echo.Context) error {
		return handlers.GetPropertyByID(dbName, c)
	})

	e.POST("/insert", func(c echo.Context) error {
		return handlers.InsertProperty(dbName, c)
	})

	e.POST("/update", func(c echo.Context) error {
		return handlers.UpdateProperty(dbName, c)
	})

	// Start the server
	err := e.Start(":8080")
	if err != nil {
		//return errors.Wrapf(err, "can not start echo server")
		e.Logger.Fatal(e.Start(":8080"))
	}
	return nil
}
