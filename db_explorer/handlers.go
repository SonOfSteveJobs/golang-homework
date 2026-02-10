package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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

func (d *DBInfo) HandleTableRecords(w http.ResponseWriter, r *http.Request, tableRecordsReq TableRecordsReq) {
	if !d.Repo.HasTable(tableRecordsReq.TableName) {
		writeJSONError(w, http.StatusNotFound, "unknown table")
		return
	}

	var (
		id     int
		limit  int = defaultLimit
		offset int = defaultOffset
	)

	if tableRecordsReq.Id != "" {
		idInt, err := strconv.Atoi(tableRecordsReq.Id)
		if err != nil || idInt <= 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid id, must be positive integer")
			return
		}
		id = idInt
	}

	if tableRecordsReq.Limit != "" {
		limitInt, err := strconv.Atoi(tableRecordsReq.Limit)
		if err != nil || limitInt <= 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid limit, must be positive integer")
			return
		}
		limit = limitInt
	}

	if tableRecordsReq.Offset != "" {
		offsetInt, err := strconv.Atoi(tableRecordsReq.Offset)
		if err != nil || offsetInt < 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid offset, must be non-negative integer")
			return
		}
		offset = offsetInt
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		records, err := d.Repo.GetTableRecords(ctx, tableRecordsReq.TableName, id, limit, offset)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if id > 0 {
			if len(records) == 1 {
				writeJSON(w, http.StatusOK, ResMap{"record": records[0]})
			}
			if len(records) == 0 {
				writeJSONError(w, http.StatusNotFound, "record not found")
			}
			return
		}

	case http.MethodPut:

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed for this endpoint")
	}
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
