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
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestFormatParseRoundTrip verifies the printer's core invariant,
// Format(Parse(x)) == x, over every fixture in testdata. This is what makes
// -rewrite churn-free: everything the handler didn't change prints back
// byte-for-byte.
func TestFormatParseRoundTrip(t *testing.T) {
	err := filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		t.Run(path, func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := Parse(path, string(src))
			if err != nil {
				t.Fatal(err)
			}
			if got := f.Format(); string(got) != string(src) {
				t.Fatalf("Format(Parse(x)) != x\ngot:\n%s\nwant:\n%s", got, src)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestFormatProgrammatic verifies Format renders a hand-built File (one with no
// raw directive lines) in canonical form: "> name args" from Name/Args, the
// Input field verbatim, no output section for a zero separator, and trailing
// trivia after the cases.
func TestFormatProgrammatic(t *testing.T) {
	f := &File{
		Cases: []*TestCase{{
			Name:  "cmd",
			Args:  Args{{Name: "a"}, {Name: "b", Value: []byte("2")}},
			Input: "line one\nline two",
		}},
		Trailing: []string{"# trailing note"},
	}
	want := "> cmd a b=2\nline one\nline two\n\n# trailing note\n"
	if got := string(f.Format()); got != want {
		t.Fatalf("Format() =\n%q\nwant\n%q", got, want)
	}
}
