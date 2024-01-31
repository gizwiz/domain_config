package models

import (
	"database/sql"
	"encoding/json"
)

// PropertyValue represents a row in the property_values view (a view calculating the value on top of the properties table)
type PropertyValue struct {
	ID          int
	Key         string
	Description sql.NullString
	Value       sql.NullString
}

// Property represents a row in the properties table
type Property struct {
	ID              int            `json:"id"`
	Key             string         `json:"key"`
	Description     sql.NullString `json:"-"`
	DefaultValue    sql.NullString `json:"-"`
	ModifiedValue   sql.NullString `json:"-"`
	CalculatedValue sql.NullString `json:"-"`
}

// PropertyJSON is a struct used for custom JSON marshaling.
type PropertyJSON struct {
	ID              int    `json:"id"`
	Key             string `json:"key"`
	Description     string `json:"description,omitempty"`
	DefaultValue    string `json:"default_value,omitempty"`
	ModifiedValue   string `json:"modified_value,omitempty"`
	CalculatedValue string `json:"calculated_value,omitempty"`
}

// MarshalJSON customizes the JSON output.
func (p Property) MarshalJSON() ([]byte, error) {
	j := PropertyJSON{
		ID:  p.ID,
		Key: p.Key,
	}

	if p.Description.Valid {
		j.Description = p.Description.String
	}
	if p.DefaultValue.Valid {
		j.DefaultValue = p.DefaultValue.String
	}
	if p.ModifiedValue.Valid {
		j.ModifiedValue = p.ModifiedValue.String
	}
	if p.CalculatedValue.Valid {
		j.CalculatedValue = p.CalculatedValue.String
	}

	return json.Marshal(j)
}

type Tag struct {
	ID  int    `json:"id"`
	Tag string `json:"tag"`
}

// function definition for isSelectedTag
type StringPredicate func(string) bool
