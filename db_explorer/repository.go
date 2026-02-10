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
	PrimaryKey    string
	Columns       []Column
	ColTypeByName map[string]string
}

type Repository struct {
	db     *sql.DB
	tables map[string]*TableSchema
}

var ErrUnknownTable = errors.New("unknown table")

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
		var primaryKey string
		for _, c := range columnsSlice {
			colTypeByName[c.Name] = strings.ToLower(c.Type)
			if c.IsPrimaryKey == "PRI" {
				primaryKey = c.Name
			}
		}

		tablesMap[key] = &TableSchema{
			Name:          key,
			PrimaryKey:    primaryKey,
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

func (repo *Repository) GetTableSchema(name string) *TableSchema {
	return repo.tables[name]
}

func (repo *Repository) GetTableNames() []string {
	tableNames := make([]string, 0, len(repo.tables))
	for name := range repo.tables {
		tableNames = append(tableNames, name)
	}
	slices.Sort(tableNames)
	return tableNames
}

func (repo *Repository) GetTableRecords(ctx context.Context, tableName string, id, limit, offset int) ([]ResMap, error) {
	_, ok := repo.tables[tableName]
	if !ok {
		return nil, ErrUnknownTable
	}

	schema := repo.tables[tableName]

	var sb strings.Builder
	sb.WriteString("SELECT * FROM `")
	sb.WriteString(tableName)
	sb.WriteString("`")

	args := make([]interface{}, 0, 3)

	if id > 0 {
		sb.WriteString(" WHERE `")
		sb.WriteString(schema.PrimaryKey)
		sb.WriteString("` = ?")
		args = append(args, id)
	}

	if limit > 0 {
		sb.WriteString(" LIMIT ?")
		args = append(args, limit)
	}

	if offset > 0 {
		sb.WriteString(" OFFSET ?")
		args = append(args, offset)
	}

	rows, err := repo.db.QueryContext(ctx, sb.String(), args...)
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

func (repo *Repository) CreateTableRecord(ctx context.Context, tableName string, body map[string]interface{}) (int64, error) {
	var colNames []string
	var placeholders []string
	var args []interface{}

	for colName, val := range body {
		colNames = append(colNames, "`"+colName+"`")
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := repo.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (repo *Repository) UpdateTableRecord(ctx context.Context, tableName string, id int, body map[string]interface{}) (int64, error) {
	schema := repo.tables[tableName]

	var setClauses []string
	var args []interface{}

	for colName, val := range body {
		setClauses = append(setClauses, "`"+colName+"` = ?")
		args = append(args, val)
	}

	if len(setClauses) == 0 {
		return 0, nil
	}

	args = append(args, id)

	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s` = ?",
		tableName,
		strings.Join(setClauses, ", "),
		schema.PrimaryKey,
	)

	result, err := repo.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (repo *Repository) DeleteTableRecord(ctx context.Context, tableName string, id int) (int64, error) {
	schema := repo.tables[tableName]

	query := fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = ?", tableName, schema.PrimaryKey)

	result, err := repo.db.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
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
