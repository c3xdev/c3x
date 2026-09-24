package terraform

// Module fetcher for non-local module sources (Terraform Registry,
// Git, generic HTTP archive). Modules are downloaded into a content-
// addressable cache at `~/.cache/c3x/modules/<hash>/`; subsequent
// runs reuse the cache without re-fetching.
//
// This module fills the gap between c3x reading a Terraform config
// and the user having to remember to run `terraform init` first.
// The fetcher detects three source shapes:
//
//	registry:  "namespace/name/provider" (optionally with subdir/sub)
//	git:       "git::https://..." or "github.com/..." or "git@..."
//	http:      "https://...zip" or ".tar.gz"
//
// Anything else falls through to the existing local-path or
// .terraform/modules/modules.json resolver in modules.go.
//
// Git fetches shell out to the `git` binary rather than embedding a
// Go git library — the binary is available in every CI runner that
// runs Terraform, and embedding go-git would add ~10 MB to the
// c3x binary for a feature most users hit once.

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"
)

// registrySourceRe matches Terraform Registry module sources:
// `namespace/name/provider` or `namespace/name/provider//submodule`.
// The hostname `registry.terraform.io` defaults; private registries
// (`my-host.com/ns/name/provider`) are also accepted.
var registrySourceRe = regexp.MustCompile(
	`^(?:([a-zA-Z0-9.-]+)/)?([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)(?://(.+))?$`)

// ModuleFetcher resolves a Terraform module source string to a
// local directory containing the module's .tf files. Cache lives at
// cacheDir; one subdirectory per source-URL hash.
type ModuleFetcher struct {
	CacheDir string
	HTTP     *http.Client
	// RegistryURL overrides the registry scheme+host for tests
	// (e.g. an httptest server URL). Empty in production: the
	// host embedded in the module source is used over HTTPS.
	RegistryURL string
}

// NewModuleFetcher builds a fetcher with the given cache root. Pass
// the empty string to default to `~/.cache/c3x/modules/`.
func NewModuleFetcher(cacheDir string) (*ModuleFetcher, error) {
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cacheDir = filepath.Join(home, ".cache", "c3x", "modules")
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}
	return &ModuleFetcher{
		CacheDir: cacheDir,
		// http.DefaultClient has no timeout: a registry host that never
		// answers would hang the parse, and CI with it.
		HTTP: &http.Client{Timeout: fetchTimeout},
	}, nil
}

// Fetch resolves the given source string. `version` is consulted
// for registry sources (empty means "latest"). Returns the local
// path containing the module's Terraform files.
func (f *ModuleFetcher) Fetch(source, version string) (string, error) {
	switch detectSourceType(source) {
	case sourceLocal:
		return "", errors.New("local sources are handled by the parser directly")
	case sourceRegistry:
		return f.fetchRegistry(source, version)
	case sourceGit:
		return f.fetchGit(source)
	case sourceHTTP:
		return f.fetchArchive(source)
	default:
		return "", fmt.Errorf("unsupported module source %q", source)
	}
}

type sourceType int

const (
	sourceUnknown sourceType = iota
	sourceLocal
	sourceRegistry
	sourceGit
	sourceHTTP
)

func detectSourceType(source string) sourceType {
	if source == "." || strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		return sourceLocal
	}
	if strings.HasPrefix(source, "git::") ||
		strings.HasPrefix(source, "git@") ||
		strings.HasPrefix(source, "github.com/") ||
		strings.Contains(source, ".git") {
		return sourceGit
	}
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Archive vs Git tarball — git tarballs use `?ref=` style
		// queries handled by the git path, so plain HTTP gets the
		// archive treatment.
		return sourceHTTP
	}
	if registrySourceRe.MatchString(source) {
		return sourceRegistry
	}
	return sourceUnknown
}

// fetchRegistry resolves a Terraform Registry module. Two-step:
//
//  1. GET https://<host>/v1/modules/<ns>/<name>/<provider>/<version>/download
//     → 204 with X-Terraform-Get header pointing at the tarball
//  2. Follow X-Terraform-Get; archive-fetch the tarball.
func (f *ModuleFetcher) fetchRegistry(source, version string) (string, error) {
	parts := registrySourceRe.FindStringSubmatch(source)
	if parts == nil {
		return "", fmt.Errorf("invalid registry source %q", source)
	}
	host, ns, name, provider, subdir := parts[1], parts[2], parts[3], parts[4], parts[5]
	if host == "" {
		host = "registry.terraform.io"
	}
	switch {
	case version == "":
		version = "latest"
	case strings.ContainsAny(version, "~><=!,*"):
		// A constraint expression ("~> 3.0", ">= 2, < 4"). The
		// registry's /download URL needs an exact version, so list
		// the published versions and pick the newest one satisfying
		// the constraint — the same selection Terraform makes.
		resolved, err := f.resolveRegistryVersion(host, ns, name, provider, version)
		if err != nil {
			return "", err
		}
		version = resolved
	}

	cacheKey := hashKey(host + "/" + ns + "/" + name + "/" + provider + "@" + version)
	dest := filepath.Join(f.CacheDir, cacheKey)
	if dirHasFiles(dest) {
		return finalSubdir(dest, subdir), nil
	}

	// Resolve the download URL.
	resolveURL := fmt.Sprintf("%s/v1/modules/%s/%s/%s/%s/download",
		f.registryBase(host), ns, name, provider, version)
	resp, err := f.HTTP.Get(resolveURL)
	if err != nil {
		return "", fmt.Errorf("registry resolve %s: %w", resolveURL, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry resolve HTTP %d for %s", resp.StatusCode, resolveURL)
	}
	downloadURL := resp.Header.Get("X-Terraform-Get")
	if downloadURL == "" {
		return "", fmt.Errorf("registry resolve missing X-Terraform-Get header")
	}
	// X-Terraform-Get often points at a `git::https://github.com/...`
	// URL rather than a tarball — Terraform Registry IS a registry of
	// git repos. Route through the git path when so.
	if strings.HasPrefix(downloadURL, "git::") {
		return f.fetchGitInto(strings.TrimPrefix(downloadURL, "git::"), dest, subdir)
	}
	return f.fetchArchiveInto(downloadURL, dest, subdir)
}

// registryBase returns the scheme+host prefix for registry API
// calls. Production always talks HTTPS; tests override RegistryURL
// to point at an httptest server.
func (f *ModuleFetcher) registryBase(host string) string {
	if f.RegistryURL != "" {
		return f.RegistryURL
	}
	return "https://" + host
}

// resolveRegistryVersion lists the module's published versions and
// returns the newest one satisfying the Terraform-style constraint
// expression ("~> 5.0", ">= 2, < 4"). Mirrors Terraform's own
// selection: highest matching version wins.
func (f *ModuleFetcher) resolveRegistryVersion(host, ns, name, provider, constraint string) (string, error) {
	constraints, err := goversion.NewConstraint(constraint)
	if err != nil {
		return "", fmt.Errorf("invalid module version constraint %q: %w", constraint, err)
	}

	listURL := fmt.Sprintf("%s/v1/modules/%s/%s/%s/versions",
		f.registryBase(host), ns, name, provider)
	resp, err := f.HTTP.Get(listURL)
	if err != nil {
		return "", fmt.Errorf("registry versions %s: %w", listURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry versions HTTP %d for %s", resp.StatusCode, listURL)
	}

	var doc struct {
		Modules []struct {
			Versions []struct {
				Version string `json:"version"`
			} `json:"versions"`
		} `json:"modules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("decode registry versions: %w", err)
	}

	var best *goversion.Version
	for _, m := range doc.Modules {
		for _, v := range m.Versions {
			parsed, err := goversion.NewVersion(v.Version)
			if err != nil {
				continue // tolerate junk entries; the registry has a few
			}
			if !constraints.Check(parsed) {
				continue
			}
			if best == nil || parsed.GreaterThan(best) {
				best = parsed
			}
		}
	}
	if best == nil {
		return "", fmt.Errorf("no published version of %s/%s/%s satisfies %q",
			ns, name, provider, constraint)
	}
	return best.Original(), nil
}

// fetchGit clones the repo at `source` into the cache. Supports
// optional `?ref=<branch-or-tag>` query for pinning.
func (f *ModuleFetcher) fetchGit(source string) (string, error) {
	// Strip the `git::` prefix Terraform sometimes adds.
	clean := strings.TrimPrefix(source, "git::")
	cacheKey := hashKey(clean)
	dest := filepath.Join(f.CacheDir, cacheKey)
	return f.fetchGitInto(clean, dest, "")
}

func (f *ModuleFetcher) fetchGitInto(source, dest, subdir string) (string, error) {
	if dirHasFiles(dest) {
		return finalSubdir(dest, subdir), nil
	}
	url := source
	ref := ""
	if i := strings.LastIndex(source, "?ref="); i >= 0 {
		url = source[:i]
		ref = source[i+len("?ref="):]
	}
	// Split off `//subdir` after host/path.
	if i := strings.LastIndex(url, "//"); i > strings.Index(url, "://")+3 {
		subdir = url[i+2:]
		url = url[:i]
	}

	// Argument-injection guard: the URL and ref come from Terraform
	// source strings, i.e. attacker-influencable when estimating an
	// untrusted repo. A leading dash would let `--upload-pack=...`
	// or similar be parsed as a git OPTION instead of a positional
	// argument. Reject outright (no legitimate URL or ref starts
	// with "-") and additionally pass `--` before positionals.
	if strings.HasPrefix(url, "-") {
		return "", fmt.Errorf("refusing git URL starting with '-': %q", url)
	}
	if strings.HasPrefix(ref, "-") {
		return "", fmt.Errorf("refusing git ref starting with '-': %q", ref)
	}

	// Clone shallow for speed; checkout the ref afterwards.
	tmp, err := os.MkdirTemp(filepath.Dir(dest), "c3x-clone-")
	if err != nil {
		return "", err
	}
	args := []string{"clone", "--quiet"}
	if ref == "" {
		args = append(args, "--depth=1")
	}
	args = append(args, "--", url, tmp)
	if err := runCommand(args); err != nil {
		_ = os.RemoveAll(tmp)
		return "", fmt.Errorf("git clone %s: %w", url, err)
	}
	if ref != "" {
		if err := runCommandIn(tmp, []string{"checkout", "--quiet", ref, "--"}); err != nil {
			_ = os.RemoveAll(tmp)
			return "", fmt.Errorf("git checkout %s: %w", ref, err)
		}
	}
	// Atomic rename into the final cache slot.
	if err := os.Rename(tmp, dest); err != nil {
		// Another goroutine may have populated it; tolerate.
		_ = os.RemoveAll(tmp)
		if !dirHasFiles(dest) {
			return "", fmt.Errorf("rename to cache: %w", err)
		}
	}
	return finalSubdir(dest, subdir), nil
}

// fetchArchive downloads a `.zip` / `.tar.gz` archive into the cache.
func (f *ModuleFetcher) fetchArchive(source string) (string, error) {
	cacheKey := hashKey(source)
	dest := filepath.Join(f.CacheDir, cacheKey)
	return f.fetchArchiveInto(source, dest, "")
}

func (f *ModuleFetcher) fetchArchiveInto(source, dest, subdir string) (string, error) {
	if dirHasFiles(dest) {
		return finalSubdir(dest, subdir), nil
	}
	resp, err := f.HTTP.Get(source)
	if err != nil {
		return "", fmt.Errorf("archive fetch %s: %w", source, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("archive HTTP %d for %s", resp.StatusCode, source)
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dest), "c3x-arch-")
	if err != nil {
		return "", err
	}
	switch {
	case strings.HasSuffix(source, ".zip"):
		if err := unzip(resp.Body, tmp); err != nil {
			_ = os.RemoveAll(tmp)
			return "", err
		}
	default: // assume .tar.gz / .tgz
		if err := untarGz(resp.Body, tmp); err != nil {
			_ = os.RemoveAll(tmp)
			return "", err
		}
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.RemoveAll(tmp)
		if !dirHasFiles(dest) {
			return "", err
		}
	}
	return finalSubdir(dest, subdir), nil
}

// maxExtractBytes caps the total decompressed size of one module
// archive (tar.gz or zip). Terraform modules are kilobytes of HCL;
// 256 MiB is three orders of magnitude of headroom while still
// stopping a decompression bomb from filling the disk.
const maxExtractBytes = 256 << 20

// withinDir reports whether the joined path stays inside dir. A bare
// strings.HasPrefix(out, dir) is bypassable because "/cache/abc" is a
// string-prefix of the sibling "/cache/abcdef"; the separator-suffixed
// comparison closes that hole.
func withinDir(dir, joined string) bool {
	dir = filepath.Clean(dir)
	joined = filepath.Clean(joined)
	return joined == dir || strings.HasPrefix(joined, dir+string(filepath.Separator))
}

func untarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	var written int64
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		out := filepath.Join(dest, hdr.Name)
		if !withinDir(dest, out) {
			return fmt.Errorf("tar entry escapes destination: %q", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
		// archive/tar's Reader normalises the legacy TypeRegA to
		// TypeReg since Go 1.11, so matching TypeReg alone is complete.
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			f, err := os.Create(out)
			if err != nil {
				return err
			}
			n, err := io.Copy(f, io.LimitReader(tr, maxExtractBytes-written+1))
			_ = f.Close()
			if err != nil {
				return err
			}
			written += n
			if written > maxExtractBytes {
				return fmt.Errorf("archive exceeds %d-byte extraction budget (decompression bomb?)", int64(maxExtractBytes))
			}
		}
	}
}

func unzip(r io.Reader, dest string) error {
	// zip.Reader needs an io.ReaderAt; buffer to a temp file first.
	tmp, err := os.CreateTemp("", "c3x-zip-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()); _ = tmp.Close() }()
	// Extraction was capped, the download itself was not: an endless
	// response body would fill the disk before extraction began.
	n, err := io.Copy(tmp, io.LimitReader(r, maxExtractBytes+1))
	if err != nil {
		return err
	}
	if n > maxExtractBytes {
		return fmt.Errorf("module archive exceeds %d bytes", maxExtractBytes)
	}
	stat, _ := tmp.Stat()
	zr, err := zip.NewReader(tmp, stat.Size())
	if err != nil {
		return err
	}
	var written int64
	for _, zf := range zr.File {
		out := filepath.Join(dest, zf.Name)
		if !withinDir(dest, out) {
			return fmt.Errorf("zip entry escapes destination: %q", zf.Name)
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		f, err := os.Create(out)
		if err != nil {
			_ = rc.Close()
			return err
		}
		n, err := io.Copy(f, io.LimitReader(rc, maxExtractBytes-written+1))
		_ = f.Close()
		_ = rc.Close()
		if err != nil {
			return err
		}
		written += n
		if written > maxExtractBytes {
			return fmt.Errorf("archive exceeds %d-byte extraction budget (decompression bomb?)", int64(maxExtractBytes))
		}
	}
	return nil
}

// finalSubdir handles the `//<subdir>` suffix on source URLs;
// the module lives at <dest>/<subdir>, not <dest>.
func finalSubdir(dest, subdir string) string {
	if subdir == "" {
		return dest
	}
	return filepath.Join(dest, subdir)
}

func hashKey(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

func dirHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}

func runCommand(args []string) error { return runCommandIn("", args) }

func runCommandIn(dir string, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// fetchTimeout bounds one module download or git command.
const fetchTimeout = 2 * time.Minute

// gitEnv restricts the transports git may use for a module source.
// GIT_ALLOW_PROTOCOL is git's own allowlist: it rules out file:// (reading
// repositories on the host running c3x) and remote helpers such as ext::,
// which run commands. GIT_TERMINAL_PROMPT=0 makes a source that needs
// credentials fail instead of waiting on a prompt nobody will answer.
func gitEnv() []string {
	return append(os.Environ(),
		"GIT_ALLOW_PROTOCOL=https:ssh:git",
		"GIT_TERMINAL_PROMPT=0",
	)
}
