package comment

// Bitbucket Cloud PR comment poster. Marker-based update-in-place
// like the GitHub and GitLab implementations.
//
// Bitbucket Cloud's REST API v2.0:
//
//   GET    /2.0/repositories/{workspace}/{repo}/pullrequests/{id}/comments
//   POST   /2.0/repositories/{workspace}/{repo}/pullrequests/{id}/comments
//   PUT    /2.0/repositories/{workspace}/{repo}/pullrequests/{id}/comments/{cid}
//
// Auth: HTTP Basic with username + an app password. We deliberately
// don't support OAuth here — the CLI's primary use case is CI, and
// every CI provider that integrates with Bitbucket already exposes
// an app password via env.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// DefaultBitbucketBaseURL is the Bitbucket Cloud API. Self-hosted
// Bitbucket Server (now Data Center) uses a different REST shape
// and is intentionally out of scope here.
const DefaultBitbucketBaseURL = "https://api.bitbucket.org/2.0"

// BitbucketTarget identifies one pull request.
type BitbucketTarget struct {
	Workspace string
	Repo      string
	PR        int
}

// AutoDetectBitbucket pulls workspace/repo/PR from the Bitbucket
// Pipelines environment. Bitbucket exposes:
//
//	BITBUCKET_REPO_FULL_NAME   "workspace/repo"
//	BITBUCKET_PR_ID            numeric pull request ID
//
// Outside a PR pipeline (e.g. a branch build) BITBUCKET_PR_ID is
// absent; we surface the error rather than posting to nowhere.
func AutoDetectBitbucket() (BitbucketTarget, error) {
	full := os.Getenv("BITBUCKET_REPO_FULL_NAME")
	if full == "" {
		return BitbucketTarget{}, errors.New("BITBUCKET_REPO_FULL_NAME not set (are you running in Bitbucket Pipelines?)")
	}
	workspace, repo, ok := strings.Cut(full, "/")
	if !ok || workspace == "" || repo == "" {
		return BitbucketTarget{}, fmt.Errorf("BITBUCKET_REPO_FULL_NAME=%q is not workspace/repo", full)
	}
	prStr := os.Getenv("BITBUCKET_PR_ID")
	if prStr == "" {
		return BitbucketTarget{}, errors.New("BITBUCKET_PR_ID not set (this pipeline is not on a PR)")
	}
	pr, err := strconv.Atoi(prStr)
	if err != nil {
		return BitbucketTarget{}, fmt.Errorf("BITBUCKET_PR_ID=%q is not numeric: %w", prStr, err)
	}
	return BitbucketTarget{Workspace: workspace, Repo: repo, PR: pr}, nil
}

// BitbucketPoster talks to api.bitbucket.org. Construct with
// [NewBitbucketPoster]; the zero value is not usable.
type BitbucketPoster struct {
	client   *http.Client
	baseURL  string
	username string
	password string
	target   BitbucketTarget
	marker   string
	recreate bool
}

// NewBitbucketPoster takes the username (Bitbucket workspace user
// or app-password user) plus the app password, and the PR target.
func NewBitbucketPoster(username, password, baseURL string, target BitbucketTarget, opts ...Options) (*BitbucketPoster, error) {
	if username == "" {
		return nil, errors.New("bitbucket username is empty (set BITBUCKET_USERNAME or pass --user)")
	}
	if password == "" {
		return nil, errors.New("bitbucket app password is empty (set BITBUCKET_APP_PASSWORD or pass --token)")
	}
	if target.Workspace == "" || target.Repo == "" || target.PR == 0 {
		return nil, fmt.Errorf("incomplete target: %+v", target)
	}
	if baseURL == "" {
		baseURL = DefaultBitbucketBaseURL
	}
	o := resolveOptions(opts)
	return &BitbucketPoster{
		client:   &http.Client{Timeout: DefaultHTTPTimeout},
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		target:   target,
		marker:   MarkerFor(o.Tag),
		recreate: o.Recreate,
	}, nil
}

// Post writes (or updates) the c3x comment on the PR.
func (p *BitbucketPoster) Post(ctx context.Context, body string) error {
	existingID, err := p.findExisting(ctx)
	if err != nil {
		return fmt.Errorf("looking up existing comment: %w", err)
	}
	fullBody := p.marker + "\n" + body
	if existingID == 0 {
		return p.createComment(ctx, fullBody)
	}
	// Recreate: delete the old comment and post a fresh one so the
	// latest estimate lands at the bottom of the PR.
	if p.recreate {
		if err := p.deleteComment(ctx, existingID); err != nil {
			return fmt.Errorf("deleting previous comment %d: %w", existingID, err)
		}
		return p.createComment(ctx, fullBody)
	}
	return p.editComment(ctx, existingID, fullBody)
}

// bitbucketComment is the subset of the API response we read.
type bitbucketComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
}

type bitbucketCommentList struct {
	Values []bitbucketComment `json:"values"`
	Next   string             `json:"next"`
}

func (p *BitbucketPoster) findExisting(ctx context.Context) (int, error) {
	next := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments?pagelen=100",
		p.baseURL, p.target.Workspace, p.target.Repo, p.target.PR)
	for i := 0; i < 50 && next != ""; i++ { // 50-page safety bound
		resp, err := p.do(ctx, http.MethodGet, next, nil)
		if err != nil {
			return 0, err
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("list comments HTTP %d: %s", resp.StatusCode, string(body))
		}
		var page bitbucketCommentList
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, fmt.Errorf("decode comments: %w", err)
		}
		for _, c := range page.Values {
			if strings.Contains(c.Content.Raw, p.marker) {
				return c.ID, nil
			}
		}
		next = page.Next
	}
	return 0, nil
}

func (p *BitbucketPoster) createComment(ctx context.Context, body string) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments",
		p.baseURL, p.target.Workspace, p.target.Repo, p.target.PR)
	payload, _ := json.Marshal(map[string]any{
		"content": map[string]string{"raw": body},
	})
	resp, err := p.do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create comment HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (p *BitbucketPoster) editComment(ctx context.Context, id int, body string) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments/%d",
		p.baseURL, p.target.Workspace, p.target.Repo, p.target.PR, id)
	payload, _ := json.Marshal(map[string]any{
		"content": map[string]string{"raw": body},
	})
	resp, err := p.do(ctx, http.MethodPut, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("edit comment %d HTTP %d: %s", id, resp.StatusCode, string(raw))
	}
	return nil
}

func (p *BitbucketPoster) deleteComment(ctx context.Context, id int) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments/%d",
		p.baseURL, p.target.Workspace, p.target.Repo, p.target.PR, id)
	resp, err := p.do(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete comment %d HTTP %d: %s", id, resp.StatusCode, string(raw))
	}
	return nil
}

func (p *BitbucketPoster) do(ctx context.Context, method, urlStr string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(p.username, p.password)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return p.client.Do(req)
}
