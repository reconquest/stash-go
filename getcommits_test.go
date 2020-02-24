package stash

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetCommits(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("wanted GET but found %s\n", r.Method)
		}
		url := *r.URL
		if url.Path != "/rest/api/1.0/projects/PROJ/repos/slug/commits" {
			t.Fatalf("Want /rest/api/1.0/projects/PROJ/repos/slug/commits but found %s\n", url.Path)
		}
		t.Log(url.Query()["since"])
		if url.Query()["since"][0] != "6782bf94782450a4e6a0d548e4c803692ca38b94" {
			t.Fatalf("Want since=6782bf94782450a4e6a0d548e4c803692ca38b94, but found %s\n", url.Query()["since"])
		}
		if url.Query()["until"][0] != "38b94f94782450a4e6a0d548e4c803692ca6782b" {
			t.Fatalf("Want until=38b94f94782450a4e6a0d548e4c803692ca6782b&limit=1000 but found %s\n", url.Query()["until"])
		}
		if url.Query()["limit"][0] != "1000" {
			t.Fatalf("Want limit=1000 but found %s\n", url.Query()["limit"])
		}
		if r.Header.Get("Authorization") != "Basic dTpw" {
			t.Fatalf("Want Basic dTpw but found %s\n", r.Header.Get("Authorization"))
		}
		w.Write(
			[]byte(`{ "values": [{
	  "id": "f94786782b2450a4e6a0d548e4c803692ca38b94",
	  "displayId": "f947867",
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
	},
{
	  "id": "38b94f94782450a4e6a0d548e4c803692ca6782b",
	  "displayId": "38b94f9",
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
	  "message": "Implement fancy new feature",
	  "parents": [
	    {
	      "id": "e00a0565a8a27cb1a130d5acce50cb3c4203c39d",
	      "displayId": "e00a056"
	    }
	  ]
	}
]}
`),
		)
	}))
	defer testServer.Close()

	url, _ := url.Parse(testServer.URL)
	stashClient := NewClient("u", "p", url)
	commits, err := stashClient.GetCommits("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94", "38b94f94782450a4e6a0d548e4c803692ca6782b")
	if err != nil {
		t.Fatalf("Not expecting error: %v\n", err)
	}

	if commits.Commits[0].ID != "f94786782b2450a4e6a0d548e4c803692ca38b94" {
		t.Fatalf("Want f94786782b2450a4e6a0d548e4c803692ca38b94 but got %s\n", commits.Commits[0].ID)
	}

	if commits.Commits[1].ID != "38b94f94782450a4e6a0d548e4c803692ca6782b" {
		t.Fatalf("Want 38b94f94782450a4e6a0d548e4c803692ca6782b but got %s\n", commits.Commits[1].ID)
	}
}

func TestGetCommits404(t *testing.T) {
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
	_, err := stashClient.GetCommits("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94", "38b94f94782450a4e6a0d548e4c803692ca6782b")
	if err == nil {
		t.Fatalf("Expecting error but did not get one\n")
	}
}

func TestGetCommits401(t *testing.T) {
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
	_, err := stashClient.GetCommits("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94", "38b94f94782450a4e6a0d548e4c803692ca6782b")
	if err == nil {
		t.Fatalf("Expecting error but did not get one\n")
	}
}

func TestGetCommits400(t *testing.T) {
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
	_, err := stashClient.GetCommits("PROJ", "slug", "6782bf94782450a4e6a0d548e4c803692ca38b94", "38b94f94782450a4e6a0d548e4c803692ca6782b")
	if err == nil {
		t.Fatalf("Expecting error but did not get one\n")
	}
}
