package stash

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const twoErrorsResponse string = `{
	"errors": [
		{
			"context": "name",
			"message": "The name should be between 1 and 255 characters.",
			"exceptionName": null
		},
		{
			"context": "email",
			"message": "The email should be a valid email address.",
			"exceptionName": null
		}
	]
}`

const invalidJSONResponse string = `{invalid: json}`

func TestStatusGreaterThan400AndTwoErrors(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprint(w, twoErrorsResponse)
	}))
	defer testServer.Close()

	request, _ := http.NewRequest("GET", testServer.URL, nil)
	actualStatus, actualBody, actualError := consumeResponse(request)
	if fmt.Sprint(actualError) != "The name should be between 1 and 255 characters. The email should be a valid email address." {
		t.Fatalf("Want error with two joined messages, but got '%v'", actualError)
	}
	if actualStatus != 400 {
		t.Fatalf("Want status 400 but found %v", actualStatus)
	}
	if string(actualBody) != twoErrorsResponse {
		t.Fatalf("Want raw response with json but found %v", actualBody)
	}
}

func TestStatusLowerThan400AndInvalidJSON(t *testing.T) {
	// here is occasional case with invalid json response and 200 OK status,
	// but we should ensure that consumeError doesn't tries to unmarshal
	// response with 200 status code
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, invalidJSONResponse)
	}))
	defer testServer.Close()

	request, _ := http.NewRequest("GET", testServer.URL, nil)
	actualStatus, actualBody, actualError := consumeResponse(request)
	if actualError != nil {
		t.Fatalf("Want error = nil, but got %v", actualError)
	}
	if actualStatus != 200 {
		t.Fatalf("Want status 400 but found %v", actualStatus)
	}
	if string(actualBody) != invalidJSONResponse {
		t.Fatalf("Want raw invalid json response but found %v", string(actualBody))
	}
}
