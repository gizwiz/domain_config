package database

import (
	"database/sql"
	"fmt"
	"github.com/gizwiz/domain_config/models"
	"github.com/pkg/errors"
	"strconv"
)

// Fetch rows from the Property table depending on the specific filter setting arguments
func FetchProperties(db *sql.DB, keyFilter string, modifiedOnly bool, selectedTags []string) ([]models.PropertyValue, error) {
	query := "SELECT p.id, p.key, p.description, p.calculated_value value FROM properties p"

	whereClause := []string{}
	if keyFilter != "" {
		whereClause = append(whereClause, "p.key like ?")
	}
	if modifiedOnly {
		whereClause = append(whereClause, "p.modified_value != ''")
	}
	if len(selectedTags) > 0 {
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

	query += " order by p.key"

	//log.Printf("query: %s", query)
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

func InsertProperty(db *sql.DB, key, description, defaultValue, modifiedValue string, tagIDs []string) error {
	stmt, err := db.Prepare("INSERT INTO properties (key, description, default_value, modified_value) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(key, description, defaultValue, modifiedValue)
	if err != nil {
		return errors.Wrapf(err, "can not insert property %s", key)
	}

	p, err := GetPropertyByKey(db, key)
	if err != nil {
		return errors.Wrapf(err, "can not get property by key %s", key)
	}

	err = UpdatePropertyTags(db, p.ID, tagIDs)
	if err != nil {
		return errors.Wrapf(err, "can not update property_tags for property %s", key)
	}

	return nil
}

func UpdateProperty(db *sql.DB, id int, key, description, defaultValue, modifiedValue string, tagIDs []string) error {
	stmt, err := db.Prepare("UPDATE properties SET key = ?, description = ?, default_value = ?, modified_value = ? WHERE id = ?")
	if err != nil {
		return errors.Wrapf(err, "can not prepare update property %d", id)
	}
	defer stmt.Close()

	_, err = stmt.Exec(key, description, defaultValue, modifiedValue, id)
	if err != nil {
		return errors.Wrapf(err, "can not update property %d", id)
	}

	err = UpdatePropertyTags(db, id, tagIDs)
	if err != nil {
		return errors.Wrapf(err, "can not update property_tags", id)
	}

	return nil
}

func UpdatePropertyTags(db *sql.DB, propertyID int, tagIDs []string) error {
	stmt2, err := db.Prepare("DELETE FROM property_tags WHERE property_id = ?")
	if err != nil {
		return err
	}
	defer stmt2.Close()
	_, err = stmt2.Exec(propertyID)
	if err != nil {
		return errors.Wrapf(err, "can not delete from property_tags for property: %d", propertyID)
	}

	for _, strTagID := range tagIDs {
		stmt, err := db.Prepare("INSERT INTO property_tags (property_id, tag_id) VALUES (?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()

		tagID, err := strconv.Atoi(strTagID)
		if err != nil {
			return errors.Wrapf(err, "can not convert %s to int", strTagID)
		}
		_, err = stmt.Exec(propertyID, tagID)
		if err != nil {
			return errors.Wrapf(err, "can not insert into property_tags, property: %d, tag: %d", propertyID, tagID)
		}
	}
	return nil
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

func GetPropertyByID(db *sql.DB, id int) (models.Property, error) {
	var prop models.Property
	query := "SELECT id, key, description, default_value, modified_value, calculated_value FROM properties where id = ?"
	err := db.QueryRow(query, id).Scan(&prop.ID, &prop.Key, &prop.Description, &prop.DefaultValue, &prop.ModifiedValue, &prop.CalculatedValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Property{}, nil
		}
		return models.Property{}, err
	}

	return prop, nil
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
