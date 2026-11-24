// Copyright 2018 The Cockroach Authors.
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
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pmezard/go-difflib/difflib"
)

// RunTest invokes a data-driven test. The test cases are contained in a
// separate test file and are dynamically loaded, parsed, and executed by this
// testing framework. By convention, test files are typically located in a
// sub-directory called "testdata". Each test file has the following format:
//
//	> <command>[,<command>...] [arg | arg=val | arg=[val1, val2, ...]]...
//	<input to the command>
//	----
//	<expected results>
//
// The command input can contain blank lines. However, by default, the expected
// results cannot contain blank lines. This alternate syntax allows the use of
// blank lines:
//
//	> <command>[,<command>...] [arg | arg=val | arg=[val1, val2, ...]]...
//	<input to the command>
//	---- BEGIN
//	<expected results>
//
//	<more expected results>
//	---- END
//
// To execute data-driven tests, pass the path of the test file as well as a
// function which can interpret and execute whatever commands are present in
// the test file. The framework invokes the function once per test case,
// passing it information about the case in a TestCase struct.
//
// The function must return the actual results of the case, which RunTest
// compares with the expected results. If the two are not equal, the test
// fails. All cases in a file run on the same testing.T, in file order; a
// failure stops the file.
//
// It is possible for a test to test for an "expected error" as follows:
//   - run the code to test
//   - if an error occurs, report the detail of the error as actual output
//   - place the expected error details in the expected results in the input
//     file
//
// It is also possible for a test to report an _unexpected_ test error by
// calling t.Error().
func RunTest(t *testing.T, path string, f func(t *testing.T, tc *TestCase) string) {
	t.Helper()
	runTest(t, path, f, configFromFlags(t))
}

// RunTestFromString is a version of RunTest which takes the contents of a
// test directly. It ignores the -focus flag (there is no file to match), and
// under -rewrite the cases run without comparison (there is no file to write
// back).
func RunTestFromString(t *testing.T, input string, f func(t *testing.T, tc *TestCase) string) {
	t.Helper()

	file, err := Parse("<string>", input)
	if err != nil {
		t.Fatal(err)
	}

	cfg := configFromFlags(t)
	cfg.focus = focusTarget{}
	runCases(t, file, f, cfg, noTarget)
}

// Walk goes through all the files in a subdirectory, creating subtests to
// match the file hierarchy; for each "leaf" file, the given function is
// called.
//
// This can be used in conjunction with RunTest. For example:
//
//	 datadriven.Walk(t, path, func (t *testing.T, path string) {
//	   // initialize per-test state
//	   datadriven.RunTest(t, path, func (t *testing.T, tc *datadriven.TestCase) string {
//	    // ...
//	   }
//	 }
//
//	Files:
//	  testdata/typing
//	  testdata/logprops/scan
//	  testdata/logprops/select
//
// If path is "testdata/typing", the function is called once and no subtests
// are created.
//
// If path is "testdata/logprops", the function is called two times, in
// separate subtests /scan, /select.
//
// If path is "testdata", the function is called three times, in subtest
// hierarchy /typing, /logprops/scan, /logprops/select.
//
// Temp/hidden files and markdown documentation (*.md) are skipped. With
// -focus, files other than the focused one are skipped.
func Walk(t *testing.T, path string, f func(t *testing.T, path string)) {
	t.Helper()
	walk(t, path, f, configFromFlags(t))
}

// walk implements Walk for a resolved runConfig.
func walk(t *testing.T, path string, f func(t *testing.T, path string), cfg runConfig) {
	t.Helper()

	finfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !finfo.IsDir() {
		if !cfg.focus.matchesFile(path) {
			t.Skipf("%s is not the -focus target", path)
		}
		f(t, path)
		return
	}

	files, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		name := file.Name()
		if tempFileRe.MatchString(name) {
			// Temp or hidden file, don't even try processing.
			continue
		}
		if strings.HasSuffix(name, ".md") {
			// Markdown documentation living alongside testdata files.
			continue
		}
		t.Run(cutExt(name), func(t *testing.T) {
			walk(t, filepath.Join(path, name), f, cfg)
		})
	}
}

// ClearResults strips expected output from a test file while keeping
// commands, inputs, and comments intact. Use this when -rewrite would produce
// garbage — for example after a serialization bug that makes every result
// malformed. Clear first, fix the bug, then rewrite cleanly.
func ClearResults(path string) error {
	file, err := ParseFile(path)
	if err != nil {
		return err
	}
	for _, tc := range file.Cases {
		tc.setResult("")
	}
	return os.WriteFile(path, file.Format(), 0o644)
}

// runConfig captures the flag-derived settings for one RunTest invocation.
// Tests construct it directly instead of mutating global flags.
type runConfig struct {
	// rewrite replaces each executed case's expected output with the
	// handler's actual output and writes the file back.
	rewrite bool
	// focus restricts execution to one file and, optionally, one case.
	focus focusTarget
}

// configFromFlags resolves the global flags into a runConfig. This is the
// only place the runner reads globals.
func configFromFlags(t *testing.T) runConfig {
	t.Helper()
	focus, err := parseFocus(*focusFlag)
	if err != nil {
		t.Fatal(err)
	}
	return runConfig{
		rewrite: *rewriteTestFiles,
		focus:   focus,
	}
}

// focusTarget is the parsed -focus flag: a testdata file path and an
// optional 1-based line selecting a single case within it.
type focusTarget struct {
	path string
	line int // 0 means the whole file
}

// parseFocus parses a "-focus <path>[:<line>]" value. A zero focusTarget
// means no focus was requested.
func parseFocus(s string) (focusTarget, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return focusTarget{}, nil
	}
	if i := strings.LastIndexByte(s, ':'); i >= 0 {
		if line, err := strconv.Atoi(s[i+1:]); err == nil {
			if line <= 0 {
				return focusTarget{}, fmt.Errorf("-focus: line must be positive: %s", s)
			}
			return focusTarget{path: s[:i], line: line}, nil
		}
	}
	return focusTarget{path: s}, nil
}

// isZero reports whether no focus was requested.
func (ft focusTarget) isZero() bool {
	return ft.path == ""
}

// matchesFile reports whether path refers to the focused file (always true
// without focus). Paths match by cleaned suffix on path-component
// boundaries, so both the package-relative form (testdata/foo) and the
// repo-relative form (tsmetric/testdata/foo) work.
func (ft focusTarget) matchesFile(path string) bool {
	if ft.isZero() {
		return true
	}
	// Normalize to slash form so the component-boundary checks below work on
	// Windows, where filepath.Clean would otherwise yield "\" separators.
	focus, path := filepath.ToSlash(filepath.Clean(ft.path)), filepath.ToSlash(filepath.Clean(path))
	if focus == path {
		return true
	}
	return strings.HasSuffix(focus, "/"+path) || strings.HasSuffix(path, "/"+focus)
}

// noTarget disables case focus in runCases: every case is compared.
const noTarget = -1

// runTest implements RunTest for a resolved runConfig: parse the file, run
// its cases, and write the updated document back under -rewrite.
func runTest(t *testing.T, path string, f func(t *testing.T, tc *TestCase) string, cfg runConfig) {
	t.Helper()

	if !cfg.focus.matchesFile(path) {
		t.Skipf("%s is not the -focus target", path)
	}

	file, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	target := noTarget
	if cfg.focus.line > 0 {
		target, err = file.caseAt(cfg.focus.line)
		if err != nil {
			t.Fatal(err)
		}
	}

	runCases(t, file, f, cfg, target)

	if cfg.rewrite {
		if err := os.WriteFile(path, file.Format(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// runCases executes the file's cases in order. With a target index, cases
// before it execute to build state but are not compared (their expected
// output may be deliberately stale while iterating on the target), the
// target case is compared (or rewritten), and cases after it do not run.
func runCases(
	t *testing.T, file *File, f func(t *testing.T, tc *TestCase) string, cfg runConfig, target int,
) {
	t.Helper()

	for i, tc := range file.Cases {
		if target != noTarget && i > target {
			// Cases after the -focus target are skipped entirely.
			return
		}

		actual := runCase(t, tc, f)
		if t.Failed() {
			// If the handler failed with .Error(), we can't hope it produced
			// useful output, and subsequent cases may depend on its state.
			// Stop processing the file.
			t.FailNow()
		}

		if target != noTarget && i < target {
			// A handler failure above still fails the test; only the
			// comparison against (possibly stale) expected output is skipped.
			continue
		}

		switch {
		case cfg.rewrite:
			tc.setResult(actual)
		case tc.Expected != actual:
			reportMismatch(t, tc, actual)
		case verbose():
			t.Logf("\n%s:\n%s  ----\n%s", tc.Pos, indentLines(tc.String()), indentLines(actual))
		}
	}
}

// runCase invokes the handler for one case, adding panic/hang diagnostics,
// and returns the actual output normalized to end in a newline.
func runCase(t *testing.T, tc *TestCase, f func(t *testing.T, tc *TestCase) string) string {
	t.Helper()

	returned := false
	defer func() {
		if r := recover(); r != nil {
			t.Logf("\npanic during %s:\n%s\n", tc.Pos, tc.Input)
			panic(r)
		}
		if !returned && !t.Skipped() {
			// t.Fatal runs the defers via runtime.Goexit, without a panic to
			// recover.
			t.Logf("\nfatal during %s:\n%s\n", tc.Pos, tc.Input)
		}
	}()

	stop := watchSlowCase(t, tc.Pos)
	defer stop()

	actual := f(t, tc)
	returned = true
	if actual != "" && !strings.HasSuffix(actual, "\n") {
		actual += "\n"
	}
	return actual
}

// watchSlowCase logs periodically while a case's handler runs; this is
// useful to pinpoint the cause of a test timeout. The returned stop function
// must be called when the handler completes; it waits for the watchdog
// goroutine to exit.
func watchSlowCase(t *testing.T, pos Pos) (stop func()) {
	done := make(chan struct{})
	go func() {
		start := time.Now()
		for {
			select {
			case <-done:
				return
			case <-time.After(10 * time.Second):
				t.Logf("%s: still running after %s\n", pos, time.Since(start))
			}
		}
	}()
	return func() {
		// The channel is unbuffered, so this waits until the goroutine is
		// exiting.
		done <- struct{}{}
	}
}

// reportMismatch fails the test with a readable comparison of the expected
// and actual output, using a unified diff when the output is long.
func reportMismatch(t *testing.T, tc *TestCase, actual string) {
	t.Helper()

	expectedLines := difflib.SplitLines(tc.Expected)
	actualLines := difflib.SplitLines(actual)
	if len(expectedLines) > 5 {
		// Print a unified diff if there is a lot of output to compare.
		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			Context: 5,
			A:       expectedLines,
			B:       actualLines,
		})
		if err == nil {
			t.Fatalf("\n%s:\n%s\noutput didn't match expected:\n%s",
				indentLines(tc.Pos.String()), indentLines(tc.String()), indentLines(diff))
		}
		t.Logf("Failed to produce diff %v", err)
	}
	t.Fatalf("\n%s:\n%s\nexpected:\n%s\nfound:\n%s",
		tc.Pos, indentLines(tc.String()), indentLines(tc.Expected), indentLines(actual))
}

// indentLines prefixes every non-empty line with two spaces, for readable
// multi-line failure messages.
func indentLines(str string) string {
	var b strings.Builder
	if str == "" {
		return ""
	}
	for i, l := range strings.Split(str, "\n") {
		if i > 0 {
			b.WriteString("\n")
		}
		if l != "" {
			b.WriteString("  ")
			b.WriteString(l)
		}
	}
	return b.String()
}

// tempFileRe matches files named .XXXX, XXX~ or #XXX#, which Walk ignores.
var tempFileRe = regexp.MustCompile(`(^\..*)|(.*~$)|(^#.*#$)`)

// cutExt returns the given file name with the extension removed, if there is
// one.
func cutExt(fileName string) string {
	extStart := len(fileName) - len(filepath.Ext(fileName))
	return fileName[:extStart]
}
