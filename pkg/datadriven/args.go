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
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
	yamlast "github.com/goccy/go-yaml/ast"
	yamlparser "github.com/goccy/go-yaml/parser"
)

// Arg is one argument on a directive line, specified in one of the forms:
//   - argument
//   - argument=value
//   - argument=[values, ...]
//   - argument={name: value, ...}
//
// A token is read as name=value only when the text left of its first "="
// is name-shaped (an identifier); otherwise the whole token is a single bare
// argument. So "./alerts?search=x", "$.items[?(@.id==1)]", "system[]=spans:all"
// and "{span:s1.trace_id}" each parse as one bare argument whose value survives
// verbatim, while "format=raw" and "query={id: [1, 2]}" parse as name=value.
// See parseLine for the full grammar.
//
// A bare argument (no "=") is both positional and named: in
// "> http GET ./alerts" the bare words are positional values read via
// Arg(i).Name, while in "> eval verbose" the bare word is a presence flag
// looked up via NamedArg("verbose"). The syntax cannot distinguish the two
// roles, so both readings are supported.
type Arg struct {
	// Name is the argument name, or the bare token itself for a positional
	// argument.
	Name string
	// Value is the raw argument value, or nil for a bare argument.
	Value []byte
}

// Args is the list of arguments on a directive line, in file order.
type Args []Arg

// IsZero reports whether the argument is a zero value (empty Name). A zero
// Arg is returned by Arg and NamedArg when the requested argument does not
// exist.
func (arg Arg) IsZero() bool {
	return arg.Name == ""
}

// String renders the argument as it would appear on a directive line.
func (arg Arg) String() string {
	if arg.Value == nil {
		return arg.Name
	}
	return arg.Name + "=" + string(arg.Value)
}

// Text returns the argument value as decoded text: a quoted value (or quoted
// bare token) is unquoted per YAML scalar rules, everything else is returned
// verbatim. For a bare argument it returns the bare token.
//
// Text is the single decoding rule for commands that consume a raw expression
// — a request path, a JSONPath, a {entity:key.field} placeholder. Unlike Scan
// it never interprets the value as a YAML flow collection, so colons, braces
// and brackets survive untouched; unlike a bracket list it is not split.
func (arg Arg) Text() string {
	s := string(arg.Value)
	if arg.Value == nil {
		s = arg.Name
	}
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		var out string
		if err := yaml.Unmarshal([]byte(s), &out); err == nil {
			return out
		}
	}
	return s
}

// Scan unmarshals the argument value into dest via YAML. If the argument is
// zero (key not found), dest is left unchanged and nil is returned. For a
// bare argument (no value), the Name is decoded as the value — this lets a
// positional argument like "> check 15m" be scanned into a typed dest.
func (arg Arg) Scan(dest any) error {
	if arg.IsZero() {
		return nil
	}
	value := arg.Value
	if arg.Value == nil {
		value = []byte(arg.Name)
	}
	return yaml.Unmarshal(value, dest)
}

// Field is one key of a flow-mapping argument value, decoded by Text rules.
type Field struct {
	// Name is the mapping key.
	Name string
	// Values holds one element for a scalar value, or N for a flow sequence.
	Values []string
	// IsList reports whether the value was written as a flow sequence, so a
	// caller that only accepts a scalar can reject "{k: [a, b]}" while still
	// treating "{k: [a]}" and "{k: a}" as equivalent one-value forms.
	IsList bool
}

// Fields parses the argument value as a YAML flow mapping — "{a: 1, b: [c, d]}"
// — and returns its keys in file order.
//
// Scalars decode by Text rules, not by YAML type inference: the raw scalar text
// is returned, so timestamps, integers and durations survive byte-for-byte,
// while a quoted scalar is unquoted. A nested mapping, or a value that is
// neither a scalar nor a flow sequence, is an error.
func (arg Arg) Fields() ([]Field, error) {
	if arg.IsZero() {
		return nil, nil
	}
	if len(arg.Value) == 0 || arg.Value[0] != '{' {
		return nil, fmt.Errorf("argument %s: expected a {name: value} mapping", arg.Name)
	}

	file, err := yamlparser.ParseBytes(arg.Value, 0)
	if err != nil {
		// goccy pretty-prints a parse error over several lines, quoting the
		// source and pointing at the column. The value here is a single short
		// line the reader already sees in the directive, so keep the message
		// itself — "[1:8] mapping key "a" already defined at [1:2]" — and drop
		// the quoted-source lines.
		message, _, _ := strings.Cut(err.Error(), "\n")
		return nil, fmt.Errorf("argument %s: %s", arg.Name, message)
	}
	if len(file.Docs) != 1 {
		return nil, fmt.Errorf("argument %s: expected a single {name: value} mapping", arg.Name)
	}

	// goccy represents a one-pair mapping as a bare MappingValueNode and a
	// multi-pair mapping as a MappingNode holding them.
	var pairs []*yamlast.MappingValueNode
	switch body := file.Docs[0].Body.(type) {
	case *yamlast.MappingNode:
		pairs = body.Values
	case *yamlast.MappingValueNode:
		pairs = []*yamlast.MappingValueNode{body}
	default:
		return nil, fmt.Errorf("argument %s: expected a {name: value} mapping", arg.Name)
	}

	fields := make([]Field, 0, len(pairs))
	for _, pair := range pairs {
		// A duplicate key is rejected by the YAML parser above, so keys here are
		// unique.
		name := pair.Key.GetToken().Value

		field := Field{Name: name}
		switch value := pair.Value.(type) {
		case *yamlast.SequenceNode:
			field.IsList = true
			field.Values = make([]string, 0, len(value.Values))
			for _, item := range value.Values {
				if _, ok := item.(*yamlast.SequenceNode); ok {
					return nil, fmt.Errorf("argument %s: key %q: nested list", arg.Name, name)
				}
				switch item.(type) {
				case *yamlast.MappingNode, *yamlast.MappingValueNode:
					return nil, fmt.Errorf("argument %s: key %q: nested mapping", arg.Name, name)
				}
				field.Values = append(field.Values, item.GetToken().Value)
			}
		case *yamlast.NullNode:
			// An empty value ("{id: }") is the empty string, matching "id=".
			field.Values = []string{""}
		case *yamlast.MappingNode, *yamlast.MappingValueNode:
			return nil, fmt.Errorf("argument %s: key %q: nested mapping", arg.Name, name)
		default:
			field.Values = []string{value.GetToken().Value}
		}
		fields = append(fields, field)
	}
	return fields, nil
}

// parseLine parses a datadriven directive line and returns the parsed
// command and Args.
//
// A directive line starts with "> " followed by a command and an optional
// list of space-separated tokens. The grammar is:
//
//	directive := "> " command token*
//	token     := quoted | bracketed | braced | word   # split on spaces,
//	                                          # honoring quotes and [ ] / { }
//	                                          # nesting
//	arg       := name "=" value   # only when the text left of the first "="
//	                              #   is name-shaped:
//	                              #   name := [A-Za-z_][A-Za-z0-9_.-]*
//	           | bareWord         # otherwise the whole token is one bare Arg
//
// The named forms are:
//   - <argname>                            # No value (bare/presence).
//   - <argname>=<value>                    # Single value.
//   - <argname>=[<value1>, <value2>, ...]  # Multiple values.
//   - <argname>={<name>: <value>, ...}     # A flow mapping, read via Fields.
//
// Values may be quoted (YAML scalar rules), bracketed or braced; nesting is
// honored, so
//
//	> cmd exprs=[a + [b + c], d + f]
//	> http GET /x query={id: [1, 2], q: 'a b'}
//
// are both valid. Because name=value is recognized only when the prefix before the
// first "=" is name-shaped, a token whose left side is not an identifier —
// such as "./alerts?search=x" or "$.items[?(@.id==1)]" — parses as one bare
// Arg with the value intact, needing no reconstruction downstream.
//
// Repeating an argument name is an error: NamedArg lookup could only ever
// see one of the duplicates. Pass multiple values as a bracket list instead
// (e.g. "> http GET /x query={id: [1, 2]}").
//
// A line that does not start with "> " returns ("", nil, nil).
func parseLine(line string) (name string, args Args, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil, nil
	}
	// Command lines must start with "> ".
	after, ok := strings.CutPrefix(line, "> ")
	if !ok {
		return "", nil, nil
	}
	origLine := line
	line = after

	defer func() {
		if r := recover(); r != nil {
			if r == (parseError{}) {
				column := len(origLine) - len(line) + 1
				name = ""
				args = nil
				err = fmt.Errorf("cannot parse directive at column %d: %s", column, origLine)
				// Note: to debug an unexpected parsing error, this is a good place to
				// add a debug.PrintStack().
			} else {
				panic(r)
			}
		}
	}()

	// until removes the prefix up to one of the given characters from line and
	// returns the prefix.
	until := func(chars string) string {
		idx := strings.IndexAny(line, chars)
		if idx == -1 {
			idx = len(line)
		}
		res := line[:idx]
		line = line[idx:]
		return res
	}

	name = until(" ")
	if name == "" {
		panic(parseError{})
	}
	line = strings.TrimSpace(line)

	for line != "" {
		var arg Arg

		// A token is name=value only when its left side, up to the first "=",
		// is name-shaped. Otherwise the whole token is one bare Arg — this is
		// what keeps URLs, JSONPaths and placeholders intact.
		if n := nameLen(line); n > 0 && n < len(line) && line[n] == '=' {
			arg.Name = line[:n]
			// Skip the name and the '='.
			line = line[n+1:]

			switch {
			case line == "" || line[0] == ' ':
				// Empty value.
				arg.Value = []byte{}
			case line[0] == '"' || line[0] == '\'':
				arg.Value, line = parseQuotedValue(line)
			case line[0] == '[':
				// Bracket value: capture raw bytes including the brackets.
				k := scanBracket(line)
				arg.Value = []byte(line[:k])
				line = strings.TrimSpace(line[k:])
			case line[0] == '{':
				// Flow-mapping value: capture raw bytes including the braces.
				k := scanBrace(line)
				arg.Value = []byte(line[:k])
				line = strings.TrimSpace(line[k:])
			default:
				// Single value: capture raw bytes up to the next space.
				arg.Value = []byte(until(" "))
			}
		} else {
			// Bare token: consume it verbatim up to the next top-level space,
			// honoring quotes and bracket nesting.
			tok := scanBareToken(line)
			if tok == "" {
				// A leading "=" (empty argument name) or otherwise unscannable
				// token.
				panic(parseError{})
			}
			arg.Name = tok
			line = line[len(tok):]
		}

		for _, prev := range args {
			if prev.Name == arg.Name {
				return "", nil, fmt.Errorf("duplicate argument %q: %s", arg.Name, origLine)
			}
		}
		args = append(args, arg)
		line = strings.TrimSpace(line)
	}
	return name, args, nil
}

// isNameStart reports whether c can begin an argument name.
func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isNameBody reports whether c can continue an argument-name identifier.
func isNameBody(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9') || c == '.' || c == '-'
}

// nameLen returns the length of the argument-name prefix at the start of line,
// or 0 if line does not begin with a name. A name is an identifier
// ([A-Za-z_][A-Za-z0-9_.-]*). A name carrying anything else — the "[]" of the
// repeated query-parameter convention, say — is written as a quoted key inside
// a flow mapping (`query={'system[]': spans:all}`).
func nameLen(line string) int {
	if line == "" || !isNameStart(line[0]) {
		return 0
	}
	i := 1
	for i < len(line) && isNameBody(line[i]) {
		i++
	}
	return i
}

// scanBareToken returns the bare token at the start of line — every byte up to
// the next top-level space, treating quoted and bracketed regions as opaque so
// their inner spaces do not split the token. It returns "" when line begins
// with "=", which the caller rejects as an empty argument name.
func scanBareToken(line string) string {
	if line == "" || line[0] == '=' {
		return ""
	}
	pos := 0
	for pos < len(line) {
		switch c := line[pos]; c {
		case ' ':
			return line[:pos]
		case '"', '\'':
			pos += scanQuoted(line[pos:])
		case '[':
			pos += scanBracket(line[pos:])
		default:
			_, size := utf8.DecodeRuneInString(line[pos:])
			pos += size
		}
	}
	return line
}

// scanBracket returns the byte length of the bracketed list beginning at
// line[0] == '[', including the outer brackets and honoring nesting. Quoted
// scalars are treated as opaque, so a bracket inside a quoted string — e.g. the
// "a]b" literal in a JSONPath filter $.items[?(@.name=="a]b")] — does not change
// nesting. A brace inside brackets is an ordinary byte: bare tokens such as
// $.items[?(@.id=={x})] must not gain a brace-balance requirement. It panics
// with parseError on an unterminated bracket.
func scanBracket(line string) int {
	return scanNested(line, '[', ']', false)
}

// scanBrace returns the byte length of the flow mapping beginning at
// line[0] == '{', including the outer braces and honoring nesting. Quoted
// scalars are opaque, and a nested bracketed list is scanned as a unit, so
// "{id: [1, 2], q: '{ a }'}" scans as one value. It panics with parseError on
// an unterminated brace.
func scanBrace(line string) int {
	return scanNested(line, '{', '}', true)
}

// scanNested returns the byte length of the region beginning at
// line[0] == openByte and ending at its matching closeByte, honoring nesting of
// that delimiter pair and treating quoted scalars as opaque. When nestBrackets
// is set, a nested bracketed list is scanned as an opaque unit; otherwise only
// the openByte/closeByte pair is tracked. It panics with parseError when the
// region is unterminated.
func scanNested(line string, openByte, closeByte byte, nestBrackets bool) int {
	pos := 1
	nestLevel := 1
	for nestLevel > 0 {
		if pos == len(line) {
			// The string ended before we found the matching close delimiter.
			panic(parseError{})
		}
		switch c := line[pos]; {
		case c == '"' || c == '\'':
			pos += scanQuoted(line[pos:])
		case nestBrackets && c == '[':
			pos += scanBracket(line[pos:])
		case c == openByte:
			nestLevel++
			pos++
		case c == closeByte:
			nestLevel--
			pos++
		default:
			_, size := utf8.DecodeRuneInString(line[pos:])
			pos += size
		}
	}
	return pos
}

// parseQuotedValue returns the raw YAML scalar bytes for a single-quoted or
// double-quoted argument value and the unparsed remainder of the directive
// line. It preserves the surrounding quotes so Arg.Scan can decode the scalar
// using YAML rules.
func parseQuotedValue(line string) ([]byte, string) {
	n := scanQuoted(line)
	return []byte(line[:n]), strings.TrimSpace(line[n:])
}

// scanQuoted returns the byte length of the single- or double-quoted scalar
// beginning at line[0], including both surrounding quotes. It panics with
// parseError on an unterminated quoted scalar.
func scanQuoted(line string) int {
	quote := line[0]
	pos := 1

	for pos < len(line) {
		switch quote {
		case '"':
			if line[pos] == '\\' {
				// A trailing backslash deliberately runs pos past len(line), so the
				// loop exits and we report an unterminated quoted scalar below.
				pos += 2
				continue
			}
			if line[pos] == '"' {
				return pos + 1
			}
		case '\'':
			if line[pos] == '\'' {
				if pos+1 < len(line) && line[pos+1] == '\'' {
					pos += 2
					continue
				}
				return pos + 1
			}
		}
		pos++
	}

	panic(parseError{})
}

// parseError is panicked by the directive-line parser as internal control
// flow; parseLine recovers it and converts it into an error value with the
// failing column.
type parseError struct{}
