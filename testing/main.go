package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	OrderFieldID   = "Id"
	OrderFieldAge  = "Age"
	OrderFieldName = "Name"
)

func isValidOrder(f string) bool {
	switch f {
	case OrderFieldID, OrderFieldAge, OrderFieldName, "":
		return true
	default:
		return false
	}
}

func isValidDirection(d int) bool {
	switch d {
	case OrderByAsc, OrderByAsIs, OrderByDesc:
		return true
	default:
		return false
	}
}

var ErrIncorrectOrderField = errors.New("ErrorBadOrderField")
var ErrIncorrectDirection = errors.New("OrderBy must be -1, 0, or 1")
var ErrIncorrectOffset = errors.New("Incorrect offset: must be positive integer")

type UserXML struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type Rows struct {
	Rows []UserXML `xml:"row"`
}

func SearchServer(req SearchRequest) ([]User, error) {
	if !isValidOrder(req.OrderField) {
		return nil, ErrIncorrectOrderField
	}
	if !isValidDirection(req.OrderBy) {
		return nil, ErrIncorrectDirection
	}
	if req.Offset < 0 {
		return nil, ErrIncorrectOffset
	}

	data, err := os.ReadFile("dataset.xml")
	if err != nil {
		return nil, err
	}

	rows := &Rows{}
	err = xml.Unmarshal(data, rows)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, req.Limit)

	for _, row := range rows.Rows {
		name := row.FirstName + " " + row.LastName
		if strings.Contains(name, req.Query) || strings.Contains(row.About, req.Query) {
			users = append(users, User{
				Id:     row.Id,
				Name:   name,
				Age:    row.Age,
				Gender: row.Gender,
				About:  row.About,
			})
		}
	}

	if req.OrderBy != OrderByAsIs {
		sort.Slice(users, func(i, j int) bool {
			var less bool
			switch req.OrderField {
			case OrderFieldID:
				less = users[i].Id < users[j].Id
			case OrderFieldAge:
				less = users[i].Age < users[j].Age
			default:
				less = users[i].Name < users[j].Name
			}

			if req.OrderBy == OrderByDesc {
				return !less
			}
			return less
		})
	}

	if req.Offset < len(users) {
		users = users[req.Offset:]
	} else {
		users = []User{}
	}

	if req.Limit < len(users) {
		users = users[:req.Limit]
	}

	return users, nil
}

func SearchServerHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := r.URL.Query()
	limit, _ := strconv.Atoi(params.Get("limit"))
	offset, _ := strconv.Atoi(params.Get("offset"))
	orderField := params.Get("order_field")
	orderBy, _ := strconv.Atoi(params.Get("order_by"))
	query := params.Get("query")

	users, err := SearchServer(SearchRequest{
		OrderField: orderField,
		OrderBy:    orderBy,
		Limit:      limit,
		Query:      query,
		Offset:     offset,
	})
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		err = encoder.Encode(SearchErrorResponse{
			Error: err.Error(),
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	err = encoder.Encode(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	server := httptest.NewServer(http.HandlerFunc(SearchServerHandler))
	defer server.Close()

	client := &SearchClient{
		URL:         server.URL,
		AccessToken: "12345",
	}

	res, err := client.FindUsers(SearchRequest{
		OrderField: "",
		OrderBy:    2,
		Limit:      25,
		Query:      "",
		Offset:     25,
	})
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%+v\n", res)
}
