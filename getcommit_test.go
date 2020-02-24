package stash

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetCommit(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("wanted GET but found %s\n", r.Method)
		}
		url := *r.URL
		if url.Path != "/rest/api/1.0/projects/PROJ/repos/slug/commits/6782bf94782450a4e6a0d548e4c803692ca38b94" {
			t.Fatalf("Want /rest/api/1.0/projects/PROJ/repos/slug/commits/6782bf94782450a4e6a0d548e4c803692ca38b94 but found %s\n", url.Path)
		}
		if r.Header.Get("Authorization") != "Basic dTpw" {
			t.Fatalf("Want Basic dTpw but found %s\n", r.Header.Get("Authorization"))
		}
		w.Write(
			[]byte(`{
  "id": "6782bf94782450a4e6a0d548e4c803692ca38b94",
  "displayId": "6782bf9",
  "author": {
    "name": "a",
    "emailAddress": "a@example.com",
    "id": 360,
    "displayName": "Bob Loblaw",
    "active": true,
    "slug": "a",
    "type": "NORMAL",
    "link": {
      "url": "/users/a",
      "rel": "self"
    },
    "links": {
      "self": [
        {
          "href": "https://git.example.com/users/a"
        }
      ]
    }
  },
  "authorTimestamp": 1459802103000,
  "message": "Updating develop poms",
  "parents": [
    {
      "id": "e00a0565a8a27cb1a130d5acce50cb3c4203c39d",
      "displayId": "e00a056"
    }
  ]
}
`),
		)
	}))
	defer testServer.Close()

	url, _ := url.Parse(testServer.URL)
	stashClient := NewClient("u", "p", url)
	commit, err := stashClient.GetCommit("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94")
	if err != nil {
		t.Fatalf("Not expecting error: %v\n", err)
	}
	if commit.ID != "6782bf94782450a4e6a0d548e4c803692ca38b94" {
		t.Fatalf("Want 6782bf94782450a4e6a0d548e4c803692ca38b94 but got %s\n", commit.ID)
	}
	if commit.AuthorTimestamp != 1459802103000 {
		t.Fatalf("Want 1459802103000 but got %d\n", commit.AuthorTimestamp)
	}
}

func TestGetCommit404(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{
        "errors": [
            {
                "context": null,
                "message": "A detailed error message.",
                "exceptionName": null
            }
        ]
    }`))
	}))
	defer testServer.Close()

	url, _ := url.Parse(testServer.URL)
	stashClient := NewClient("u", "p", url)
	_, err := stashClient.GetCommit("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94")
	if err == nil {
		t.Fatalf("Expecting error but did not get one\n")
	}
}

func TestGetCommit401(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{
        "errors": [
            {
                "context": null,
                "message": "A detailed error message.",
                "exceptionName": null
            }
        ]
    }`))
	}))
	defer testServer.Close()

	url, _ := url.Parse(testServer.URL)
	stashClient := NewClient("u", "p", url)
	_, err := stashClient.GetCommit("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94")
	if err == nil {
		t.Fatalf("Expecting error but did not get one\n")
	}
}

func TestGetCommit400(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{
        "errors": [
            {
                "context": null,
                "message": "A detailed error message.",
                "exceptionName": null
            }
        ]
    }`))
	}))
	defer testServer.Close()

	url, _ := url.Parse(testServer.URL)
	stashClient := NewClient("u", "p", url)
	_, err := stashClient.GetCommit("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94")
	if err == nil {
		t.Fatalf("Expecting error but did not get one\n")
	}
}
