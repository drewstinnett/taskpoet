#!/usr/bin/env bash
# Conventional Commits, in one place: CI uses this to enforce the format and
# the release job uses it to pick the next version, so the two can never
# disagree about what a commit means.
#
#   conventional-commits.sh check-title 'feat: add a thing'
#   conventional-commits.sh check origin/main..HEAD
#   conventional-commits.sh next [REV]        # prints e.g. v2.1.0, or nothing
#
# Bumps: 'type!:' or a 'BREAKING CHANGE:' footer is major, feat is minor, fix
# and perf are patch. Every other type is valid, but does not cause a release.
#
# Written for bash 3.2 too, so it can be tried on a Mac.
set -euo pipefail

TYPES='feat|fix|perf|docs|style|refactor|test|build|ci|chore|revert'
SUBJECT_RE="^(${TYPES})(\\(([a-z0-9._/-]+)\\))?(!)?: .+"
BREAKING_FOOTER_RE='^BREAKING[ -]CHANGE(:| #)'

usage() {
    sed -n '2,/^set /p' "$0" | sed '$d; s/^# \{0,1\}//' >&2
    exit 2
}

# Merge commits, and the ones GitHub writes for a revert, have no say in
# versioning and nobody types them by hand.
is_ignored() {
    case "$1" in
    "Merge "* | 'Revert "'*) return 0 ;;
    esac
    return 1
}

is_valid() {
    [[ $1 =~ $SUBJECT_RE ]]
}

format_help() {
    cat >&2 <<EOF

Commit subjects and PR titles must follow Conventional Commits:

    <type>[(scope)][!]: <description>

  types: ${TYPES//|/, }
  e.g.   feat: add a --wait flag
         fix(import): keep annotations on recurring tasks
         feat!: drop the v0 database format

  Use '!' after the type, or a 'BREAKING CHANGE:' footer, for a change that
  breaks users. Only feat, fix and perf (and breaking changes) cause a release.
EOF
}

# bump_of REV prints 3, 2, 1 or 0: major, minor, patch, nothing
bump_of() {
    local rev=$1 subject
    subject=$(git log -1 --format=%s "$rev")
    is_ignored "$subject" && { echo 0; return; }
    is_valid "$subject" || { echo 0; return; }
    if [[ ${BASH_REMATCH[4]} == '!' ]] ||
        git log -1 --format=%b "$rev" | grep -Eq "$BREAKING_FOOTER_RE"; then
        echo 3
        return
    fi
    case ${subject%%[(!:]*} in
    feat) echo 2 ;;
    fix | perf) echo 1 ;;
    *) echo 0 ;;
    esac
}

cmd_check_title() {
    [[ $# -eq 1 ]] || usage
    is_ignored "$1" && return 0
    if ! is_valid "$1"; then
        echo "Not a Conventional Commit title: $1" >&2
        format_help
        return 1
    fi
}

cmd_check() {
    [[ $# -eq 1 ]] || usage
    local bad=0 rev subject revs
    revs=$(git rev-list --no-merges --reverse "$1") # not in the loop: a bad range must fail
    for rev in $revs; do
        subject=$(git log -1 --format=%s "$rev")
        is_ignored "$subject" && continue
        if ! is_valid "$subject"; then
            echo "$(git log -1 --format=%h "$rev") $subject" >&2
            bad=$((bad + 1))
        fi
    done
    if [[ $bad -gt 0 ]]; then
        echo "^ $bad commit(s) that are not Conventional Commits" >&2
        format_help
        return 1
    fi
}

# The major version a Go module may release is fixed by its import path: a
# module called .../v2 can only ever be tagged v2.x.y.
module_major() {
    local mod
    mod=$(sed -n 's/^module[[:space:]]\{1,\}//p' go.mod | head -n1)
    if [[ $mod =~ /v([0-9]+)$ ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        echo 1
    fi
}

cmd_next() {
    local rev=${1:-HEAD} base range revs best=0 r b
    base=$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' --exclude 'v*-*' "$rev" 2>/dev/null || true)
    range=${base:+$base..}$rev

    revs=$(git rev-list --no-merges "$range")
    for r in $revs; do
        b=$(bump_of "$r")
        [[ $b -gt $best ]] && best=$b
    done
    [[ $best -gt 0 ]] || return 0

    local major=0 minor=0 patch=0
    if [[ $base =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
        major=${BASH_REMATCH[1]} minor=${BASH_REMATCH[2]} patch=${BASH_REMATCH[3]}
    fi
    case $best in
    3) major=$((major + 1)) minor=0 patch=0 ;;
    2) minor=$((minor + 1)) patch=0 ;;
    1) patch=$((patch + 1)) ;;
    esac

    local want
    want=$(module_major)
    if [[ $major -gt $want ]]; then
        echo "A breaking change would make this v$major.0.0, but go.mod is still /v$want." >&2
        echo "Bump the module path to /v$major first, or drop the breaking change." >&2
        return 1
    fi
    # Not on the right major yet (v0.x to a /v2 module): jump straight to it.
    # Without a /vN suffix both v0 and v1 are fine, so there is nothing to catch up on.
    if [[ $want -ge 2 && $major -lt $want ]]; then
        major=$want minor=0 patch=0
    fi
    echo "v$major.$minor.$patch"
}

[[ $# -ge 1 ]] || usage
cmd=$1
shift
case $cmd in
check-title) cmd_check_title "$@" ;;
check) cmd_check "$@" ;;
next) cmd_next "$@" ;;
*) usage ;;
esac
