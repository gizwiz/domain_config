
CREATE TABLE properties (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE,
    description TEXT,
    default_value TEXT,   -- Stores a formula/function (if starting with '=') or a text value otherwise
    calculated_value TEXT, -- Stores the result of the calculated formula/function, or the same as default_value if it's not a formula
    modified_value TEXT   -- Stores the modified value if available; otherwise, it should be considered as NULL
);

CREATE VIEW property_values AS
SELECT 
    id,
    key,
    description,
    CASE 
        WHEN modified_value IS NOT NULL THEN modified_value 
        ELSE calculated_value 
    END AS value
FROM properties;

