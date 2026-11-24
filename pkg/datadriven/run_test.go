// Copyright 2019 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package datadriven

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

func TestNewLineBetweenDirectives(t *testing.T) {
	RunTestFromString(t, `
> foo
----
unknown command

> bar
----
unknown command




> bar
----
unknown command
`, func(t *testing.T, d *TestCase) string {
		if d.Input != "sentence" {
			return "unknown command"
		}
		return ""
	})
}

func TestSkip(t *testing.T) {
	RunTestFromString(t, `
> skip
----

> error
----
`, func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "skip":
			// Verify that calling t.Skip() does not fail with an API error on
			// testing.T.
			t.Skip("woo")
		case "error":
			// The skip should mask the error afterwards.
			t.Error("never reached")
		}
		return d.Expected
	})
}

// TestComments verifies that #-prefixed lines are always inert comments —
// single lines, blocks, bare #, and #==== separators alike — and never open
// subtests or affect execution.
func TestComments(t *testing.T) {
	RunTestFromString(t, `
# a single comment line directly above a directive
> name
----
TestComments

#
# ============================================================
# a block with bare # lines and an ==== separator
# ============================================================
#

> name
----
TestComments

#====
> name
----
TestComments

## nested-looking heading is a comment too
> name
----
TestComments
`, func(t *testing.T, d *TestCase) string {
		// The test name proves no subtest was opened for any # line.
		return t.Name()
	})
}

func TestMultiLineTest(t *testing.T) {
	RunTest(t, "testdata/multiline", func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "small":
			return `just
two lines of output`
		case "large":
			return `more
than
five
lines
of
output`
		default:
			d.Fatalf(t, "unknown directive: %s", d.Name)
		}
		return d.Expected
	})
}

func TestWalk(t *testing.T) {
	Walk(t, "testdata/walk", func(t *testing.T, path string) {
		RunTest(t, path, func(t *testing.T, d *TestCase) string {
			return fmt.Sprintf("test name: %s\n", t.Name())
		})
	})
}

// rewriteHandler implements the simple directives used by the rewrite and
// clear-results tests.
func rewriteHandler(t *testing.T, d *TestCase) string {
	switch d.Name {
	case "noop":
		return d.Input

	case "duplicate":
		return fmt.Sprintf("%s\n%s", d.Input, d.Input)

	case "duplicate-with-blank":
		return fmt.Sprintf("%s\n\n%s", d.Input, d.Input)

	case "no-output":
		return ""

	default:
		d.Fatalf(t, "unknown directive: %s", d.Name)
		return ""
	}
}

// TestRewrite runs each testdata/rewrite/*-before file in rewrite mode and
// requires the result to match the corresponding *-after file. Running the
// package tests with -rewrite regenerates the *-after files instead.
func TestRewrite(t *testing.T) {
	const testDir = "testdata/rewrite"
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	var tests []string
	for _, file := range files {
		if name := file.Name(); strings.HasSuffix(name, "-before") {
			tests = append(tests, strings.TrimSuffix(name, "-before"))
		} else if !strings.HasSuffix(name, "-after") {
			t.Fatalf("all files in %s must end in either -before or -after: %s", testDir, name)
		}
	}
	sort.Strings(tests)

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			before, err := os.ReadFile(filepath.Join(testDir, test+"-before"))
			if err != nil {
				t.Fatal(err)
			}

			// Rewrite a scratch copy so the -before fixture stays intact.
			path := filepath.Join(t.TempDir(), test)
			if err := os.WriteFile(path, before, 0o644); err != nil {
				t.Fatal(err)
			}
			runTest(t, path, rewriteHandler, runConfig{rewrite: true})
			rewritten, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			afterPath := filepath.Join(testDir, test+"-after")
			if *rewriteTestFiles {
				// We are rewriting the rewrite tests themselves: dump the
				// output into the -after file.
				if err := os.WriteFile(afterPath, rewritten, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			expected, err := os.ReadFile(afterPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(rewritten, expected) {
				diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
					A: difflib.SplitLines(string(expected)),
					B: difflib.SplitLines(string(rewritten)),
				})
				if err != nil {
					t.Fatal(err)
				}
				t.Errorf("expected didn't match actual:\n%s", diff)
			}
		})
	}
}

// TestClearResults verifies that ClearResults blanks every case's expected
// output while preserving directives, inputs, and comments — including
// collapsing ---- BEGIN/END sections to a plain ----.
func TestClearResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clear")
	err := os.WriteFile(path, []byte(`# a comment to preserve
> noop
some input
----
stale output

> duplicate-with-blank
more input
---- BEGIN
stale

output
---- END
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	if err := ClearResults(path); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `# a comment to preserve
> noop
some input
----

> duplicate-with-blank
more input
----
`
	if string(got) != want {
		t.Fatalf("cleared file:\n%s\nwant:\n%s", got, want)
	}
}

// TestClearResultsMissing verifies ClearResults surfaces the read error for a
// nonexistent path.
func TestClearResultsMissing(t *testing.T) {
	if err := ClearResults(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("ClearResults(missing) = nil error, want an error")
	}
}
