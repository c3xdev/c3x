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

func TestAutoDetectAzureDevOpsFromPipelinesEnv(t *testing.T) {
	t.Setenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI", "https://dev.azure.com/acme/")
	t.Setenv("SYSTEM_TEAMPROJECT", "platform")
	t.Setenv("BUILD_REPOSITORY_NAME", "widgets")
	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "73")
	target, baseURL, err := comment.AutoDetectAzureDevOps()
	if err != nil {
		t.Fatalf("AutoDetectAzureDevOps: %v", err)
	}
	if target.Org != "acme" || target.Project != "platform" ||
		target.Repo != "widgets" || target.PR != 73 {
		t.Errorf("target = %+v", target)
	}
	if baseURL != "https://dev.azure.com" {
		t.Errorf("baseURL = %q", baseURL)
	}
}

func TestAutoDetectAzureDevOpsRejectsNonPRPipeline(t *testing.T) {
	t.Setenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI", "https://dev.azure.com/acme/")
	t.Setenv("SYSTEM_TEAMPROJECT", "p")
	t.Setenv("BUILD_REPOSITORY_NAME", "r")
	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "")
	if _, _, err := comment.AutoDetectAzureDevOps(); err == nil {
		t.Error("expected error when not on a PR pipeline")
	}
}

func TestAzureDevOpsPosterRequiresAllFields(t *testing.T) {
	if _, err := comment.NewAzureDevOpsPoster("", "", comment.AzureDevOpsTarget{Org: "a", Project: "p", Repo: "r", PR: 1}); err == nil {
		t.Error("expected error on empty token")
	}
	if _, err := comment.NewAzureDevOpsPoster("t", "", comment.AzureDevOpsTarget{}); err == nil {
		t.Error("expected error on zero target")
	}
}

func TestAzureDevOpsCreatesThreadWhenNoneExists(t *testing.T) {
	var posts int
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_, _ = io.WriteString(w, `{"value":[]}`)
		case r.Method == http.MethodPost:
			posts++
			_ = json.NewDecoder(r.Body).Decode(&sent)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":5,"comments":[{"id":7}]}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	p, err := comment.NewAzureDevOpsPoster("pat", srv.URL,
		comment.AzureDevOpsTarget{Org: "acme", Project: "platform", Repo: "widgets", PR: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Post(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if posts != 1 {
		t.Errorf("expected 1 POST, got %d", posts)
	}
	// Confirm payload includes the marker via the nested comments slice.
	comments, _ := sent["comments"].([]any)
	if len(comments) == 0 {
		t.Fatal("posted thread had no comments")
	}
	c0, _ := comments[0].(map[string]any)
	content, _ := c0["content"].(string)
	if !strings.Contains(content, comment.Marker) {
		t.Errorf("posted content missing marker: %q", content)
	}
}

// TestAzureDevOpsRecreateDeletesThenCreates verifies --recreate on
// Azure DevOps: the marked thread's root comment is DELETEd (removing
// the thread) and a fresh thread POSTed, not PATCHed in place.
func TestAzureDevOpsRecreateDeletesThenCreates(t *testing.T) {
	var deleted, created, patched bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			payload := map[string]any{
				"value": []map[string]any{
					{
						"id": 99,
						"comments": []map[string]any{
							{"id": 11, "content": comment.Marker + "\n(old)"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			created = true
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":100}`)
		case http.MethodPatch:
			patched = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	p, _ := comment.NewAzureDevOpsPoster("pat", srv.URL,
		comment.AzureDevOpsTarget{Org: "o", Project: "p", Repo: "r", PR: 1},
		comment.Options{Recreate: true})
	if err := p.Post(context.Background(), "fresh"); err != nil {
		t.Fatal(err)
	}
	if !deleted || !created {
		t.Errorf("recreate should DELETE then POST; deleted=%v created=%v", deleted, created)
	}
	if patched {
		t.Error("recreate must not PATCH in place")
	}
}

func TestAzureDevOpsEditsExistingMarkedComment(t *testing.T) {
	var patched bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			payload := map[string]any{
				"value": []map[string]any{
					{
						"id": 99,
						"comments": []map[string]any{
							{"id": 11, "content": comment.Marker + "\n(stale)"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case r.Method == http.MethodPatch:
			patched = true
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":11}`)
		case r.Method == http.MethodPost:
			t.Error("created a new thread; should have PATCHed existing")
			w.WriteHeader(http.StatusConflict)
		}
	}))
	defer srv.Close()
	p, _ := comment.NewAzureDevOpsPoster("pat", srv.URL,
		comment.AzureDevOpsTarget{Org: "o", Project: "p", Repo: "r", PR: 1})
	if err := p.Post(context.Background(), "fresh"); err != nil {
		t.Fatal(err)
	}
	if !patched {
		t.Error("expected PATCH against the marked comment")
	}
}
