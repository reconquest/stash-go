// Atlassian Stash API package.
// Stash API Reference: https://developer.atlassian.com/static/rest/stash/3.0.1/stash-rest.html
package stash

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ae6rt/retry"
)

var Log *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

type (
	Stash interface {
		CreateRepository(projectKey, slug string) (Repository, error)
		RenameRepository(projectKey, slug, newslug string) error
		GetRepositories() (map[int]Repository, error)
		GetBranches(projectKey, repositorySlug string) (map[string]Branch, error)
		GetTags(projectKey, repositorySlug string) (map[string]Tag, error)
		CreateBranchRestriction(projectKey, repositorySlug, branch, user string) (BranchRestriction, error)
		GetBranchRestrictions(projectKey, repositorySlug string) (BranchRestrictions, error)
		DeleteBranchRestriction(projectKey, repositorySlug string, id int) error
		GetRepository(projectKey, repositorySlug string) (Repository, error)
		GetPullRequests(projectKey, repositorySlug, state string) ([]PullRequest, error)
		GetPullRequest(projectKey, repositorySlug, identifier string) (PullRequest, error)
		GetRawFile(projectKey, repositorySlug, branch, filePath string) ([]byte, error)
		CreatePullRequest(projectKey, repositorySlug, title, description, fromRef, toRef string, reviewers []string) (PullRequest, error)
		UpdatePullRequest(projectKey, repositorySlug, identifier string, version int, title, description, toRef string, reviewers []string) (PullRequest, error)
		DeleteBranch(projectKey, repositorySlug, branchName string) error
		GetCommit(projectKey, repositorySlug, commitHash string) (Commit, error)
		GetCommits(projectKey, repositorySlug, commitSinceHash string, commitUntilHash string) (Commits, error)
		CreateComment(projectKey, repositorySlug, pullRequest, text string) (Comment, error)
	}

	Client struct {
		userName string
		password string
		baseURL  *url.URL
		Stash
	}

	Page struct {
		IsLastPage    bool `json:"isLastPage"`
		Size          int  `json:"size"`
		Start         int  `json:"start"`
		NextPageStart int  `json:"nextPageStart"`
	}

	Repositories struct {
		IsLastPage    bool         `json:"isLastPage"`
		Size          int          `json:"size"`
		Start         int          `json:"start"`
		NextPageStart int          `json:"nextPageStart"`
		Repository    []Repository `json:"values"`
	}

	Repository struct {
		ID      int     `json:"id"`
		Name    string  `json:"name"`
		Slug    string  `json:"slug"`
		Project Project `json:"project"`
		ScmID   string  `json:"scmId"`
		Links   Links   `json:"links"`
	}

	Project struct {
		Key string `json:"key"`
	}

	Links struct {
		Clones []Clone `json:"clone"`
	}

	Clone struct {
		HREF string `json:"href"`
		Name string `json:"name"`
	}

	Branches struct {
		IsLastPage    bool     `json:"isLastPage"`
		Size          int      `json:"size"`
		Start         int      `json:"start"`
		NextPageStart int      `json:"nextPageStart"`
		Branch        []Branch `json:"values"`
	}

	Branch struct {
		ID              string `json:"id"`
		DisplayID       string `json:"displayId"`
		LatestChangeSet string `json:"latestChangeset"`
		IsDefault       bool   `json:"isDefault"`
	}

	Tags struct {
		Page
		Tags []Tag `json:"values"`
	}

	Tag struct {
		ID        string `json:"id"`
		DisplayID string `json:"displayId"`
		Hash      string `json:"hash"`
	}

	BranchRestrictions struct {
		BranchRestriction []BranchRestriction `json:"values"`
	}

	BranchRestriction struct {
		Id     int    `json:"id"`
		Branch Branch `json:"branch"`
	}

	BranchPermission struct {
		Type   string   `json:"type"`
		Branch string   `json:"value"`
		Users  []string `json:"users"`
		Groups []string `json:"groups"`
	}

	PullRequests struct {
		Page
		PullRequests []PullRequest `json:"values"`
	}

	PullRequest struct {
		ID          int        `id:"closed"`
		Version     int        `json:"version"`
		Closed      bool       `json:"closed"`
		Open        bool       `json:"open"`
		State       string     `json:"state"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		FromRef     Ref        `json:"fromRef"`
		ToRef       Ref        `json:"toRef"`
		CreatedDate int64      `json:"createdDate"`
		UpdatedDate int64      `json:"updatedDate"`
		Reviewers   []Reviewer `json:"reviewers"`
	}

	Comment struct {
		ID int `json:"id"`
	}

	Ref struct {
		DisplayID string `json:"displayId"`
	}

	errorResponse struct {
		StatusCode int
		Reason     string
		error
	}

	stashError struct {
		Errors []struct {
			Context       string `json:"context"`
			Message       string `json:"message"`
			ExceptionName string `json:"exceptionName"`
		} `json:"errors"`
	}

	// Pull Request Types

	User struct {
		Name string `json:"name"`
	}

	Reviewer struct {
		User User `json:"user"`
	}

	PullRequestProject struct {
		Key string `json:"key"`
	}

	PullRequestRepository struct {
		Slug    string             `json:"slug"`
		Name    string             `json:"name,omitempty"`
		Project PullRequestProject `json:"project"`
	}

	PullRequestRef struct {
		Id         string                `json:"id"`
		Repository PullRequestRepository `json:"repository"`
	}

	PullRequestResource struct {
		Version     int    `json:"version"`
		Title       string `json:"title,omitempty"`
		Description string `json:"description,omitempty"`
		// FromRef and ToRef should be PullRequestRef but there is interface{}
		// for omitting empty values. encoding/json can't handle empty structs
		// and omit them.
		FromRef   interface{} `json:"fromRef,omitempty"`
		ToRef     interface{} `json:"toRef,omitempty"`
		Reviewers []Reviewer  `json:"reviewers,omitempty"`
	}

	CommentResource struct {
		Text string `json:"text"`
	}

	Commit struct {
		ID        string `json:"id"`
		DisplayID string `json:"displayId"`
		Author    struct {
			Name         string `json:"name"`
			EmailAddress string `json:"emailAddress"`
		} `json:"author"`
		AuthorTimestamp int64 `json:"authorTimestamp"` // in milliseconds since the epoch
		Attributes      struct {
			JiraKeys []string `json:"jira-key"`
		} `json:"attributes"`
	}

	Commits struct {
		Commits []Commit `json:"values"`
	}
)

const (
	stashPageLimit int = 25
)

var (
	httpTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
)

var (
	httpClient *http.Client = &http.Client{Timeout: 10 * time.Second, Transport: httpTransport}
)

func (e errorResponse) Error() string {
	return fmt.Sprintf("%s (%d)", e.Reason, e.StatusCode)
}

func NewClient(userName, password string, baseURL *url.URL) Stash {
	return Client{userName: userName, password: password, baseURL: baseURL}
}

func (client Client) CreateRepository(projectKey, projectSlug string) (Repository, error) {
	slug := fmt.Sprintf(`{"name": "%s", "scmId": "git"}`, projectSlug)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos", client.baseURL.String(), projectKey), bytes.NewBuffer([]byte(slug)))
	if err != nil {
		return Repository{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	req.SetBasicAuth(client.userName, client.password)

	responseCode, data, err := consumeResponse(req)
	if err != nil {
		return Repository{}, err
	}
	if responseCode != http.StatusCreated {
		var reason string = "unknown reason"
		switch {
		case responseCode == http.StatusBadRequest:
			reason = "The repository was not created due to a validation error."
		case responseCode == http.StatusUnauthorized:
			reason = "The currently authenticated user has insufficient permissions to create a repository."
		case responseCode == http.StatusNotFound:
			reason = "The resource was not found. Does the project key exist?"
		case responseCode == http.StatusConflict:
			reason = "A repository with same name already exists."
		}
		return Repository{}, errorResponse{StatusCode: responseCode, Reason: reason}
	}

	var t Repository
	err = json.Unmarshal(data, &t)
	if err != nil {
		return Repository{}, err
	}

	return t, nil
}

func (client Client) RenameRepository(projectKey, projectSlug, newSlug string) error {
	payload := fmt.Sprintf(`{"name":"%s"}`, newSlug)

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf(
			"%s/rest/api/1.0/projects/%s/repos/%s",
			client.baseURL.String(),
			projectKey,
			projectSlug,
		),
		bytes.NewBuffer([]byte(payload)),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	req.SetBasicAuth(client.userName, client.password)

	responseCode, _, err := consumeResponse(req)
	if err != nil {
		return err
	}
	if responseCode != http.StatusCreated {
		var reason string = "unknown reason"
		switch {
		case responseCode == http.StatusBadRequest:
			reason = "The repository was not renamed due to a validation error."
		case responseCode == http.StatusUnauthorized:
			reason = "The currently authenticated user has insufficient permissions to create a repository."
		case responseCode == http.StatusNotFound:
			reason = "The resource was not found. Does the project key exist?"
		case responseCode == http.StatusConflict:
			reason = "A repository with same name already exists."
		}
		return errorResponse{StatusCode: responseCode, Reason: reason}
	}

	return nil
}

// GetRepositories returns a map of repositories indexed by repository URL.
func (client Client) GetRepositories() (map[int]Repository, error) {
	start := 0
	repositories := make(map[int]Repository)
	morePages := true
	for morePages {
		retry := retry.New(3, retry.DefaultBackoffFunc)
		var data []byte
		work := func() error {
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/1.0/repos?start=%d&limit=%d", client.baseURL.String(), start, stashPageLimit), nil)
			if err != nil {
				return err
			}
			Log.Printf("stash.GetRepositories URL %s\n", req.URL)
			req.Header.Set("Accept", "application/json")
			// use credentials if we have them.  If not, the repository must be public.
			if client.userName != "" && client.password != "" {
				req.SetBasicAuth(client.userName, client.password)
			}

			var responseCode int
			responseCode, data, err = consumeResponse(req)
			if err != nil {
				return err
			}
			if responseCode != http.StatusOK {
				var reason string = "unhandled reason"
				switch {
				case responseCode == http.StatusBadRequest:
					reason = "Bad request."
				}
				return errorResponse{StatusCode: responseCode, Reason: reason}
			}
			return nil
		}
		if err := retry.Try(work); err != nil {
			return nil, err
		}

		var r Repositories
		err := json.Unmarshal(data, &r)
		if err != nil {
			return nil, err
		}
		for _, repo := range r.Repository {
			repositories[repo.ID] = repo
		}
		morePages = !r.IsLastPage
		start = r.NextPageStart
	}
	return repositories, nil
}

// GetBranches returns a map of branches indexed by branch display name for the given repository.
func (client Client) GetBranches(projectKey, repositorySlug string) (map[string]Branch, error) {
	start := 0
	branches := make(map[string]Branch)
	morePages := true
	for morePages {
		var data []byte
		retry := retry.New(3, retry.DefaultBackoffFunc)
		workit := func() error {
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/branches?start=%d&limit=%d", client.baseURL.String(), projectKey, repositorySlug, start, stashPageLimit), nil)
			if err != nil {
				return err
			}
			req.Header.Set("Accept", "application/json")
			req.SetBasicAuth(client.userName, client.password)

			var responseCode int
			responseCode, data, err = consumeResponse(req)
			if err != nil {
				return err
			}

			if responseCode != http.StatusOK {
				var reason string = "unhandled reason"
				switch {
				case responseCode == http.StatusNotFound:
					reason = "Not found"
				case responseCode == http.StatusUnauthorized:
					reason = "Unauthorized"
				}
				return errorResponse{StatusCode: responseCode, Reason: reason}
			}
			return nil
		}
		if err := retry.Try(workit); err != nil {
			return nil, err
		}

		var r Branches
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		for _, branch := range r.Branch {
			branches[branch.DisplayID] = branch
		}
		morePages = !r.IsLastPage
		start = r.NextPageStart
	}
	return branches, nil
}

// GetTags returns a map of tags indexed by tag display name for the given repository.
func (client Client) GetTags(projectKey, repositorySlug string) (map[string]Tag, error) {
	start := 0
	tags := make(map[string]Tag)
	morePages := true
	for morePages {
		var data []byte
		retry := retry.New(3, retry.DefaultBackoffFunc)
		work := func() error {
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/tags?start=%d&limit=%d", client.baseURL.String(), projectKey, repositorySlug, start, stashPageLimit), nil)
			if err != nil {
				return err
			}
			req.Header.Set("Accept", "application/json")

			// use credentials if we have them.  If not, the repository must be public.
			if client.userName != "" && client.password != "" {
				req.SetBasicAuth(client.userName, client.password)
			}

			var responseCode int
			responseCode, data, err = consumeResponse(req)
			if err != nil {
				return err
			}

			if responseCode != http.StatusOK {
				var reason string = "unhandled reason"
				switch {
				case responseCode == http.StatusNotFound:
					reason = "Not found"
				case responseCode == http.StatusUnauthorized:
					reason = "Unauthorized"
				}
				return errorResponse{StatusCode: responseCode, Reason: reason}
			}
			return nil
		}
		if err := retry.Try(work); err != nil {
			return nil, err
		}

		var r Tags
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		for _, tag := range r.Tags {
			tags[tag.DisplayID] = tag
		}
		morePages = !r.IsLastPage
		start = r.NextPageStart
	}
	return tags, nil
}

// GetRepository returns a repository representation for the given Stash Project key and repository slug.
func (client Client) GetRepository(projectKey, repositorySlug string) (Repository, error) {
	retry := retry.New(3, retry.DefaultBackoffFunc)

	var r Repository
	work := func() error {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s", client.baseURL.String(), projectKey, repositorySlug), nil)
		if err != nil {
			return err
		}
		Log.Printf("stash.GetRepository %s\n", req.URL)
		req.Header.Set("Accept", "application/json")
		// use credentials if we have them.  If not, the repository must be public.
		if client.userName != "" && client.password != "" {
			req.SetBasicAuth(client.userName, client.password)
		}

		responseCode, data, err := consumeResponse(req)
		if err != nil {
			return err
		}

		if responseCode != http.StatusOK {
			var reason string = "unhandled reason"
			switch {
			case responseCode == http.StatusNotFound:
				reason = "Not found"
			case responseCode == http.StatusUnauthorized:
				reason = "Unauthorized"
			}
			return errorResponse{StatusCode: responseCode, Reason: reason}
		}

		err = json.Unmarshal(data, &r)
		if err != nil {
			return err
		}
		return nil
	}

	return r, retry.Try(work)
}

func (client Client) CreateBranchRestriction(projectKey, repositorySlug, branch, user string) (BranchRestriction, error) {

	branchPermission := BranchPermission{
		Type:   "BRANCH",
		Branch: branch,
		Users:  []string{user},
		Groups: []string{},
	}

	data, err := json.Marshal(branchPermission)
	if err != nil {
		return BranchRestriction{}, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rest/branch-permissions/1.0/projects/%s/repos/%s/restricted", client.baseURL.String(), projectKey, repositorySlug), bytes.NewReader(data))
	if err != nil {
		return BranchRestriction{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	req.SetBasicAuth(client.userName, client.password)

	responseCode, data, err := consumeResponse(req)
	if err != nil {
		return BranchRestriction{}, err
	}
	if responseCode != http.StatusOK {
		var reason string = "unknown reason"
		switch {
		case responseCode == http.StatusBadRequest:
			reason = "The branch restriction was not created due to a validation error."
		case responseCode == http.StatusUnauthorized:
			reason = "The currently authenticated user has insufficient permissions to create a branch restriction."
		case responseCode == http.StatusNotFound:
			reason = "The resource was not found. Does the project key exist? What about the repo?  The user?  The branch?"
		case responseCode == http.StatusConflict:
			reason = "A branch restriction with same name already exists."
		}
		return BranchRestriction{}, errorResponse{StatusCode: responseCode, Reason: reason}
	}

	var t BranchRestriction
	err = json.Unmarshal(data, &t)
	if err != nil {
		return BranchRestriction{}, err
	}

	return t, nil
}

func (client Client) GetBranchRestrictions(projectKey, repositorySlug string) (BranchRestrictions, error) {
	retry := retry.New(3, retry.DefaultBackoffFunc)

	var branchRestrictions BranchRestrictions
	work := func() error {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/branch-permissions/1.0/projects/%s/repos/%s/restricted", client.baseURL.String(), projectKey, repositorySlug), nil)
		if err != nil {
			return err
		}
		Log.Printf("stash.GetBranchRestrictions %s\n", req.URL)
		req.Header.Set("Accept", "application/json")
		req.SetBasicAuth(client.userName, client.password)

		responseCode, data, err := consumeResponse(req)
		if err != nil {
			return err
		}

		if responseCode != http.StatusOK {
			var reason string = "unhandled reason"
			switch {
			case responseCode == http.StatusNotFound:
				reason = "Not found"
			case responseCode == http.StatusUnauthorized:
				reason = "Unauthorized"
			}
			return errorResponse{StatusCode: responseCode, Reason: reason}
		}

		err = json.Unmarshal(data, &branchRestrictions)
		if err != nil {
			return err
		}
		return nil
	}

	return branchRestrictions, retry.Try(work)
}

// GetRepository returns a repository representation for the given Stash Project key and repository slug.
func (client Client) DeleteBranchRestriction(projectKey, repositorySlug string, id int) error {
	retry := retry.New(3, retry.DefaultBackoffFunc)

	work := func() error {
		req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/rest/branch-permissions/1.0/projects/%s/repos/%s/restricted/%d", client.baseURL.String(), projectKey, repositorySlug, id), nil)
		if err != nil {
			return err
		}
		Log.Printf("stash.DeleteBranchRestriction %s\n", req.URL)
		req.Header.Set("Accept", "application/json")
		req.SetBasicAuth(client.userName, client.password)

		responseCode, _, err := consumeResponse(req)
		if err != nil {
			return err
		}

		if responseCode != http.StatusNoContent {
			var reason string = "unhandled reason"
			switch {
			case responseCode == http.StatusNotFound:
				reason = "Not found"
			case responseCode == http.StatusUnauthorized:
				reason = "Unauthorized"
			}
			return errorResponse{StatusCode: responseCode, Reason: reason}
		}

		return nil
	}

	return retry.Try(work)
}

// GetPullRequests returns a list of pull requests for a project / slug.
func (client Client) GetPullRequests(projectKey, projectSlug, state string) ([]PullRequest, error) {
	start := 0
	pullRequests := make([]PullRequest, 0)
	morePages := true
	for morePages {
		retry := retry.New(3, retry.DefaultBackoffFunc)
		var data []byte
		work := func() error {
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests?state=%s&start=%d&limit=%d", client.baseURL.String(), projectKey, projectSlug, state, start, stashPageLimit), nil)
			if err != nil {
				return err
			}
			req.Header.Set("Accept", "application/json")
			req.SetBasicAuth(client.userName, client.password)

			var responseCode int
			responseCode, data, err = consumeResponse(req)
			if err != nil {
				return err
			}
			if responseCode != http.StatusOK {
				var reason string = "unhandled reason"
				switch {
				case responseCode == http.StatusBadRequest:
					reason = "Bad request."
				}
				return errorResponse{StatusCode: responseCode, Reason: reason}
			}
			return nil
		}
		if err := retry.Try(work); err != nil {
			return nil, err
		}

		var r PullRequests
		err := json.Unmarshal(data, &r)
		if err != nil {
			return nil, err
		}
		for _, pr := range r.PullRequests {
			pullRequests = append(pullRequests, pr)
		}
		morePages = !r.IsLastPage
		start = r.NextPageStart
	}
	return pullRequests, nil
}

// GetPullRequest returns a pull request for a project/slug with specified
// identifier.
func (client Client) GetPullRequest(projectKey, projectSlug, identifier string) (PullRequest, error) {
	retry := retry.New(3, retry.DefaultBackoffFunc)
	var data []byte
	work := func() error {
		req, err := http.NewRequest(
			"GET",
			fmt.Sprintf(
				"%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s",
				client.baseURL.String(), projectKey, projectSlug, identifier,
			),
			nil,
		)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		// use credentials if we have them.  If not, the repository must be public.
		if client.userName != "" && client.password != "" {
			req.SetBasicAuth(client.userName, client.password)
		}

		var responseCode int
		responseCode, data, err = consumeResponse(req)
		if err != nil {
			return err
		}
		if responseCode != http.StatusOK {
			var reason string = "unhandled reason"
			switch {
			case responseCode == http.StatusBadRequest:
				reason = "Bad request."
			case responseCode == http.StatusUnauthorized:
				reason = "The currently authenticated user has insufficient permissions to see a pull request."
			case responseCode == http.StatusNotFound:
				reason = "The resource was not found. Does the project key exist?"
			}
			return errorResponse{StatusCode: responseCode, Reason: reason}
		}
		return nil
	}
	if err := retry.Try(work); err != nil {
		return PullRequest{}, err
	}

	var r PullRequest
	err := json.Unmarshal(data, &r)
	if err != nil {
		return PullRequest{}, err
	}

	return r, nil
}

// CreateComment creates a comment for a pull-request.
func (client Client) CreateComment(projectKey, repositorySlug, pullRequest, text string) (Comment, error) {
	resource := CommentResource{
		Text: text,
	}

	reqBody, err := json.Marshal(resource)
	if err != nil {
		return Comment{}, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf(
			"%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s/comments",
			client.baseURL.String(),
			projectKey,
			repositorySlug,
			pullRequest,
		),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return Comment{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	req.SetBasicAuth(client.userName, client.password)

	responseCode, data, err := consumeResponse(req)
	if err != nil {
		return Comment{}, err
	}
	if responseCode != http.StatusCreated {
		var reason string = "unknown reason"
		switch {
		case responseCode == http.StatusBadRequest:
			reason = "The comment was not created due to a validation error."
		case responseCode == http.StatusUnauthorized:
			reason = "The currently authenticated user has insufficient permissions to create a comment."
		case responseCode == http.StatusNotFound:
			reason = "The resource was not found. Does the project key exist?"
		}

		return Comment{}, errorResponse{StatusCode: responseCode, Reason: reason}
	}

	var t Comment
	err = json.Unmarshal(data, &t)
	if err != nil {
		return Comment{}, err
	}

	return t, nil
}

// CreatePullRequest creates a pull request between branches.
func (client Client) CreatePullRequest(projectKey, repositorySlug, title, description, fromRef, toRef string, reviewers []string) (PullRequest, error) {

	var revs []Reviewer
	for _, rev := range reviewers {
		revs = append(revs, Reviewer{
			User: User{Name: rev},
		})
	}

	pullRequestResource := PullRequestResource{
		Title:       title,
		Description: description,
		FromRef: PullRequestRef{
			Id: fromRef,
			Repository: PullRequestRepository{
				Slug: repositorySlug,
				Project: PullRequestProject{
					Key: projectKey,
				},
			},
		},
		ToRef: PullRequestRef{
			Id: toRef,
			Repository: PullRequestRepository{
				Slug: repositorySlug,
				Project: PullRequestProject{
					Key: projectKey,
				},
			},
		},
		Reviewers: revs,
	}

	reqBody, err := json.Marshal(pullRequestResource)
	if err != nil {
		return PullRequest{}, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests", client.baseURL.String(), projectKey, repositorySlug), bytes.NewBuffer(reqBody))
	if err != nil {
		return PullRequest{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	req.SetBasicAuth(client.userName, client.password)

	responseCode, data, err := consumeResponse(req)
	if err != nil {
		return PullRequest{}, err
	}
	if responseCode != http.StatusCreated {
		var reason string = "unknown reason"
		switch {
		case responseCode == http.StatusBadRequest:
			reason = "The pull-request was not created due to a validation error."
		case responseCode == http.StatusUnauthorized:
			reason = "The currently authenticated user has insufficient permissions to create a pull-request."
		case responseCode == http.StatusNotFound:
			reason = "The resource was not found. Does the project key exist?"
		case responseCode == http.StatusConflict:
			reason = "A pull-request with same name already exists."
		}
		return PullRequest{}, errorResponse{StatusCode: responseCode, Reason: reason}
	}

	var t PullRequest
	err = json.Unmarshal(data, &t)
	if err != nil {
		return PullRequest{}, err
	}

	return t, nil
}

// UpdatePullRequest update a pull request.
func (client Client) UpdatePullRequest(projectKey, repositorySlug, identifier string, version int, title, description, toRef string, reviewers []string) (PullRequest, error) {
	var revs []Reviewer
	for _, rev := range reviewers {
		revs = append(revs, Reviewer{
			User: User{Name: rev},
		})
	}

	pullRequestResource := PullRequestResource{
		Version:     version,
		Title:       title,
		Description: description,
		Reviewers:   revs,
	}

	if toRef != "" {
		pullRequestResource.ToRef = PullRequestRef{
			Id: toRef,
			Repository: PullRequestRepository{
				Slug: repositorySlug,
				Project: PullRequestProject{
					Key: projectKey,
				},
			},
		}
	}

	reqBody, err := json.Marshal(pullRequestResource)
	if err != nil {
		return PullRequest{}, err
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf(
			"%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s",
			client.baseURL.String(),
			projectKey,
			repositorySlug,
			identifier,
		),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return PullRequest{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	req.SetBasicAuth(client.userName, client.password)

	responseCode, data, err := consumeResponse(req)
	if err != nil {
		return PullRequest{}, err
	}

	if responseCode != http.StatusOK {
		var reason string = "unknown reason"
		switch {
		case responseCode == http.StatusBadRequest:
			reason = "The pull-request was not updated due to a validation error."
		case responseCode == http.StatusUnauthorized:
			reason = "The currently authenticated user has insufficient permissions to edit a pull-request."
		case responseCode == http.StatusNotFound:
			reason = "The resource was not found. Does the project key exist?"
		case responseCode == http.StatusConflict:
			reason = "The pull-request was not updated due to a conflicts. Does the `from` and new `to` branch are different?"
		}
		return PullRequest{}, errorResponse{StatusCode: responseCode, Reason: reason}
	}

	var t PullRequest
	err = json.Unmarshal(data, &t)
	if err != nil {
		return PullRequest{}, err
	}

	return t, nil
}

func (client Client) DeleteBranch(projectKey, repositorySlug, branchName string) error {
	work := func() error {
		buffer := bytes.NewBufferString((fmt.Sprintf(`{"name":"refs/heads/%s","dryRun":false}`, branchName)))
		req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/rest/branch-utils/1.0/projects/%s/repos/%s/branches", client.baseURL.String(), projectKey, repositorySlug), buffer)
		if err != nil {
			return err
		}
		req.Header.Set("Content-type", "application/json")
		req.SetBasicAuth(client.userName, client.password)

		responseCode, _, err := consumeResponse(req)
		if err != nil {
			return err
		}

		switch responseCode {
		case http.StatusNoContent:
			return nil
		case http.StatusBadRequest:
			return errorResponse{StatusCode: responseCode, Reason: "Bad Requeest"}
		case http.StatusUnauthorized:
			return errorResponse{StatusCode: responseCode, Reason: "Unauthorized"}
		default:
			return errorResponse{StatusCode: responseCode, Reason: "(unhandled reason)"}
		}
	}
	return retry.New(3, retry.DefaultBackoffFunc).Try(work)
}

func (client Client) GetRawFile(repositoryProjectKey, repositorySlug, filePath, branch string) ([]byte, error) {
	retry := retry.New(3, retry.DefaultBackoffFunc)

	var data []byte
	work := func() error {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/projects/%s/repos/%s/browse/%s?at=%s&raw", client.baseURL.String(), strings.ToLower(repositoryProjectKey), strings.ToLower(repositorySlug), filePath, branch), nil)
		if err != nil {
			return err
		}
		Log.Printf("stash.GetRawFile %s\n", req.URL)
		req.SetBasicAuth(client.userName, client.password)

		var responseCode int
		responseCode, data, err = consumeResponse(req)
		if err != nil {
			return err
		}
		if responseCode != http.StatusOK {
			var reason string = "unhandled reason"
			switch {
			case responseCode == http.StatusNotFound:
				reason = "Not found"
			case responseCode == http.StatusUnauthorized:
				reason = "Unauthorized"
			}
			return errorResponse{StatusCode: responseCode, Reason: reason}
		}
		return nil
	}

	return data, retry.Try(work)
}

// GetCommit returns a representation of the given commit hash.
func (client Client) GetCommit(projectKey, repositorySlug, commitHash string) (Commit, error) {
	retry := retry.New(3, retry.DefaultBackoffFunc)

	var data []byte
	work := func() error {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/commits/%s", client.baseURL.String(), projectKey, repositorySlug, commitHash), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")

		if client.userName != "" && client.password != "" {
			req.SetBasicAuth(client.userName, client.password)
		}

		var responseCode int
		responseCode, data, err = consumeResponse(req)
		if err != nil {
			return err
		}

		if responseCode != http.StatusOK {
			var reason string = "unhandled reason"
			switch {
			case responseCode == http.StatusBadRequest:
				reason = "Bad Request"
			case responseCode == http.StatusUnauthorized:
				reason = "Unauthorized"
			case responseCode == http.StatusNotFound:
				reason = "Not found"
			}
			return errorResponse{StatusCode: responseCode, Reason: reason}
		}
		return nil
	}

	if err := retry.Try(work); err != nil {
		return Commit{}, err
	}

	var commit Commit
	err := json.Unmarshal(data, &commit)
	return commit, err
}

// GetCommits returns the commits between two hashes, inclusively.
func (client Client) GetCommits(projectKey, repositorySlug, commitSinceHash string, commitUntilHash string) (Commits, error) {
	retry := retry.New(3, retry.DefaultBackoffFunc)

	var data []byte
	work := func() error {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/commits?since=%s&until=%s&limit=1000", client.baseURL.String(), projectKey, repositorySlug, commitSinceHash, commitUntilHash), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")

		if client.userName != "" && client.password != "" {
			req.SetBasicAuth(client.userName, client.password)
		}

		var responseCode int
		responseCode, data, err = consumeResponse(req)
		if err != nil {
			return err
		}

		if responseCode != http.StatusOK {
			var reason string = "unhandled reason"
			switch {
			case responseCode == http.StatusBadRequest:
				reason = "Bad Request"
			case responseCode == http.StatusUnauthorized:
				reason = "Unauthorized"
			case responseCode == http.StatusNotFound:
				reason = "Not found"
			}
			return errorResponse{StatusCode: responseCode, Reason: reason}
		}
		return nil
	}

	if err := retry.Try(work); err != nil {
		return Commits{}, err
	}

	var commits Commits
	err := json.Unmarshal(data, &commits)
	return commits, err
}

func HasRepository(repositories map[int]Repository, url string) (Repository, bool) {
	for _, repo := range repositories {
		for _, clone := range repo.Links.Clones {
			if clone.HREF == url {
				return repo, true
			}
		}
	}
	return Repository{}, false
}

func IsRepositoryExists(err error) bool {
	if err == nil {
		return false
	}
	if response, ok := err.(errorResponse); ok {
		return response.StatusCode == http.StatusConflict
	}
	return false
}

func IsRepositoryNotFound(err error) bool {
	if err == nil {
		return false
	}
	if response, ok := err.(errorResponse); ok {
		return response.StatusCode == http.StatusNotFound
	}
	return false
}

func consumeResponse(req *http.Request) (int, []byte, error) {
	response, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, err
	}

	defer response.Body.Close()

	if response.StatusCode >= 400 {
		var errResponse stashError
		if err := json.Unmarshal(data, &errResponse); err == nil {
			var messages []string
			for _, e := range errResponse.Errors {
				messages = append(messages, e.Message)
			}
			return response.StatusCode, data, errors.New(strings.Join(messages, " "))
		} else {
			return response.StatusCode, nil, err
		}
	}

	return response.StatusCode, data, nil
}

// SshUrl extracts the SSH-based URL from the repository metadata.
func (repo Repository) SshUrl() string {
	for _, clone := range repo.Links.Clones {
		if clone.Name == "ssh" {
			return clone.HREF
		}
	}
	return ""
}
