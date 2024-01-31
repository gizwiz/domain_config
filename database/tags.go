package database

import (
	"database/sql"
	"github.com/gizwiz/domain_config/models"
)

// Fetch all rows from the tags table
func FetchTags(db *sql.DB) ([]models.Tag, error) {
	query := "SELECT id, tag FROM tags"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// Fetch all tags for a property
func FetchPropertyTagIDs(db *sql.DB, propertyID int) ([]int, error) {
	query := "select t.tag_id from property_tags t where t.property_id = :1"
	rows, err := db.Query(query, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tagIDs []int
	for rows.Next() {
		var tagID int
		if err := rows.Scan(&tagID); err != nil {
			return nil, err
		}
		tagIDs = append(tagIDs, tagID)
	}

	return tagIDs, nil
}
