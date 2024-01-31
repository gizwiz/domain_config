package database

import (
	"database/sql"
	"fmt"
	"github.com/gizwiz/domain_config/models"
	"log"
)

// Fetch rows from the Property table depending on the specific filter setting arguments
func FetchProperties(dbName string, keyFilter string, modifiedOnly bool, selectedTags []string) ([]models.PropertyValue, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT p.id, p.key, p.description, p.calculated_value value FROM properties p"

	whereClause := []string{}
	if keyFilter != "" {
		whereClause = append(whereClause, "p.key like ?")
	}
	if modifiedOnly {
		whereClause = append(whereClause, "p.modified_value != ''")
	}
	if len(selectedTags) > 0 {
		//// tag or tag or tag...
		//query += " join property_tags pt on p.id = pt.property_id join tags t on pt.tag_id = t.id"
		//selectedTagList := strings.Join(selectedTags, ",")
		//whereClause = append(whereClause, fmt.Sprintf("t.ID in (%s)", selectedTagList))

		// tag and tag and tag
		for idx, selectedTag := range selectedTags {
			query += fmt.Sprintf(" join property_tags pt%[1]d on p.id = pt%[1]d.property_id join tags t%[1]d on pt%[1]d.tag_id = t%[1]d.id", idx)
			whereClause = append(whereClause, fmt.Sprintf("t%[1]d.ID = %s", idx, selectedTag))
		}
	}

	for index, filter := range whereClause {
		if index == 0 {
			query += " WHERE " + filter
		} else {
			query += " AND " + filter
		}
	}

	log.Printf("query: %s", query)
	rows, err := db.Query(query, keyFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var propertyValues []models.PropertyValue
	for rows.Next() {
		var pv models.PropertyValue
		if err := rows.Scan(&pv.ID, &pv.Key, &pv.Description, &pv.Value); err != nil {
			return nil, err
		}
		propertyValues = append(propertyValues, pv)
	}

	return propertyValues, nil
}

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
	query := "SELECT id, key, description, default_value, modified_value, calculated_value FROM properties where id = ?"
	err = db.QueryRow(query, id).Scan(&prop.ID, &prop.Key, &prop.Description, &prop.DefaultValue, &prop.ModifiedValue, &prop.CalculatedValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &prop, nil
}

func GetPropertyByKey(db *sql.DB, key string) (*models.Property, error) {

	var prop models.Property
	query := "SELECT id, key, description, default_value, modified_value, calculated_value FROM properties where key = ?"
	err := db.QueryRow(query, key).Scan(&prop.ID, &prop.Key, &prop.Description, &prop.DefaultValue, &prop.ModifiedValue, &prop.CalculatedValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &prop, nil
}
