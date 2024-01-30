package handlers

import (
	"database/sql"
	"github.com/gizwiz/domain_config/database"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"os"
)

func ExportTablesToJson(dbName string, c echo.Context) error {

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	const exportDir = "./_export"
	err = os.MkdirAll(exportDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "can not mkdir %s", exportDir)
	}
	for _, tableName := range []string{"properties", "tags", "property_tags"} {
		err = database.ExportTableToJson(db, exportDir, tableName)
		if err != nil {
			return errors.Wrapf(err, "can not export %s", tableName)
		}
	}
	return nil
}
