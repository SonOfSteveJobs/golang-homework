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

func (d *DBInfo) HandleTableRecords(w http.ResponseWriter, r *http.Request, req TableRecordsReq) {
	if !d.Repo.HasTable(req.TableName) {
		writeJSONError(w, http.StatusNotFound, "unknown table")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if req.Id != "" {
		switch r.Method {
		case http.MethodGet:
			d.HandleGetRecord(ctx, w, req)
		case http.MethodPost:
			d.HandleUpdateRecord(ctx, w, req)
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
		d.HandleCreateRecord(ctx, w, req)
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

	records, err := d.Repo.GetTableRecords(ctx, req.TableName, id, 0, 0)
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

func (d *DBInfo) HandleCreateRecord(ctx context.Context, w http.ResponseWriter, req TableRecordsReq) {

}

func (d *DBInfo) HandleUpdateRecord(ctx context.Context, w http.ResponseWriter, req TableRecordsReq) {

}

func (d *DBInfo) HandleDeleteRecord(ctx context.Context, w http.ResponseWriter, req TableRecordsReq) {

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
