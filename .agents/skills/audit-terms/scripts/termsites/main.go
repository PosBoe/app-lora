// Command termsites prints the evidence of a term audit from Go syntax alone.
// It does not type-check, so it runs on a tree that does not build.
//
//	go run ./.agents/skills/audit-terms/scripts/termsites [-importers] sites <root>... <term>
//	go run ./.agents/skills/audit-terms/scripts/termsites surface <dir>...
//	go run ./.agents/skills/audit-terms/scripts/termsites [-importers] glossary <CONTEXT.md> <root>...
//	go run ./.agents/skills/audit-terms/scripts/termsites importers <dir>
//
// sites lists every declaration and parameter whose name contains the term,
// grouped by the type of the value it names. Two groups are the tell of one
// name for two concepts. It also lists the string literals that contain the
// term.
//
// surface lists the exported names, the string enum values, and the wire keys
// of each package. It includes methods and declarations inside blocks. It also
// lists look-alike name pairs, names that carry no concept, type names reused
// for values of another type, and clipped words whose full form the scope
// also writes.
//
// glossary compares each headword with the identifier it names. It also sweeps
// the ambiguity index and the _Avoid_ lines against the code.
//
// With -importers, each root expands to the package in it plus every package
// in the module that imports it.
//
// importers prints the package directory and the directory of every package
// in the module that imports it. Use it as the search root of a scope that no
// CONTEXT.md owns.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func main() {
	tests := flag.Bool("tests", false, "include _test.go files")
	withImporters := flag.Bool("importers", false, "expand each root to the package and its importers")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: termsites [-tests] [-importers] sites <root>... <term> | surface <dir>... | glossary <CONTEXT.md> <root>... | importers <dir>")
	}
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		flag.Usage()
		os.Exit(2)
	}

	switch args[0] {
	case "sites":
		if len(args) < 3 {
			flag.Usage()
			os.Exit(2)
		}
		roots := args[1 : len(args)-1]
		if *withImporters {
			roots = expandImporters(roots)
		}
		runSites(roots, args[len(args)-1], *tests)
	case "surface":
		for _, dir := range args[1:] {
			runSurface(dir, *tests)
		}
	case "glossary":
		if len(args) < 3 {
			flag.Usage()
			os.Exit(2)
		}
		roots := args[2:]
		if *withImporters {
			roots = expandImporters(roots)
		}
		runGlossary(args[1], roots, *tests)
	case "importers":
		fmt.Println(strings.Join(importers(args[1]), "\n"))
	default:
		flag.Usage()
		os.Exit(2)
	}
}

//------------------------------------------------------------------------------

// site is one declaration that carries a name.
type site struct {
	kind  string // type, func, method, field, const, var
	pos   string // file:line
	pkg   string // package directory
	name  string // qualified for methods and fields: Recv.Name
	typ   string // the type of the value the name holds or returns
	value string // the string value of a const, if any
	doc   string // the first sentence of a type's doc comment
}

type parsedFile struct {
	path string
	file *ast.File
	fset *token.FileSet
}

func parseTree(root string, tests bool) ([]parsedFile, int) {
	var files []parsedFile
	skipped := 0
	fset := token.NewFileSet()
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "vendor" || name == "testdata" || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !tests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}
		if ast.IsGenerated(f) {
			skipped++
			return nil
		}
		files = append(files, parsedFile{path: path, file: f, fset: fset})
		return nil
	})
	return files, skipped
}

// declSites returns every named declaration in the file: package-level types,
// funcs, methods, consts and vars, and the fields of struct types.
func declSites(pf parsedFile) []site {
	var out []site
	pos := func(p token.Pos) string {
		position := pf.fset.Position(p)
		return fmt.Sprintf("%s:%d", position.Filename, position.Line)
	}
	pkg := filepath.Dir(pf.path)

	for _, decl := range pf.file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			s := site{kind: "func", pos: pos(decl.Name.Pos()), pkg: pkg, name: decl.Name.Name, typ: resultType(decl.Type)}
			if decl.Recv != nil && len(decl.Recv.List) > 0 {
				s.kind = "method"
				s.name = recvName(decl.Recv.List[0].Type) + "." + decl.Name.Name
			}
			out = append(out, s)
			if decl.Type.Params != nil {
				for _, field := range decl.Type.Params.List {
					for _, name := range field.Names {
						if name.Name == "_" {
							continue
						}
						out = append(out, site{kind: "param", pos: pos(name.Pos()), pkg: pkg,
							name: s.name + "(" + name.Name + ")", typ: types.ExprString(field.Type)})
					}
				}
			}
		case *ast.GenDecl:
			var lastType string
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					doc := spec.Doc
					if doc == nil && len(decl.Specs) == 1 {
						doc = decl.Doc
					}
					out = append(out, site{kind: "type", pos: pos(spec.Name.Pos()), pkg: pkg, name: spec.Name.Name,
						typ: underlying(spec.Type), doc: firstSentence(doc)})
					if st, ok := spec.Type.(*ast.StructType); ok {
						out = append(out, fieldSites(pkg, spec.Name.Name, st, pos)...)
					}
				case *ast.ValueSpec:
					kind := "var"
					if decl.Tok == token.CONST {
						kind = "const"
					}
					typ := ""
					if spec.Type != nil {
						typ = types.ExprString(spec.Type)
						lastType = typ
					} else if kind == "const" && len(spec.Values) == 0 {
						typ = lastType // iota repetition keeps the type
					}
					for i, name := range spec.Names {
						if name.Name == "_" {
							continue
						}
						s := site{kind: kind, pos: pos(name.Pos()), pkg: pkg, name: name.Name, typ: typ}
						if i < len(spec.Values) {
							if lit, ok := spec.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
								s.value, _ = strconv.Unquote(lit.Value)
								if s.typ == "" {
									s.typ = "untyped string"
								}
							} else if s.typ == "" {
								s.typ = "= " + types.ExprString(spec.Values[i])
							}
						}
						out = append(out, s)
					}
				}
			}
		}
	}
	return out
}

func fieldSites(pkg, owner string, st *ast.StructType, pos func(token.Pos) string) []site {
	var out []site
	for _, field := range st.Fields.List {
		typ := types.ExprString(field.Type)
		if len(field.Names) == 0 { // embedded
			out = append(out, site{kind: "field", pos: pos(field.Pos()), pkg: pkg, name: owner + "." + typ, typ: typ})
			continue
		}
		for _, name := range field.Names {
			s := site{kind: "field", pos: pos(name.Pos()), pkg: pkg, name: owner + "." + name.Name, typ: typ}
			if field.Tag != nil {
				s.value = jsonKey(field.Tag.Value)
			}
			out = append(out, s)
		}
	}
	return out
}

// firstSentence returns the first sentence of a doc comment, on one line.
func firstSentence(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	text := strings.Join(strings.Fields(cg.Text()), " ")
	if i := strings.Index(text, ". "); i >= 0 {
		text = text[:i+1]
	}
	return text
}

func resultType(ft *ast.FuncType) string {
	if ft.Results == nil || len(ft.Results.List) == 0 {
		return "(no result)"
	}
	var parts []string
	for _, r := range ft.Results.List {
		n := max(len(r.Names), 1)
		for range n {
			parts = append(parts, types.ExprString(r.Type))
		}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func recvName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return recvName(e.X)
	case *ast.IndexExpr:
		return recvName(e.X)
	case *ast.IndexListExpr:
		return recvName(e.X)
	case *ast.Ident:
		return e.Name
	}
	return types.ExprString(expr)
}

func underlying(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface"
	}
	return types.ExprString(expr)
}

var jsonTagRe = regexp.MustCompile(`json:"([^",]*)`)

func jsonKey(tag string) string {
	m := jsonTagRe.FindStringSubmatch(tag)
	if m == nil || m[1] == "" || m[1] == "-" {
		return ""
	}
	return m[1]
}

//------------------------------------------------------------------------------

// splitWords splits camelCase, PascalCase, snake_case and kebab-case into
// lower-case words. HTTPClient gives http, client.
func splitWords(s string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r):
			prevLower := i > 0 && (unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1]))
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			prevUpper := i > 0 && unicode.IsUpper(rs[i-1])
			if prevLower || (prevUpper && nextLower) {
				flush()
			}
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

// wordForms returns the forms of one lower-case word a name may use.
func wordForms(w string) []string {
	forms := []string{w, w + "s", w + "es"}
	if strings.HasSuffix(w, "y") && len(w) > 1 {
		forms = append(forms, w[:len(w)-1]+"ies")
	}
	return forms
}

// containsTerm reports whether the words hold the term's words in sequence.
// The last term word may be plural.
func containsTerm(words, term []string) bool {
	if len(term) == 0 || len(words) < len(term) {
		return false
	}
	for i := 0; i+len(term) <= len(words); i++ {
		ok := true
		for j, tw := range term {
			w := words[i+j]
			if j == len(term)-1 {
				if !slices.Contains(wordForms(tw), w) {
					ok = false
				}
			} else if w != tw {
				ok = false
			}
			if !ok {
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// lastName returns the part of a qualified site name after the last dot.
// For a parameter, Func(name), it returns the parameter name.
func lastName(name string) string {
	if strings.HasSuffix(name, ")") {
		if i := strings.LastIndexByte(name, '('); i >= 0 {
			return name[i+1 : len(name)-1]
		}
	}
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return name
}

//------------------------------------------------------------------------------

func runSites(roots []string, term string, tests bool) {
	var files []parsedFile
	skipped := 0
	for _, root := range roots {
		f, n := parseTree(root, tests)
		files = append(files, f...)
		skipped += n
	}
	root := strings.Join(roots, " ")
	termWords := splitWords(term)

	var hits []site
	type lit struct{ pos, text string }
	var lits []lit
	commentLines := map[string]int{}

	for _, pf := range files {
		for _, s := range declSites(pf) {
			if containsTerm(splitWords(lastName(s.name)), termWords) ||
				(s.value != "" && containsTerm(splitWords(s.value), termWords)) {
				hits = append(hits, s)
			}
		}
		ast.Inspect(pf.file, func(n ast.Node) bool {
			bl, ok := n.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING {
				return true
			}
			text, err := strconv.Unquote(bl.Value)
			if err != nil || !containsTerm(splitWords(text), termWords) {
				return true
			}
			p := pf.fset.Position(bl.Pos())
			lits = append(lits, lit{fmt.Sprintf("%s:%d", p.Filename, p.Line), text})
			return true
		})
		for _, cg := range pf.file.Comments {
			for _, c := range cg.List {
				if containsTerm(splitWords(c.Text), termWords) {
					commentLines[pf.path]++
				}
			}
		}
	}

	fmt.Printf("term %q under %s: %d files parsed, %d generated files skipped\n", term, root, len(files), skipped)

	groups := map[string][]site{}
	for _, s := range hits {
		groups[s.typ] = append(groups[s.typ], s)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(groups[keys[i]]) != len(groups[keys[j]]) {
			return len(groups[keys[i]]) > len(groups[keys[j]])
		}
		return keys[i] < keys[j]
	})

	fmt.Printf("\n== declarations: %d, grouped by the type of the value named (%d types)\n", len(hits), len(keys))
	if len(keys) > 1 {
		fmt.Println("   More than one type: check each pair for one name used for two concepts.")
	}
	for _, k := range keys {
		fmt.Printf("\n%s\n", k)
		for _, s := range groups[k] {
			extra := ""
			if s.value != "" {
				extra = fmt.Sprintf("  = %q", s.value)
			}
			fmt.Printf("  %-6s %-50s %s%s\n", s.kind, s.name, s.pos, extra)
		}
	}

	fmt.Printf("\n== string literals: %d (operator-facing text, log keys, SQL)\n", len(lits))
	for _, l := range lits {
		fmt.Printf("  %s  %q\n", l.pos, truncate(l.text, 100))
	}

	total := 0
	for _, n := range commentLines {
		total += n
	}
	fmt.Printf("\n== comment lines: %d in %d files\n", total, len(commentLines))
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

//------------------------------------------------------------------------------

func runSurface(dir string, tests bool) {
	files, skipped := parseTree(dir, tests)
	byPkg := map[string][]site{}
	commentWords := map[string]map[string]bool{}
	for _, pf := range files {
		for _, s := range declSites(pf) {
			byPkg[s.pkg] = append(byPkg[s.pkg], s)
		}
		pkg := filepath.Dir(pf.path)
		if commentWords[pkg] == nil {
			commentWords[pkg] = map[string]bool{}
		}
		for _, cg := range pf.file.Comments {
			for _, w := range strings.FieldsFunc(strings.ToLower(cg.Text()), func(r rune) bool { return !unicode.IsLetter(r) }) {
				commentWords[pkg][w] = true
			}
		}
	}
	pkgs := make([]string, 0, len(byPkg))
	for p := range byPkg {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)

	fmt.Printf("surface of %s: %d files parsed, %d generated files skipped\n", dir, len(files), skipped)
	for _, p := range pkgs {
		var exported, enums, wire []string
		for _, s := range byPkg[p] {
			n := lastName(s.name)
			switch {
			case s.kind == "param":
			case s.kind == "field" && s.value != "":
				wire = append(wire, fmt.Sprintf("  %-28s %s  (%s)", s.value, s.name, s.pos))
			case s.kind == "const" && s.value != "":
				enums = append(enums, fmt.Sprintf("  %-40s = %-24q %s  (%s)", s.name, s.value, s.typ, s.pos))
			case ast.IsExported(n) && (s.kind != "field" && s.kind != "method" || ast.IsExported(strings.Split(s.name, ".")[0])):
				exported = append(exported, fmt.Sprintf("  %-6s %-50s %s", s.kind, s.name, s.typ))
			}
		}
		fmt.Printf("\n### %s\n", p)
		printSection("exported", exported)
		printSection("string constants (enum values)", enums)
		printSection("json keys", wire)
		printSection("look-alike pairs: one letter apart, or one word inflected, with a different kind or type", lookAlikes(byPkg[p]))
		printSection("names that carry no concept (a reader learns nothing from the word)", emptyNames(byPkg[p]))
		printSection("type definitions: the first doc sentence of each type (run the definition test on each)", typeDefinitions(byPkg[p]))
		printSection("type names reused for a value of another type (one name, two concepts)", typeNameReuse(byPkg[p]))
		printSection("clipped words whose full form the package also writes", clippings(byPkg[p], commentWords[p]))
	}
}

// typeDefinitions returns each type with the first sentence of its doc.
// A sentence that explains the name with a plainer word is the signal of the
// definition test: "session is one TCP connection" names a connection.
func typeDefinitions(sites []site) []string {
	var out []string
	for _, s := range sites {
		if s.kind != "type" {
			continue
		}
		doc := s.doc
		if doc == "" {
			doc = "(no doc)"
		}
		out = append(out, fmt.Sprintf("  %-28s %s", s.name, truncate(doc, 150)))
	}
	return out
}

// typeNameReuse returns the parameters, fields, vars and funcs whose name is a
// type's name, or its plural, while the value is not of that type.
func typeNameReuse(sites []site) []string {
	typeNames := map[string]string{} // lower name -> declared name
	for _, s := range sites {
		if s.kind == "type" {
			typeNames[strings.ToLower(s.name)] = s.name
		}
	}
	var out []string
	for _, s := range sites {
		if s.kind == "type" || s.kind == "method" {
			continue
		}
		n := strings.ToLower(lastName(s.name))
		tn, ok := typeNames[n]
		if !ok {
			tn, ok = typeNames[strings.TrimSuffix(n, "s")]
		}
		if !ok || strings.Contains(s.typ, tn) {
			continue
		}
		out = append(out, fmt.Sprintf("  type %-20s  vs %-6s %-40s %-22s %s", tn, s.kind, s.name, truncate(s.typ, 22), s.pos))
	}
	return out
}

// commonClips holds the short forms Go code uses everywhere; they are not
// private abbreviations.
var commonClips = map[string]bool{
	"ctx": true, "err": true, "errs": true, "buf": true, "db": true, "id": true, "ids": true, "url": true, "cfg": true,
	"conf": true, "req": true, "resp": true, "res": true, "fn": true, "idx": true, "msg": true, "msgs": true,
	"num": true, "str": true, "src": true, "dst": true, "tmp": true, "val": true, "vals": true, "max": true,
	"min": true, "len": true, "cap": true, "ch": true, "wg": true, "mu": true, "ok": true, "rw": true,
	"tx": true, "env": true, "opt": true, "opts": true, "info": true, "spec": true, "stat": true, "stats": true,
	"attr": true, "attrs": true, "addr": true, "auth": true, "doc": true, "docs": true, "pkg": true, "app": true,
	"api": true, "sql": true, "dir": true, "ext": true, "pos": true, "prev": true, "cur": true, "next": true,
	"ref": true, "refs": true, "repo": true, "sub": true, "sum": true, "sync": true, "sys": true, "util": true,
	"ts": true, "dur": true, "col": true, "cols": true, "ptr": true, "impl": true, "init": true, "args": true,
	"arg": true, "params": true, "param": true, "prop": true, "props": true, "elem": true, "elems": true,
	"int": true, "uint": true, "bool": true, "byte": true, "bytes": true, "rune": true, "http": true,
	"json": true, "yaml": true, "html": true, "uuid": true, "otel": true, "ttl": true, "slo": true,
	"pg": true, "is": true, "has": true, "get": true, "set": true, "new": true, "add": true, "del": true,
	"key": true, "keys": true, "sec": true, "ms": true, "ns": true, "gc": true, "io": true, "os": true,
	"end": true, "run": true, "use": true, "log": true, "out": true, "in": true, "to": true, "of": true, "at": true,
	"by": true, "on": true, "or": true, "and": true, "all": true, "any": true, "not": true, "no": true, "one": true,
	"two": true, "per": true, "raw": true, "row": true, "rows": true, "top": true, "day": true, "sort": true,
	"func": true, "with": true, "go": true, "re": true, "conn": true, "exec": true, "iter": true, "cmd": true,
	"proc": true, "hash": true, "enc": true, "dec": true, "lit": true, "gen": true, "test": true, "mock": true,
}

// inflection holds the endings that make a longer form of the same word.
var inflection = map[string]bool{
	"s": true, "es": true, "ed": true, "d": true, "ing": true, "er": true, "ers": true,
	"ne": true, "e": true, "ne's": true, "y": true, "ly": true,
}

// clippings returns short words (two to four letters) in names that are the
// start of a longer word the same package writes in a name or a comment,
// such as sig next to signal.
func clippings(sites []site, comments map[string]bool) []string {
	full := map[string]bool{}
	for w := range comments {
		full[w] = true
	}
	uses := map[string][]string{}
	for _, s := range sites {
		for _, w := range splitWords(lastName(s.name)) {
			full[w] = true
			if len(w) >= 2 && len(w) <= 4 && !commonClips[w] {
				uses[w] = append(uses[w], s.name)
			}
		}
	}
	var words []string
	for w := range uses {
		words = append(words, w)
	}
	sort.Strings(words)
	var out []string
	for _, w := range words {
		var longs []string
		for f := range full {
			if len(f) < len(w)+2 || !strings.HasPrefix(f, w) {
				continue
			}
			rest := f[len(w):]
			if inflection[rest] || (len(rest) > 1 && inflection[rest[1:]] && rest[0] == w[len(w)-1]) || startsWithWord(rest, full) {
				continue // does, dropped, droppartition: the short word is a word here
			}
			longs = append(longs, f)
		}
		if len(longs) == 0 {
			continue
		}
		sort.Strings(longs)
		out = append(out, fmt.Sprintf("  %-5s full form written: %-30s used in: %s", w, strings.Join(firstN(longs, 3), ", "), strings.Join(firstN(uses[w], 4), ", ")))
	}
	return out
}

// lookAlikes returns pairs of package-level or method names that a reader
// can confuse: one or two edits apart, or the same stem inflected (Own and
// Owned), where the two declarations differ in kind or type.
func lookAlikes(sites []site) []string {
	type named struct {
		name string
		s    site
	}
	var ns []named
	seen := map[string]bool{}
	for _, s := range sites {
		if s.kind == "field" || s.kind == "param" {
			continue
		}
		n := lastName(s.name)
		if len(n) < 3 || seen[s.kind+s.name] {
			continue
		}
		seen[s.kind+s.name] = true
		ns = append(ns, named{n, s})
	}
	var out []string
	for i := range ns {
		for j := i + 1; j < len(ns); j++ {
			a, b := ns[i], ns[j]
			if a.name == b.name || a.s.kind+a.s.typ == b.s.kind+b.s.typ {
				continue
			}
			la, lb := strings.ToLower(a.name), strings.ToLower(b.name)
			if conventionPair(a.name, b.name, a.s, b.s) {
				continue
			}
			if !(sameStem(la, lb) || editDistance(la, lb) == 1) {
				continue
			}
			out = append(out, fmt.Sprintf("  %-24s %-6s %-22s | %-24s %-6s %s", a.s.name, a.s.kind, truncate(a.s.typ, 22), b.s.name, b.s.kind, truncate(b.s.typ, 22)))
		}
	}
	return out
}

// conventionPair reports the pairs Go conventions create on purpose: an
// exported name and its unexported twin (ApplyOption, applyOptions), a name
// and a longer name that extends it (migrationDirectory, migrationDirectoryRE),
// and a collection type of its element (GoFunc, GoFuncs).
func conventionPair(an, bn string, as, bs site) bool {
	la, lb := strings.ToLower(an), strings.ToLower(bn)
	if strings.TrimSuffix(la, "s") == strings.TrimSuffix(lb, "s") && ast.IsExported(an) != ast.IsExported(bn) {
		return true
	}
	if la == lb || strings.TrimSuffix(strings.TrimSuffix(la, "s"), "e") == strings.TrimSuffix(strings.TrimSuffix(lb, "s"), "e") {
		return true // a name and its plural
	}
	for _, pair := range [][2]string{{"get", "set"}, {"trip", "untrip"}, {"marshal", "unmarshal"}, {"lock", "unlock"}} {
		if strings.Replace(la, pair[0], pair[1], 1) == lb || strings.Replace(lb, pair[0], pair[1], 1) == la {
			return true // an accessor pair or an inverse pair
		}
	}
	short, long := la, lb
	if len(short) > len(long) {
		short, long = long, short
	}
	if (strings.HasPrefix(long, short) || strings.HasSuffix(long, short)) && !sameStem(short, long) {
		return true
	}
	return strings.Contains(as.typ, bn) || strings.Contains(bs.typ, an)
}

func sameStem(a, b string) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	for _, suf := range []string{"ed", "d", "ing"} {
		if b == a+suf {
			return true
		}
	}
	return false
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// contentFree lists words that name no concept on their own.
var contentFree = map[string]bool{
	"data": true, "info": true, "item": true, "items": true, "object": true, "thing": true,
	"manager": true, "helper": true, "util": true, "utils": true, "misc": true, "stuff": true,
	"value": true, "values": true, "entry": true, "elem": true, "impl": true, "base": true,
	"common": true, "generic": true, "wrapper": true, "holder": true,
}

// emptyNames returns exported types and package-level funcs whose whole
// name is one letter, or only words from contentFree.
func emptyNames(sites []site) []string {
	var out []string
	for _, s := range sites {
		if s.kind != "type" && s.kind != "func" {
			continue
		}
		n := lastName(s.name)
		if !ast.IsExported(n) {
			continue
		}
		words := splitWords(n)
		empty := len(n) <= 2
		if !empty && len(words) > 0 {
			empty = true
			for _, w := range words {
				if !contentFree[w] {
					empty = false
					break
				}
			}
		}
		if empty {
			out = append(out, fmt.Sprintf("  %-6s %-30s %s  (%s)", s.kind, n, truncate(s.typ, 40), s.pos))
		}
	}
	return out
}

func printSection(title string, lines []string) {
	if len(lines) == 0 {
		fmt.Printf("-- %s: none\n", title)
		return
	}
	fmt.Printf("-- %s (%d)\n", title, len(lines))
	for _, l := range lines {
		fmt.Println(l)
	}
}

//------------------------------------------------------------------------------

var (
	headRe   = regexp.MustCompile("^\\*\\*([^*]+)\\*\\*(.*)$")
	identRe  = regexp.MustCompile("`([A-Za-z_][\\w.]*)(?:\\(\\))?`")
	indexRow = regexp.MustCompile(`^\|\s*\*\*([^*]+)\*\*\s*\|`)
)

type headword struct {
	line  int
	word  string
	ident string
	avoid string // the text of the entry's _Avoid_ line, joined across wraps
}

func runGlossary(contextPath string, roots []string, tests bool) {
	f, err := os.Open(contextPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	var heads []headword
	var index []string
	inAvoid := false
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if inAvoid {
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "**") || strings.HasPrefix(line, "_") ||
				strings.HasPrefix(line, "|") || strings.HasPrefix(line, "#") {
				inAvoid = false
			} else {
				heads[len(heads)-1].avoid += " " + strings.TrimSpace(line)
				continue
			}
		}
		if rest, ok := strings.CutPrefix(line, "_Avoid_:"); ok && len(heads) > 0 {
			heads[len(heads)-1].avoid = strings.TrimSpace(rest)
			inAvoid = true
			continue
		}
		if m := indexRow.FindStringSubmatch(line); m != nil {
			index = append(index, strings.ToLower(strings.TrimSpace(m[1])))
			continue
		}
		m := headRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// An entry opens with the headword, then "(`ident`)", ":" or a dash.
		// Bold text that opens a body line is not an entry.
		rest := m[2]
		opens := rest == "" || strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, " (") || strings.HasPrefix(rest, ", **") ||
			strings.HasPrefix(rest, " —") || strings.HasPrefix(rest, " -") || strings.HasSuffix(m[1], ":")
		if !opens {
			continue
		}
		h := headword{line: n, word: strings.TrimSpace(strings.TrimSuffix(m[1], ":"))}
		if strings.HasPrefix(rest, " (") {
			if im := identRe.FindStringSubmatch(rest); im != nil {
				h.ident = im[1]
			}
		}
		heads = append(heads, h)
	}

	fmt.Printf("glossary %s: %d headwords, %d ambiguity-index words\n", contextPath, len(heads), len(index))

	fmt.Println("\n== headword pairing: words of the headword the identifier drops")
	fmt.Println("   A dropped word on the ambiguity index is a finding. A package qualifier supplies a word only if it contains it.")
	for _, h := range heads {
		if h.ident == "" {
			continue
		}
		identWords := splitWords(strings.ReplaceAll(h.ident, ".", " "))
		var missing []string
		for _, w := range splitWords(h.word) {
			if !containsTerm(identWords, []string{w}) {
				marker := w
				if slices.Contains(index, w) {
					marker = w + " [INDEX]"
				}
				missing = append(missing, marker)
			}
		}
		if len(missing) > 0 {
			fmt.Printf("  %s:%d  %q -> %s  drops: %s\n", contextPath, h.line, h.word, h.ident, strings.Join(missing, ", "))
		}
	}
	unpaired := 0
	for _, h := range heads {
		if h.ident == "" {
			unpaired++
		}
	}
	_ = unpaired

	var files []parsedFile
	for _, root := range roots {
		f, _ := parseTree(root, tests)
		files = append(files, f...)
	}
	var sites []site
	for _, pf := range files {
		sites = append(sites, declSites(pf)...)
	}

	fmt.Println("\n== headwords with no identifier beside them: the declarations that spell each one")
	fmt.Println("   A headword no declaration spells is drift: find what the code calls the concept.")
	for _, h := range heads {
		if h.ident != "" {
			continue
		}
		hw := splitWords(h.word)
		var spell []string
		for _, s := range sites {
			if s.kind != "param" && containsTerm(splitWords(lastName(s.name)), hw) {
				spell = append(spell, s.name)
			}
		}
		if len(spell) == 0 {
			fmt.Printf("  %s:%d  %-28q NO DECLARATION SPELLS IT\n", contextPath, h.line, h.word)
		} else {
			fmt.Printf("  %s:%d  %-28q %d: %s\n", contextPath, h.line, h.word, len(spell), strings.Join(firstN(spell, 4), ", "))
		}
	}

	fmt.Println("\n== _Avoid_ lines: each avoided word, its stated reason, and the code that still spells it")
	fmt.Println("   An avoided word with no reason cannot drop a finding. An avoided word still in code is drift.")
	for _, h := range heads {
		if h.avoid == "" {
			continue
		}
		for _, item := range splitTopLevel(h.avoid) {
			word, reason := avoidItem(item)
			if word == "" {
				continue
			}
			var inCode []string
			for _, s := range sites {
				if s.kind != "param" && containsTerm(splitWords(lastName(s.name)), splitWords(word)) {
					inCode = append(inCode, s.name)
				}
			}
			if reason == "" {
				reason = "NO REASON"
			}
			line := fmt.Sprintf("  %s:%d  %-24q avoid %-22q %s", contextPath, h.line, h.word, word, truncate(reason, 70))
			if len(inCode) > 0 {
				line += fmt.Sprintf("  | still in code: %d (%s)", len(inCode), strings.Join(firstN(inCode, 4), ", "))
			}
			fmt.Println(line)
		}
	}

	// bare[word] = sites whose whole name is that one word.
	bare := map[string][]site{}
	{
		for _, s := range sites {
			if s.kind == "field" || s.kind == "method" {
				continue
			}
			words := splitWords(lastName(s.name))
			if len(words) == 1 && ast.IsExported(lastName(s.name)) {
				bare[words[0]] = append(bare[words[0]], s)
			}
		}
	}

	fmt.Println("\n== index words spelled bare by an exported package-level name")
	for _, w := range index {
		for _, s := range bare[w] {
			fmt.Printf("  %-12s %-6s %s.%s  %s\n", w, s.kind, filepath.Base(s.pkg), s.name, s.pos)
		}
	}

	fmt.Println("\n== bare exported names in two or more packages that the index does not list")
	fmt.Println("   Listed only where the kinds or types differ. One shape in every package is a convention, such as Module.")
	var words []string
	for w, ss := range bare {
		pkgs := map[string]bool{}
		shapes := map[string]bool{}
		for _, s := range ss {
			pkgs[s.pkg] = true
			shapes[s.kind+" "+s.typ] = true
		}
		if len(pkgs) >= 2 && len(shapes) >= 2 && !slices.Contains(index, w) {
			words = append(words, w)
		}
	}
	sort.Strings(words)
	for _, w := range words {
		var where []string
		for _, s := range bare[w] {
			where = append(where, fmt.Sprintf("%s.%s (%s %s)", filepath.Base(s.pkg), s.name, s.kind, s.typ))
		}
		fmt.Printf("  %-14s %s\n", w, strings.Join(where, "; "))
	}
}

// splitTopLevel splits an _Avoid_ list on commas outside parentheses.
func splitTopLevel(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',', ';':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}

// avoidItem splits one _Avoid_ item into the avoided word and its reason.
// A reason is a parenthesis or the text after a dash.
func avoidItem(item string) (word, reason string) {
	item = strings.TrimSuffix(strings.TrimSpace(item), ".")
	if i := strings.IndexByte(item, '('); i >= 0 {
		word, reason = item[:i], strings.Trim(item[i:], "()")
	} else if i := strings.Index(item, " — "); i >= 0 {
		word, reason = item[:i], item[i+len(" — "):]
	} else {
		word = item
	}
	word = stripMarkup(word)
	word = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(word), "bare "))
	// "enforcer as a name for the sweep": the avoided word is the head.
	for _, sep := range []string{" as ", " for ", " when ", " unqualified", " in "} {
		if i := strings.Index(word, sep); i > 0 && len(strings.Fields(word)) > 2 {
			if reason == "" {
				reason = strings.TrimSpace(word[i:])
			}
			word = strings.TrimSpace(word[:i])
			break
		}
	}
	if len(strings.Fields(word)) > 4 {
		return "", "" // prose that followed the list, not an avoided word
	}
	return word, strings.TrimSpace(reason)
}

func firstN(ss []string, n int) []string {
	if len(ss) <= n {
		return ss
	}
	return append(ss[:n:n], "…")
}

//------------------------------------------------------------------------------

// expandImporters returns the roots plus the importers of each, deduplicated.
func expandImporters(roots []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range roots {
		for _, d := range importers(r) {
			if !seen[d] {
				seen[d] = true
				out = append(out, d)
			}
		}
	}
	return out
}

// importers returns dir and the directory of every package in the module that
// imports the package in dir. The directories are relative to the module root
// when dir is.
func importers(dir string) []string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	modRoot, modPath := findModule(abs)
	if modRoot == "" {
		fmt.Fprintln(os.Stderr, "termsites: no go.mod above", dir)
		os.Exit(1)
	}
	rel, _ := filepath.Rel(modRoot, abs)
	importPath := modPath + "/" + filepath.ToSlash(rel)

	dirs := map[string]bool{dir: true}
	fset := token.NewFileSet()
	_ = filepath.WalkDir(modRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != modRoot && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "vendor" || name == "testdata" || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == importPath {
				r, _ := filepath.Rel(modRoot, filepath.Dir(path))
				dirs[r] = true
			}
		}
		return nil
	})
	out := make([]string, 0, len(dirs))
	for d := range dirs {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func findModule(dir string) (root, path string) {
	for d := dir; ; d = filepath.Dir(d) {
		data, err := os.ReadFile(filepath.Join(d, "go.mod"))
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
					return d, strings.TrimSpace(rest)
				}
			}
			return "", ""
		}
		if filepath.Dir(d) == d {
			return "", ""
		}
	}
}

// startsWithWord reports whether s starts with a word of three or more
// letters from words, which makes w+s a compound, not a longer form of w.
func startsWithWord(s string, words map[string]bool) bool {
	for i := len(s); i >= 3; i-- {
		if words[s[:i]] {
			return true
		}
	}
	return false
}

var (
	mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdEmphRe = regexp.MustCompile(`(?:^|\b)_([^_]+)_`)
)

// stripMarkup removes Markdown emphasis, code marks and links from s.
func stripMarkup(s string) string {
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = mdEmphRe.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("**", "", "`", "", "*", "").Replace(s)
	return strings.Trim(strings.TrimSpace(s), "_\"'")
}
