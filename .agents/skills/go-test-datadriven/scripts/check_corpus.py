#!/usr/bin/env python3
"""Find mechanical datadriven corpus defects that a green run cannot show.

Usage: check_corpus.py <package dir> [<package dir> ...]

It reports two kinds of finding:

  swallowed  A directive has no "----" line, and a later line that looks
             like a directive falls into its input. The parser then reads
             that later directive as input text, so it never runs and its
             expected output is never compared.
  unwalked   A directory under testdata/ holds corpus files, but no
             *_test.go file of the package names it. Check each one by
             hand: a test can build the path at run time.

Fixture files (*.yml, *.yaml, *.json) and testdata/fixtures/ are not corpora
and are skipped. The exit status is 1 when there is a finding.
"""

import os
import re
import sys

DIRECTIVE = re.compile(r"^>(\s|$)")
SEPARATORS = ("----", "---- BEGIN")
NON_CORPUS_EXT = (".yml", ".yaml", ".json", ".go", ".sql", ".csv", ".gz", ".zst")


def main(argv):
    findings = 0
    for pkg in argv[1:]:
        testdata = os.path.join(pkg, "testdata")
        if not os.path.isdir(testdata):
            continue
        corpus_dirs = set()
        for path in corpus_files(testdata):
            corpus_dirs.add(os.path.dirname(path))
            findings += check_swallowed(path)
        findings += check_unwalked(pkg, testdata, corpus_dirs)
    return 1 if findings else 0


def corpus_files(testdata):
    for root, dirs, files in os.walk(testdata):
        dirs[:] = [d for d in dirs if d != "fixtures"]
        for name in sorted(files):
            if name.endswith(NON_CORPUS_EXT) or name.startswith("."):
                continue
            path = os.path.join(root, name)
            if looks_like_corpus(path):
                yield path


def looks_like_corpus(path):
    try:
        with open(path, encoding="utf-8") as f:
            return any(DIRECTIVE.match(line) for line in f)
    except (UnicodeDecodeError, OSError):
        return False


def check_swallowed(path):
    with open(path, encoding="utf-8") as f:
        lines = f.read().split("\n")

    findings = 0
    i = 0
    while i < len(lines):
        if not DIRECTIVE.match(lines[i]):
            i += 1
            continue
        start = i
        i += 1
        # ">"-continuation lines directly after the directive belong to it.
        while i < len(lines) and DIRECTIVE.match(lines[i]):
            i += 1
        # The input runs to the separator or EOF. A blank line does not end it.
        swallowed = None
        while i < len(lines) and lines[i] not in SEPARATORS:
            if swallowed is None and DIRECTIVE.match(lines[i]):
                swallowed = i
            i += 1
        if swallowed is not None:
            findings += 1
            print(f"swallowed {path}:{start + 1}: {lines[start].strip()!r} has no "
                  f"'----', so {lines[swallowed].strip()!r} at line {swallowed + 1} "
                  f"is read as its input and never runs")
        if i < len(lines) and lines[i] == "---- BEGIN":
            while i < len(lines) and lines[i] != "---- END":
                i += 1
        # Skip the expected output, which runs to the next blank line.
        while i < len(lines) and lines[i] != "":
            i += 1
    return findings


def check_unwalked(pkg, testdata, corpus_dirs):
    sources = []
    for name in os.listdir(pkg):
        if name.endswith("_test.go"):
            with open(os.path.join(pkg, name), encoding="utf-8") as f:
                sources.append(f.read())
    text = "\n".join(sources)

    findings = 0
    for d in sorted(corpus_dirs):
        rel = os.path.relpath(d, testdata)
        if rel == ".":
            continue
        # A test names the directory itself or one of its ancestors.
        parts = rel.split(os.sep)
        names = {"/".join(parts[:n]) for n in range(1, len(parts) + 1)}
        names |= set(parts)
        if any(re.search(r"[\"/`]" + re.escape(n) + r"[\"/`]", text) for n in names):
            continue
        findings += 1
        print(f"unwalked  {d}: no *_test.go in {pkg} names this directory; "
              f"confirm no test walks it, then delete it")
    return findings


if __name__ == "__main__":
    sys.exit(main(sys.argv))
