#!/usr/bin/env bash
#
# The validation flow for claude-repo-factory.
#
#   gofmt  ->  go vet  ->  go test  ->  golangci-lint  ->  go build
#
# This script is the single definition of that order. The Makefile, CI and
# every documented checklist defer to it, so the sequence cannot drift between
# a developer machine and the pipeline. Steps run in order and the script
# stops at the first failure.
#
# Usage:
#   scripts/check.sh              run the full flow
#   scripts/check.sh --skip-lint  skip golangci-lint (it is the slow step)
#
set -euo pipefail

# Pinned so that a local run and CI apply exactly the same ruleset.
GOLANGCI_VERSION="v2.1.6"

SKIP_LINT=0
for arg in "$@"; do
	case "$arg" in
	--skip-lint) SKIP_LINT=1 ;;
	-h | --help)
		# Print the header comment block, stopping at the first line that is
		# not a comment, so the help text cannot drift as the script grows.
		awk 'NR < 3 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "$0"
		exit 0
		;;
	*)
		echo "unknown option: $arg" >&2
		exit 2
		;;
	esac
done

cd "$(dirname "$0")/.."

step=0
total=5
banner() {
	step=$((step + 1))
	printf '\n=== [%d/%d] %s ===\n' "$step" "$total" "$1"
}

# 1. gofmt. Cheapest check, and formatting noise obscures every later diff.
banner "gofmt"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
	echo "not gofmt-formatted:" >&2
	echo "$unformatted" >&2
	echo >&2
	echo "fix with: gofmt -w ." >&2
	exit 1
fi
echo "all files formatted"

# 2. go vet. First step that type-checks, so a compile break surfaces here.
banner "go vet ./..."
go vet ./...
echo "clean"

# 3. go test. Behaviour before style: a lint finding on broken code is noise.
banner "go test ./..."
go test ./...

# 4. golangci-lint. The slow, broad pass, run once the code is known correct.
banner "golangci-lint run"
if [ "$SKIP_LINT" -eq 1 ]; then
	echo "skipped (--skip-lint)"
elif command -v golangci-lint >/dev/null 2>&1; then
	golangci-lint run
	echo "0 issues"
else
	echo "golangci-lint not on PATH; running the pinned version through go run"
	echo "(the first run compiles it and takes several minutes)"
	go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION}" run
	echo "0 issues"
fi

# 5. go build. Final gate: every package, including the command, links.
banner "go build ./..."
go build ./...
echo "builds"

printf '\nvalidation flow passed\n'
