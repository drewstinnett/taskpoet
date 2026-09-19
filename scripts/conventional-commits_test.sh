#!/usr/bin/env bash
# Tests for conventional-commits.sh, against throwaway repos with known history.
set -euo pipefail

SCRIPT=$(cd "$(dirname "$0")" && pwd)/conventional-commits.sh
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

fails=0
pass() { printf 'ok   %s\n' "$1"; }
fail() {
    printf 'FAIL %s\n     want: %s\n     got:  %s\n' "$1" "$2" "$3"
    fails=$((fails + 1))
}

# newrepo MODULE: an empty repo whose go.mod says MODULE
newrepo() {
    rm -rf "$TMP/repo"
    mkdir "$TMP/repo"
    cd "$TMP/repo"
    git init -q .
    git config user.email t@example.com
    git config user.name t
    git config commit.gpgsign false
    printf 'module %s\n' "$1" >go.mod
}

# commit SUBJECT [BODY]
commit() {
    git commit -q --allow-empty -m "$1" ${2:+-m "$2"}
}

# next NAME WANT: what 'next' prints ('' for no release)
next() {
    local got
    got=$("$SCRIPT" next 2>&1) || true
    if [[ $got == "$2" ]]; then pass "$1"; else fail "$1" "$2" "$got"; fi
}

# check NAME WANT_RC: exit status of 'check' over the whole history
check() {
    local rc=0
    "$SCRIPT" check "$(git rev-list --max-parents=0 HEAD | tail -n1)..HEAD" >/dev/null 2>&1 || rc=$?
    if [[ $rc -eq $2 ]]; then pass "$1"; else fail "$1" "$2" "$rc"; fi
}

# title NAME WANT_RC TITLE
title() {
    local rc=0
    "$SCRIPT" check-title "$3" >/dev/null 2>&1 || rc=$?
    if [[ $rc -eq $2 ]]; then pass "$1"; else fail "$1" "$2" "$rc"; fi
}

# --- versions -----------------------------------------------------------

newrepo example.com/m
commit 'chore: start'
git tag v1.2.3
next 'nothing since the tag: no release' ''
commit 'docs: words'
commit 'chore: deps'
next 'docs and chore do not release' ''
commit 'fix: a bug'
next 'fix is a patch' 'v1.2.4'
commit 'perf(db): faster'
next 'perf is a patch too' 'v1.2.4'
commit 'feat(cli): a thing'
next 'feat beats fix: minor' 'v1.3.0'
commit 'refactor!: rename it'
next 'bang is major, but the module is v1 not v2' 'A breaking change would make this v2.0.0, but go.mod is still /v1.
Bump the module path to /v2 first, or drop the breaking change.'

newrepo example.com/m
commit 'chore: start'
git tag v1.2.3
commit 'fix: a bug' 'BREAKING CHANGE: it all changed'
next 'BREAKING CHANGE footer is major' 'A breaking change would make this v2.0.0, but go.mod is still /v1.
Bump the module path to /v2 first, or drop the breaking change.'

newrepo example.com/m/v2
commit 'chore: start'
git tag v2.4.1
commit 'feat: x'
commit 'feat!: y'
next 'v2 module with a v2 tag, breaking is refused' 'A breaking change would make this v3.0.0, but go.mod is still /v2.
Bump the module path to /v3 first, or drop the breaking change.'

newrepo example.com/m/v2
commit 'chore: start'
git tag v0.3.1
commit 'Add some things without a type'
next 'unconventional commits do not release' ''
commit 'fix: small'
next 'v0.x tag on a /v2 module jumps to v2.0.0' 'v2.0.0'

newrepo example.com/m
commit 'feat: first thing'
next 'no tags at all starts from v0.0.0' 'v0.1.0'

newrepo example.com/m
commit 'chore: start'
git tag v0.3.1
commit 'feat: x'
next 'v0 module minor' 'v0.4.0'
git tag v0.4.0
git tag v0.4.0-rc.1
commit 'fix: y'
next 'the pre-release style tag is not a base' 'v0.4.1'

newrepo example.com/m
commit 'chore: start'
git tag v1.0.0
git checkout -q -b topic
commit 'feat: on a branch'
git checkout -q -
commit 'fix: on main'
git merge -q --no-ff topic -m 'Merge pull request #1 from x/topic'
next 'merge commits are skipped, their commits are not' 'v1.1.0'

# --- checking -----------------------------------------------------------

newrepo example.com/m
commit 'feat: fine'
commit 'fix(scope): fine'
commit 'chore!: fine'
commit 'Revert "feat: fine"'
check 'all good history' 0
commit 'Fixed the thing'
check 'a bad subject fails' 1

newrepo example.com/m
commit 'feat: fine'
commit 'feat:missing space'
check 'no space after the colon fails' 1

newrepo example.com/m
commit 'feat: fine'
commit 'Feat: capital'
check 'the type is lowercase' 1

newrepo example.com/m
commit 'feat: fine'
rc=0
"$SCRIPT" check nosuchref..HEAD >/dev/null 2>&1 || rc=$?
if [[ $rc -ne 0 ]]; then pass 'a bad range is an error, not a pass'; else fail 'a bad range is an error, not a pass' 'non-zero' 0; fi

title 'good title' 0 'feat(cli): add update'
title 'breaking title' 0 'fix!: a thing'
title 'plain title fails' 1 'Add update'
title 'unknown type fails' 1 'feature: add update'
title 'empty description fails' 1 'feat: '
title 'merge title is ignored' 0 'Merge branch main into x'

if [[ $fails -gt 0 ]]; then
    printf '\n%d failed\n' "$fails"
    exit 1
fi
printf '\nall passed\n'
