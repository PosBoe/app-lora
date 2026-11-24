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

import "testing"

func TestString(t *testing.T) {
	RunTest(t, "testdata/string", func(t *testing.T, d *TestCase) string {
		return d.String()
	})
}

// TestCaseAt exercises the -focus line-to-case mapping: a case's span runs
// from its directive line to the line before the next directive, and lines
// outside any case are errors.
func TestCaseAt(t *testing.T) {
	f, err := Parse("f", `# comment
> first
----
out

> second
input
----
out
`)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		line    int
		want    int // case index
		wantErr string
	}{
		{line: 1, wantErr: "f:1: before the first directive (line 2)"},
		{line: 2, want: 0}, // directive line
		{line: 3, want: 0}, // separator
		{line: 4, want: 0}, // expected output
		{line: 5, want: 0}, // blank line before the next case
		{line: 6, want: 1}, // second directive
		{line: 8, want: 1}, // second separator
		{line: 9, want: 1}, // last line
		{line: 42, wantErr: "f:42: past the end of file (9 lines)"},
	}
	for _, tt := range tests {
		got, err := f.caseAt(tt.line)
		if tt.wantErr != "" {
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("caseAt(%d) error = %v, want %q", tt.line, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("caseAt(%d): %v", tt.line, err)
		}
		if got != tt.want {
			t.Fatalf("caseAt(%d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

// TestCaseAtNoCases verifies caseAt reports an error, rather than selecting a
// phantom case, when the file has no directives (only comments).
func TestCaseAtNoCases(t *testing.T) {
	f, err := Parse("f", "# just a comment\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.caseAt(1); err == nil || err.Error() != "f: no test cases" {
		t.Fatalf("caseAt on a comment-only file = %v, want %q", err, "f: no test cases")
	}
}
