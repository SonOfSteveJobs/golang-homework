package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"sort"
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

var ErrIncorrectOrderField = errors.New("Incorrect order field: must be one of 'Id', 'Age', or 'Name'")
var ErrIncorrectDirection = errors.New(ErrorBadOrderField)
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

func main() {
	users, err := SearchServer(SearchRequest{
		OrderField: "",
		OrderBy:    OrderByAsc,
		Limit:      900,
		Query:      "",
		Offset:     100,
	})
	if err != nil {
		fmt.Println(err)
	}

	for _, user := range users {
		fmt.Printf("User: %v\n", user)
	}
}
