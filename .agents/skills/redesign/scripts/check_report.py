#!/usr/bin/env python3
"""Check a saved /redesign report for the slips that recur in long reports.

Usage: check_report.py <report.md> [--repo <repo root>]

Fails (exit 1) on:
  - a bare goal, non-goal, frozen-contract or record id (G3, N1, F2, D12) that no
    document qualifies; a range that opens with the qualified form
    (devices:G1-G3) is allowed, and a draft's `D?` is allowed; a fenced
    block holding **Decision.** is a drafted record, and its bare ids name its
    own document, so they are allowed there;
  - a qualifier that names no design document (invite:G1 where the document is
    core/invite), and an id joined to its document by a slash (core/invite/G1);
  - an absolute path in the Handoff section that does not exist, or a
    <placeholder> left in it.

Lists for review (never fails): each scoped claim about the code — "every
caller", "no consumer", "only build", "0 of N" — which must state the builds,
roles and config switches it covered.
"""

import argparse
import os
import re
import subprocess
import sys

ID = r"[GNFD]\d+"
QUALIFIED = re.compile(r"([\w./-]+):(" + ID + r")\b")
BARE = re.compile(r"(?<![\w:/.-])(" + ID + r")\b")
SLASHED = re.compile(r"([\w.-]+(?:/[\w.-]+)*)/(" + ID + r")\b")
RANGE_TAIL = re.compile(r"[\w./-]+:" + ID + r"\s*(?:–|-|\.\.)\s*$")
SUBJECT = r"(?:call(?:er| site)|consumer|build|role|package|file|test|path|reader|writer|subscriber|importer|site|use)s?"
CLAIM = re.compile(
    r"\b(?:(?:every|all|no|only|none of the)\s+(?:[\w-]+\s+){0,3}" + SUBJECT + r"|0 of \d+)\b",
    re.IGNORECASE,
)
ABS_PATH = re.compile(r"(?<![\w.])(/(?:[\w.@+-]+/)*[\w.@+-]+)")
URL = re.compile(r"\b[a-z][a-z0-9+.-]*://\S+", re.IGNORECASE)
PLACEHOLDER = re.compile(r"<[a-z][\w -]*>")


def repo_root(given):
    if given:
        return given
    out = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"], capture_output=True, text=True
    )
    return out.stdout.strip() or os.getcwd()


def qualifier_exists(root, q):
    # Mirrors the placement rule of design-doc/DESIGN-FORMAT.md § Scope.
    q = q.strip("`*_([")
    path = q if q.endswith(".md") else os.path.join(q, "DESIGN.md")
    if not is_design_doc_path(path):
        return False
    return os.path.isfile(os.path.join(root, path))


def is_design_doc_path(path):
    if os.path.basename(path) == "DESIGN.md":
        return True
    return path.startswith("docs/design/") and path.endswith(".md") and not path.endswith(".evidence.md")


def draft_lines(lines):
    """Return the line numbers inside fenced blocks that hold a drafted record."""
    out, block, inside = set(), [], False
    for n, line in enumerate(lines, 1):
        if line.lstrip().startswith("```"):
            if inside and any("**Decision.**" in l for _, l in block):
                out.update(k for k, _ in block)
            inside, block = not inside, []
            continue
        if inside:
            block.append((n, line))
    return out


def section_of(lines, heading_prefix):
    out, inside = [], False
    for n, line in enumerate(lines, 1):
        if line.startswith("## "):
            inside = line[3:].lstrip("0123456789. ").lower().startswith(heading_prefix)
            continue
        if inside:
            out.append((n, line))
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("report")
    ap.add_argument("--repo")
    args = ap.parse_args()
    root = repo_root(args.repo)
    lines = open(args.report, encoding="utf-8").read().split("\n")

    drafts = draft_lines(lines)
    failures, claims = [], []
    for n, line in enumerate(lines, 1):
        for m in QUALIFIED.finditer(line):
            if not qualifier_exists(root, m.group(1)):
                failures.append(f"{n}: qualifier `{m.group(1)}` names no design document ({m.group(0)})")
        for m in BARE.finditer(line):
            if n in drafts:
                continue
            if RANGE_TAIL.search(line[: m.start()]):
                continue
            failures.append(f"{n}: bare id {m.group(1)} — qualify it with its document")
        for m in SLASHED.finditer(line):
            failures.append(f"{n}: {m.group(0)} joins its document with a slash — write {m.group(1)}:{m.group(2)}")
        if CLAIM.search(line):
            claims.append(f"{n}: {line.strip()[:160]}")

    handoff = section_of(lines, "handoff")
    if not handoff:
        failures.append("no `## 9. Handoff` section")
    for n, line in handoff:
        for m in ABS_PATH.finditer(URL.sub(" ", line)):
            p = m.group(1).rstrip(".,;:)")
            if p.count("/") >= 2 and not os.path.exists(p):
                failures.append(f"{n}: handoff path does not exist: {p}")
        for m in PLACEHOLDER.finditer(line):
            failures.append(f"{n}: placeholder left in the handoff: {m.group(0)}")

    if failures:
        print("FAIL")
        print("\n".join(failures))
    else:
        print("PASS: ids qualified, qualifiers resolve, handoff paths exist")
    if claims:
        print(f"\nREVIEW {len(claims)} scope claims — each states what it covered:")
        print("\n".join(claims))
    sys.exit(1 if failures else 0)


if __name__ == "__main__":
    main()
