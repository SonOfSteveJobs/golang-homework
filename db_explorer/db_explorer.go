package main

import (
	"database/sql"
	"net/http"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
type DBInfo struct {
	Repo *Repository
}

type ResMap map[string]interface{}

type TableRecordsReq struct {
	TableName string
	Id        string
	Limit     string
	Offset    string
}

func (d *DBInfo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	if path == "/" {
		d.HandleGetTables(w, r)
		return
	}

	pathSegments := strings.Split(path, "/")
	if len(pathSegments) < 2 {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}

	tableRecordsReq := TableRecordsReq{
		TableName: pathSegments[1],
		Id:        "",
		Limit:     limit,
		Offset:    offset,
	}
	if len(pathSegments) >= 3 {
		tableRecordsReq.Id = pathSegments[2]
	}
	d.HandleTableRecords(w, r, tableRecordsReq)
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	repo, err := NewRepository(db)
	if err != nil {
		return nil, err
	}

	return &DBInfo{
		Repo: repo,
	}, nil
}
