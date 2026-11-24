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
	"path/filepath"
	"reflect"
	"testing"
)

// TestParseFileMissing verifies ParseFile surfaces the read error for a
// nonexistent path instead of returning a nil file and nil error.
func TestParseFileMissing(t *testing.T) {
	if _, err := ParseFile(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("ParseFile(missing) = nil error, want an error")
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name:    "missing directive prefix",
			src:     "cmd without prefix\n",
			wantErr: `f:1: directive line must start with "> ", got: cmd without prefix`,
		},
		{
			name:    "bad directive syntax",
			src:     "> xx =\n",
			wantErr: "f:1: cannot parse directive at column 6: > xx =",
		},
		{
			name:    "duplicate argument",
			src:     "> cmd a=1 a=2\n",
			wantErr: `f:1: duplicate argument "a": > cmd a=1 a=2`,
		},
		{
			name:    "duplicate bare argument",
			src:     "> cmd verbose verbose\n",
			wantErr: `f:1: duplicate argument "verbose": > cmd verbose verbose`,
		},
		{
			// Multiple values for one key use a bracket list instead of
			// repeating the argument.
			name: "bracket list is the repeated-value form",
			src:  "> http GET /x query={id: [1, 2]}\n----\n",
		},
		{
			// The old wrapping syntax is rejected outright instead of taking
			// the "\" as an argument.
			name:    "trailing backslash",
			src:     "> cmd a \\\n>   b\n----\n",
			wantErr: `f:1: trailing "\" in directive; continue long lines with a leading ">" instead`,
		},
		{
			name:    "EOF without ---- END",
			src:     "> cmd\n---- BEGIN\noutput\n",
			wantErr: `f:1: EOF without "---- END"`,
		},
		{
			name:    "non-blank line after ---- END",
			src:     "> cmd\n---- BEGIN\noutput\n---- END\n> next\n----\n",
			wantErr: `f:5: non-blank line after "---- END"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("f", tt.src)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err)
			}
		})
	}
}

// TestParseTrivia verifies comment/blank attachment: trivia above a
// directive becomes its Leading, trivia after the last case becomes the
// file's Trailing, and #-lines inside input are content.
func TestParseTrivia(t *testing.T) {
	f, err := Parse("f", `# header

> cmd
# part of input
----
out

# trailing note
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(f.Cases))
	}
	tc := f.Cases[0]
	if want := []string{"# header", ""}; !reflect.DeepEqual(tc.Leading, want) {
		t.Fatalf("Leading = %q, want %q", tc.Leading, want)
	}
	if want := "# part of input"; tc.Input != want {
		t.Fatalf("Input = %q, want %q", tc.Input, want)
	}
	if want := "out\n"; tc.Expected != want {
		t.Fatalf("Expected = %q, want %q", tc.Expected, want)
	}
	if want := []string{"# trailing note"}; !reflect.DeepEqual(f.Trailing, want) {
		t.Fatalf("Trailing = %q, want %q", f.Trailing, want)
	}
	if tc.Pos.Line != 3 {
		t.Fatalf("Pos.Line = %d, want 3", tc.Pos.Line)
	}
}

// TestLineContinuation verifies that a directive continued with ">"-prefixed
// lines parses as one logical line (args from all physical lines, input
// untouched) and that the wrapped layout survives rewrite byte-for-byte.
func TestLineContinuation(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "args wrapped across lines",
			src: `> http GET ./alerts
>   query={status: open, limit: 10}
>   format=status
body line
----
"http" [GET ./alerts query={status: open, limit: 10} format=status] input="body line"
`,
		},
		{
			// A flow mapping split mid-value: the continuation joins with a
			// single space, so the value reassembles.
			name: "flow mapping split mid-value",
			src: `> http GET ./alerts query={status: open,
>   limit: 10} format=status
body line
----
"http" [GET ./alerts query={status: open, limit: 10} format=status] input="body line"
`,
		},
	}

	handler := func(t *testing.T, tc *TestCase) string {
		return fmt.Sprintf("%q %+v input=%q", tc.Name, tc.Args, tc.Input)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse("f", tt.src)
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Cases) != 1 {
				t.Fatalf("expected 1 case, got %d", len(f.Cases))
			}

			RunTestFromString(t, tt.src, handler)

			// The raw wrapped lines print back verbatim.
			if got := f.Format(); string(got) != tt.src {
				t.Fatalf("Format(Parse(x)) != x\ngot:\n%s\nwant:\n%s", got, tt.src)
			}
		})
	}
}

// TestInputStartingWithMarker documents the cost of marker-based
// continuation: a ">" line right below the directive is always a
// continuation, so input that must start with ">" is preceded by a blank
// line.
func TestInputStartingWithMarker(t *testing.T) {
	f, err := Parse("f", `> cmd

> not a continuation
----
out
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(f.Cases))
	}
	tc := f.Cases[0]
	if len(tc.Args) != 0 {
		t.Fatalf("Args = %+v, want none", tc.Args)
	}
	if want := "\n> not a continuation"; tc.Input != want {
		t.Fatalf("Input = %q, want %q", tc.Input, want)
	}
}
