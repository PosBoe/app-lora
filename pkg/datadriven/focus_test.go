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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// focusFixture is a stateful counter file used by the focus tests. The
// expected output of the two inc cases is deliberately stale: focused runs
// must execute them for state but not compare them.
const focusFixture = `# counter fixture
> inc
----
stale-not-compared

> inc
----
stale-not-compared

> get
----
2

> boom
----
never reached
`

// writeFocusFixture writes the counter fixture to a temp file and returns
// its path plus a handler that counts inc calls in *n and fails on boom.
func writeFocusFixture(t *testing.T, n *int) (string, func(*testing.T, *TestCase) string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "counter")
	if err := os.WriteFile(path, []byte(focusFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := func(t *testing.T, tc *TestCase) string {
		switch tc.Name {
		case "inc":
			*n++
			return ""
		case "get":
			return fmt.Sprint(*n)
		case "boom":
			tc.Fatalf(t, "case after the focus target must not run")
			return ""
		default:
			tc.Fatalf(t, "unknown directive: %s", tc.Name)
			return ""
		}
	}
	return path, handler
}

// TestFocusLine focuses the "get" case (line 12 points at its expected
// output): the two inc cases must execute without being compared to their
// stale expected output, get is compared, and boom never runs.
func TestFocusLine(t *testing.T) {
	var n int
	path, handler := writeFocusFixture(t, &n)

	runTest(t, path, handler, runConfig{
		focus: focusTarget{path: path, line: 12},
	})

	if n != 2 {
		t.Fatalf("prefix cases must run to build state: inc ran %d times, want 2", n)
	}
}

// TestFocusLineRewrite combines -focus with -rewrite: only the focused
// case's expected output is replaced; the stale prefix, the skipped tail,
// and comments print back verbatim.
func TestFocusLineRewrite(t *testing.T) {
	var n int
	path, handler := writeFocusFixture(t, &n)

	runTest(t, path, handler, runConfig{
		rewrite: true,
		focus:   focusTarget{path: path, line: 12},
	})

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != focusFixture {
		// The focused case's actual output ("2") equals its expected output,
		// so a correct focused rewrite reproduces the file byte-for-byte —
		// stale prefix included.
		t.Fatalf("focused rewrite:\n%s\nwant:\n%s", got, focusFixture)
	}
}

// TestFocusOtherFileSkipped verifies that with -focus pointing at another
// file, runTest skips without invoking the handler.
func TestFocusOtherFileSkipped(t *testing.T) {
	var n int
	path, handler := writeFocusFixture(t, &n)

	t.Run("other", func(t *testing.T) {
		runTest(t, path, handler, runConfig{
			focus: focusTarget{path: "testdata/other_file"},
		})
	})

	if n != 0 {
		t.Fatal("handler ran for a file that is not the -focus target")
	}
}

// TestFocusWalk verifies that Walk visits only the focused file, including
// when a walk root contains no matching file at all (a package with several
// Walk roots skips the non-matching ones entirely).
func TestFocusWalk(t *testing.T) {
	dir := t.TempDir()
	rootA := filepath.Join(dir, "roota")
	rootB := filepath.Join(dir, "rootb")
	for _, p := range []string{
		filepath.Join(rootA, "one"),
		filepath.Join(rootA, "two"),
		filepath.Join(rootB, "three"),
	} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("> noop\n----\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := runConfig{focus: focusTarget{path: filepath.Join(rootA, "two")}}
	var visited []string
	visit := func(t *testing.T, path string) {
		visited = append(visited, filepath.Base(path))
	}

	walk(t, rootA, visit, cfg)
	walk(t, rootB, visit, cfg)

	if want := []string{"two"}; !reflect.DeepEqual(visited, want) {
		t.Fatalf("visited %q, want %q", visited, want)
	}
}

func TestParseFocus(t *testing.T) {
	tests := []struct {
		in      string
		want    focusTarget
		wantErr bool
	}{
		{in: "", want: focusTarget{}},
		{in: "testdata/foo", want: focusTarget{path: "testdata/foo"}},
		{in: "testdata/foo:42", want: focusTarget{path: "testdata/foo", line: 42}},
		// A non-numeric suffix is part of the file name, not a line.
		{in: "testdata/foo:bar", want: focusTarget{path: "testdata/foo:bar"}},
		{in: "testdata/foo:0", wantErr: true},
		{in: "testdata/foo:-1", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseFocus(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("parseFocus(%q): expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseFocus(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("parseFocus(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestFocusMatchesFile(t *testing.T) {
	tests := []struct {
		focus string
		path  string
		want  bool
	}{
		// No focus matches everything.
		{focus: "", path: "testdata/foo", want: true},
		{focus: "testdata/foo", path: "testdata/foo", want: true},
		// Repo-relative focus matches a package-relative path and vice versa.
		{focus: "tsmetric/testdata/foo", path: "testdata/foo", want: true},
		{focus: "testdata/foo", path: "tsmetric/testdata/foo", want: true},
		// Suffixes match only on path-component boundaries.
		{focus: "data/foo", path: "testdata/foo", want: false},
		{focus: "testdata/foo", path: "testdata/foobar", want: false},
		{focus: "testdata/foo", path: "testdata/bar", want: false},
	}
	for _, tt := range tests {
		ft := focusTarget{path: tt.focus}
		if got := ft.matchesFile(tt.path); got != tt.want {
			t.Fatalf("focus %q matchesFile(%q) = %v, want %v", tt.focus, tt.path, got, tt.want)
		}
	}
}
