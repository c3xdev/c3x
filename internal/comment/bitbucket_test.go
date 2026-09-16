package comment_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/comment"
)

func TestAutoDetectBitbucketFromPipelinesEnv(t *testing.T) {
	t.Setenv("BITBUCKET_REPO_FULL_NAME", "acme/widgets")
	t.Setenv("BITBUCKET_PR_ID", "55")
	target, err := comment.AutoDetectBitbucket()
	if err != nil {
		t.Fatalf("AutoDetectBitbucket: %v", err)
	}
	if target.Workspace != "acme" || target.Repo != "widgets" || target.PR != 55 {
		t.Errorf("target = %+v", target)
	}
}

func TestAutoDetectBitbucketRejectsBranchPipeline(t *testing.T) {
	t.Setenv("BITBUCKET_REPO_FULL_NAME", "acme/widgets")
	t.Setenv("BITBUCKET_PR_ID", "")
	if _, err := comment.AutoDetectBitbucket(); err == nil {
		t.Error("expected error when not on a PR pipeline")
	}
}

func TestBitbucketPosterRequiresAllFields(t *testing.T) {
	if _, err := comment.NewBitbucketPoster("", "pw", "", comment.BitbucketTarget{Workspace: "w", Repo: "r", PR: 1}); err == nil {
		t.Error("expected error on empty user")
	}
	if _, err := comment.NewBitbucketPoster("u", "", "", comment.BitbucketTarget{Workspace: "w", Repo: "r", PR: 1}); err == nil {
		t.Error("expected error on empty password")
	}
	if _, err := comment.NewBitbucketPoster("u", "p", "", comment.BitbucketTarget{}); err == nil {
		t.Error("expected error on zero target")
	}
}

func TestBitbucketPostCreatesWhenNoneExists(t *testing.T) {
	var posts int
	var createdBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			_, _ = io.WriteString(w, `{"values":[],"next":""}`)
		case r.Method == http.MethodPost:
			posts++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if c, ok := body["content"].(map[string]any); ok {
				if raw, ok := c["raw"].(string); ok {
					createdBody = raw
				}
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":7}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	p, err := comment.NewBitbucketPoster("u", "pw", srv.URL,
		comment.BitbucketTarget{Workspace: "acme", Repo: "widgets", PR: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Post(context.Background(), "hello"); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if posts != 1 {
		t.Errorf("expected 1 POST, got %d", posts)
	}
	if !strings.Contains(createdBody, comment.Marker) {
		t.Errorf("missing marker: %q", createdBody)
	}
}

// TestBitbucketRecreateDeletesThenCreates verifies --recreate on
// Bitbucket: the marked comment is DELETEd and a fresh one POSTed, not
// PUT in place.
func TestBitbucketRecreateDeletesThenCreates(t *testing.T) {
	var deleted, created, put bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			payload := map[string]any{
				"values": []map[string]any{
					{"id": 42, "content": map[string]string{"raw": comment.Marker + "\n(old)"}},
				},
				"next": "",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			created = true
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":43}`)
		case http.MethodPut:
			put = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	p, _ := comment.NewBitbucketPoster("u", "pw", srv.URL,
		comment.BitbucketTarget{Workspace: "w", Repo: "r", PR: 1},
		comment.Options{Recreate: true})
	if err := p.Post(context.Background(), "fresh"); err != nil {
		t.Fatal(err)
	}
	if !deleted || !created {
		t.Errorf("recreate should DELETE then POST; deleted=%v created=%v", deleted, created)
	}
	if put {
		t.Error("recreate must not PUT in place")
	}
}

func TestBitbucketPostUpdatesMarkedComment(t *testing.T) {
	var put bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			payload := map[string]any{
				"values": []map[string]any{
					{"id": 42, "content": map[string]string{"raw": comment.Marker + "\n(stale)"}},
				},
				"next": "",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case r.Method == http.MethodPut:
			put = true
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":42}`)
		case r.Method == http.MethodPost:
			t.Error("unexpected POST — should have edited the marked comment")
		}
	}))
	defer srv.Close()
	p, _ := comment.NewBitbucketPoster("u", "pw", srv.URL,
		comment.BitbucketTarget{Workspace: "w", Repo: "r", PR: 1})
	if err := p.Post(context.Background(), "fresh"); err != nil {
		t.Fatal(err)
	}
	if !put {
		t.Error("expected PUT against the marked comment")
	}
}
