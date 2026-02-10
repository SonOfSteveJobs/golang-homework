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

func (d *DBInfo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// params := r.URL.Query().Get("limit")

	if path == "/" {
		d.HandleGetTables(w, r)
		return
	}

	pathSegments := strings.Split(path, "/")
	switch len(pathSegments) {
	case 2:
		d.HandleTableRecords(w, r, pathSegments[1], "")
	case 3:
		d.HandleTableRecords(w, r, pathSegments[1], pathSegments[2])
	default:
		writeJSONError(w, http.StatusBadRequest, "invalid path")
	}
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
