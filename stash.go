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
)

var Log *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

type (
	Stash interface {
		CreateRepository(projectKey, slug string) (Repository, error)
		RenameRepository(projectKey, slug, newslug string) error
		MoveRepository(projectKey, slug, newslug string) error
		RemoveRepository(projectKey, slug string) error
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
		Version     int    `json:"version,omitempty"`
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
	stashPageLimit        = 25
	stashUnexpectedStatus = "unexpected server status"
)

var (
	httpTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
)

var (
	httpClient *http.Client = &http.Client{
		Timeout:   10 * time.Second,
		Transport: httpTransport,
	}
)

func (e errorResponse) Error() string {
	return fmt.Sprintf("%s (%d)", e.Reason, e.StatusCode)
}

func NewClient(userName, password string, baseURL *url.URL) Stash {
	return Client{userName: userName, password: password, baseURL: baseURL}
}

func (client Client) CreateRepository(
	projectKey, repositorySlug string,
) (Repository, error) {
	data, err := client.request(
		"POST", fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos",
			projectKey,
		),
		struct {
			Name string `json:"name"`
			Scm  string `json:"scmId"`
		}{repositorySlug, "git"},
		http.StatusCreated,
	)
	if err != nil {
		return Repository{}, err
	}

	var response Repository
	err = json.Unmarshal(data, &response)
	if err != nil {
		return Repository{}, err
	}

	return response, nil
}
func (client Client) MoveRepository(projectKey, repositorySlug, newProjectKey string) error {
	payload := struct {
		Project struct {
			Key string `json:"key"`
		} `json:"project"`
	}{}

	payload.Project.Key = newProjectKey

	_, err := client.request(
		"PUT",
		"/rest/api/1.0/projects/%s/repos/%s",
		payload,
		http.StatusCreated,
	)
	if err != nil {
		return err
	}

	return nil
}

func (client Client) RemoveRepository(projectKey, repositorySlug string) error {
	_, err := client.request(
		"DELETE",
		fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s",
			projectKey, repositorySlug,
		),
		nil,
		http.StatusAccepted,
		http.StatusNoContent,
	)
	if err != nil {
		return err
	}

	return nil
}

func (client Client) RenameRepository(projectKey, repositorySlug, newSlug string) error {
	_, err := client.request(
		"PUT",
		fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s",
			projectKey,
			repositorySlug,
		),
		struct {
			Name string `json:"name"`
		}{
			Name: newSlug,
		},
		http.StatusCreated,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetRepositories returns a map of repositories indexed by repository URL.
func (client Client) GetRepositories() (map[int]Repository, error) {
	start := 0
	repositories := make(map[int]Repository)
	morePages := true
	for morePages {
		data, err := client.request(
			"GET",
			fmt.Sprintf(
				"/rest/api/1.0/repos?start=%d&limit=%d",
				start, stashPageLimit,
			),
			nil,
			http.StatusOK,
		)
		if err != nil {
			return nil, err
		}

		var response Repositories
		err = json.Unmarshal(data, &response)
		if err != nil {
			return nil, err
		}

		for _, repo := range response.Repository {
			repositories[repo.ID] = repo
		}

		morePages = !response.IsLastPage
		start = response.NextPageStart
	}

	return repositories, nil
}

// GetBranches returns a map of branches indexed by branch display name for the given repository.
func (client Client) GetBranches(projectKey, repositorySlug string) (map[string]Branch, error) {
	start := 0
	branches := make(map[string]Branch)
	morePages := true
	for morePages {
		data, err := client.request(
			"GET",
			fmt.Sprintf(
				"/rest/api/1.0/projects/%s/repos/%s/branches?start=%d&limit=%d",
				projectKey, repositorySlug, start, stashPageLimit,
			),
			nil,
			http.StatusOK,
		)
		if err != nil {
			return nil, err
		}

		var response Branches
		if err := json.Unmarshal(data, &response); err != nil {
			return nil, err
		}

		for _, branch := range response.Branch {
			branches[branch.DisplayID] = branch
		}
		morePages = !response.IsLastPage
		start = response.NextPageStart
	}
	return branches, nil
}

// GetTags returns a map of tags indexed by tag display name for the given repository.
func (client Client) GetTags(projectKey, repositorySlug string) (map[string]Tag, error) {
	start := 0
	tags := make(map[string]Tag)
	morePages := true
	for morePages {
		data, err := client.request(
			"GET",
			fmt.Sprintf(
				"/rest/api/1.0/projects/%s/repos/%s/tags?start=%d&limit=%d",
				projectKey, repositorySlug, start, stashPageLimit,
			),
			nil,
			http.StatusOK,
		)
		if err != nil {
			return nil, err
		}

		var response Tags
		if err := json.Unmarshal(data, &response); err != nil {
			return nil, err
		}

		for _, tag := range response.Tags {
			tags[tag.DisplayID] = tag
		}

		morePages = !response.IsLastPage
		start = response.NextPageStart
	}

	return tags, nil
}

// GetRepository returns a repository representation for the given Stash Project key and repository slug.
func (client Client) GetRepository(
	projectKey, repositorySlug string,
) (Repository, error) {
	data, err := client.request(
		"GET",
		fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s",
			projectKey, repositorySlug,
		),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return Repository{}, err
	}

	var response Repository
	err = json.Unmarshal(data, &response)
	if err != nil {
		return Repository{}, err
	}

	return response, nil
}

func (client Client) CreateBranchRestriction(
	projectKey, repositorySlug, branch, user string,
) (BranchRestriction, error) {
	payload := BranchPermission{
		Type:   "BRANCH",
		Branch: branch,
		Users:  []string{user},
		Groups: []string{},
	}

	data, err := client.request(
		"POST", fmt.Sprintf(
			"/rest/branch-permissions/1.0/projects/%s/repos/%s/restricted",
			projectKey, repositorySlug,
		),
		payload,
		http.StatusOK,
	)
	if err != nil {
		return BranchRestriction{}, err
	}

	var response BranchRestriction
	err = json.Unmarshal(data, &response)
	if err != nil {
		return BranchRestriction{}, err
	}

	return response, nil
}

func (client Client) GetBranchRestrictions(
	projectKey, repositorySlug string,
) (BranchRestrictions, error) {

	data, err := client.request(
		"GET", fmt.Sprintf(
			"/rest/branch-permissions/1.0/projects/%s/repos/%s/restricted",
			projectKey, repositorySlug,
		),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return BranchRestrictions{}, err

	}

	var branchRestrictions BranchRestrictions
	err = json.Unmarshal(data, &branchRestrictions)
	if err != nil {
		return BranchRestrictions{}, err
	}

	return branchRestrictions, nil
}

// GetRepository returns a repository representation for the given Stash Project key and repository slug.
func (client Client) DeleteBranchRestriction(
	projectKey, repositorySlug string, id int,
) error {

	_, err := client.request(
		"DELETE",
		fmt.Sprintf(
			"/rest/branch-permissions/1.0/projects/%s/repos/%s/restricted/%d",
			projectKey, repositorySlug, id,
		),
		nil,
		http.StatusNoContent,
	)
	if err != nil {
		return err

	}

	return nil
}

// GetPullRequests returns a list of pull requests for a project / slug.
func (client Client) GetPullRequests(
	projectKey, repositorySlug, state string,
) ([]PullRequest, error) {
	start := 0
	pullRequests := make([]PullRequest, 0)
	morePages := true
	for morePages {
		data, err := client.request(
			"GET",
			fmt.Sprintf(
				"/rest/api/1.0/projects/%s/repos/%s/pull-requests?state=%s&start=%d&limit=%d",
				projectKey, repositorySlug, state, start, stashPageLimit,
			),
			nil,
			http.StatusOK,
		)
		if err != nil {
			return nil, err
		}

		var response PullRequests
		err = json.Unmarshal(data, &response)
		if err != nil {
			return nil, err
		}

		for _, pr := range response.PullRequests {
			pullRequests = append(pullRequests, pr)
		}

		morePages = !response.IsLastPage
		start = response.NextPageStart
	}

	return pullRequests, nil
}

// GetPullRequest returns a pull request for a project/slug with specified
// identifier.
func (client Client) GetPullRequest(
	projectKey, repositorySlug, identifier string,
) (PullRequest, error) {
	data, err := client.request(
		"GET",
		fmt.Sprintf(
			"rest/api/1.0/projects/%s/repos/%s/pull-requests/%s",
			projectKey, repositorySlug, identifier,
		),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return PullRequest{}, err
	}

	var response PullRequest
	err = json.Unmarshal(data, &response)
	if err != nil {
		return PullRequest{}, err
	}

	return response, nil
}

// CreateComment creates a comment for a pull-request.
func (client Client) CreateComment(
	projectKey, repositorySlug, pullRequest, text string,
) (Comment, error) {
	payload := CommentResource{
		Text: text,
	}

	data, err := client.request(
		"POST",
		fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s/comments",
			projectKey,
			repositorySlug,
			pullRequest,
		),
		payload,
		http.StatusCreated,
	)
	if err != nil {
		return Comment{}, err
	}

	var response Comment
	err = json.Unmarshal(data, &response)
	if err != nil {
		return Comment{}, err
	}

	return response, nil
}

// CreatePullRequest creates a pull request between branches.
func (client Client) CreatePullRequest(
	projectKey, repositorySlug, title, description, fromRef, toRef string,
	reviewers []string,
) (PullRequest, error) {
	var users []Reviewer
	for _, rev := range reviewers {
		users = append(users, Reviewer{
			User: User{Name: rev},
		})
	}

	payload := PullRequestResource{
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
		Reviewers: users,
	}

	data, err := client.request(
		"POST", fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s/pull-requests",
			projectKey, repositorySlug,
		),
		payload,
		http.StatusCreated,
	)
	if err != nil {
		return PullRequest{}, err
	}

	var response PullRequest
	err = json.Unmarshal(data, &response)
	if err != nil {
		return PullRequest{}, err
	}

	return response, nil
}

func (client *Client) request(
	method, url string, payload interface{}, statuses ...int,
) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(
		method,
		strings.TrimRight(client.baseURL.String(), "/")+url,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-type", "application/json")

	if client.userName != "" && client.password != "" {
		request.SetBasicAuth(client.userName, client.password)
	}

	status, data, err := consumeResponse(request)
	if err != nil {
		return nil, err
	}

	for _, expectedStatus := range statuses {
		if status == expectedStatus {
			return data, nil
		}
	}

	return nil, errorResponse{
		StatusCode: status,
		Reason:     stashUnexpectedStatus,
	}
}

// UpdatePullRequest update a pull request.
func (client Client) UpdatePullRequest(
	projectKey, repositorySlug, identifier string,
	version int,
	title, description, toRef string,
	reviewers []string,
) (PullRequest, error) {
	var users []Reviewer
	for _, rev := range reviewers {
		users = append(users, Reviewer{
			User: User{Name: rev},
		})
	}

	payload := PullRequestResource{
		Version:     version,
		Title:       title,
		Description: description,
		Reviewers:   users,
	}

	if toRef != "" {
		payload.ToRef = PullRequestRef{
			Id: toRef,
			Repository: PullRequestRepository{
				Slug: repositorySlug,
				Project: PullRequestProject{
					Key: projectKey,
				},
			},
		}
	}

	data, err := client.request(
		"PUT",
		fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s",
			projectKey,
			repositorySlug,
			identifier,
		),
		payload,
		http.StatusOK,
	)
	if err != nil {
		return PullRequest{}, err
	}

	var response PullRequest
	err = json.Unmarshal(data, &response)
	if err != nil {
		return PullRequest{}, err
	}

	return response, nil
}

func (client Client) DeleteBranch(
	projectKey, repositorySlug, branchName string,
) error {
	_, err := client.request(
		"DELETE",
		fmt.Sprintf(
			"/rest/branch-utils/1.0/projects/%s/repos/%s/branches",
			projectKey, repositorySlug,
		),
		struct {
			Name   string `json:"name"`
			DryRun bool   `json:"dryRun"`
		}{"refs/heads/" + branchName, false},
		http.StatusNoContent,
	)
	if err != nil {
		return err
	}

	return nil
}

func (client Client) GetRawFile(
	repositoryProjectKey, repositorySlug, filePath, branch string,
) ([]byte, error) {
	return client.request(
		"GET",
		fmt.Sprintf(
			"/projects/%s/repos/%s/browse/%s?at=%s&raw",
			strings.ToLower(repositoryProjectKey),
			strings.ToLower(repositorySlug),
			filePath, branch,
		),
		nil,
		http.StatusOK,
	)
}

// GetCommit returns a representation of the given commit hash.
func (client Client) GetCommit(
	projectKey, repositorySlug, commitHash string,
) (Commit, error) {
	data, err := client.request(
		"GET", fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s/commits/%s",
			projectKey, repositorySlug, commitHash,
		),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return Commit{}, err
	}

	var commit Commit
	err = json.Unmarshal(data, &commit)
	return commit, err
}

// GetCommits returns the commits between two hashes, inclusively.
func (client Client) GetCommits(
	projectKey, repositorySlug, commitSinceHash, commitUntilHash string,
) (Commits, error) {
	data, err := client.request(
		"GET",
		fmt.Sprintf(
			"/rest/api/1.0/projects/%s/repos/%s/commits?since=%s&until=%s&limit=1000",
			projectKey, repositorySlug, commitSinceHash, commitUntilHash,
		),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return Commits{}, err
	}

	var commits Commits
	err = json.Unmarshal(data, &commits)
	if err != nil {
		return Commits{}, err
	}

	return commits, nil
}

func HasRepository(
	repositories map[int]Repository,
	url string,
) (Repository, bool) {
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
