#!/usr/bin/env bash
# buildbin.sh builds the test binary of one package at one revision.
#
#   buildbin.sh [-t TAGS] REV PKG OUT
#
# REV is a git revision, or "." for the working tree as it stands.
# PKG is a package path relative to the repository root, e.g. ./pkg/datadriven.
# OUT is the path of the binary to write.
#
# A revision other than "." builds in a detached worktree under the scratch
# directory of OUT, so the working tree is never touched.
#
# GOAMD64=v3 and GOARM64=v8.0 pin the microarchitecture, so the base and the
# head binaries run the same instructions.
set -euo pipefail

tags=""
while getopts "t:" opt; do
  case $opt in
    t) tags=$OPTARG ;;
    *) echo "usage: $0 [-t TAGS] REV PKG OUT" >&2; exit 2 ;;
  esac
done
shift $((OPTIND - 1))
[ $# -eq 3 ] || { echo "usage: $0 [-t TAGS] REV PKG OUT" >&2; exit 2; }
rev=$1 pkg=$2 out=$3

root=$(git rev-parse --show-toplevel)
out=$(realpath -m "$out")
mkdir -p "$(dirname "$out")"

export GOAMD64=${GOAMD64:-v3} GOARM64=${GOARM64:-v8.0}
tagflag=()
[ -n "$tags" ] && tagflag=(-tags="$tags")

if [ "$rev" = "." ]; then
  (cd "$root" && go test "${tagflag[@]}" -c -o "$out" "$pkg")
  echo "built $out from the working tree"
  exit 0
fi

sha=$(git -C "$root" rev-parse --short "$rev^{commit}")
wt=$(dirname "$out")/wt-$sha
if [ ! -d "$wt" ]; then
  git -C "$root" worktree add --detach -q "$wt" "$sha"
fi
(cd "$wt" && go test "${tagflag[@]}" -c -o "$out" "$pkg")
echo "built $out from $sha (worktree $wt)"
echo "remove the worktree when done: git worktree remove $wt"
