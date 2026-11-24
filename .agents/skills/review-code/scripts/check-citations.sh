#!/usr/bin/env bash
# Print the line each citation of a review report points at, so a wrong or
# drifted citation shows before the report goes out.
#
# Usage: check-citations.sh <report-file> [<rev>]
#   Run it from the repository root. With <rev>, it reads each cited file at
#   that revision (git show <rev>:<path>), for the base side of a change.
#   Without it, it reads the working tree.
#
# A citation is a repo-relative path with a line or a range: path/to/file.go:42
# or path/to/file.go:42-45. For a range, it prints the first and the last line.
# In a JSON report, a code reference ({"kind": "code", "path", "line",
# "revision"}) is a citation too. It prints at its own revision, when it names
# one. The labels of the review protocol name no git ref: head reads the working
# tree, and base, merge-base or mergeBase read <rev>. Without <rev>, a base label
# prints UNRESOLVED.
set -euo pipefail

report=${1:?usage: check-citations.sh <report-file> [<rev>]}
rev=${2:-}

show() {
  local path=$1 line=$2
  if [[ -n $rev ]]; then
    git show "$rev:$path" 2>/dev/null | sed -n "${line}p"
  else
    sed -n "${line}p" "$path"
  fi
}

exists() {
  local path=$1
  if [[ -n $rev ]]; then
    git cat-file -e "$rev:$path" 2>/dev/null
  else
    [[ -f $path ]]
  fi
}

# check prints the line that one citation points at, read at the revision in
# $rev.
check() {
  local cite=$1 path lines first last text
  path=${cite%:*}
  lines=${cite##*:}
  first=${lines%-*}
  last=${lines#*-}
  if ! exists "$path"; then
    echo "MISSING  $cite"
    return
  fi
  text=$(show "$path" "$first")
  if [[ -z ${text//[[:space:]]/} ]]; then
    echo "BLANK    $cite"
  else
    echo "$cite  ${text#"${text%%[![:space:]]*}"}"
  fi
  if [[ $last != "$first" ]]; then
    echo "  ..$last  $(show "$path" "$last" | sed 's/^[[:space:]]*//')"
  fi
}

refs=""
if jq -e . "$report" >/dev/null 2>&1; then
  refs=$(jq -r '.. | objects | select(.kind == "code" and (.path | type) == "string" and (.line | type) == "number")
    | "\(.revision // "")|\(.path):\(.line)"' "$report" | sort -u)
fi
cites=$(grep -oE '[A-Za-z0-9_./@{}-]+\.[A-Za-z0-9]+:[0-9]+(-[0-9]+)?' "$report" | sort -u || true)
if [[ -z $refs && -z $cites ]]; then
  echo "no citations in $report"
  exit 0
fi

if [[ -n $refs ]]; then
  echo "== code references"
  default_rev=$rev
  while IFS="|" read -r ref_rev cite; do
    case $ref_rev in
      "") rev=$default_rev ;;
      head) rev="" ;;
      base | merge-base | mergeBase)
        if [[ -z $default_rev ]]; then
          echo "UNRESOLVED  $cite (revision $ref_rev: pass the base revision as <rev>)"
          continue
        fi
        rev=$default_rev
        ;;
      *) rev=$ref_rev ;;
    esac
    check "$cite"
  done <<<"$refs"
  rev=$default_rev
  echo "== prose citations"
fi
if [[ -n $cites ]]; then
  while IFS= read -r cite; do
    check "$cite"
  done <<<"$cites"
fi
