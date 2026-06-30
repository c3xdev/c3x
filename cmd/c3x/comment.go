package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/c3xdev/c3x/internal/comment"
	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/spf13/cobra"
)

// commentBody computes the markdown body for a PR/MR comment. With no
// baseline it's an absolute estimate; when baselinePath is set it's a
// cost diff against that saved baseline — the per-PR delta ("+$144
// this PR") rather than the project total. Shared by every forge
// subcommand so they stay identical.
func commentBody(
	ctx context.Context,
	rawPath string,
	resolved config.Resolved,
	varFiles, vars []string,
	baselinePath string,
) (string, error) {
	current, err := computeCurrent(ctx, rawPath, resolved, varFiles, vars)
	if err != nil {
		return "", err
	}
	if baselinePath == "" {
		return comment.FormatComment(current)
	}
	baseline, err := loadBaseline(baselinePath)
	if err != nil {
		return "", fmt.Errorf("loading baseline %s: %w", baselinePath, err)
	}
	return comment.FormatCommentDiff(domain.ComputeDiff(baseline, current))
}

// newCommentCmd assembles `c3x comment <forge>` for GitHub, GitLab,
// and Bitbucket, each plugged into the same dispatcher.
func newCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Post the cost estimate as a comment on a pull / merge request.",
	}
	cmd.AddCommand(newCommentGitHubCmd())
	cmd.AddCommand(newCommentGitLabCmd())
	cmd.AddCommand(newCommentBitbucketCmd())
	cmd.AddCommand(newCommentAzureDevOpsCmd())
	return cmd
}

func newCommentAzureDevOpsCmd() *cobra.Command {
	var (
		path         string
		token        string
		org          string
		project      string
		repo         string
		pr           int
		baseURL      string
		offline      bool
		varFiles     []string
		vars         []string
		baselinePath string
	)
	cmd := &cobra.Command{
		Use:     "azuredevops",
		Aliases: []string{"ado"},
		Short:   "Post a c3x estimate as an Azure DevOps PR comment.",
		Long: `Posts the markdown-rendered estimate as a thread on an Azure DevOps
pull request. Like the other forges, the comment carries a versioned
marker so re-runs update in place.

In CI: when running inside an Azure Pipelines pull-request job,
org / project / repo / PR auto-detect from
SYSTEM_TEAMFOUNDATIONCOLLECTIONURI, SYSTEM_TEAMPROJECT,
BUILD_REPOSITORY_NAME, and SYSTEM_PULLREQUEST_PULLREQUESTID. The
token reads from AZURE_DEVOPS_TOKEN or SYSTEM_ACCESSTOKEN.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, detectedBaseURL, err := resolveAzureDevOpsTarget(org, project, repo, pr)
			if err != nil {
				return err
			}
			if baseURL == "" {
				baseURL = detectedBaseURL
			}
			if token == "" {
				token = os.Getenv("AZURE_DEVOPS_TOKEN")
				if token == "" {
					token = os.Getenv("SYSTEM_ACCESSTOKEN")
				}
			}
			poster, err := comment.NewAzureDevOpsPoster(token, baseURL, target)
			if err != nil {
				return err
			}

			projectDir, err := resolveProjectDir(path)
			if err != nil {
				return err
			}
			flags := map[string]any{}
			if offline {
				flags["offline"] = true
			}
			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			body, err := commentBody(cmd.Context(), path, resolved, varFiles, vars, baselinePath)
			if err != nil {
				return err
			}
			if err := poster.Post(cmd.Context(), body); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(),
				"posted c3x comment to %s/%s/%s PR !%d\n",
				target.Org, target.Project, target.Repo, target.PR)
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", ".", "Terraform input (directory, .tf, plan JSON)")
	cmd.Flags().StringVar(&token, "token", "",
		"Azure DevOps PAT (default: $AZURE_DEVOPS_TOKEN or $SYSTEM_ACCESSTOKEN)")
	cmd.Flags().StringVar(&org, "org", "", "organisation (default: auto-detect)")
	cmd.Flags().StringVar(&project, "project", "", "project (default: auto-detect)")
	cmd.Flags().StringVar(&repo, "repo", "", "repo (default: auto-detect)")
	cmd.Flags().IntVar(&pr, "pr", 0, "PR ID (default: auto-detect)")
	cmd.Flags().StringVar(&baseURL, "base-url", "",
		"override the API host (default: dev.azure.com)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars file (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "",
		"saved baseline JSON (from `c3x estimate --save-baseline`); posts a cost delta instead of an absolute estimate")
	return cmd
}

func resolveAzureDevOpsTarget(org, project, repo string, pr int) (comment.AzureDevOpsTarget, string, error) {
	if org != "" && project != "" && repo != "" && pr > 0 {
		return comment.AzureDevOpsTarget{Org: org, Project: project, Repo: repo, PR: pr}, "", nil
	}
	detected, baseURL, err := comment.AutoDetectAzureDevOps()
	if err != nil {
		return comment.AzureDevOpsTarget{}, "",
			fmt.Errorf("auto-detect failed (supply --org/--project/--repo/--pr): %w", err)
	}
	if org != "" {
		detected.Org = org
	}
	if project != "" {
		detected.Project = project
	}
	if repo != "" {
		detected.Repo = repo
	}
	if pr > 0 {
		detected.PR = pr
	}
	if detected.PR == 0 {
		return comment.AzureDevOpsTarget{}, "",
			errors.New("PR ID is required (--pr) and was not auto-detectable")
	}
	return detected, baseURL, nil
}

func newCommentBitbucketCmd() *cobra.Command {
	var (
		path         string
		user         string
		password     string
		workspace    string
		repo         string
		pr           int
		baseURL      string
		offline      bool
		varFiles     []string
		vars         []string
		baselinePath string
	)
	cmd := &cobra.Command{
		Use:   "bitbucket",
		Short: "Post a c3x estimate as a Bitbucket Cloud PR comment.",
		Long: `Posts the markdown-rendered estimate as a comment on a Bitbucket
Cloud PR. Like the other forges, the comment carries a versioned
marker so re-runs update in place.

In CI: when running inside a Bitbucket Pipelines pull-request
pipeline, workspace / repo / PR auto-detect from
BITBUCKET_REPO_FULL_NAME and BITBUCKET_PR_ID. Username defaults to
$BITBUCKET_USERNAME, the app password to $BITBUCKET_APP_PASSWORD.

Bitbucket Server (Data Center) is intentionally not supported —
its REST shape differs and self-hosted users will be better served
by a dedicated implementation.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, err := resolveBitbucketTarget(workspace, repo, pr)
			if err != nil {
				return err
			}
			if user == "" {
				user = os.Getenv("BITBUCKET_USERNAME")
			}
			if password == "" {
				password = os.Getenv("BITBUCKET_APP_PASSWORD")
			}
			poster, err := comment.NewBitbucketPoster(user, password, baseURL, target)
			if err != nil {
				return err
			}

			projectDir, err := resolveProjectDir(path)
			if err != nil {
				return err
			}
			flags := map[string]any{}
			if offline {
				flags["offline"] = true
			}
			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			body, err := commentBody(cmd.Context(), path, resolved, varFiles, vars, baselinePath)
			if err != nil {
				return err
			}
			if err := poster.Post(cmd.Context(), body); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(),
				"posted c3x comment to %s/%s PR #%d\n", target.Workspace, target.Repo, target.PR)
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", ".", "Terraform input (directory, .tf, plan JSON)")
	cmd.Flags().StringVar(&user, "user", "", "Bitbucket username (default: $BITBUCKET_USERNAME)")
	cmd.Flags().StringVar(&password, "token", "",
		"Bitbucket app password (default: $BITBUCKET_APP_PASSWORD)")
	cmd.Flags().StringVar(&workspace, "workspace", "",
		"Bitbucket workspace (default: auto-detect from BITBUCKET_REPO_FULL_NAME)")
	cmd.Flags().StringVar(&repo, "repo", "", "Bitbucket repo (default: auto-detect)")
	cmd.Flags().IntVar(&pr, "pr", 0, "pull request ID (default: auto-detect from BITBUCKET_PR_ID)")
	cmd.Flags().StringVar(&baseURL, "base-url", "",
		"override the API base URL (default: api.bitbucket.org/2.0)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars file (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "",
		"saved baseline JSON (from `c3x estimate --save-baseline`); posts a cost delta instead of an absolute estimate")
	return cmd
}

func resolveBitbucketTarget(workspace, repo string, pr int) (comment.BitbucketTarget, error) {
	if workspace != "" && repo != "" && pr > 0 {
		return comment.BitbucketTarget{Workspace: workspace, Repo: repo, PR: pr}, nil
	}
	detected, err := comment.AutoDetectBitbucket()
	if err != nil {
		return comment.BitbucketTarget{}, fmt.Errorf("auto-detect failed (supply --workspace/--repo/--pr): %w", err)
	}
	if workspace != "" {
		detected.Workspace = workspace
	}
	if repo != "" {
		detected.Repo = repo
	}
	if pr > 0 {
		detected.PR = pr
	}
	if detected.PR == 0 {
		return comment.BitbucketTarget{}, errors.New("PR number is required (--pr) and was not auto-detectable")
	}
	return detected, nil
}

func newCommentGitLabCmd() *cobra.Command {
	var (
		path         string
		token        string
		project      string
		baseURL      string
		mr           int
		offline      bool
		varFiles     []string
		vars         []string
		baselinePath string
	)
	cmd := &cobra.Command{
		Use:   "gitlab",
		Short: "Post a c3x estimate as a GitLab merge-request comment.",
		Long: `Posts the markdown-rendered estimate as a note on a GitLab MR.
Like the GitHub poster, the note carries a versioned marker so
re-runs update in place.

In CI: when running inside a GitLab merge-request pipeline, project
and MR auto-detect from CI_PROJECT_ID and CI_MERGE_REQUEST_IID. The
API base URL also reads CI_API_V4_URL automatically (works on
self-hosted GitLab without flags). Token reads from GITLAB_TOKEN
or CI_JOB_TOKEN, or pass --token.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, detectedBaseURL, err := resolveGitLabTarget(project, mr)
			if err != nil {
				return err
			}
			if baseURL == "" {
				baseURL = detectedBaseURL
			}
			if token == "" {
				token = os.Getenv("GITLAB_TOKEN")
				if token == "" {
					token = os.Getenv("CI_JOB_TOKEN")
				}
			}
			poster, err := comment.NewGitLabPoster(token, baseURL, target)
			if err != nil {
				return err
			}

			projectDir, err := resolveProjectDir(path)
			if err != nil {
				return err
			}
			flags := map[string]any{}
			if offline {
				flags["offline"] = true
			}
			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			body, err := commentBody(cmd.Context(), path, resolved, varFiles, vars, baselinePath)
			if err != nil {
				return err
			}
			if err := poster.Post(cmd.Context(), body); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(),
				"posted c3x comment to project %s MR !%d\n", target.ProjectID, target.MR)
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", ".", "Terraform input (directory, .tf, plan JSON)")
	cmd.Flags().StringVar(&token, "token", "",
		"GitLab API token (default: $GITLAB_TOKEN or $CI_JOB_TOKEN)")
	cmd.Flags().StringVar(&project, "project", "",
		"GitLab project ID or url-escaped group/project (default: auto-detect from CI_PROJECT_ID)")
	cmd.Flags().StringVar(&baseURL, "base-url", "",
		"GitLab API base URL (default: auto-detect from CI_API_V4_URL or gitlab.com)")
	cmd.Flags().IntVar(&mr, "mr", 0, "merge-request IID (default: auto-detect from CI_MERGE_REQUEST_IID)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars file (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "",
		"saved baseline JSON (from `c3x estimate --save-baseline`); posts a cost delta instead of an absolute estimate")
	return cmd
}

// resolveGitLabTarget merges explicit flags with GitLab CI env
// auto-detection. Returns the resolved target + the detected
// base URL so the caller can pass it to the poster.
func resolveGitLabTarget(project string, mr int) (comment.GitLabTarget, string, error) {
	if project != "" && mr > 0 {
		return comment.GitLabTarget{ProjectID: project, MR: mr}, "", nil
	}
	detected, baseURL, err := comment.AutoDetectGitLab()
	if err != nil {
		return comment.GitLabTarget{}, "", fmt.Errorf("auto-detect failed (supply --project/--mr): %w", err)
	}
	if project != "" {
		detected.ProjectID = project
	}
	if mr > 0 {
		detected.MR = mr
	}
	if detected.MR == 0 {
		return comment.GitLabTarget{}, "", errors.New("MR IID is required (--mr) and was not auto-detectable")
	}
	return detected, baseURL, nil
}

func newCommentGitHubCmd() *cobra.Command {
	var (
		path         string
		token        string
		owner        string
		repo         string
		pr           int
		offline      bool
		varFiles     []string
		vars         []string
		baselinePath string
	)
	cmd := &cobra.Command{
		Use:   "github",
		Short: "Post a c3x estimate as a GitHub PR comment.",
		Long: `Posts the markdown-rendered estimate as a comment on a GitHub PR.
The comment carries a versioned marker so subsequent runs update it in
place rather than stacking duplicates.

In CI: when running inside a GitHub Actions pull-request workflow,
owner / repo / pr auto-detect from GITHUB_REPOSITORY and GITHUB_REF.
The token reads from GITHUB_TOKEN (set automatically by Actions) or
--token. Outside CI, supply --owner --repo --pr explicitly.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			target, err := resolveCommentTarget(owner, repo, pr)
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("GITHUB_TOKEN")
			}
			poster, err := comment.NewGitHubPoster(token, target)
			if err != nil {
				return err
			}

			projectDir, err := resolveProjectDir(path)
			if err != nil {
				return err
			}
			flags := map[string]any{}
			if offline {
				flags["offline"] = true
			}
			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			body, err := commentBody(cmd.Context(), path, resolved, varFiles, vars, baselinePath)
			if err != nil {
				return err
			}
			if err := poster.Post(cmd.Context(), body); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(),
				"posted c3x comment to %s/%s#%d\n", target.Owner, target.Repo, target.PR)
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", ".", "Terraform input (directory, .tf, plan JSON)")
	cmd.Flags().StringVar(&token, "token", "",
		"GitHub API token (default: $GITHUB_TOKEN)")
	cmd.Flags().StringVar(&owner, "owner", "", "GitHub repository owner (default: auto-detect from GITHUB_REPOSITORY)")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repository name (default: auto-detect)")
	cmd.Flags().IntVar(&pr, "pr", 0, "pull request number (default: auto-detect from GITHUB_REF)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars file (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "",
		"saved baseline JSON (from `c3x estimate --save-baseline`); posts a cost delta instead of an absolute estimate")
	return cmd
}

// resolveCommentTarget merges explicit flags with environment-based
// auto-detection. Any explicitly-supplied flag wins; missing values
// are filled in from GITHUB_REPOSITORY / GITHUB_REF.
func resolveCommentTarget(owner, repo string, pr int) (comment.Target, error) {
	if owner != "" && repo != "" && pr > 0 {
		return comment.Target{Owner: owner, Repo: repo, PR: pr}, nil
	}
	detected, err := comment.AutoDetect()
	if err != nil {
		return comment.Target{}, fmt.Errorf("auto-detect failed (supply --owner/--repo/--pr): %w", err)
	}
	if owner != "" {
		detected.Owner = owner
	}
	if repo != "" {
		detected.Repo = repo
	}
	if pr > 0 {
		detected.PR = pr
	}
	if detected.PR == 0 {
		return comment.Target{}, errors.New("PR number is required (--pr) and was not auto-detectable")
	}
	return detected, nil
}
