package comment

// GitLab merge-request comment poster. Mirrors the GitHub
// implementation: marker-based update-in-place so re-running c3x on
// the same MR edits the existing c3x comment instead of stacking.
//
// We avoid the official `xanzy/go-gitlab` dependency to keep the
// transitive dep footprint flat — the REST surface we need is two
// endpoints (list notes, edit note) plus one (create note). Hand-
// rolling them is ~80 lines and avoids pulling in 30+ MB of unused
// resource clients.

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

// DefaultGitLabBaseURL is gitlab.com's API. Self-hosted GitLab
// instances override via [NewGitLabPoster]'s baseURL parameter.
const DefaultGitLabBaseURL = "https://gitlab.com/api/v4"

// GitLabTarget identifies one MR.
type GitLabTarget struct {
	ProjectID string // either numeric ID or URL-escaped "group/project"
	MR        int
}

// AutoDetectGitLab populates a Target from GitLab CI environment
// variables. GitLab exposes:
//
//	CI_PROJECT_ID        numeric ID of the project (e.g. "278964")
//	CI_MERGE_REQUEST_IID merge-request internal ID (project-scoped)
//	CI_API_V4_URL        base URL of the API (handles self-hosted)
//
// If running outside an MR pipeline, CI_MERGE_REQUEST_IID is empty;
// we return a usage error rather than silently posting nowhere.
func AutoDetectGitLab() (GitLabTarget, string, error) {
	project := os.Getenv("CI_PROJECT_ID")
	if project == "" {
		return GitLabTarget{}, "", errors.New("CI_PROJECT_ID not set (are you running in GitLab CI?)")
	}
	mrStr := os.Getenv("CI_MERGE_REQUEST_IID")
	if mrStr == "" {
		return GitLabTarget{}, "", errors.New("CI_MERGE_REQUEST_IID not set (this pipeline is not on an MR)")
	}
	mr, err := strconv.Atoi(mrStr)
	if err != nil {
		return GitLabTarget{}, "", fmt.Errorf("CI_MERGE_REQUEST_IID=%q is not numeric: %w", mrStr, err)
	}
	baseURL := os.Getenv("CI_API_V4_URL")
	if baseURL == "" {
		baseURL = DefaultGitLabBaseURL
	}
	return GitLabTarget{ProjectID: project, MR: mr}, baseURL, nil
}

// GitLabPoster talks to gitlab.com (or self-hosted). Construct with
// [NewGitLabPoster]; the zero value is not usable.
type GitLabPoster struct {
	client   *http.Client
	baseURL  string
	token    string
	target   GitLabTarget
	marker   string
	recreate bool
}

// NewGitLabPoster takes a personal-access or CI job token plus the
// MR target. baseURL defaults to gitlab.com — pass a self-hosted
// instance's `CI_API_V4_URL` for on-prem.
func NewGitLabPoster(token, baseURL string, target GitLabTarget, opts ...Options) (*GitLabPoster, error) {
	if token == "" {
		return nil, errors.New("gitlab token is empty (set GITLAB_TOKEN, CI_JOB_TOKEN, or pass --token)")
	}
	if target.ProjectID == "" || target.MR == 0 {
		return nil, fmt.Errorf("incomplete target: %+v", target)
	}
	if baseURL == "" {
		baseURL = DefaultGitLabBaseURL
	}
	o := resolveOptions(opts)
	return &GitLabPoster{
		client:   &http.Client{Timeout: DefaultHTTPTimeout},
		baseURL:  strings.TrimRight(baseURL, "/"),
		token:    token,
		target:   target,
		marker:   MarkerFor(o.Tag),
		recreate: o.Recreate,
	}, nil
}

// Post writes (or updates) the c3x comment on the MR. Mirrors the
// GitHub Poster contract.
func (p *GitLabPoster) Post(ctx context.Context, body string) error {
	existingID, err := p.findExisting(ctx)
	if err != nil {
		return fmt.Errorf("looking up existing note: %w", err)
	}
	fullBody := p.marker + "\n" + body
	if existingID == 0 {
		return p.createNote(ctx, fullBody)
	}
	// Recreate: delete the old note and post a fresh one so the latest
	// estimate lands at the bottom of the MR discussion.
	if p.recreate {
		if err := p.deleteNote(ctx, existingID); err != nil {
			return fmt.Errorf("deleting previous note %d: %w", existingID, err)
		}
		return p.createNote(ctx, fullBody)
	}
	return p.editNote(ctx, existingID, fullBody)
}

// gitlabNote is the subset of GitLab's `/notes` response we care
// about.
type gitlabNote struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

// findExisting paginates the MR's notes looking for one containing
// the c3x marker. Returns 0 + nil error when no existing note found.
func (p *GitLabPoster) findExisting(ctx context.Context) (int, error) {
	project := url.PathEscape(p.target.ProjectID)
	page := 1
	for {
		listURL := fmt.Sprintf("%s/projects/%s/merge_requests/%d/notes?per_page=100&page=%d",
			p.baseURL, project, p.target.MR, page)
		resp, err := p.do(ctx, http.MethodGet, listURL, nil)
		if err != nil {
			return 0, err
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("list notes HTTP %d: %s", resp.StatusCode, string(body))
		}
		var notes []gitlabNote
		if err := json.Unmarshal(body, &notes); err != nil {
			return 0, fmt.Errorf("decode notes: %w", err)
		}
		for _, n := range notes {
			if strings.Contains(n.Body, p.marker) {
				return n.ID, nil
			}
		}
		if len(notes) < 100 {
			return 0, nil // last page
		}
		page++
		// Safety bound: GitLab's note list is per-MR; even a noisy MR
		// hits maybe a few hundred notes. 50 pages × 100 = 5000 is
		// more than generous.
		if page > 50 {
			return 0, nil
		}
	}
}

func (p *GitLabPoster) createNote(ctx context.Context, body string) error {
	project := url.PathEscape(p.target.ProjectID)
	endpoint := fmt.Sprintf("%s/projects/%s/merge_requests/%d/notes",
		p.baseURL, project, p.target.MR)
	payload, _ := json.Marshal(map[string]string{"body": body})
	resp, err := p.do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create note HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (p *GitLabPoster) editNote(ctx context.Context, noteID int, body string) error {
	project := url.PathEscape(p.target.ProjectID)
	endpoint := fmt.Sprintf("%s/projects/%s/merge_requests/%d/notes/%d",
		p.baseURL, project, p.target.MR, noteID)
	payload, _ := json.Marshal(map[string]string{"body": body})
	resp, err := p.do(ctx, http.MethodPut, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("edit note %d HTTP %d: %s", noteID, resp.StatusCode, string(raw))
	}
	return nil
}

func (p *GitLabPoster) deleteNote(ctx context.Context, noteID int) error {
	project := url.PathEscape(p.target.ProjectID)
	endpoint := fmt.Sprintf("%s/projects/%s/merge_requests/%d/notes/%d",
		p.baseURL, project, p.target.MR, noteID)
	resp, err := p.do(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete note %d HTTP %d: %s", noteID, resp.StatusCode, string(raw))
	}
	return nil
}

// do is a small wrapper that attaches the auth header and the
// content type for write methods.
func (p *GitLabPoster) do(ctx context.Context, method, urlStr string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", p.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// No per-request context timeout: p.client carries
	// DefaultHTTPTimeout (set in NewGitLabPoster), which bounds the
	// whole exchange including body reads. Layering a context
	// deadline on top would either leak its cancel func or kill the
	// caller's body read.
	return p.client.Do(req)
}
