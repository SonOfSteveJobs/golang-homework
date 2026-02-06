package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// код писать тут
//
//	go test -v
//	go test -cover
//	go test -coverprofile=cover.out
//	go tool cover -html=cover.out -o cover.html

type TestCase struct {
	ID                string
	Request           SearchRequest
	ResponseIds       []int
	isErr             bool
	ErrContainsString string
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
		ResponseIds:       []int{13},
		isErr:             false,
		ErrContainsString: "",
	},
	{
		ID: "success len(data) != req.Limit",
		Request: SearchRequest{
			OrderField: "",
			OrderBy:    1,
			Limit:      30,
			Query:      "Boyd Wolf",
			Offset:     0,
		},
		ResponseIds: []int{0},
		isErr:       false,
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
		ResponseIds:       []int{},
		isErr:             true,
		ErrContainsString: "limit must be > 0",
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
		ResponseIds:       []int{13, 33, 18, 26, 9, 27, 31, 4, 14, 20, 7, 25, 34, 21, 6, 1, 10, 24, 8, 11, 23, 3, 17, 30, 12},
		isErr:             false,
		ErrContainsString: "",
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
		ResponseIds:       []int{},
		isErr:             true,
		ErrContainsString: "offset must be > 0",
	},
	{
		ID: "incorrect order by",
		Request: SearchRequest{
			OrderField: "",
			OrderBy:    2,
			Limit:      30,
			Query:      "",
			Offset:     0,
		},
		ResponseIds:       []int{},
		isErr:             true,
		ErrContainsString: "unknown bad request error: OrderBy must be -1, 0, or 1",
	},
	{
		ID: "incorrect order field",
		Request: SearchRequest{
			OrderField: "12345",
			OrderBy:    1,
			Limit:      30,
			Query:      "",
			Offset:     0,
		},
		ResponseIds:       []int{},
		isErr:             true,
		ErrContainsString: "OrderFeld 12345 invalid",
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

	for _, tc := range testCases {
		t.Run(tc.ID, func(t *testing.T) {
			res, err := client.FindUsers(tc.Request)
			if err != nil && !tc.isErr {
				t.Errorf("expected no error, got: %v", err)
				return
			}

			if tc.isErr {
				if nil == err {
					t.Error("expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tc.ErrContainsString) {
					t.Errorf("expected error: %v, got: %v", tc.ErrContainsString, err.Error())
				}
				return
			}

			// users := []int{}
			for _, user := range res.Users {
				// users = append(users, user.Id)
				if !contains(tc.ResponseIds, user.Id) {
					t.Errorf("expected user id(s): %v, failed id: %d", tc.ResponseIds, user.Id)
				}
			}
			// t.Logf("%+v", users)
		})
	}
}

type TestCaseServerError struct {
	ID                string
	ErrContainsString string
	URL               string
	AccessToken       string
	Handler           http.HandlerFunc
}

var testCasesServerError = []TestCaseServerError{
	{
		ID:                "timeout",
		ErrContainsString: "timeout for",
		AccessToken:       "12345",
		Handler:           SlowSearchServerHandler,
	},
	{
		ID:                "unknown error",
		ErrContainsString: "unknown error",
		URL:               "http://invalidURL:12345",
		AccessToken:       "12345",
		Handler:           nil,
	},
	{
		ID:                "unauthorized",
		ErrContainsString: "Bad AccessToken",
		AccessToken:       "",
		Handler:           SearchServerHandler,
	},
	{
		ID:                "internal server error",
		ErrContainsString: "SearchServer fatal error",
		AccessToken:       "12345",
		Handler:           InternalErrorHandler,
	},
	{
		ID:                "bad request",
		ErrContainsString: "cant unpack error json",
		AccessToken:       "12345",
		Handler:           BadRequestHandler,
	},
	{
		ID:                "invalid json 200 response",
		ErrContainsString: "cant unpack result json",
		AccessToken:       "12345",
		Handler:           InvalidJSONHandler,
	},
}

func TestServerErrors(t *testing.T) {
	for _, tc := range testCasesServerError {
		t.Run(tc.ID, func(t *testing.T) {
			url := tc.URL

			if tc.Handler != nil {
				server := httptest.NewServer(tc.Handler)
				defer server.Close()
				url = server.URL
			}

			client := &SearchClient{
				URL:         url,
				AccessToken: tc.AccessToken,
			}

			_, err := client.FindUsers(SearchRequest{Limit: 1})

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.ErrContainsString)
			}

			if !strings.Contains(err.Error(), tc.ErrContainsString) {
				t.Errorf("expected error containing %q, got: %v", tc.ErrContainsString, err)
			}
		})
	}
}

func SlowSearchServerHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
	w.Write([]byte("timeout"))
}

func InternalErrorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func BadRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func InvalidJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("invalid json"))
}
