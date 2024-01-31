package database

import (
	"database/sql"
	"github.com/gizwiz/domain_config/models"
)

// Fetch all rows from the tags table
func FetchTags(dbName string) ([]models.Tag, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

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