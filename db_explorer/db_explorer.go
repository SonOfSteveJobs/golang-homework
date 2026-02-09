package main

import (
	"database/sql"
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

		columns, err := db.Query("SHOW FULL COLUMNS FROM `" + tableName + "`")
		if err != nil {
			return nil, err
		}

		var columnsSlice []Column
		for columns.Next() {
			col := Column{}
			var skip interface{}

			err := columns.Scan(&col.Name, &col.Type, &skip, &col.Nullable, &col.IsPrimaryKey, &col.HasDefaultValue, &skip, &skip, &skip) //всегда 9 колонок
			if err != nil {
				return nil, err
			}
			fmt.Printf("column: %v\n", col)

			columnsSlice = append(columnsSlice, col)
		}
		columns.Close()

		tablesMap[tableName] = columnsSlice
	}

	return &DBInfo{
		DB:     db,
		Tables: tablesMap,
	}, nil
}
