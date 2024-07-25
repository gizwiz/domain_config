package handlers

import (
	"database/sql"
	"fmt"
	"github.com/expr-lang/expr"
	"github.com/gizwiz/domain_config/database"
	"github.com/gizwiz/domain_config/models"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
)

func CalculateProperties(db *sql.DB, c echo.Context) error {
	// all none formulas the calculated field is the same as the default_value
	err := calculateNoneFunctionPropertiesDB(db)
	if err != nil {
		return errors.Wrapf(err, "can not calculate none function properties")
	}

	// for all formula properties, calculate one by one
	err = calculateFunctionPropertiesDB(db)
	if err != nil {
		return errors.Wrapf(err, "can not calculate function properties")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
	})

	return nil
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

type Env struct {
	Calculations map[string]string
	db           *sql.DB
}

func (Env) Sprintf(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}

func (env *Env) Get(key string) (string, error) {
	if calculated, ok := env.Calculations[key]; ok {
		return calculated, nil
	}

	// not in env yet, so get it from the db
	property, err := database.GetPropertyByKey(env.db, key)
	if err != nil {
		return "", errors.Wrapf(err, "can not get property for key %s", key) //todo can we return errors?
	}

	var calculatedValue string
	if property.CalculatedValue.Valid {
		calculatedValue = property.CalculatedValue.String
	} else {
		calculatedValue, err = calculateProperty(property, env)
		if err != nil {
			return "", errors.Wrapf(err, "can not calculateProperty %v", property)
		}
	}
	env.Calculations[key] = calculatedValue

	return calculatedValue, nil
}

func calculateFunctionPropertiesDB(db *sql.DB) error {

	// loop through all formula properties and calculate them compiling the expressions
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

	env := Env{
		Calculations: map[string]string{
			// property.Key: property,
		},
		db: db,
	}

	// Now iterate over the slice of properties to update them
	for _, p := range properties {
		calculatedValue, err := env.Get(p.Key)
		if err != nil {
			return errors.Wrapf(err, "can not calculate property value %+v", p)
		}
		if err := database.UpdatePropertyCalculatedValue(db, p.ID, calculatedValue); err != nil {
			return errors.Wrap(err, "can not update calculate_value in none-calculated properties")
		}
	}

	if err = rows.Err(); err != nil {
		return errors.Wrapf(err, "can not iterate over none-calculated properties")
	}

	return nil
}

func calculateProperty(property *models.Property, env *Env) (string, error) {
	//log.Printf("calculate: %s started", property.Key)
	code := property.DefaultValue.String[1:]
	program, err := expr.Compile(code, expr.Env(env))
	if err != nil {
		return "", errors.Wrapf(err, "can not expr.compile %s", property.DefaultValue.String[1:])
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return "", errors.Wrapf(err, "can not expr.compile %s", property.DefaultValue.String[1:])
	}

	//log.Printf("calculate: %s -> %s done", property.Key, output)
	return fmt.Sprintf("%s", output), nil
}
