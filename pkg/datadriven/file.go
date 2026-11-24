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
	"regexp"
	"strings"
	"testing"
)

// Pos identifies a location in a testdata file.
type Pos struct {
	// Path is the testdata file path as passed to RunTest or Parse.
	Path string
	// Line is the 1-based line number.
	Line int
}

// String renders the position as "path:line" — the format used in failure
// messages and accepted by the -focus flag.
func (p Pos) String() string {
	return fmt.Sprintf("%s:%d", p.Path, p.Line)
}

// Separator styles for a test case's expected-output section.
const (
	// sepNone marks a case without an output section (a directive at EOF).
	sepNone = ""
	// sepSingle is the common "----" separator; the expected output runs to
	// the first blank line.
	sepSingle = "----"
	// sepBlank is the "---- BEGIN" separator, paired with a closing
	// "---- END"; it brackets expected output that itself contains blank
	// lines.
	sepBlank = "---- BEGIN"
	// sepBlankEnd closes a sepBlank section.
	sepBlankEnd = "---- END"
)

// File is a parsed testdata file: an ordered list of test cases plus any
// trailing trivia. For valid input, Format(Parse(x)) == x.
type File struct {
	// Path is the file path the document was parsed from.
	Path string

	// Cases are the test cases in file order.
	Cases []*TestCase

	// Trailing holds verbatim comment/blank lines after the last case.
	Trailing []string

	// numLines is the number of lines in the source, used to bounds-check
	// -focus line targets.
	numLines int
}

// caseAt returns the index of the case containing the given 1-based line. A
// case's span runs from its directive line to the line before the next
// case's directive (or EOF), so pointing anywhere inside the case — input,
// separator, or expected output — selects it.
func (f *File) caseAt(line int) (int, error) {
	if len(f.Cases) == 0 {
		return 0, fmt.Errorf("%s: no test cases", f.Path)
	}
	if first := f.Cases[0].Pos.Line; line < first {
		return 0, fmt.Errorf("%s:%d: before the first directive (line %d)", f.Path, line, first)
	}
	if line > f.numLines {
		return 0, fmt.Errorf("%s:%d: past the end of file (%d lines)", f.Path, line, f.numLines)
	}
	target := 0
	for i, tc := range f.Cases {
		if tc.Pos.Line > line {
			break
		}
		target = i
	}
	return target, nil
}

// TestCase is one directive in a testdata file: a command with optional
// arguments, input, and expected output.
type TestCase struct {
	// Leading holds verbatim comment/blank lines above the directive line;
	// they are preserved by -rewrite.
	Leading []string

	// Pos is the position of the directive line, suitable for inclusion in
	// logs and error messages (and for the -focus flag).
	Pos Pos

	// Name is the command: the first word on the directive line.
	Name string

	// Args contains the arguments that follow the command.
	Args Args

	// Input is the text between the directive line and the ---- separator.
	Input string

	// Expected is the text below the ---- separator. Most handlers need not
	// read this: they return their own actual output and the framework
	// compares the two.
	Expected string

	// directiveLines are the raw "> cmd args" line and any ">"-continuation
	// lines as they appeared in the file, so Format reproduces them
	// byte-for-byte (extra spaces and wrapping included).
	directiveLines []string

	// inputLines are the raw input lines as they appeared in the file, for
	// byte-for-byte reproduction by Format. Input is the same text joined.
	inputLines []string

	// sep is the separator style of the expected-output section: sepSingle,
	// sepBlank, or sepNone.
	sep string
}

// String renders the directive and its input in canonical form. The result
// ends in a newline.
func (tc *TestCase) String() string {
	fields := make([]string, 0, len(tc.Args)+1)
	fields = append(fields, tc.Name)
	for i := range tc.Args {
		fields = append(fields, tc.Args[i].String())
	}
	s := "> " + strings.Join(fields, " ")
	if tc.Input == "" {
		return s + "\n"
	}
	return fmt.Sprintf("%s\n%s", s, tc.Input)
}

// Arg returns the argument at idx. Returns a zero Arg if idx is out of
// range — check with IsZero before use.
func (tc *TestCase) Arg(idx int) (arg Arg) {
	if idx < 0 || idx > len(tc.Args)-1 {
		return arg
	}
	return tc.Args[idx]
}

// NamedArg returns the argument with the given key. Returns a zero Arg if no
// such argument exists — check with IsZero before use.
func (tc *TestCase) NamedArg(key string) (arg Arg) {
	for _, a := range tc.Args {
		if a.Name == key {
			return a
		}
	}
	return arg
}

// Logf is a wrapper for tb.Logf which adds file position information, so
// that it's easy to locate the source of the log.
func (tc *TestCase) Logf(tb testing.TB, format string, args ...any) {
	tb.Helper()
	tb.Logf("%s: %s", tc.Pos, fmt.Sprintf(format, args...))
}

// Fatalf wraps a fatal testing error with test file position information, so
// that it's easy to locate the source of the error.
func (tc *TestCase) Fatalf(tb testing.TB, format string, args ...any) {
	tb.Helper()
	tb.Fatalf("%s: %s", tc.Pos, fmt.Sprintf(format, args...))
}

// setResult replaces the expected output with actual and picks the matching
// separator style. Used by -rewrite and ClearResults.
func (tc *TestCase) setResult(actual string) {
	tc.Expected = actual
	if hasBlankLine(actual) {
		tc.sep = sepBlank
	} else {
		tc.sep = sepSingle
	}
}

// hasBlankLine returns true iff `s` contains at least one line that's empty
// or contains only whitespace.
func hasBlankLine(s string) bool {
	return blankLineRe.MatchString(s)
}

// blankLineRe matches lines that contain only whitespaces (or entirely
// empty/blank lines). We use the "m" flag for "multiline" mode so that "^"
// can match the beginning of individual lines inside the input, not just the
// beginning of the input. In multiline mode, "$" also matches the end of
// lines. However, note how the regexp uses "\n" to match the end of lines
// instead of "$". This is because of an oddity in the Go regexp engine: at
// the very end of the input, *after the final \n in the input*, Go estimates
// there is still one more line containing no characters but that matches the
// "^.*$" regexp. The result of this oddity is that an input text like
// "foo\n" would match as "foo\n" (no match) + "" (yes match). We don't want
// that final match to be included, so we force the end-of-line match using
// "\n" specifically.
var blankLineRe = regexp.MustCompile(`(?m)^[\t ]*\n`)
