package database

import (
	"database/sql"

	"github.com/gizwiz/domain_config/models"
)

func InsertProperty(dbName string, key, description, defaultValue, modifiedValue string) error {
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

func UpdateProperty(dbName string, id int, key, description, defaultValue, modifiedValue string) error {
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

func UpdatePropertyCalculatedValue(db *sql.DB, id int, calculatedValue string) error {
	// Prepare update statement
	stmt, err := db.Prepare("UPDATE properties SET calculated_value = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(calculatedValue, id)
	return err
}

func GetPropertyByID(dbName string, id int) (*models.Property, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var prop models.Property
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
