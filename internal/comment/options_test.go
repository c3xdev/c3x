package comment_test

// Tests for comment tagging (custom marker namespacing) and the
// recreate (delete-and-new) behaviour. GitLab is used as the exercise
// vehicle since it's the most common self-hosted target, but the marker
// logic is shared by every forge.

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

func TestMarkerFor(t *testing.T) {
	cases := []struct {
		name string
		tag  string
		want string
	}{
		{"empty falls back to base", "", comment.Marker},
		{"whitespace-only falls back to base", "   ", comment.Marker},
		{"simple tag", "staging-vpc", "<!-- c3x-comment:v1:staging-vpc -->"},
		{"path-like tag kept", "staging/modules/vpc", "<!-- c3x-comment:v1:staging/modules/vpc -->"},
		{"dots and underscores kept", "prod.us_east", "<!-- c3x-comment:v1:prod.us_east -->"},
		{"unsafe chars neutralised", "a b!c", "<!-- c3x-comment:v1:a-b-c -->"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := comment.MarkerFor(tc.tag); got != tc.want {
				t.Errorf("MarkerFor(%q) = %q, want %q", tc.tag, got, tc.want)
			}
		})
	}
}

// TestMarkerForCannotBreakOutOfComment guards the sanitiser: a tag that
// tries to smuggle a `-->` terminator must not produce a second one, or
// it could close the HTML comment early and corrupt the rendered body.
func TestMarkerForCannotBreakOutOfComment(t *testing.T) {
	m := comment.MarkerFor("evil --> <script>")
	if n := strings.Count(m, "-->"); n != 1 {
		t.Errorf("marker %q has %d '-->' terminators, want exactly 1", m, n)
	}
	// Isolate the tag portion (between the fixed prefix and the ` -->`
	// terminator); it must carry no '>' that could close the comment.
	inner := strings.TrimSuffix(strings.TrimPrefix(m, "<!-- c3x-comment:v1:"), " -->")
	if strings.Contains(inner, ">") {
		t.Errorf("marker %q leaked a '>' into the tag portion %q", m, inner)
	}
}

// TestTagScopedCommentsAreIndependent proves two runs with different
// tags don't clobber each other: a poster tagged "prod" ignores an
// existing "staging"-tagged note and creates its own.
func TestTagScopedCommentsAreIndependent(t *testing.T) {
	stagingNote := comment.MarkerFor("staging") + "\n(staging estimate)"
	var posted, updated bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/notes"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 5, "body": stagingNote}})
		case r.Method == http.MethodPost:
			posted = true
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":9}`)
		case r.Method == http.MethodPut:
			updated = true
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":5}`)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	poster, err := comment.NewGitLabPoster("tok", srv.URL,
		comment.GitLabTarget{ProjectID: "1", MR: 1},
		comment.Options{Tag: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if err := poster.Post(context.Background(), "prod estimate"); err != nil {
		t.Fatal(err)
	}
	if !posted {
		t.Error("prod-tagged run should create its own note, not reuse the staging one")
	}
	if updated {
		t.Error("prod-tagged run must not update the staging-tagged note")
	}
}

// TestTagMatchesOwnNote is the complement: a "staging"-tagged run finds
// and updates the staging note in place.
func TestTagMatchesOwnNote(t *testing.T) {
	stagingNote := comment.MarkerFor("staging") + "\n(stale)"
	var updated bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/notes"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 5, "body": stagingNote}})
		case r.Method == http.MethodPut:
			updated = true
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":5}`)
		case r.Method == http.MethodPost:
			t.Error("should have updated the staging note, not created a new one")
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	poster, _ := comment.NewGitLabPoster("tok", srv.URL,
		comment.GitLabTarget{ProjectID: "1", MR: 1},
		comment.Options{Tag: "staging"})
	if err := poster.Post(context.Background(), "fresh"); err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Error("expected the staging note to be updated in place")
	}
}

// TestRecreateDeletesThenCreates verifies the delete-and-new behaviour:
// with Recreate set, an existing note is DELETEd and a fresh one POSTed
// (so the latest estimate lands at the bottom) instead of a PUT.
func TestRecreateDeletesThenCreates(t *testing.T) {
	existing := comment.Marker + "\n(old)"
	var deleted, created, updated bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/notes"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 42, "body": existing}})
		case r.Method == http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost:
			created = true
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":43}`)
		case r.Method == http.MethodPut:
			updated = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	poster, _ := comment.NewGitLabPoster("tok", srv.URL,
		comment.GitLabTarget{ProjectID: "1", MR: 1},
		comment.Options{Recreate: true})
	if err := poster.Post(context.Background(), "fresh"); err != nil {
		t.Fatal(err)
	}
	if !deleted || !created {
		t.Errorf("recreate should DELETE then POST; deleted=%v created=%v", deleted, created)
	}
	if updated {
		t.Error("recreate must not PUT/update in place")
	}
}
