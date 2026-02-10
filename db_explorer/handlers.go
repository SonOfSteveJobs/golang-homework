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

func (d *DBInfo) HandleTableRecords(w http.ResponseWriter, r *http.Request, table string, idStr string) {
	if !d.Repo.HasTable(table) {
		writeJSONError(w, http.StatusNotFound, "unknown table")
		return
	}

	var id int
	if idStr != "" {
		idInt, err := strconv.Atoi(idStr)
		if err != nil || idInt <= 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid id, must be positive integer")
			return
		}
		id = idInt
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	records, err := d.Repo.GetTableRecords(ctx, table, id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ResMap{"records": records})
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
