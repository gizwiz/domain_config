CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag TEXT UNIQUE,
    description TEXT
);


CREATE TABLE property_tags (
    property_id INTEGER,
    tag_id INTEGER,
    PRIMARY KEY (property_id, tag_id),
    FOREIGN KEY (property_id) REFERENCES properties(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);


