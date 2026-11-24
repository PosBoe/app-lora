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
	"strings"
)

// ParseFile reads and parses the testdata file at path.
func ParseFile(path string) (*File, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is a testdata file the test itself names
	if err != nil {
		return nil, err
	}
	return Parse(path, string(data))
}

// Parse parses testdata file contents into a document. The name is recorded
// as File.Path and used in positions and error messages; it does not have to
// exist on disk.
//
// Errors carry a "path:line" position. Parsing is pure: it touches neither
// the testing package nor any global flags.
func Parse(name, src string) (*File, error) {
	p := &parser{
		file:  &File{Path: name},
		lines: splitLines(src),
	}
	p.file.numLines = len(p.lines)

	if err := p.parse(); err != nil {
		return nil, err
	}
	return p.file, nil
}

// parser holds the state of one Parse call: the source split into lines and
// a cursor.
type parser struct {
	file  *File
	lines []string
	idx   int // 0-based index of the next line to consume
}

// pos returns the position of the next line to consume.
func (p *parser) pos() Pos {
	return Pos{Path: p.file.Path, Line: p.idx + 1}
}

// parse consumes all lines, appending cases and trailing trivia to p.file.
func (p *parser) parse() error {
	// trivia accumulates comment/blank lines until the next directive claims
	// them as Leading (or EOF leaves them as the file's Trailing).
	var trivia []string

	for p.idx < len(p.lines) {
		raw := p.lines[p.idx]
		line := strings.TrimSpace(raw)

		if line == "" || strings.HasPrefix(line, "#") {
			// Comments and blank lines at directive scope are inert trivia,
			// preserved verbatim.
			trivia = append(trivia, raw)
			p.idx++
			continue
		}

		tc, err := p.parseCase(raw, line)
		if err != nil {
			return err
		}
		tc.Leading = trivia
		trivia = nil
		p.file.Cases = append(p.file.Cases, tc)
	}

	p.file.Trailing = trivia
	return nil
}

// parseCase consumes one directive line (plus any ">"-continuation lines),
// its input, and its expected-output section. raw is the verbatim directive
// line, line its trimmed form.
func (p *parser) parseCase(raw, line string) (*TestCase, error) {
	pos := p.pos()

	tc := &TestCase{
		Pos:            pos,
		directiveLines: []string{raw},
	}
	p.idx++

	// A long directive continues on the next line when that line starts with
	// ">" too:
	//
	//	> http GET ./alerts
	//	>   query={status: open}
	//
	// Continuation is decided by the marker alone, so the first line of a
	// case's input can never start with ">"; an input that must begin that
	// way starts with a blank line instead.
	for {
		if strings.HasSuffix(line, `\`) {
			// The old wrapping syntax, on the directive line or on one of its
			// continuations. Reject it rather than letting parseLine take the
			// bare "\" as an argument.
			return nil, fmt.Errorf(
				`%s: trailing "\" in directive; continue long lines with a leading ">" instead`, pos)
		}
		if p.idx == len(p.lines) {
			break
		}
		next := p.lines[p.idx]
		cont, ok := continuation(next)
		if !ok {
			break
		}
		tc.directiveLines = append(tc.directiveLines, next)
		if cont != "" {
			line += " " + cont
		}
		p.idx++
	}

	name, args, err := parseLine(line)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", pos, err)
	}
	if name == "" {
		// Non-blank, non-comment line that is not a directive. This is almost
		// certainly a missing "> " prefix — fail loudly rather than silently
		// dropping the case.
		return nil, fmt.Errorf("%s: directive line must start with \"> \", got: %s", pos, line)
	}
	tc.Name = name
	tc.Args = args

	// Input runs from the directive line to the separator (or EOF, for a
	// directive without an output section). Blank lines and #-lines inside
	// the input are content, not trivia.
	for p.idx < len(p.lines) {
		l := p.lines[p.idx]
		if l == sepSingle || l == sepBlank {
			tc.sep = l
			p.idx++
			break
		}
		tc.inputLines = append(tc.inputLines, l)
		p.idx++
	}
	tc.Input = strings.Join(tc.inputLines, "\n")

	switch tc.sep {
	case sepSingle:
		tc.Expected = p.parseExpected()
	case sepBlank:
		tc.Expected, err = p.parseExpectedBlank(pos)
		if err != nil {
			return nil, err
		}
	}
	return tc, nil
}

// continuation reports whether raw continues the directive above it — a line
// whose trimmed form is ">" or starts with "> " or ">\t" — and returns its
// contribution to the logical directive line, with the marker and the
// surrounding whitespace stripped.
func continuation(raw string) (string, bool) {
	line := strings.TrimSpace(raw)
	rest, ok := strings.CutPrefix(line, ">")
	if !ok {
		return "", false
	}
	if rest != "" && !strings.HasPrefix(rest, " ") && !strings.HasPrefix(rest, "\t") {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

// parseExpected consumes a "----" expected-output section: it runs to the
// first blank line (consumed as the case's structural terminator) or EOF.
func (p *parser) parseExpected() string {
	var b strings.Builder
	for p.idx < len(p.lines) {
		l := p.lines[p.idx]
		if strings.TrimSpace(l) == "" {
			// The blank line terminating the section belongs to the case;
			// Format regenerates it.
			p.idx++
			break
		}
		b.WriteString(l)
		b.WriteByte('\n')
		p.idx++
	}
	return b.String()
}

// parseExpectedBlank consumes a "---- BEGIN" expected-output section: it
// runs to the "---- END" marker and may contain blank lines. casePos is the
// directive position, used in error messages.
func (p *parser) parseExpectedBlank(casePos Pos) (string, error) {
	var b strings.Builder
	for {
		if p.idx >= len(p.lines) {
			return "", fmt.Errorf("%s: EOF without %q", casePos, sepBlankEnd)
		}
		l := p.lines[p.idx]
		if l == sepBlankEnd {
			p.idx++
			break
		}
		b.WriteString(l)
		b.WriteByte('\n')
		p.idx++
	}

	// Consume the blank line after ---- END (the case's structural
	// terminator, like the blank ending a ---- section).
	if p.idx < len(p.lines) {
		if p.lines[p.idx] != "" {
			return "", fmt.Errorf("%s: non-blank line after %q", p.pos(), sepBlankEnd)
		}
		p.idx++
	}
	return b.String(), nil
}

// splitLines splits src into lines without their trailing newline, matching
// bufio.Scanner: a final newline does not produce an empty last line.
func splitLines(src string) []string {
	lines := strings.Split(src, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines
}
