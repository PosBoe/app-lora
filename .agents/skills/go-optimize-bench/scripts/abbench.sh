#!/usr/bin/env bash
# abbench.sh runs two or more test binaries of one package in alternating
# short blocks, then compares them with benchstat.
#
#   abbench.sh -d PKGDIR -p PATTERN -o OUTPREFIX [-n ROUNDS] [-c COUNT]
#              [-t BENCHTIME] [-a] BASE.test HEAD.test [MORE.test ...]
#
# -d  package directory to run from; benchmarks often read testdata by a
#     relative path, and a binary run elsewhere skips them silently
# -p  -test.bench pattern; anchor it (^BenchmarkX$/...) so it matches only
#     what you mean
# -o  output prefix; writes OUTPREFIX.<name>.txt per binary and OUTPREFIX.load
# -n  rounds (default 10); samples per side = ROUNDS x COUNT
# -c  -test.count per block (default 1)
# -t  -test.benchtime (default: the binary's default)
# -a  allow byte-identical binaries: an A/A run that measures the noise floor
#
# Each round runs every binary once, and the order rotates each round, so
# thermal and load drift lands on all sides equally. The script stops when a
# block prints no benchmark line (a skip or a crash), when two binaries are
# byte-identical (a stale build), and when a benchmark fails.
set -euo pipefail

usage() { sed -n '5,6p' "$0" >&2; exit 2; }
dir="" pat="" out="" rounds=10 count=1 benchtime="" aa=0
while getopts "d:p:o:n:c:t:a" opt; do
  case $opt in
    d) dir=$OPTARG ;;
    p) pat=$OPTARG ;;
    o) out=$OPTARG ;;
    n) rounds=$OPTARG ;;
    c) count=$OPTARG ;;
    t) benchtime=$OPTARG ;;
    a) aa=1 ;;
    *) usage ;;
  esac
done
shift $((OPTIND - 1))
[ -n "$dir" ] && [ -n "$pat" ] && [ -n "$out" ] && [ $# -ge 2 ] || usage
[ $((rounds * count)) -ge 6 ] || { echo "abbench: $rounds x $count < 6 samples per side" >&2; exit 2; }

export PATH="$(go env GOPATH)/bin:$PATH"
command -v benchstat >/dev/null || { echo "abbench: install benchstat: go install golang.org/x/perf/cmd/benchstat@latest" >&2; exit 2; }

out=$(realpath -m "$out")
mkdir -p "$(dirname "$out")"
bins=() names=()
for b in "$@"; do
  b=$(realpath "$b")
  n=$(basename "$b" .test)
  for m in "${names[@]}"; do
    [ "$m" != "$n" ] || { echo "abbench: two binaries named $n" >&2; exit 2; }
  done
  bins+=("$b") names+=("$n")
done
for ((i = 1; i < ${#bins[@]}; i++)); do
  if [ $aa -eq 0 ] && cmp -s "${bins[0]}" "${bins[$i]}"; then
    echo "abbench: ${names[0]} and ${names[$i]} are byte-identical; rebuild" >&2
    exit 1
  fi
done

flags=(-test.run='^$' -test.bench="$pat" -test.benchmem -test.count="$count")
[ -n "$benchtime" ] && flags+=(-test.benchtime="$benchtime")

for n in "${names[@]}"; do : > "$out.$n.txt"; done
: > "$out.load"
cd "$dir"
k=${#bins[@]}
for ((r = 0; r < rounds; r++)); do
  echo "round $((r + 1)) load $(cut -d' ' -f1-3 /proc/loadavg)" >> "$out.load"
  for ((j = 0; j < k; j++)); do
    i=$(((r + j) % k))
    tmp=$(mktemp)
    if ! "${bins[$i]}" "${flags[@]}" > "$tmp" 2>&1; then
      cat "$tmp" >&2
      echo "abbench: ${names[$i]} failed in round $((r + 1))" >&2
      exit 1
    fi
    if ! grep -q '^Benchmark' "$tmp"; then
      cat "$tmp" >&2
      echo "abbench: ${names[$i]} printed no benchmark in round $((r + 1)); check -p and -d" >&2
      exit 1
    fi
    cat "$tmp" >> "$out.${names[$i]}.txt"
    rm -f "$tmp"
  done
done
echo "done $(cut -d' ' -f1-3 /proc/loadavg)" >> "$out.load"

args=()
for n in "${names[@]}"; do args+=("$n=$out.$n.txt"); done
benchstat "${args[@]}"
