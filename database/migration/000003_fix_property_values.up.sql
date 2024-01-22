drop VIEW property_values; 
CREATE VIEW property_values AS
SELECT 
    id,
    key,
    description,
    calculated_value AS value
FROM properties;
