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
	"strings"
	"testing"
	"time"
)

func TestParseLine(t *testing.T) {
	RunTestFromString(t, `
> parse

> xx =
----
here: cannot parse directive at column 6: > xx =

> parse

> xx a=b a=c
----
here: duplicate argument "a": > xx a=b a=c

> parse

> xx a=b b=c c=[1,2,3]
----
"xx" [a=b b=c c=[1,2,3]]

> parse

> http GET ./alerts query={search: "alert 1"} header={X-Note: 'it''s ok'}
----
"http" [GET ./alerts query={search: "alert 1"} header={X-Note: 'it''s ok'}]

> parse

> http GET ./alerts query={search: "unterminated
----
here: cannot parse directive at column 27: > http GET ./alerts query={search: "unterminated

# A request target whose left-of-"=" is not name-shaped is one bare Arg: the
# "?search=x" query is not split into a name=value pair.
> parse

> http GET ./alerts?search=x format=raw
----
"http" [GET ./alerts?search=x format=raw]

# A JSONPath with "==" survives verbatim as a single bare Arg.
> parse

> http extract $.items[?(@.id==1)].name
----
"http" [extract $.items[?(@.id==1)].name]

# A name is a plain identifier: "system[]=spans:all" is not name-shaped, so it
# stays one bare Arg. The repeated query-parameter convention is written as a
# quoted key instead — see the query={'system[]': …} form.
> parse

> http GET /x system[]=spans:all
----
"http" [GET /x system[]=spans:all]

# An unterminated bracket list is a parse error.
> parse

> cmd x=[1, 2
----
here: cannot parse directive at column 9: > cmd x=[1, 2

# A flow mapping is one value: its inner spaces do not split the token.
> parse

> http GET /x query={search: alert 1, limit: 10} format=status
----
"http" [GET /x query={search: alert 1, limit: 10} format=status]

# Brackets nested in a flow mapping are scanned as a unit.
> parse

> http GET /x query={id: [1, 2], q: a b}
----
"http" [GET /x query={id: [1, 2], q: a b}]

# A brace inside a quoted value is opaque and does not change nesting.
> parse

> http GET /x query={q: '{ a }'}
----
"http" [GET /x query={q: '{ a }'}]

# An unterminated flow mapping is a parse error.
> parse

> cmd x={a: 1
----
here: cannot parse directive at column 9: > cmd x={a: 1

# A backslash escapes the next character inside a double-quoted value.
> parse

> cmd x="a\"b"
----
"cmd" [x="a\"b"]

# A quoted region inside a bare token keeps its inner spaces from splitting it.
> parse

> cmd abc"d e"f
----
"cmd" [abc"d e"f]
`, func(t *testing.T, d *TestCase) string {
		// Each case's input is itself a directive line, so it is preceded by
		// a blank line: a ">" line directly below the directive would be
		// parsed as a continuation of "> parse".
		cmd, args, err := parseLine(strings.TrimPrefix(d.Input, "\n"))
		if err != nil {
			return fmt.Errorf("here: %w", err).Error()
		}
		return fmt.Sprintf("%q %+v", cmd, args)
	})
}

// TestArgText verifies Arg.Text: quoted values and quoted bare tokens are
// unquoted per YAML scalar rules, while raw expressions (paths, JSONPaths,
// placeholders, bare words) survive verbatim.
func TestArgText(t *testing.T) {
	RunTestFromString(t, `
> text a="alert 1" b=raw c={span:s1.trace_id} d=$.items[?(@.id==1)] path e='a b'
----
a="alert 1" -> "alert 1"
b=raw -> "raw"
c={span:s1.trace_id} -> "{span:s1.trace_id}"
d=$.items[?(@.id==1)] -> "$.items[?(@.id==1)]"
path -> "path"
e='a b' -> "a b"
`, func(t *testing.T, d *TestCase) string {
		var b strings.Builder
		for _, arg := range d.Args {
			fmt.Fprintf(&b, "%s -> %q\n", arg.String(), arg.Text())
		}
		return strings.TrimRight(b.String(), "\n")
	})
}

// TestArgFields verifies Arg.Fields: keys keep file order, scalars decode by
// Text rules (verbatim text, quoted scalars unquoted), lists are flagged, and
// anything that is not a flat flow mapping is an error.
func TestArgFields(t *testing.T) {
	RunTestFromString(t, `
# Keys keep their file order, not YAML map order.
> fields v={b: 1, a: 2, c: 3}
----
b=[1]
a=[2]
c=[3]

# Scalars are raw text: timestamps, integers and durations survive verbatim.
> fields v={time_gte: 2025-01-01T00:00:00Z, limit: 30000, timeout: 300ms}
----
time_gte=[2025-01-01T00:00:00Z]
limit=[30000]
timeout=[300ms]

# Quoted scalars are unquoted; quoting is what carries spaces, commas and
# YAML flow indicators such as a JSONPath or a {entity:key} placeholder.
> fields v={search: "alert 1", path: '$.alerts[0].name', since: '{clock:epoch}'}
----
search=[alert 1]
path=[$.alerts[0].name]
since=[{clock:epoch}]

# A flow sequence yields N values and is flagged as a list.
> fields v={id: [1, 2]}
----
id=[1 2] list

# A one-element list holds the same value as the scalar form but stays flagged.
> fields v={id: [1]}
----
id=[1] list

# A value carrying a YAML flow indicator — here the ":" of a system name — must
# be quoted, in a list as well as on its own.
> fields v={system: ['spans:all', 'logs:all'], q: 'a, b'}
----
system=[spans:all logs:all] list
q=[a, b]

# A quoted key carries the repeated query-parameter "[]" suffix.
> fields v={'system[]': spans:all}
----
system[]=[spans:all]

# An empty value decodes as the empty string.
> fields v={id: }
----
id=[]

# An empty mapping has no fields.
> fields v={}
----
<no fields>

> fields v=[1, 2]
----
error: argument v: expected a {name: value} mapping

> fields v=plain
----
error: argument v: expected a {name: value} mapping

> fields v={a: {b: 1}}
----
error: argument v: key "a": nested mapping

> fields v={a: [{b: 1}]}
----
error: argument v: key "a": nested mapping

> fields v={a: 1, a: 2}
----
error: argument v: [1:8] mapping key "a" already defined at [1:2]
`, func(t *testing.T, d *TestCase) string {
		fields, err := d.NamedArg("v").Fields()
		if err != nil {
			return "error: " + err.Error()
		}
		if len(fields) == 0 {
			return "<no fields>"
		}
		var b strings.Builder
		for _, field := range fields {
			fmt.Fprintf(&b, "%s=%v", field.Name, field.Values)
			if field.IsList {
				b.WriteString(" list")
			}
			b.WriteByte('\n')
		}
		return strings.TrimRight(b.String(), "\n")
	})
}

// TestArgScan verifies Arg.Scan across scalar, slice, and map destination
// types, quoted-scalar decoding, and the missing-argument short-circuit that
// leaves the destination untouched.
func TestArgScan(t *testing.T) {
	scan := func(d *TestCase, dest any) {
		if err := d.NamedArg("vals").Scan(dest); err != nil {
			d.Fatalf(t, "scan vals: %v", err)
		}
	}

	RunTest(t, "testdata/scan_args", func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "string":
			var dest string
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "bool":
			var dest bool
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "float64":
			var dest float64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "time.Duration":
			var dest time.Duration
			scan(d, &dest)
			return dest.String()
		case "[]string":
			var dest []string
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]int":
			var dest []int
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]int64":
			var dest []int64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]uint64":
			var dest []uint64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]float64":
			var dest []float64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "map[string]string":
			var dest map[string]string
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		default:
			return fmt.Sprintf("unrecognized type %s", d.Name)
		}
	})
}

// TestArgPositional covers positional access via TestCase.Arg: an in-range
// index returns the argument, an out-of-range (or negative) index returns a
// zero Arg, and a bare positional token scans into a typed destination using
// its name as the value.
func TestArgPositional(t *testing.T) {
	f, err := Parse("f", "> cmd a b=2 15m\n----\n")
	if err != nil {
		t.Fatal(err)
	}
	tc := f.Cases[0]

	if got := tc.Arg(0); got.IsZero() || got.String() != "a" {
		t.Fatalf("Arg(0) = %q (zero=%v), want %q", got.String(), got.IsZero(), "a")
	}
	if got := tc.Arg(1); got.String() != "b=2" {
		t.Fatalf("Arg(1) = %q, want %q", got.String(), "b=2")
	}
	if got := tc.Arg(-1); !got.IsZero() {
		t.Fatalf("Arg(-1) = %q, want a zero Arg", got.String())
	}
	if got := tc.Arg(3); !got.IsZero() {
		t.Fatalf("Arg(3) = %q, want a zero Arg", got.String())
	}

	// The bare positional "15m" scans into a typed dest via its name.
	var d time.Duration
	if err := tc.Arg(2).Scan(&d); err != nil {
		t.Fatalf("Scan bare positional: %v", err)
	}
	if d != 15*time.Minute {
		t.Fatalf("Scan(15m) = %s, want 15m0s", d)
	}
}

// TestDirective renders each case's parsed command and arguments, exercising
// how the directive-line grammar (bare words, name=value, bracket lists with
// nesting, extra spaces, quoted scalars) lands in TestCase.Name/Args.
func TestDirective(t *testing.T) {
	RunTest(t, "testdata/directive", func(t *testing.T, d *TestCase) string {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "cmd: %s\n%d arguments\n", d.Name, len(d.Args))
		for _, a := range d.Args {
			fmt.Fprintf(&buf, "%s\n", a.String())
		}
		return buf.String()
	})
}
