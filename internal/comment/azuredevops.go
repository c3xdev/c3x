package comment

// Azure DevOps PR comment poster. Marker-based update-in-place like
// the other forges.
//
// Azure DevOps REST API (api-version=7.1):
//
//   GET    /{org}/{project}/_apis/git/repositories/{repo}/pullRequests/{prId}/threads
//   POST   /{org}/{project}/_apis/git/repositories/{repo}/pullRequests/{prId}/threads
//   PATCH  /{org}/{project}/_apis/git/repositories/{repo}/pullRequests/{prId}/threads/{threadId}/comments/{commentId}
//
// Azure DevOps's model is a tiny bit richer than GitHub/GitLab —
// comments are children of "threads". Each c3x post creates one
// thread containing one comment. On re-run we identify the existing
// thread by its first comment's marker, then PATCH that comment.
//
// Auth: a personal access token sent via HTTP Basic with an empty
// username (the Azure DevOps convention).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	// DefaultAzureDevOpsBaseURL is dev.azure.com. Older azure.com /
	// visualstudio.com hosts also work; users override via baseURL.
	DefaultAzureDevOpsBaseURL = "https://dev.azure.com"

	// apiVersion is pinned to a stable REST version so a server-side
	// upgrade doesn't silently change response shapes under us.
	azureDevOpsAPIVersion = "7.1"
)

// AzureDevOpsTarget identifies one PR.
type AzureDevOpsTarget struct {
	Org     string
	Project string
	Repo    string
	PR      int
}

// AutoDetectAzureDevOps populates a Target from Azure Pipelines
// environment variables:
//
//	SYSTEM_TEAMFOUNDATIONCOLLECTIONURI    e.g. "https://dev.azure.com/acme/"
//	SYSTEM_TEAMPROJECT                    project name
//	BUILD_REPOSITORY_NAME                 repo name
//	SYSTEM_PULLREQUEST_PULLREQUESTID      numeric PR ID
//
// All four are required; missing values surface a usage error.
func AutoDetectAzureDevOps() (AzureDevOpsTarget, string, error) {
	collectionURL := strings.TrimRight(os.Getenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI"), "/")
	if collectionURL == "" {
		return AzureDevOpsTarget{}, "", errors.New("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI not set (are you running in Azure Pipelines?)")
	}
	// Collection URL ends with the org name. Strip dev.azure.com/...
	org := collectionURL[strings.LastIndex(collectionURL, "/")+1:]
	if org == "" {
		return AzureDevOpsTarget{}, "", fmt.Errorf("could not extract org from SYSTEM_TEAMFOUNDATIONCOLLECTIONURI=%q", collectionURL)
	}
	project := os.Getenv("SYSTEM_TEAMPROJECT")
	if project == "" {
		return AzureDevOpsTarget{}, "", errors.New("SYSTEM_TEAMPROJECT not set")
	}
	repo := os.Getenv("BUILD_REPOSITORY_NAME")
	if repo == "" {
		return AzureDevOpsTarget{}, "", errors.New("BUILD_REPOSITORY_NAME not set")
	}
	prStr := os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID")
	if prStr == "" {
		return AzureDevOpsTarget{}, "", errors.New("SYSTEM_PULLREQUEST_PULLREQUESTID not set (this pipeline is not on a PR)")
	}
	pr, err := strconv.Atoi(prStr)
	if err != nil {
		return AzureDevOpsTarget{}, "", fmt.Errorf("SYSTEM_PULLREQUEST_PULLREQUESTID=%q is not numeric: %w", prStr, err)
	}
	// Derive base URL: everything before the org segment is the
	// collection URL, which is also the API host.
	baseURL := collectionURL[:strings.LastIndex(collectionURL, "/"+org)]
	if baseURL == "" {
		baseURL = DefaultAzureDevOpsBaseURL
	}
	return AzureDevOpsTarget{Org: org, Project: project, Repo: repo, PR: pr}, baseURL, nil
}

// AzureDevOpsPoster talks to the Azure DevOps REST API.
type AzureDevOpsPoster struct {
	client   *http.Client
	baseURL  string
	token    string
	target   AzureDevOpsTarget
	marker   string
	recreate bool
}

// NewAzureDevOpsPoster takes a personal access token + the target.
// baseURL defaults to dev.azure.com.
func NewAzureDevOpsPoster(token, baseURL string, target AzureDevOpsTarget, opts ...Options) (*AzureDevOpsPoster, error) {
	if token == "" {
		return nil, errors.New("azure devops PAT is empty (set AZURE_DEVOPS_TOKEN or SYSTEM_ACCESSTOKEN, or pass --token)")
	}
	if target.Org == "" || target.Project == "" || target.Repo == "" || target.PR == 0 {
		return nil, fmt.Errorf("incomplete target: %+v", target)
	}
	if baseURL == "" {
		baseURL = DefaultAzureDevOpsBaseURL
	}
	o := resolveOptions(opts)
	return &AzureDevOpsPoster{
		client:   &http.Client{Timeout: DefaultHTTPTimeout},
		baseURL:  strings.TrimRight(baseURL, "/"),
		token:    token,
		target:   target,
		marker:   MarkerFor(o.Tag),
		recreate: o.Recreate,
	}, nil
}

// adoThread / adoComment mirror the response shapes we read.
type adoThread struct {
	ID       int          `json:"id"`
	Comments []adoComment `json:"comments"`
}

type adoComment struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

type adoThreadList struct {
	Value []adoThread `json:"value"`
}

// Post writes (or updates) the c3x thread on the PR.
func (p *AzureDevOpsPoster) Post(ctx context.Context, body string) error {
	threadID, commentID, err := p.findExisting(ctx)
	if err != nil {
		return fmt.Errorf("looking up existing thread: %w", err)
	}
	fullBody := p.marker + "\n" + body
	if threadID == 0 {
		return p.createThread(ctx, fullBody)
	}
	// Recreate: delete the old thread's root comment (which removes the
	// thread) and post a fresh thread so the latest estimate lands at
	// the bottom of the PR.
	if p.recreate {
		if err := p.deleteComment(ctx, threadID, commentID); err != nil {
			return fmt.Errorf("deleting previous comment %d: %w", commentID, err)
		}
		return p.createThread(ctx, fullBody)
	}
	return p.editComment(ctx, threadID, commentID, fullBody)
}

func (p *AzureDevOpsPoster) findExisting(ctx context.Context) (int, int, error) {
	endpoint := p.threadsURL("")
	resp, err := p.do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, 0, err
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("list threads HTTP %d: %s", resp.StatusCode, string(body))
	}
	var list adoThreadList
	if err := json.Unmarshal(body, &list); err != nil {
		return 0, 0, fmt.Errorf("decode threads: %w", err)
	}
	for _, th := range list.Value {
		for _, c := range th.Comments {
			if strings.Contains(c.Content, p.marker) {
				return th.ID, c.ID, nil
			}
		}
	}
	return 0, 0, nil
}

func (p *AzureDevOpsPoster) createThread(ctx context.Context, body string) error {
	endpoint := p.threadsURL("")
	payload, _ := json.Marshal(map[string]any{
		"comments": []map[string]any{
			{"parentCommentId": 0, "content": body, "commentType": 1},
		},
		"status": 1, // active
	})
	resp, err := p.do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create thread HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (p *AzureDevOpsPoster) editComment(ctx context.Context, threadID, commentID int, body string) error {
	endpoint := p.threadsURL(fmt.Sprintf("/%d/comments/%d", threadID, commentID))
	payload, _ := json.Marshal(map[string]any{
		"content":     body,
		"commentType": 1,
	})
	resp, err := p.do(ctx, http.MethodPatch, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("edit comment %d HTTP %d: %s", commentID, resp.StatusCode, string(raw))
	}
	return nil
}

func (p *AzureDevOpsPoster) deleteComment(ctx context.Context, threadID, commentID int) error {
	endpoint := p.threadsURL(fmt.Sprintf("/%d/comments/%d", threadID, commentID))
	resp, err := p.do(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete comment %d HTTP %d: %s", commentID, resp.StatusCode, string(raw))
	}
	return nil
}

// threadsURL builds the /threads[<sub>]?api-version=... URL.
func (p *AzureDevOpsPoster) threadsURL(sub string) string {
	return fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/pullRequests/%d/threads%s?api-version=%s",
		p.baseURL,
		url.PathEscape(p.target.Org),
		url.PathEscape(p.target.Project),
		url.PathEscape(p.target.Repo),
		p.target.PR,
		sub,
		azureDevOpsAPIVersion,
	)
}

func (p *AzureDevOpsPoster) do(ctx context.Context, method, urlStr string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	// Azure DevOps PAT is sent via Basic auth with an empty username.
	req.SetBasicAuth("", p.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return p.client.Do(req)
}
