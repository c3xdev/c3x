package terraform

import (
	"crypto/md5"  //nolint:gosec // filemd5() is a checksum, not security.
	"crypto/sha1" //nolint:gosec // filesha1() likewise.
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// evalScope is where an expression is evaluated from: the root module's
// directory and the directory of the module the expression sits in. It
// supplies Terraform's path.module / path.root / path.cwd and the
// filesystem functions (file, templatefile, fileset, ...).
//
// Relative paths in those functions resolve against the root directory,
// as Terraform resolves them against the working directory it runs in,
// which is why configurations write file("${path.module}/x").
type evalScope struct {
	rootDir   string
	moduleDir string
	// allowFiles registers the filesystem functions. Without it they stay
	// unregistered, so a call is an unknown function and nothing is read.
	allowFiles bool
	// budget bounds the whole parse; shared by every module's scope.
	budget *parseBudget
	// untrusted confines local module sources to the root directory.
	untrusted bool
}

func newRootScope(dir string, allowFiles, untrusted bool) evalScope {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	return evalScope{rootDir: abs, moduleDir: abs, allowFiles: allowFiles, budget: newParseBudget(), untrusted: untrusted}
}

func (s evalScope) child(dir string) evalScope {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	return evalScope{rootDir: s.rootDir, moduleDir: abs, allowFiles: s.allowFiles, budget: s.budget, untrusted: s.untrusted}
}

// evalContext is buildEvalContext plus the path object and the
// filesystem functions for this scope.
func (s evalScope) evalContext(vars, locals, data cty.Value, extras map[string]cty.Value) *hcl.EvalContext {
	ctx := buildEvalContext(vars, locals, data, extras)
	if s.rootDir == "" {
		return ctx
	}
	ctx.Variables["path"] = s.pathObject()
	if !s.allowFiles {
		return ctx
	}
	for name, fn := range s.fileFunctions() {
		ctx.Functions[name] = fn
	}
	return ctx
}

func (s evalScope) pathObject() cty.Value {
	module := "."
	if rel, err := filepath.Rel(s.rootDir, s.moduleDir); err == nil {
		module = filepath.ToSlash(rel)
	}
	return cty.ObjectVal(map[string]cty.Value{
		"module": cty.StringVal(module),
		"root":   cty.StringVal("."),
		"cwd":    cty.StringVal(s.rootDir),
	})
}

// maxFileBytes caps a single read. Configuration files are small; the cap
// keeps a pathological input from making the parser read something huge.
const maxFileBytes = 4 << 20

// outsideProjectMsg is the refusal text, matched by warnUnknownFunctions
// so a refused read is reported rather than silently priced as a default.
const outsideProjectMsg = "c3x reads files only inside the project directory"

// errOutsideProject marks a path that resolves outside the allowed roots.
var errOutsideProject = errors.New(outsideProjectMsg)

// resolve maps a path argument to the real file inside the project.
//
// Reads are confined to the root module's directory and the current
// module's directory (which, for a remote module, is its cache dir). c3x
// also parses untrusted configuration server-side, where an unconfined
// file("/proc/self/environ") would disclose the host's environment into
// the estimate. Symlinks are resolved before the check, so a link inside
// the project pointing out of it is refused too.
func (s evalScope) resolve(p string) (string, error) {
	if !filepath.IsAbs(p) {
		p = filepath.Join(s.rootDir, p)
	}
	target, err := filepath.EvalSymlinks(filepath.Clean(p))
	if err != nil {
		return "", err
	}
	for _, root := range []string{s.rootDir, s.moduleDir} {
		r, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		if within(r, target) {
			return target, nil
		}
	}
	return "", fmt.Errorf("%w: %s", errOutsideProject, p)
}

// containsDir reports whether dir, symlinks resolved, lies within the root
// directory.
func (s evalScope) containsDir(dir string) bool {
	target, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return false
	}
	root, err := filepath.EvalSymlinks(s.rootDir)
	return err == nil && within(root, target)
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func (s evalScope) read(p string) ([]byte, error) {
	target, err := s.resolve(p)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", p)
	}
	if info.Size() > maxFileBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", p, maxFileBytes)
	}
	return os.ReadFile(target) //nolint:gosec // confined by resolve above
}

func (s evalScope) readText(p string) (string, error) {
	b, err := s.read(p)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", fmt.Errorf("%s contains invalid UTF-8; use filebase64 for binary files", p)
	}
	return string(b), nil
}

func (s evalScope) fileFunctions() map[string]function.Function {
	pathFn := func(impl func(string) (cty.Value, error), ret cty.Type) function.Function {
		return function.New(&function.Spec{
			Params: []function.Parameter{{Name: "path", Type: cty.String}},
			Type:   function.StaticReturnType(ret),
			Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
				return impl(args[0].AsString())
			},
		})
	}
	fileHash := func(newHash func() hash.Hash, encode func([]byte) string) function.Function {
		return pathFn(func(p string) (cty.Value, error) {
			b, err := s.read(p)
			if err != nil {
				return cty.NilVal, err
			}
			h := newHash()
			h.Write(b)
			return cty.StringVal(encode(h.Sum(nil))), nil
		}, cty.String)
	}

	fns := map[string]function.Function{
		"file": pathFn(func(p string) (cty.Value, error) {
			t, err := s.readText(p)
			return cty.StringVal(t), err
		}, cty.String),
		"filebase64": pathFn(func(p string) (cty.Value, error) {
			b, err := s.read(p)
			return cty.StringVal(base64.StdEncoding.EncodeToString(b)), err
		}, cty.String),
		"fileexists": pathFn(func(p string) (cty.Value, error) {
			target, err := s.resolve(p)
			if errors.Is(err, errOutsideProject) {
				return cty.NilVal, err
			}
			if err != nil {
				return cty.False, nil
			}
			info, err := os.Stat(target)
			if err != nil {
				return cty.False, nil
			}
			if info.IsDir() {
				return cty.NilVal, fmt.Errorf("%s is a directory, not a file", p)
			}
			return cty.True, nil
		}, cty.Bool),
		"filemd5":          fileHash(md5.New, hex.EncodeToString),
		"filesha1":         fileHash(sha1.New, hex.EncodeToString),
		"filesha256":       fileHash(sha256.New, hex.EncodeToString),
		"filesha512":       fileHash(sha512.New, hex.EncodeToString),
		"filebase64sha256": fileHash(sha256.New, base64.StdEncoding.EncodeToString),
		"filebase64sha512": fileHash(sha512.New, base64.StdEncoding.EncodeToString),
		"abspath": pathFn(func(p string) (cty.Value, error) {
			if !filepath.IsAbs(p) {
				p = filepath.Join(s.rootDir, p)
			}
			return cty.StringVal(filepath.ToSlash(filepath.Clean(p))), nil
		}, cty.String),
		"fileset": function.New(&function.Spec{
			Params: []function.Parameter{{Name: "path", Type: cty.String}, {Name: "pattern", Type: cty.String}},
			Type:   function.StaticReturnType(cty.Set(cty.String)),
			Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
				return s.fileset(args[0].AsString(), args[1].AsString())
			},
		}),
	}
	// templatefile renders with the same functions, except itself:
	// Terraform does not allow templatefile to be called recursively.
	fns["templatefile"] = function.New(&function.Spec{
		Params: []function.Parameter{{Name: "path", Type: cty.String}, {Name: "vars", Type: cty.DynamicPseudoType}},
		Type:   function.StaticReturnType(cty.DynamicPseudoType),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			src, err := s.readText(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			expr, diags := hclsyntax.ParseTemplate([]byte(src), args[0].AsString(), hcl.InitialPos)
			if diags.HasErrors() {
				return cty.NilVal, diags
			}
			vars := map[string]cty.Value{}
			if v := args[1]; v.CanIterateElements() {
				for it := v.ElementIterator(); it.Next(); {
					k, e := it.Element()
					vars[k.AsString()] = e
				}
			}
			tplFns := terraformFunctions()
			for k, f := range fns {
				if k != "templatefile" {
					tplFns[k] = f
				}
			}
			val, diags := expr.Value(&hcl.EvalContext{Variables: vars, Functions: tplFns})
			if diags.HasErrors() {
				return cty.NilVal, diags
			}
			return val, nil
		},
	})
	return fns
}

// fileset lists regular files under dir whose path relative to dir
// matches pattern, with doublestar semantics: `**` spans directories,
// `*` and `?` stay within one, and `{a,b}` offers alternatives.
func (s evalScope) fileset(dir, pattern string) (cty.Value, error) {
	base, err := s.resolve(dir)
	if err != nil {
		if errors.Is(err, errOutsideProject) {
			return cty.NilVal, err
		}
		return cty.SetValEmpty(cty.String), nil
	}
	patterns := expandBraces(filepath.ToSlash(pattern))
	var out []string
	err = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		for _, pat := range patterns {
			if globMatch(strings.Split(pat, "/"), strings.Split(rel, "/")) {
				out = append(out, rel)
				break
			}
		}
		return nil
	})
	if err != nil {
		return cty.NilVal, err
	}
	if len(out) == 0 {
		return cty.SetValEmpty(cty.String), nil
	}
	sort.Strings(out)
	vals := make([]cty.Value, len(out))
	for i, f := range out {
		vals[i] = cty.StringVal(f)
	}
	return cty.SetVal(vals), nil
}

// globMatch matches path segments, letting a `**` segment consume zero
// or more segments.
func globMatch(pat, segs []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			for i := 0; i <= len(segs); i++ {
				if globMatch(pat[1:], segs[i:]) {
					return true
				}
			}
			return false
		}
		if len(segs) == 0 {
			return false
		}
		if ok, _ := path.Match(pat[0], segs[0]); !ok {
			return false
		}
		pat, segs = pat[1:], segs[1:]
	}
	return len(segs) == 0
}

// expandBraces turns "a/{b,c}/*.tf" into ["a/b/*.tf", "a/c/*.tf"].
func expandBraces(p string) []string {
	open := strings.IndexByte(p, '{')
	if open < 0 {
		return []string{p}
	}
	depth := 0
	for i := open; i < len(p); i++ {
		switch p[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				var out []string
				for _, alt := range splitTopLevel(p[open+1 : i]) {
					out = append(out, expandBraces(p[:open]+alt+p[i+1:])...)
				}
				return out
			}
		}
	}
	return []string{p}
}

func splitTopLevel(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}
