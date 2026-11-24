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
	"bytes"
	"strings"
)

// Format renders the document back to the testdata file format. Comment and
// blank-line trivia, directive lines, and input print back verbatim, so a
// parsed-then-formatted file is byte-identical to its source; only cases
// whose expected output was replaced (by -rewrite or ClearResults) change.
// Formatting is pure: it performs no I/O.
func (f *File) Format() []byte {
	var b bytes.Buffer
	for i, tc := range f.Cases {
		if i > 0 {
			// A blank line separates a case's expected output from what
			// follows; the parser consumed it as the section terminator.
			b.WriteByte('\n')
		}
		tc.format(&b)
	}
	if len(f.Trailing) > 0 {
		if len(f.Cases) > 0 {
			b.WriteByte('\n')
		}
		writeLines(&b, f.Trailing)
	}
	return b.Bytes()
}

// format renders one case: leading trivia, directive line, input, and the
// expected-output section in its separator style.
func (tc *TestCase) format(b *bytes.Buffer) {
	writeLines(b, tc.Leading)

	if len(tc.directiveLines) > 0 {
		writeLines(b, tc.directiveLines)
	} else {
		// A programmatically built case has no raw directive line; render
		// the canonical form.
		b.WriteString("> " + tc.Name)
		for _, arg := range tc.Args {
			b.WriteString(" " + arg.String())
		}
		b.WriteByte('\n')
	}

	if tc.inputLines != nil {
		writeLines(b, tc.inputLines)
	} else if tc.Input != "" {
		b.WriteString(tc.Input)
		if !strings.HasSuffix(tc.Input, "\n") {
			b.WriteByte('\n')
		}
	}

	switch tc.sep {
	case sepSingle:
		b.WriteString(sepSingle)
		b.WriteByte('\n')
		b.WriteString(tc.Expected)
	case sepBlank:
		b.WriteString(sepBlank)
		b.WriteByte('\n')
		b.WriteString(tc.Expected)
		b.WriteString(sepBlankEnd)
		b.WriteByte('\n')
	case sepNone:
		// Directive without an output section.
	}
}

// writeLines writes each line verbatim, newline-terminated.
func writeLines(b *bytes.Buffer, lines []string) {
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
}
