package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
type DBInfo struct {
	DB     *sql.DB
	Tables map[string][]Column
}

type Column struct {
	Name            string
	Type            string
	Nullable        string
	IsPrimaryKey    string
	HasDefaultValue sql.NullString
}

type ResMap map[string]interface{}

func (d *DBInfo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method
	// params := r.URL.Query().Get("limit")

	if path == "/" {
		if method != http.MethodGet {
			resMap := ResMap{
				"error": "unknown method (only GET supported)",
			}
			createJSONResponse(w, http.StatusBadRequest, resMap)
			return
		}
		tableNames := d.GetTables()

		resMap := ResMap{
			"response": ResMap{
				"tables": tableNames,
			},
		}

		createJSONResponse(w, http.StatusOK, resMap)
		return
	}

	pathSegments := strings.Split(path, "/")
	var table string
	var id int

	switch len(pathSegments) {
	case 2:
		table = pathSegments[1]
	case 3:
		table = pathSegments[1]
		idStr := pathSegments[2]
		if idStr != "" {
			idInt, err := strconv.Atoi(idStr)
			if err != nil || idInt <= 0 {
				resMap := ResMap{
					"error": "invalid id, must be positive integer",
				}
				createJSONResponse(w, http.StatusBadRequest, resMap)
				return
			}

			id = idInt
		}
	default:
		resMap := ResMap{
			"error": "invalid path",
		}
		createJSONResponse(w, http.StatusBadRequest, resMap)
		return
	}

	_, ok := d.Tables[table]
	if !ok {
		resMap := ResMap{
			"error": "unknown table",
		}
		createJSONResponse(w, http.StatusNotFound, resMap)
		return
	}

	resMap := ResMap{}

	if table != "" {
		records, err := d.getTableRecords(table, id)
		if err != nil {
			fmt.Printf("Error getting table records: %v\n", err)
			resMap["error"] = err.Error()
			createJSONResponse(w, http.StatusInternalServerError, resMap)
			return
		}

		resMap["response"] = map[string][]ResMap{"records": records}
	}
	createJSONResponse(w, http.StatusOK, resMap)
}

func (d *DBInfo) GetTables() []string {
	tableNames := make([]string, 0, len(d.Tables))
	for name := range d.Tables {
		tableNames = append(tableNames, name)
	}
	slices.Sort(tableNames)
	return tableNames
}

func (d *DBInfo) getTableRecords(tableName string, id int) ([]ResMap, error) {
	colTypeByName := make(map[string]string, len(d.Tables[tableName]))
	for _, c := range d.Tables[tableName] {
		colTypeByName[c.Name] = strings.ToLower(c.Type)
	}

	var (
		rows *sql.Rows
		err  error
	)

	if id > 0 {
		query := fmt.Sprintf("SELECT * FROM `%s` WHERE id = ?", tableName)
		rows, err = d.DB.Query(query, id)
	} else {
		query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
		rows, err = d.DB.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var records []ResMap

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		record := make(ResMap, len(columns))
		for i, colName := range columns {
			val := values[i]

			b, ok := val.([]byte)
			if ok {
				colType := colTypeByName[colName]

				if strings.HasPrefix(colType, "int") {
					intVal, err := strconv.Atoi(string(b))
					if nil == err {
						record[colName] = intVal
						continue
					}
				}
				record[colName] = string(b)
				continue
			}

			record[colName] = val
		}
		records = append(records, record)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return records, nil
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	rows, err := db.Query(`SHOW TABLES`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tablesMap := make(map[string][]Column)
	for rows.Next() {
		tableName := ""
		err := rows.Scan(&tableName)

		if err != nil {
			return nil, err
		}
		// хз можно ли так писать, делаю чтобы не создавать промежуточные структуры, например слайс с именами
		tablesMap[tableName] = nil
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	for key, _ := range tablesMap {
		columnsSlice, err := loadColumns(db, key)
		if err != nil {
			return nil, err
		}
		tablesMap[key] = columnsSlice
	}

	return &DBInfo{
		DB:     db,
		Tables: tablesMap,
	}, nil
}

func loadColumns(db *sql.DB, tableName string) ([]Column, error) {
	//Задание говорит использовать SHOW FULL COLUMNS, но мне плохо от записи columns.Scan(&col.Name, &col.Type, &skip, &col.Nullable, &col.IsPrimaryKey, &col.HasDefaultValue, &skip, &skip, &skip)
	const query = `
SELECT
  COLUMN_NAME,
  COLUMN_TYPE,
  IS_NULLABLE,
  COLUMN_KEY,
  COLUMN_DEFAULT
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY ORDINAL_POSITION;
`

	columns, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer columns.Close()

	var columnsSlice []Column
	for columns.Next() {
		var col Column
		err := columns.Scan(
			&col.Name,
			&col.Type,
			&col.Nullable,
			&col.IsPrimaryKey,
			&col.HasDefaultValue,
		)
		if err != nil {
			return nil, err
		}

		columnsSlice = append(columnsSlice, col)
	}
	if columns.Err() != nil {
		return nil, columns.Err()
	}

	return columnsSlice, nil
}

func createJSONResponse(w http.ResponseWriter, status int, data ResMap) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
