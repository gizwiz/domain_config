package main

import (
	"database/sql"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Excel struct {
	XMLName xml.Name `xml:"Excel"`
	Sheets  []Sheet  `xml:"Sheet"`
}

type Sheet struct {
	Name string `xml:"name,attr"`
	Rows []Row  `xml:"Row"`
}

type Row struct {
	Key struct {
		Cell string `xml:"cell,attr"`
		Text string `xml:",chardata"`
	} `xml:"Key"`
	Value struct {
		Cell         string `xml:"cell,attr"`
		DefinedName  string `xml:"DefinedName"`
		DefaultValue string `xml:"DefaultValue"`
	} `xml:"V"`
	SeparatorCell string `xml:"separatorCell,attr"`
	Separator     string `xml:"separator,attr"`
}

func main() {
	xmlPath := flag.String("xml", "", "Path to the config.xml file")
	newDB := flag.Bool("newdb", false, "Remove the previous database if true")
	flag.Parse()

	if *xmlPath == "" {
		log.Fatal("Usage: main --xml <config.xml> [--newdb=true]")
	}

	if *newDB {
		if err := os.Remove("./domain_config.db"); err != nil && !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	xmlFile, err := os.Open(*xmlPath)
	if err != nil {
		log.Fatal(err)
	}
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var excel Excel
	xml.Unmarshal(byteValue, &excel)

	db, err := sql.Open("sqlite3", "./domain_config.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS properties (
		"key" TEXT NOT NULL PRIMARY KEY,
		"value" TEXT
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	insertSQL := `INSERT INTO properties (key, value) VALUES (?, ?)`
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, sheet := range excel.Sheets {
		var currentSeparator string
		for _, row := range sheet.Rows {
			if row.Separator != "" {
				currentSeparator = row.Separator
			}
			if row.Key.Text != "" && row.Value.DefaultValue != "" {
				var compositeKey string
				if currentSeparator != "" {
					compositeKey = fmt.Sprintf("%s.%s.%s", sheet.Name, currentSeparator, row.Key.Text)
				} else {
					compositeKey = fmt.Sprintf("%s.%s", sheet.Name, row.Key.Text)
				}
				_, err = stmt.Exec(compositeKey, row.Value.DefaultValue)
				if err != nil {
					log.Printf("UNIQUE constraint failed for key: %s, value: %s, sheet: %s\n", compositeKey, row.Value.DefaultValue, sheet.Name)
				}
			}
		}
	}

	fmt.Println("Database created successfully.")
}
