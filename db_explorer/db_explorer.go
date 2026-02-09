package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
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

func (d *DBInfo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method
	fmt.Println("PATH:", path, "METHOD:", method)

	if method == http.MethodGet && path == "/" {
		tableNames := make([]string, 0, len(d.Tables))
		for tableName := range d.Tables {
			tableNames = append(tableNames, tableName)
		}

		resMap := map[string]interface{}{
			"response": map[string]interface{}{
				"tables": tableNames,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resMap)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(d.Tables)
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
			rows.Close()
			return nil, err
		}
		// хз можно ли так писать, делаю чтобы не создавать промежуточные структуры, например слайс с именами
		tablesMap[tableName] = nil
	}

	if rows.Err() != nil {
		rows.Close()
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
