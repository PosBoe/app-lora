package lora

import (
	"bufio"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skippedDirs are not read. The skills keep worked examples from the reference
// repository, which name paths this tree does not hold on purpose
// (.agents/skills/README.md § Origin).
var skippedDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"dist":          true,
	".agents":       true,
	".claude":       true,
	".vite":         true,
	".vitest-cache": true,
}

// citationAllowances are cited paths that a fresh checkout does not hold.
// Each one states why.
var citationAllowances = map[string]string{
	"config/dev.yml":  "task init copies it from config/dev.default.yml; git ignores it",
	"config/test.yml": "task init copies it from config/test.default.yml; git ignores it",
	"bin/lora":        "task build writes it; git ignores it",
	"vue/dist":        "task build:vue writes it; git ignores it",
}

// docAllowances are path prefixes that one document cites on purpose although
// the tree does not hold them. Each states why.
var docAllowances = map[string]map[string]string{
	"pkg/datadriven/DESIGN.md": {
		"pkg/ddtest":             "the reference repository's harness over this package; it was not copied",
		"tracing/":               "a consumer in the reference repository",
		"tsmetric/":              "a consumer in the reference repository",
		"cockroachdb/datadriven": "the upstream repository this package forks",
	},
	"pkg/datadriven/README.md": {
		"testdata/":  "an example layout of a consumer package",
		"tsmetric/":  "a consumer in the reference repository",
		"pkg/ddtest": "the reference repository's harness over this package; it was not copied",
	},
}

var (
	backtickRE = regexp.MustCompile("`([^`\n]+)`")
	linkRE     = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	// A citation may end in :42, :42-45, or :Symbol.
	lineRE = regexp.MustCompile(`:(\d+(-\d+)?|[A-Za-z_][\w.]*)$`)
)

// TestDocCitationsExist fails a document that cites a repository path the
// tree no longer holds. It reads backticked spans that hold a slash, and
// relative Markdown links, outside fenced code blocks.
//
// A path resolves against the repository root, the document's own directory,
// or vue/ (a frontend document writes `src/...`). A bare file name such as
// `CONTEXT.md` names a kind of document, not a path, so it is not checked.
// Neither is a one-segment directory (`internal/`), a route (`/api/v1`), or a
// standard library package (`log/slog`).
func TestDocCitationsExist(t *testing.T) {
	docs := walkTree(t)
	require.NotEmpty(t, docs)

	for _, doc := range docs {
		for _, c := range citations(t, doc) {
			if allowed(doc, c.path) {
				continue
			}
			if !resolves(doc, c.path) {
				assert.Failf(t, "dead citation", "%s:%d cites `%s`, which the tree does not hold", doc, c.line, c.path)
			}
		}
	}
}

func allowed(doc, path string) bool {
	if _, ok := citationAllowances[path]; ok {
		return true
	}
	for prefix := range docAllowances[doc] {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

type citation struct {
	path string
	line int
}

func walkTree(t *testing.T) (docs []string) {
	t.Helper()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." && skippedDirs[d.Name()] {
			return filepath.SkipDir
		}
		// CLAUDE.md is a symlink to AGENTS.md; read the target once.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			docs = append(docs, path)
		}
		return nil
	})
	require.NoError(t, err)
	return docs
}

func citations(t *testing.T, doc string) []citation {
	t.Helper()
	f, err := os.Open(doc)
	require.NoError(t, err)
	defer f.Close()

	var out []citation
	fenced := false
	scanner := bufio.NewScanner(f)
	for n := 1; scanner.Scan(); n++ {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		for _, m := range backtickRE.FindAllStringSubmatch(line, -1) {
			if p, ok := pathLike(m[1]); ok {
				out = append(out, citation{path: p, line: n})
			}
		}
		for _, m := range linkRE.FindAllStringSubmatch(line, -1) {
			if p, ok := linkTarget(m[1]); ok {
				out = append(out, citation{path: p, line: n})
			}
		}
	}
	require.NoError(t, scanner.Err())
	return out
}

// pathLike reports whether a backticked span names a repository path, and
// returns it without a trailing :line suffix.
func pathLike(s string) (string, bool) {
	if strings.ContainsAny(s, " *?<>{}$()[]|=,'\"") {
		return "", false
	}
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "/") || strings.Contains(s, "://") {
		return "", false
	}
	s = lineRE.ReplaceAllString(s, "")
	trimmed := strings.TrimSuffix(s, "/")
	if !strings.Contains(trimmed, "/") {
		return "", false
	}
	if isStdlib(trimmed) {
		return "", false
	}
	// A module path such as github.com/goccy/go-yaml or go.uber.org/fx.
	if first, _, ok := strings.Cut(s, "/"); ok && strings.Contains(first, ".") && !strings.HasPrefix(first, ".") {
		return "", false
	}
	return s, true
}

func linkTarget(s string) (string, bool) {
	if strings.Contains(s, "://") || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "mailto:") {
		return "", false
	}
	s, _, _ = strings.Cut(s, "#")
	// A link target is a path only when it holds a slash or a file extension;
	// `[*T](err)` in prose is not a link.
	return s, strings.ContainsAny(s, "/.")
}

func resolves(doc, path string) bool {
	clean := strings.TrimSuffix(path, "/")
	return exists(clean) ||
		exists(filepath.Join(filepath.Dir(doc), clean)) ||
		exists(filepath.Join("vue", clean))
}

func isStdlib(path string) bool {
	return exists(filepath.Join(build.Default.GOROOT, "src", path))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
