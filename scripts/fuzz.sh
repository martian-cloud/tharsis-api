#!/usr/bin/env bash
#
# Runs every Go fuzz target for a fixed duration each.
#
# `go test -fuzz` only accepts a single package and a single target per
# invocation, so this enumerates all targets and runs them one at a time.
#
# Environment:
#   FUZZ_TIME        duration per target (default: 1m)
#   FUZZ_PARALLEL    value for -parallel (default: Go's default, GOMAXPROCS)
#   CI_NODE_INDEX    1-based shard index (default: 1)
#   CI_NODE_TOTAL    number of shards (default: 1)
#   FUZZ_OUTPUT_DIR  where new crashing inputs are copied (default: ../fuzz-failures)
#
# Run from the oss directory. Exits non-zero if any target fails.

set -uo pipefail

FUZZ_TIME="${FUZZ_TIME:-1m}"
NODE_INDEX="${CI_NODE_INDEX:-1}"
NODE_TOTAL="${CI_NODE_TOTAL:-1}"
OUTPUT_DIR="${FUZZ_OUTPUT_DIR:-../fuzz-failures}"
TAGS="noui"

parallel_flag=()
if [ -n "${FUZZ_PARALLEL:-}" ]; then
  parallel_flag=(-parallel "$FUZZ_PARALLEL")
fi

# Build the list of "<package> <target>" pairs, then take this shard's slice. Each command is run on its
# own, not inside a process substitution, so a failure (e.g. a package that doesn't compile) aborts the
# script instead of silently dropping that package's targets.
packages=$(go list -tags="$TAGS" -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...) || exit 1
packages=$(grep -v /internal/db <<<"$packages" || true)

targets=()
while read -r pkg; do
  [ -n "$pkg" ] || continue
  listed=$(go test -tags="$TAGS" -list '^Fuzz' "$pkg") || { echo "failed to list fuzz targets in ${pkg}" >&2; exit 1; }
  while read -r name; do
    targets+=("$pkg $name")
  done < <(grep '^Fuzz' <<<"$listed" || true)
done <<<"$packages"

if [ ${#targets[@]} -eq 0 ]; then
  echo "No fuzz targets found" >&2
  exit 1
fi

echo "Found ${#targets[@]} fuzz targets; running shard ${NODE_INDEX}/${NODE_TOTAL} for ${FUZZ_TIME} each"

failed=()
for i in "${!targets[@]}"; do
  if [ $((i % NODE_TOTAL)) -ne $((NODE_INDEX - 1)) ]; then
    continue
  fi

  read -r pkg name <<<"${targets[$i]}"
  echo "=== FUZZ ${pkg} ${name}"

  if ! go test -tags="$TAGS" -run='^$' -fuzz="^${name}\$" -fuzztime="$FUZZ_TIME" ${parallel_flag[@]+"${parallel_flag[@]}"} "$pkg"; then
    failed+=("${pkg}.${name}")
  fi
done

# Go writes crashing inputs to testdata/fuzz/<Target>/ inside the package.
# Collect any that aren't already committed so they can be kept as artifacts.
if git rev-parse --git-dir >/dev/null 2>&1; then
  while read -r f; do
    mkdir -p "${OUTPUT_DIR}/$(dirname "$f")"
    cp "$f" "${OUTPUT_DIR}/${f}"
  done < <(git ls-files --others --exclude-standard . | grep '/testdata/fuzz/')
fi

if [ ${#failed[@]} -gt 0 ]; then
  echo "Failed fuzz targets:"
  printf '  %s\n' "${failed[@]}"
  exit 1
fi
