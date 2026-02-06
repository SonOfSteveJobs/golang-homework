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
	ExpectedIds       []int // id в правильном порядке
	ExpectedCount     int
	ExpectedNextPage  bool
	isErr             bool
	ErrContainsString string
}

var testCases = []TestCase{
	// успешные кейсы
	{
		ID: "success basic",
		Request: SearchRequest{
			OrderField: "",
			OrderBy:    OrderByDesc,
			Limit:      1,
			Query:      "",
			Offset:     0,
		},
		ExpectedIds:      []int{13},
		ExpectedCount:    1,
		ExpectedNextPage: true,
	},
	{
		ID: "search by Name",
		Request: SearchRequest{
			Limit: 10,
			Query: "Boyd Wolf",
		},
		ExpectedIds:      []int{0},
		ExpectedCount:    1,
		ExpectedNextPage: false,
	},
	{
		ID: "search in About field",
		Request: SearchRequest{
			Limit: 5,
			Query: "adipisicing",
		},
		ExpectedCount:    5,
		ExpectedNextPage: true,
	},
	{
		ID: "query no results",
		Request: SearchRequest{
			Limit: 10,
			Query: "люблюписатьтесты)",
		},
		ExpectedIds:      []int{},
		ExpectedCount:    0,
		ExpectedNextPage: false,
	},

	//сортировка
	{
		ID: "sort by Name asc",
		Request: SearchRequest{
			OrderField: "Name",
			OrderBy:    OrderByAsc,
			Limit:      3,
		},
		ExpectedIds:      []int{15, 16, 19},
		ExpectedCount:    3,
		ExpectedNextPage: true,
	},
	{
		ID: "sort by Age asc",
		Request: SearchRequest{
			OrderField: "Age",
			OrderBy:    OrderByAsc,
			Limit:      3,
		},
		ExpectedIds:      []int{1, 15, 23},
		ExpectedCount:    3,
		ExpectedNextPage: true,
	},
	{
		ID: "sort by Id desc",
		Request: SearchRequest{
			OrderField: "Id",
			OrderBy:    OrderByDesc,
			Limit:      3,
		},
		ExpectedIds:      []int{34, 33, 32},
		ExpectedCount:    3,
		ExpectedNextPage: true,
	},
	{
		ID: "OrderByAsIs - no sorting",
		Request: SearchRequest{
			OrderField: "Id",
			OrderBy:    OrderByAsIs,
			Limit:      3,
		},
		ExpectedIds:      []int{0, 1, 2},
		ExpectedCount:    3,
		ExpectedNextPage: true,
	},

	// Limit
	{
		ID: "limit capped at 25",
		Request: SearchRequest{
			OrderField: "Id",
			OrderBy:    OrderByAsc,
			Limit:      100,
		},
		ExpectedCount:    25,
		ExpectedNextPage: true,
	},

	// Offset
	{
		ID: "offset skips records",
		Request: SearchRequest{
			OrderField: "Id",
			OrderBy:    OrderByAsc,
			Limit:      3,
			Offset:     5,
		},
		ExpectedIds:      []int{5, 6, 7},
		ExpectedCount:    3,
		ExpectedNextPage: true,
	},
	{
		ID: "offset near end",
		Request: SearchRequest{
			OrderField: "Id",
			OrderBy:    OrderByAsc,
			Limit:      10,
			Offset:     32,
		},
		ExpectedIds:      []int{32, 33, 34},
		ExpectedCount:    3,
		ExpectedNextPage: false,
	},
	{
		ID: "empty result",
		Request: SearchRequest{
			Limit:  10,
			Offset: 100,
		},
		ExpectedIds:      []int{},
		ExpectedCount:    0,
		ExpectedNextPage: false,
	},

	// ошибки
	{
		ID: "negative limit",
		Request: SearchRequest{
			Limit: -1,
		},
		isErr:             true,
		ErrContainsString: "limit must be > 0",
	},
	{
		ID: "negative offset",
		Request: SearchRequest{
			Limit:  10,
			Offset: -1,
		},
		isErr:             true,
		ErrContainsString: "offset must be > 0",
	},
	{
		ID: "invalid order by",
		Request: SearchRequest{
			OrderBy: 2,
			Limit:   10,
		},
		isErr:             true,
		ErrContainsString: "unknown bad request error",
	},
	{
		ID: "invalid order field",
		Request: SearchRequest{
			OrderField: "InvalidField",
			Limit:      10,
		},
		isErr:             true,
		ErrContainsString: "OrderFeld InvalidField invalid",
	},
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

			if tc.isErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.ErrContainsString) {
					t.Errorf("expected error containing %q, got: %v", tc.ErrContainsString, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(res.Users) != tc.ExpectedCount {
				t.Errorf("expected %d users, got %d", tc.ExpectedCount, len(res.Users))
			}

			if res.NextPage != tc.ExpectedNextPage {
				t.Errorf("expected NextPage=%v, got %v", tc.ExpectedNextPage, res.NextPage)
			}

			if len(tc.ExpectedIds) > 0 {
				if len(res.Users) != len(tc.ExpectedIds) {
					t.Errorf("expected %d users, got %d", len(tc.ExpectedIds), len(res.Users))
					return
				}
				for i, user := range res.Users {
					if user.Id != tc.ExpectedIds[i] {
						t.Errorf("position %d: expected id %d, got %d", i, tc.ExpectedIds[i], user.Id)
					}
				}
			}
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
