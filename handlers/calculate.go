package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gizwiz/domain_config/database"
	"github.com/gizwiz/domain_config/models"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func CalculateProperties(dbName string, c echo.Context) error {

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	// all none formulas the calculated field is the same as the default_value
	err = calculateNoneFunctionPropertiesDB(db)
	if err != nil {
		return errors.Wrapf(err, "can not calculate none function properties")
	}

	// for all formula properties, calculate one by one
	err = calculateFunctionPropertiesDB(db)
	if err != nil {
		return errors.Wrapf(err, "can not calculate function properties")
	}

	// Then redirect to the same URL, effectively reloading the page
	return c.Redirect(http.StatusFound, c.Request().RequestURI)
}

func calculateNoneFunctionPropertiesDB(db *sql.DB) error {

	//stmt, err := db.Prepare("update properties set calculated_value = default_value where default_value not like '=%'")
	stmt, err := db.Prepare(`UPDATE properties
		SET calculated_value = CASE
				WHEN modified_value != '' THEN modified_value
				WHEN default_value NOT LIKE '=%' THEN default_value
				ELSE NULL
		END`)
	if err != nil {
		return errors.Wrapf(err, "can not prepare update calculated_value")
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return errors.Wrapf(err, "can not exec update calculated_value")
	}
	return nil
}

func calculateFunctionPropertiesDB(db *sql.DB) error {

	// todo loop through all formula properties and calculate them compiling the expressions
	rows, err := db.Query("select id, key, description, default_value, modified_value from properties where calculated_value is NULL")
	if err != nil {
		return errors.Wrap(err, "can not select none-calculated properties")
	}
	//defer rows.Close()

	var properties []models.Property
	for rows.Next() {
		var p models.Property
		err := rows.Scan(&p.ID, &p.Key, &p.Description, &p.DefaultValue, &p.ModifiedValue)
		if err != nil {
			return errors.Wrap(err, "can not scan none-calculated properties")
		}
		properties = append(properties, p)

	}
	rows.Close()

	// Now iterate over the slice of properties to update them
	for _, p := range properties {
		// todo implement using expr-engine
		if err := database.UpdatePropertyCalculatedValue(db, p.ID, "CALCULATED"); err != nil {
			return errors.Wrap(err, "can not update calculate_value in none-calculated properties")
		}
	}

	if err = rows.Err(); err != nil {
		return errors.Wrapf(err, "can not iterate over none-calculated properties")
	}

	return nil
}
