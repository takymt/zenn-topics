#!/usr/bin/env bash
set -euo pipefail

remote="${BUMP_REMOTE:-origin}"
branch="${BUMP_BRANCH:-$(git branch --show-current)}"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "ERROR: not a git repository" >&2
  exit 1
fi

if [[ -z "$branch" ]]; then
  echo "ERROR: could not detect current branch; set BUMP_BRANCH" >&2
  exit 1
fi

if ! git remote get-url "$remote" >/dev/null 2>&1; then
  echo "ERROR: remote '$remote' not found; set BUMP_REMOTE" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "ERROR: working tree is not clean" >&2
  echo "Commit or stash changes before running bump." >&2
  exit 1
fi

if ! command -v gobump >/dev/null 2>&1; then
  echo "ERROR: gobump command not found" >&2
  echo "Run through mise: 'mise run bump'" >&2
  exit 1
fi

gobump patch -w .

new_version="$(sed -nE 's/^[[:space:]]*const[[:space:]]+version[[:space:]]*=[[:space:]]*"([^"]+)".*$/\1/p' version.go | head -n 1)"
if [[ -z "$new_version" ]]; then
  echo "ERROR: failed to read version from version.go" >&2
  exit 1
fi

tag="v${new_version}"

if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
  echo "ERROR: local tag '${tag}' already exists" >&2
  exit 1
fi

if git ls-remote --exit-code --tags "$remote" "refs/tags/${tag}" >/dev/null 2>&1; then
  echo "ERROR: remote tag '${tag}' already exists on '${remote}'" >&2
  exit 1
fi

git add version.go
if git diff --cached --quiet; then
  echo "ERROR: no version change detected in version.go" >&2
  exit 1
fi

git commit -m "chore(bump): ${tag}"
git tag -a "${tag}" -m "${tag}"

git push "$remote" "$branch"
git push "$remote" "${tag}"

echo "Bumped to ${tag} and pushed ${branch} + tag to ${remote}."
