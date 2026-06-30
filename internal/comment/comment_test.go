package comment_test

// Tests for the GitHub PR comment poster. The HTTP path is exercised
// against a stub server (httptest) so we never touch the real
// api.github.com; the env-driven AutoDetect and marker-recognition
// logic are pure-Go and tested directly.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/comment"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/google/go-github/v84/github"
	"github.com/shopspring/decimal"
)

func TestAutoDetectFromGitHubActionsEnv(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "acme/widgets")
	t.Setenv("GITHUB_REF", "refs/pull/42/merge")
	got, err := comment.AutoDetect()
	if err != nil {
		t.Fatalf("AutoDetect: %v", err)
	}
	if got.Owner != "acme" || got.Repo != "widgets" || got.PR != 42 {
		t.Errorf("AutoDetect = %+v, want acme/widgets#42", got)
	}
}

func TestAutoDetectRejectsMissingEnv(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "")
	if _, err := comment.AutoDetect(); err == nil {
		t.Error("expected error when GITHUB_REPOSITORY is empty")
	}
}

func TestAutoDetectRejectsMalformedRepo(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "no-slash")
	t.Setenv("GITHUB_REF", "refs/pull/1/merge")
	if _, err := comment.AutoDetect(); err == nil {
		t.Error("expected error for non-owner/repo GITHUB_REPOSITORY")
	}
}

func TestAutoDetectRejectsNonPullRequestRef(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "acme/widgets")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	if _, err := comment.AutoDetect(); err == nil {
		t.Error("expected error when GITHUB_REF is a branch ref")
	}
}

func TestNewGitHubPosterRejectsEmptyToken(t *testing.T) {
	if _, err := comment.NewGitHubPoster("", comment.Target{Owner: "a", Repo: "b", PR: 1}); err == nil {
		t.Error("expected error on empty token")
	}
}

func TestNewGitHubPosterRejectsIncompleteTarget(t *testing.T) {
	if _, err := comment.NewGitHubPoster("tok", comment.Target{}); err == nil {
		t.Error("expected error on zero Target")
	}
}

// TestPostCreatesCommentWhenNoneExists drives the full HTTP path
// against an httptest server impersonating the GitHub REST API. The
// happy path: list returns empty → POST creates a new comment.
func TestPostCreatesCommentWhenNoneExists(t *testing.T) {
	var createdBody string
	var posts, gets int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			gets++
			_, _ = io.WriteString(w, "[]")
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			posts++
			var ic github.IssueComment
			_ = json.NewDecoder(r.Body).Decode(&ic)
			if ic.Body != nil {
				createdBody = *ic.Body
			}
			id := int64(99)
			_ = json.NewEncoder(w).Encode(&github.IssueComment{ID: &id, Body: ic.Body})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	poster := newPosterWithBaseURL(t, srv.URL)
	if err := poster.Post(context.Background(), "hello"); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if gets != 1 {
		t.Errorf("expected 1 GET, got %d", gets)
	}
	if posts != 1 {
		t.Errorf("expected 1 POST, got %d", posts)
	}
	if !strings.Contains(createdBody, comment.Marker) {
		t.Errorf("created body missing marker: %q", createdBody)
	}
	if !strings.Contains(createdBody, "hello") {
		t.Errorf("created body missing payload: %q", createdBody)
	}
}

// TestPostUpdatesExistingComment confirms the marker-based "find +
// edit" path: list returns a comment containing the marker → PATCH
// updates it in place rather than POSTing a duplicate.
func TestPostUpdatesExistingComment(t *testing.T) {
	var patched bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			existing := comment.Marker + "\n(stale)"
			id := int64(7)
			_ = json.NewEncoder(w).Encode([]github.IssueComment{{ID: &id, Body: &existing}})
		case r.Method == http.MethodPatch:
			patched = true
			var ic github.IssueComment
			_ = json.NewDecoder(r.Body).Decode(&ic)
			_ = json.NewEncoder(w).Encode(&ic)
		case r.Method == http.MethodPost:
			t.Errorf("Post created a new comment; should have edited the marked existing one")
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	poster := newPosterWithBaseURL(t, srv.URL)
	if err := poster.Post(context.Background(), "fresh body"); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if !patched {
		t.Error("expected PATCH for marker-marked existing comment, none observed")
	}
}

// newPosterWithBaseURL constructs a GitHubPoster pointing at a
// stub server instead of api.github.com.
func newPosterWithBaseURL(t *testing.T, baseURL string) *comment.GitHubPoster {
	t.Helper()
	p, err := comment.NewGitHubPoster("test-token", comment.Target{Owner: "acme", Repo: "widgets", PR: 1})
	if err != nil {
		t.Fatalf("NewGitHubPoster: %v", err)
	}
	// The poster embeds an unexported github.Client; rebuild it
	// against the test server via the exported helper.
	if err := comment.SetClientBaseURL(p, baseURL); err != nil {
		t.Fatalf("SetClientBaseURL: %v", err)
	}
	return p
}

// TestFormatCommentDiffRendersDelta verifies the baseline path renders a
// markdown cost diff (the "+$X this PR" body) rather than an absolute
// estimate, including the signed total-delta line.
func TestFormatCommentDiffRendersDelta(t *testing.T) {
	base := domain.Estimate{
		ProjectTotal: decimal.RequireFromString("894.32"),
		Currency:     domain.CurrencyUSD,
		Costs: []domain.Cost{{
			Resource:        domain.Reference{Kind: "aws_instance", Name: "web"},
			MonthlySubtotal: decimal.RequireFromString("894.32"),
			Currency:        domain.CurrencyUSD,
		}},
	}
	current := domain.Estimate{
		ProjectTotal: decimal.RequireFromString("1038.32"),
		Currency:     domain.CurrencyUSD,
		Costs: []domain.Cost{{
			Resource:        domain.Reference{Kind: "aws_instance", Name: "web"},
			MonthlySubtotal: decimal.RequireFromString("1038.32"),
			Currency:        domain.CurrencyUSD,
		}},
	}

	body, err := comment.FormatCommentDiff(domain.ComputeDiff(base, current))
	if err != nil {
		t.Fatalf("FormatCommentDiff: %v", err)
	}
	for _, want := range []string{"c3x diff", "894.32", "1038.32", "144"} {
		if !strings.Contains(body, want) {
			t.Errorf("diff comment body missing %q\n---\n%s", want, body)
		}
	}
}
