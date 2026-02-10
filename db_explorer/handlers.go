package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLimit  = 5
	defaultOffset = 0
)

func (d *DBInfo) HandleGetTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusBadRequest, "unknown method (only GET supported)")
		return
	}

	tableNames := d.Repo.GetTableNames()

	writeJSON(w, http.StatusOK, ResMap{"tables": tableNames})
}

func (d *DBInfo) HandleTableRecords(w http.ResponseWriter, r *http.Request, req TableRecordsReq) {
	if !d.Repo.HasTable(req.TableName) {
		writeJSONError(w, http.StatusNotFound, "unknown table")
		return
	}

	//в задании не требудется, но мало ли..
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if req.Id != "" {
		switch r.Method {
		case http.MethodGet:
			d.HandleGetRecord(ctx, w, req)
		case http.MethodPost:
			d.HandleUpdateRecord(ctx, w, req, r.Body)
		case http.MethodDelete:
			d.HandleDeleteRecord(ctx, w, req)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		d.HandleListRecords(ctx, w, req)
	case http.MethodPut:
		d.HandleCreateRecord(ctx, w, req, r.Body)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (d *DBInfo) HandleListRecords(ctx context.Context, w http.ResponseWriter, req TableRecordsReq) {
	limit := defaultLimit
	offset := defaultOffset

	if req.Limit != "" {
		limitInt, err := strconv.Atoi(req.Limit)
		if err == nil && limitInt > 0 {
			limit = limitInt
		}
	}

	if req.Offset != "" {
		offsetInt, err := strconv.Atoi(req.Offset)
		if err == nil && offsetInt >= 0 {
			offset = offsetInt
		}
	}

	records, err := d.Repo.GetTableRecords(ctx, req.TableName, 0, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ResMap{"records": records})
}

func (d *DBInfo) HandleGetRecord(ctx context.Context, w http.ResponseWriter, req TableRecordsReq) {
	id, err := strconv.Atoi(req.Id)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid id, must be positive integer")
		return
	}

	const (
		limitSkip  = 0
		offsetSkip = 0
	)

	records, err := d.Repo.GetTableRecords(ctx, req.TableName, id, limitSkip, offsetSkip)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(records) == 0 {
		writeJSONError(w, http.StatusNotFound, "record not found")
		return
	}

	writeJSON(w, http.StatusOK, ResMap{"record": records[0]})
}

func (d *DBInfo) HandleCreateRecord(ctx context.Context, w http.ResponseWriter, req TableRecordsReq, body io.ReadCloser) {
	var bodyMap map[string]interface{}
	err := json.NewDecoder(body).Decode(&bodyMap)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	schema := d.Repo.GetTableSchema(req.TableName)

	cleanBody := make(map[string]interface{})
	for _, col := range schema.Columns {
		if col.IsPrimaryKey == "PRI" {
			continue
		}

		val, exists := bodyMap[col.Name]
		if exists {
			cleanBody[col.Name] = val
			continue
		}

		if col.Nullable == "YES" || col.HasDefaultValue.Valid {
			continue
		}

		if strings.HasPrefix(schema.ColTypeByName[col.Name], "int") {
			cleanBody[col.Name] = 0
		} else {
			cleanBody[col.Name] = ""
		}
	}

	newID, err := d.Repo.CreateTableRecord(ctx, req.TableName, cleanBody)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ResMap{schema.PrimaryKey: newID})
}

func (d *DBInfo) HandleUpdateRecord(ctx context.Context, w http.ResponseWriter, req TableRecordsReq, body io.ReadCloser) {
	id, err := strconv.Atoi(req.Id)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var bodyMap map[string]interface{}
	err = json.NewDecoder(body).Decode(&bodyMap)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	schema := d.Repo.GetTableSchema(req.TableName)

	cleanBody := make(map[string]interface{})
	for key, val := range bodyMap {
		var col *Column
		for i := range schema.Columns {
			if schema.Columns[i].Name == key {
				col = &schema.Columns[i]
				break
			}
		}

		if col == nil {
			continue
		}

		if col.IsPrimaryKey == "PRI" {
			writeJSONError(w, http.StatusBadRequest, "field "+key+" have invalid type")
			return
		}

		if val == nil {
			if col.Nullable != "YES" {
				writeJSONError(w, http.StatusBadRequest, "field "+key+" have invalid type")
				return
			}
			cleanBody[key] = nil
			continue
		}

		colType := schema.ColTypeByName[key]
		isIntCol := strings.HasPrefix(colType, "int")

		switch val.(type) {
		case float64:
			if !isIntCol {
				writeJSONError(w, http.StatusBadRequest, "field "+key+" have invalid type")
				return
			}
			cleanBody[key] = val
		case string:
			if isIntCol {
				writeJSONError(w, http.StatusBadRequest, "field "+key+" have invalid type")
				return
			}
			cleanBody[key] = val
		default:
			writeJSONError(w, http.StatusBadRequest, "field "+key+" have invalid type")
			return
		}
	}

	updated, err := d.Repo.UpdateTableRecord(ctx, req.TableName, id, cleanBody)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ResMap{"updated": updated})
}

func (d *DBInfo) HandleDeleteRecord(ctx context.Context, w http.ResponseWriter, req TableRecordsReq) {
	id, err := strconv.Atoi(req.Id)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	deleted, err := d.Repo.DeleteTableRecord(ctx, req.TableName, id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ResMap{"deleted": deleted})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ResMap{"response": data})
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ResMap{"error": msg})
}
