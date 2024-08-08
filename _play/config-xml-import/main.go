package main

import (
	"database/sql"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

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

func transformFormula(formula string, cellToKey map[string]string, sheetName string, compositeKey string) string {
	if strings.HasPrefix(formula, "=CONCATENATE(") {
		// Remove the "=CONCATENATE(" prefix and the closing ")"
		args := formula[len("=CONCATENATE(") : len(formula)-1]
		// Split the arguments by comma
		argList := strings.Split(args, ",")
		// Replace cell references with keys
		for i, arg := range argList {
			argList[i] = replaceCellReferences(arg, cellToKey, sheetName, compositeKey)
		}
		// Join the arguments with " + "
		return "=" + strings.Join(argList, " + ")
	}
	return replaceCellReferences(formula, cellToKey, sheetName, compositeKey)
}

func replaceCellReferences(formula string, cellToKey map[string]string, sheetName string, compositeKey string) string {
	re := regexp.MustCompile(`\$?[A-Z]+\$?\d+|\w+!\$?[A-Z]+\$?\d+`)
	return re.ReplaceAllStringFunc(formula, func(cellRef string) string {
		// Check if the cell reference includes a sheet name
		var normalizedCellRef, fullCellRef string
		if strings.Contains(cellRef, "!") {
			parts := strings.Split(cellRef, "!")
			normalizedCellRef = strings.ReplaceAll(parts[1], "$", "")
			fullCellRef = fmt.Sprintf("%s!%s", parts[0], normalizedCellRef)
		} else {
			normalizedCellRef = strings.ReplaceAll(cellRef, "$", "")
			fullCellRef = fmt.Sprintf("%s!%s", sheetName, normalizedCellRef)
		}

		if key, exists := cellToKey[fullCellRef]; exists {
			log.Printf("Cell reference FOUND: %s in formula: %s, sheet: %s, compositeKey: %s, key found: %s", cellRef, formula, sheetName, compositeKey, key)
			return fmt.Sprintf(`Get("%s")`, key)
		}

		log.Printf("Cell reference not found: %s in formula: %s, sheet: %s, compositeKey: %s", cellRef, formula, sheetName, compositeKey)
		return cellRef
	})
}

func main() {
	// Configure logging to output to standard error
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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

	byteValue, _ := io.ReadAll(xmlFile)

	var excel Excel
	xml.Unmarshal(byteValue, &excel)

	db, err := sql.Open("sqlite3", "./domain_config.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS properties (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT UNIQUE,
		description TEXT,
		default_value TEXT,
		calculated_value TEXT,
		modified_value TEXT
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	insertSQL := `INSERT INTO properties (key, description, default_value, calculated_value, modified_value) VALUES (?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	cellToKey := make(map[string]string)

	// First pass: Collect cell references and their corresponding keys
	for _, sheet := range excel.Sheets {
		var currentSeparator string
		for _, row := range sheet.Rows {
			if row.Separator != "" {
				currentSeparator = strings.ReplaceAll(row.Separator, " ", "_")
				cellToKey[fmt.Sprintf("%s!%s", sheet.Name, row.SeparatorCell)] = fmt.Sprintf("%s.%s.key", sheet.Name, currentSeparator)
			}
			if row.Key.Text != "" {
				var compositeKey string
				if currentSeparator != "" {
					compositeKey = fmt.Sprintf("%s.%s.%s", sheet.Name, currentSeparator, row.Key.Text)
				} else {
					compositeKey = fmt.Sprintf("%s.%s", sheet.Name, row.Key.Text)
				}
				cellToKey[fmt.Sprintf("%s!%s", sheet.Name, row.Key.Cell)] = compositeKey
				cellToKey[fmt.Sprintf("%s!%s", sheet.Name, row.Value.Cell)] = compositeKey
			}
		}
	}
	// Adding the Deployments sheet cells which are not in config.xml file
	cellToKey["Deployments!B2"] = "Domain.Domain.ilias.domain.name"
	cellToKey["Deployments!C2"] = "DeploymentGroups.key"
	cellToKey["SetupserverApps!C2"] = "DeploymentGroups.key"

	// Log the full cellToKey content to a file, sorted by the hashmap index
	logFile, err := os.Create("cellToKey.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	keys := make([]string, 0, len(cellToKey))
	for k := range cellToKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(logFile, "Cell: %s, Key: %s\n", k, cellToKey[k])
	}

	// Second pass: Insert data into the database with transformed formulas
	_, err = stmt.Exec("DeploymentGroups.key", "DeploymentGroups key description", "=Config.ISC_only.domain.contextRootOverwrite", "", "")
	if err != nil {
		log.Printf("UNIQUE constraint failed for key: %s, value: %s, sheet: %s\n", "DeploymentGroups.key", "=Config.ISC_only.domain.contextRootOverwrite", "DeploymentGroups")
	}
	for _, sheet := range excel.Sheets {
		var currentSeparator string
		for _, row := range sheet.Rows {
			if row.Separator != "" {
				currentSeparator = strings.ReplaceAll(row.Separator, " ", "_")

				if sheet.Name == "Deployments" {
					compositeKey := fmt.Sprintf("%s.%s.key", sheet.Name, currentSeparator)
					value := currentSeparator
					_, err = stmt.Exec(compositeKey, "Deployments key description", value, "", "")
					if err != nil {
						log.Printf("UNIQUE constraint failed for key: %s, value: %s, sheet: %s\n", compositeKey, value, sheet.Name)
					}
				}
			}
			if row.Key.Text != "" && row.Value.DefaultValue != "" {
				var compositeKey string
				if currentSeparator != "" {
					compositeKey = fmt.Sprintf("%s.%s.%s", sheet.Name, currentSeparator, row.Key.Text)
				} else {
					compositeKey = fmt.Sprintf("%s.%s", sheet.Name, row.Key.Text)
				}

				value := row.Value.DefaultValue
				if strings.HasPrefix(value, "=") {
					value = transformFormula(value, cellToKey, sheet.Name, compositeKey)
				}

				_, err = stmt.Exec(compositeKey, row.Key.Text, value, "", "")
				if err != nil {
					log.Printf("UNIQUE constraint failed for key: %s, value: %s, sheet: %s\n", compositeKey, value, sheet.Name)
				}
			}
		}
	}

	fmt.Println("Database created successfully.")
}
