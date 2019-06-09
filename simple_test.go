package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var s Samwise

// func SetUp() {
// 	s.Initialize()
// }

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)
	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

// func TestEmptyDB(t *testing.T) {
// 	// SetUp()
// 	req, _ := http.NewRequest("GET", "api/v1/test/example", nil)
// 	response := executeRequest(req)

// 	checkResponseCode(t, http.StatusNotFound, response.Code)
// }

// func TestEmptyFolderGET(t *testing.T) {

// 	req, _ := http.NewRequest("GET", "api/v1/test/missing", nil)
// 	response := executeRequest(req)

// 	checkResponseCode(t, http.StatusNotFound, response.Code)

// 	// TODO replace with 404 response
// 	if body := response.Body.String(); body != "[]" {
// 		t.Errorf("Expected an empty array. Got %s", body)
// 	}
// }

func TestEmptyFolderKeys(t *testing.T) {

	req, _ := http.NewRequest("GET", "api/v1/keys/test/", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, response.Code)

	if body := response.Body.String(); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func TestMain(m *testing.M) {
	s = Samwise{}
	s.Initialize()
	os.Exit(m.Run())
}
