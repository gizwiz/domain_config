This is just a play project to keep track of some "properties" in a sqllite db.
The idea is to work towards a more generic appraoch to keep track of configuration properties where properties can store values, but can also depend on other properties to calculate default values.

Preparation:

    sqlite3 main.db < ddl.sql
    sqlite3 main.db < example_dml.sql

Running:

    templ generate; go run .

Browser:

    open http://localhost:8080/properties
