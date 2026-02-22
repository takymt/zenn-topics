#!/usr/bin/env bash
set -euo pipefail

commit_msg_file="$1"
commit_msg=$(head -1 "$commit_msg_file")

pattern='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?!?: .+'

if ! echo "$commit_msg" | grep -qE "$pattern"; then
  echo "ERROR: Commit message must follow Conventional Commits format." >&2
  echo "" >&2
  echo "  Format: <type>[optional scope]: <description>" >&2
  echo "" >&2
  echo "  Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert" >&2
  echo "" >&2
  echo "  Examples:" >&2
  echo "    feat: add page list command" >&2
  echo "    fix(config): handle missing profile" >&2
  echo "    docs: update README" >&2
  echo "" >&2
  echo "  Your message: $commit_msg" >&2
  exit 1
fi
