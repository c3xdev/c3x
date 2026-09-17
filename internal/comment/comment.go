// Package comment posts cost-estimate comments to pull requests. The
// design is marker-based: every comment c3x writes carries a sentinel
// (`<!-- c3x-comment:v1 -->`) so a subsequent run finds and edits the
// existing comment instead of stacking new ones.
//
// GitHub, GitLab, Bitbucket, and Azure DevOps are supported; the
// [Poster] interface is the seam for adding more.
package comment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/google/go-github/v84/github"
)

// DefaultHTTPTimeout caps a single GitHub API request. PR-comment
// posting is part of the user-visible critical path; we never want
// to hang the CLI indefinitely on a flaky network.
const DefaultHTTPTimeout = 20 * time.Second

// Marker is the HTML-comment sentinel embedded in every body c3x
// posts. Versioned so we can change the layout later without losing
// the ability to find old comments. This is the base (untagged)
// marker; [MarkerFor] derives tag-namespaced variants.
const Marker = "<!-- c3x-comment:v1 -->"

// Options tunes how a Poster manages its comment. The zero value is
// the default behaviour c3x has always had: one shared marker,
// updated in place.
type Options struct {
	// Tag namespaces the comment marker so multiple c3x runs on the
	// same PR/MR — one per environment, or per Terraform directory in
	// a monorepo — keep independent comments instead of overwriting
	// each other. Empty uses the default [Marker].
	Tag string

	// Recreate deletes the previous c3x comment and posts a fresh one
	// instead of editing in place. Useful on busy PRs where "newest at
	// the bottom" reads better than an edit buried up-thread.
	Recreate bool
}

// resolveOptions collapses the variadic option slug the poster
// constructors accept into a single value. Passing no Options keeps
// the historical behaviour, which is why the constructors stayed
// backward-compatible when tagging was added.
func resolveOptions(opts []Options) Options {
	if len(opts) > 0 {
		return opts[0]
	}
	return Options{}
}

// MarkerFor returns the sentinel a Poster embeds for the given tag.
// An empty (or all-invalid) tag yields the base [Marker], so comments
// written before tagging existed are still found. A non-empty tag is
// namespaced as `<!-- c3x-comment:v1:<tag> -->` so each tag finds and
// edits only its own comment.
func MarkerFor(tag string) string {
	t := sanitizeTag(tag)
	if t == "" {
		return Marker
	}
	return "<!-- c3x-comment:v1:" + t + " -->"
}

// sanitizeTag restricts a user-supplied tag to a safe charset so it
// cannot break out of the surrounding HTML comment. Notably it drops
// '>' (mapping any disallowed rune to '-'), which makes a premature
// `-->` sequence impossible. The result stays a tidy single token,
// keeping the common `$WORKSPACE-$DIR` shape intact (letters, digits,
// '.', '_', '-', '/').
func sanitizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range tag {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '_', r == '-', r == '/':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	const maxLen = 80
	out := b.String()
	if len(out) > maxLen {
		out = out[:maxLen]
	}
	return strings.Trim(out, "-")
}

// Poster is the contract a forge integration satisfies. Implementing
// a new backend means returning a new Poster from a constructor and
// wiring a CLI subcommand.
type Poster interface {
	Post(ctx context.Context, body string) error
}

// Target is what `c3x comment github --owner=X --repo=Y --pr=Z`
// builds. The CLI fills it from flags or auto-detects from the
// GITHUB_* environment variables CI providers expose.
type Target struct {
	Owner string
	Repo  string
	PR    int
}

// AutoDetect populates a Target from the standard GitHub Actions
// environment. Missing values are reported via the returned error so
// the CLI can produce an actionable diagnostic.
//
//	GITHUB_REPOSITORY   "owner/repo"
//	GITHUB_REF          "refs/pull/123/merge" or "refs/heads/foo"
//	GITHUB_EVENT_NAME   "pull_request" (the only context we post in)
func AutoDetect() (Target, error) {
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return Target{}, errors.New("GITHUB_REPOSITORY not set (are you running in GitHub Actions?)")
	}
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" {
		return Target{}, fmt.Errorf("GITHUB_REPOSITORY=%q is not owner/repo", repo)
	}
	pr, err := prFromEnv()
	if err != nil {
		return Target{}, err
	}
	return Target{Owner: owner, Repo: name, PR: pr}, nil
}

// prFromEnv reads the PR number from the GitHub Actions environment.
// GitHub exposes the number directly via GITHUB_REF (e.g.
// `refs/pull/42/merge`) on pull-request events.
func prFromEnv() (int, error) {
	ref := os.Getenv("GITHUB_REF")
	if !strings.HasPrefix(ref, "refs/pull/") {
		return 0, fmt.Errorf("GITHUB_REF=%q is not a pull-request ref (need refs/pull/N/...)", ref)
	}
	parts := strings.Split(ref, "/")
	if len(parts) < 3 {
		return 0, fmt.Errorf("GITHUB_REF=%q lacks a PR number", ref)
	}
	n, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("GITHUB_REF=%q PR component is not numeric: %w", ref, err)
	}
	return n, nil
}

// GitHubPoster talks to api.github.com. Construct with [NewGitHubPoster];
// the zero value is not usable.
type GitHubPoster struct {
	client   *github.Client
	target   Target
	marker   string
	recreate bool
}

// NewGitHubPoster takes the API token (typically from GITHUB_TOKEN)
// and the target PR. An empty token is a usage error.
//
// The constructor installs a [DefaultHTTPTimeout] on the underlying
// HTTP client. go-github otherwise defers to http.DefaultClient,
// which has no timeout — a network-side hang would block the CLI
// indefinitely.
func NewGitHubPoster(token string, target Target, opts ...Options) (*GitHubPoster, error) {
	if token == "" {
		return nil, errors.New("github token is empty (set GITHUB_TOKEN or pass --token)")
	}
	if target.Owner == "" || target.Repo == "" || target.PR == 0 {
		return nil, fmt.Errorf("incomplete target: %+v", target)
	}
	o := resolveOptions(opts)
	httpClient := &http.Client{Timeout: DefaultHTTPTimeout}
	return &GitHubPoster{
		client:   github.NewClient(httpClient).WithAuthToken(token),
		target:   target,
		marker:   MarkerFor(o.Tag),
		recreate: o.Recreate,
	}, nil
}

// Post writes (or updates) the c3x comment on the PR. The body
// passed in must NOT already include the marker — Post prepends it.
func (p *GitHubPoster) Post(ctx context.Context, body string) error {
	existing, err := p.findExisting(ctx)
	if err != nil {
		return fmt.Errorf("looking up existing comment: %w", err)
	}
	fullBody := p.marker + "\n" + body
	if existing == nil {
		return p.create(ctx, fullBody)
	}
	// Recreate: drop the old comment and post a fresh one so the
	// latest estimate lands at the bottom of the thread.
	if p.recreate {
		if _, err := p.client.Issues.DeleteComment(ctx, p.target.Owner, p.target.Repo, *existing.ID); err != nil {
			return fmt.Errorf("deleting previous PR comment %d: %w", *existing.ID, err)
		}
		return p.create(ctx, fullBody)
	}
	existing.Body = &fullBody
	_, _, err = p.client.Issues.EditComment(ctx, p.target.Owner, p.target.Repo, *existing.ID,
		&github.IssueComment{Body: &fullBody})
	if err != nil {
		return fmt.Errorf("updating PR comment %d: %w", *existing.ID, err)
	}
	return nil
}

// create posts a new comment carrying the (already-marker-prefixed)
// body.
func (p *GitHubPoster) create(ctx context.Context, fullBody string) error {
	_, _, err := p.client.Issues.CreateComment(ctx, p.target.Owner, p.target.Repo, p.target.PR,
		&github.IssueComment{Body: &fullBody})
	if err != nil {
		return fmt.Errorf("creating PR comment: %w", err)
	}
	return nil
}

// findExisting walks the PR's issue comments looking for the marker.
// We paginate because long-running PRs can accumulate hundreds of
// comments; GitHub's default page size is 30.
func (p *GitHubPoster) findExisting(ctx context.Context) (*github.IssueComment, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		comments, resp, err := p.client.Issues.ListComments(ctx,
			p.target.Owner, p.target.Repo, p.target.PR, opts)
		if err != nil {
			return nil, err
		}
		for _, c := range comments {
			if c.Body != nil && strings.Contains(*c.Body, p.marker) {
				return c, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return nil, nil
}

// FormatComment renders an Estimate as the body c3x will post. By
// default this is the collapsible summary+details layout
// ([render.RenderMarkdownComment]): a one-line headline with the
// per-resource tables tucked into a <details> block, so a long estimate
// doesn't flood the thread. When expand is true it posts the flat,
// always-expanded layout ([render.RenderMarkdown]) instead.
//
// Markdown is hard-pinned in both cases because PR comments are
// markdown-rendered; the user's `--format` (e.g. text) would post
// box-drawing characters that render as garbage on GitHub.
func FormatComment(est domain.Estimate, expand bool) (string, error) {
	if expand {
		return render.RenderMarkdown(est), nil
	}
	return render.RenderMarkdownComment(est), nil
}

// FormatCommentDiff renders a Diff as the body c3x will post when a
// baseline is supplied — a per-PR cost delta ("Total: $894/mo →
// $1,038/mo +$144") as the headline. Collapsible by default (the
// Added/Modified/Removed tables sit in a <details> block); expand posts
// the flat layout. Markdown is hard-pinned as in [FormatComment].
func FormatCommentDiff(d domain.Diff, expand bool) (string, error) {
	if expand {
		return render.RenderMarkdownDiff(d), nil
	}
	return render.RenderMarkdownDiffComment(d), nil
}

// SetClientBaseURL redirects a poster's GitHub client to the given
// URL. Exported only for tests that stub api.github.com via
// httptest. Production code constructs the client with the default
// (api.github.com) base via [NewGitHubPoster]; this hook is
// intentionally narrow and unsafe for general use.
func SetClientBaseURL(p *GitHubPoster, baseURL string) error {
	if p == nil {
		return errors.New("nil poster")
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	c, err := p.client.WithEnterpriseURLs(baseURL, baseURL)
	if err != nil {
		return err
	}
	p.client = c
	return nil
}
