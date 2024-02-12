This is just a play project to keep track of some "properties" in a sqllite db.
The idea is to work towards a more generic approach to keep track of configuration properties where properties can store values, but can also depend on other properties to calculate default values.

## Prepare the database using migrations:

Assuming here that sqlite3 is already installed. On mac this can be done using `brew install sqlite3`

    go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

    cd domain_config
    migrate -path database/migration/ -database "sqlite3://main.db" -verbose up

## Optionally example data can be imported:

    sqlite3 main.db < example_dml.sql

## Running:

    templ generate; go run .

## Browser:

    open http://localhost:8080/properties

## continuous dev

    tmux pane1: npm run watch
    tmux pane2: air
    tmux pane3: nvim .
