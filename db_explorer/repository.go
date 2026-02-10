package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Column struct {
	Name            string
	Type            string
	Nullable        string
	IsPrimaryKey    string
	HasDefaultValue sql.NullString
}

type TableSchema struct {
	Name          string
	Columns       []Column
	ColTypeByName map[string]string
}

type Repository struct {
	db     *sql.DB
	tables map[string]*TableSchema
}

func NewRepository(db *sql.DB) (*Repository, error) {
	rows, err := db.Query(`SHOW TABLES`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tablesMap := make(map[string]*TableSchema)
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

	for key := range tablesMap {
		columnsSlice, err := loadColumns(db, key)
		if err != nil {
			return nil, err
		}

		colTypeByName := make(map[string]string, len(columnsSlice))
		for _, c := range columnsSlice {
			colTypeByName[c.Name] = strings.ToLower(c.Type)
		}

		tablesMap[key] = &TableSchema{
			Name:          key,
			Columns:       columnsSlice,
			ColTypeByName: colTypeByName,
		}
	}

	return &Repository{
		db:     db,
		tables: tablesMap,
	}, nil
}

func (repo *Repository) HasTable(name string) bool {
	_, ok := repo.tables[name]
	return ok
}

func (repo *Repository) GetTableNames() []string {
	tableNames := make([]string, 0, len(repo.tables))
	for name := range repo.tables {
		tableNames = append(tableNames, name)
	}
	slices.Sort(tableNames)
	return tableNames
}

func (repo *Repository) GetTableRecords(ctx context.Context, tableName string, id int) ([]ResMap, error) {
	_, ok := repo.tables[tableName]
	if !ok {
		return nil, errors.New("unknown table")
	}

	schema := repo.tables[tableName]

	var (
		rows *sql.Rows
		err  error
	)

	if id > 0 {
		query := fmt.Sprintf("SELECT * FROM `%s` WHERE id = ?", tableName)
		rows, err = repo.db.QueryContext(ctx, query, id)
	} else {
		query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
		rows, err = repo.db.QueryContext(ctx, query)
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
				colType := schema.ColTypeByName[colName]

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
