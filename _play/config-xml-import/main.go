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
	Value []struct {
		Cell         string `xml:"cell,attr"`
		DefinedName  string `xml:"DefinedName"`
		Comments     string `xml:"Comments"`
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

func processDatasourcesSheet(sheet Sheet, stmt *sql.Stmt, cellToKey map[string]string, definedNames []string, definedNameToKey map[string]string) {
	columnToKey := map[int]string{
		0:  "Name",
		1:  "JNDI",
		2:  "DriverClassName",
		3:  "Database",
		4:  "User",
		5:  "Password",
		6:  "InitialCapacity",
		7:  "MinimumCapacity",
		8:  "MaximumCapacity",
		9:  "Targets",
		10: "_",
		11: "_",
		12: "InitSQL",
	}

	for _, row := range sheet.Rows {
		if row.Key.Text == "" {
			continue
		}
		rowKey := row.Key.Text
		for i, value := range row.Value {
			columnKey := columnToKey[i]
			// ignore _ cells (these contain duplicate formulas)
			if columnKey == "_" {
				continue
			}
			compositeKey := fmt.Sprintf("%s.%s.%s", sheet.Name, rowKey, columnKey)

			v := value.DefaultValue
			if strings.HasPrefix(v, "=") {
				v = transformFormula(v, cellToKey, sheet.Name, compositeKey)

				// Replace defined names in the formula
				v = replaceDefinedNames(v, definedNames, definedNameToKey)

				v = strings.ReplaceAll(v, fmt.Sprintf(`Get("DataSources.%s")`, rowKey), fmt.Sprintf(`Get("DataSources.%s.Key")`, rowKey))
			}

			description := value.Comments

			// adding a Key field as well, because some formulas refer to the .Key field
			if i == 0 {
				compositeKey1 := fmt.Sprintf("%s.%s.Key", sheet.Name, rowKey)
				_, err := stmt.Exec(compositeKey1, "key of the datasource", rowKey, "", "")
				if err != nil {
					log.Printf("Error inserting Datasources Key: %s, value: %s\n", compositeKey1, rowKey)
				}
			}

			// adding a key/value in the db for this excel column (cell)
			_, err := stmt.Exec(compositeKey, description, v, "", "")
			if err != nil {
				log.Printf("Error inserting Datasources key: %s, value: %s\n", compositeKey, v)
			}

		}
	}
}

func processWorkmanagersSheet(sheet Sheet, stmt *sql.Stmt, cellToKey map[string]string, definedNames []string, definedNameToKey map[string]string) {
	columnToKey := map[int]string{
		0: "_",
		1: "Name",
		2: "MinimumThreads",
		3: "MaximumThreads",
		4: "MaximumCapacity",
		5: "Targets",
	}

	for _, row := range sheet.Rows {
		if row.Key.Text == "" {
			continue
		}
		rowKey := row.Key.Text
		for i, value := range row.Value {
			columnKey := columnToKey[i]
			// ignore _ cells (these contain duplicate formulas)
			if columnKey == "_" {
				continue
			}
			compositeKey := fmt.Sprintf("%s.%s.%s", sheet.Name, rowKey, columnKey)

			v := value.DefaultValue
			if strings.HasPrefix(v, "=") {
				v = transformFormula(v, cellToKey, sheet.Name, compositeKey)

				// Replace defined names in the formula
				v = replaceDefinedNames(v, definedNames, definedNameToKey)

				v = strings.ReplaceAll(v, fmt.Sprintf(`Get("WorkManagers.%s")`, rowKey), fmt.Sprintf(`Get("WorkManagers.%s.Key")`, rowKey))
			}

			description := value.Comments

			// adding a Key field as well, because some formulas refer to the .Key field
			if i == 1 {
				compositeKey1 := fmt.Sprintf("%s.%s.Key", sheet.Name, rowKey)
				_, err := stmt.Exec(compositeKey1, "key of the Workmanager", rowKey, "", "")
				if err != nil {
					log.Printf("Error inserting Workmanager Key: %s, value: %s\n", compositeKey1, rowKey)
				}
			}

			// adding a key/value in the db for this excel column (cell)
			_, err := stmt.Exec(compositeKey, description, v, "", "")
			if err != nil {
				log.Printf("Error inserting Workmanager key: %s, value: %s\n", compositeKey, v)
			}

		}
	}
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

func replaceDefinedNames(formula string, definedNames []string, definedNameToKey map[string]string) string {
	// Split the formula into parts using a regular expression that matches the delimiters
	re := regexp.MustCompile(`([=+()\s\t])`)
	parts := re.Split(formula, -1)
	delimiters := re.FindAllString(formula, -1)

	// Replace the defined names in the parts
	for i, part := range parts {
		for _, definedName := range definedNames {
			if part == definedName {
				parts[i] = fmt.Sprintf(`Get("%s")`, definedNameToKey[definedName])
			}
		}
	}

	// Join the parts back together with the delimiters
	var result strings.Builder
	for i := 0; i < len(parts); i++ {
		result.WriteString(parts[i])
		if i < len(delimiters) {
			result.WriteString(delimiters[i])
		}
	}

	return result.String()
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
	definedNameToKey := make(map[string]string)

	// First pass: Collect cell references and their corresponding keys
	for _, sheet := range excel.Sheets {
		var currentSeparator string
		for _, row := range sheet.Rows {
			if row.Separator != "" {
				currentSeparator = cleanupSeparator(row.Separator)
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
				cellToKey[fmt.Sprintf("%s!%s", sheet.Name, row.Value[0].Cell)] = compositeKey
				if row.Value[0].DefinedName != "" {
					definedNameToKey[row.Value[0].DefinedName] = compositeKey
				}
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

	// Sort defined names by length in descending order
	definedNames := make([]string, 0, len(definedNameToKey))
	for definedName := range definedNameToKey {
		definedNames = append(definedNames, definedName)
	}
	sort.Slice(definedNames, func(i, j int) bool {
		return len(definedNames[i]) > len(definedNames[j])
	})

	// Log the full cellToKey content to a file, sorted by the hashmap index
	logFile2, err := os.Create("definedNames.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer logFile2.Close()
	for _, definedName := range definedNames {
		fmt.Fprintf(logFile2, "Defined name: %s, Key: %s\n", definedName, definedNameToKey[definedName])
	}
	// Second pass: Insert data into the database with transformed formulas and replace defined names
	for _, sheet := range excel.Sheets {
		switch sheet.Name {
		case "DataSources":
			processDatasourcesSheet(sheet, stmt, cellToKey, definedNames, definedNameToKey)
			continue
		case "WorkManagers":
			processWorkmanagersSheet(sheet, stmt, cellToKey, definedNames, definedNameToKey)
			continue
		default:
			// single value sheets here
			var currentSeparator string
			for _, row := range sheet.Rows {
				if row.Separator != "" {
					currentSeparator = cleanupSeparator(row.Separator)

					if sheet.Name == "Deployments" || sheet.Name == "StandaloneApps" || sheet.Name == "SetupserverApps" {
						compositeKey := fmt.Sprintf("%s.%s.key", sheet.Name, currentSeparator)
						value := currentSeparator
						_, err = stmt.Exec(compositeKey, "Deployments key description", value, "", "")
						if err != nil {
							log.Printf("UNIQUE constraint failed for key: %s, value: %s, sheet: %s\n", compositeKey, value, sheet.Name)
						}
					}
				}
				// if row.Key.Text != "" && row.Value.DefaultValue != "" {
				if row.Key.Text != "" {
					var compositeKey string
					if currentSeparator != "" {
						compositeKey = fmt.Sprintf("%s.%s.%s", sheet.Name, currentSeparator, row.Key.Text)
					} else {
						compositeKey = fmt.Sprintf("%s.%s", sheet.Name, row.Key.Text)
					}

					value := row.Value[0].DefaultValue
					if strings.HasPrefix(value, "=") {
						value = transformFormula(value, cellToKey, sheet.Name, compositeKey)

						// Replace defined names in the formula
						value = replaceDefinedNames(value, definedNames, definedNameToKey)
					}

					description := row.Value[0].Comments

					_, err = stmt.Exec(compositeKey, description, value, "", "")
					if err != nil {
						log.Printf("UNIQUE constraint failed for key: %s, value: %s, sheet: %s\n", compositeKey, value, sheet.Name)
					}
				}
			}

		} // switch
	}

	fmt.Println("Database created successfully.")
}

func cleanupSeparator(currentSeparator string) string {
	var result string
	result = strings.ReplaceAll(currentSeparator, " / ", "_")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, " - ", "_")
	result = strings.ReplaceAll(result, "> ", "")
	result = strings.ReplaceAll(result, ">", "")
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "Relative_portnumber_(useful_when_multiple_domains_on_same_server)", "DELTA")
	return result
}
