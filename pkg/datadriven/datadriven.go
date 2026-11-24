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

// Package datadriven runs data-driven tests. Test cases live in plain text
// files: each case declares a command with optional arguments and input,
// followed by the expected output:
//
//	> <command> [arg | arg=val | arg=[val1, val2, ...]]...
//	<input to the command>
//	----
//	<expected output>
//
// A handler receives each case as a *TestCase and returns the actual output,
// which the framework compares against the expected output. Pass -rewrite to
// replace the expected output with the actual output instead of failing.
//
// Lines starting with # are comments; they never affect execution and are
// preserved byte-for-byte by -rewrite. Expected output that itself contains
// blank lines is bracketed with "---- BEGIN" / "---- END" instead of "----".
//
// The package is a pipeline over an explicit document model: Parse produces
// a *File, RunTest executes its cases, and Format renders the (possibly
// updated) document back to disk. See README.md for the full format
// specification and usage patterns.
//
// Forked from github.com/cockroachdb/datadriven.
package datadriven

import (
	"flag"
	"testing"
)

var (
	// rewriteTestFiles is the -rewrite flag: replace expected output with
	// actual output and write the test files back.
	rewriteTestFiles = flag.Bool(
		"rewrite", false,
		"ignore the expected results and rewrite the test files with the actual results from this "+
			"run. Used to update tests when a change affects many cases; please verify the testfile "+
			"diffs carefully!",
	)

	// quietLog is the -datadriven-quiet flag: suppress per-case logging.
	quietLog = flag.Bool(
		"datadriven-quiet", false,
		"avoid echoing the directives and responses from test files.",
	)

	// focusFlag is the -focus flag: run only one testdata file, or with
	// ":<line>", only the case containing that line (cases above it run
	// un-compared to build state; cases below it are skipped).
	focusFlag = flag.String(
		"focus", "",
		"run only the testdata file at <path>; with <path>:<line>, run the file up to the case "+
			"containing that line and compare only that case. Copy the path:line from a test "+
			"failure message.",
	)
)

// Rewriting reports whether -rewrite was passed.
//
// A test that narrows its corpus for speed — usually under testing.Short() —
// must widen back to the full set when this is true. A rewrite that reaches
// only part of a corpus leaves the rest stale while the corpus looks freshly
// regenerated, which is the worst of both: the files disagree with each other
// and nothing on the push path says so.
func Rewriting() bool { return *rewriteTestFiles }

// verbose reports whether per-case logging is enabled (-v without
// -datadriven-quiet).
func verbose() bool {
	return testing.Verbose() && !*quietLog
}
