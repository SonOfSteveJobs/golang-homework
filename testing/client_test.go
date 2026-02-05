package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// код писать тут
//
//	go test -v
//	go test -cover
//	go test -coverprofile=cover.out
//	go tool cover -html=cover.out -o cover.html

type TestCase struct {
	ID          string
	Request     SearchRequest
	ResponseIds []int
	isErr       bool
	Err         string
}

var testCases = []TestCase{
	{
		ID: "success",
		Request: SearchRequest{
			OrderField: "",
			OrderBy:    1,
			Limit:      1,
			Query:      "",
			Offset:     0,
		},
		ResponseIds: []int{13},
		isErr:       false,
		Err:         "",
	},
	{
		ID: "negative limit",
		Request: SearchRequest{
			OrderField: "",
			OrderBy:    1,
			Limit:      -1,
			Query:      "",
			Offset:     0,
		},
		ResponseIds: []int{},
		isErr:       true,
		Err:         "limit must be > 0",
	},
	{
		ID: "limit > 25",
		Request: SearchRequest{
			OrderField: "",
			OrderBy:    1,
			Limit:      30,
			Query:      "",
			Offset:     0,
		},
		ResponseIds: []int{13, 33, 18, 26, 9, 27, 31, 4, 14, 20, 7, 25, 34, 21, 6, 1, 10, 24, 8, 11, 23, 3, 17, 30, 12},
		isErr:       false,
		Err:         "",
	},
	{
		ID: "negative offset",
		Request: SearchRequest{
			OrderField: "",
			OrderBy:    1,
			Limit:      30,
			Query:      "",
			Offset:     -1,
		},
		ResponseIds: []int{},
		isErr:       true,
		Err:         "offset must be > 0",
	},
}

// надо так как старая версия го не содержит функцию slices.Contains
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func TestSearchServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServerHandler))
	defer server.Close()

	client := &SearchClient{
		URL:         server.URL,
		AccessToken: "12345",
	}

	for _, ts := range testCases {
		t.Run(ts.ID, func(t *testing.T) {
			res, err := client.FindUsers(ts.Request)
			if err != nil && !ts.isErr {
				t.Errorf("expected no error, got: %v", err)
				return
			}

			if ts.isErr {
				if nil == err {
					t.Error("expected error, got nil")
					return
				}
				if err.Error() != ts.Err {
					t.Errorf("expected error: %v, got: %v", ts.Err, err.Error())
				}
				return
			}

			// users := []int{}
			for _, user := range res.Users {
				// users = append(users, user.Id)
				if !contains(ts.ResponseIds, user.Id) {
					t.Errorf("expected user id(s): %v, failed id: %d", ts.ResponseIds, user.Id)
				}
			}
			// t.Logf("%+v", users)
		})
	}
}
