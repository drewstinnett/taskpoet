# Contributing

## Commit messages

Releases are made from commit messages, so they follow
[Conventional Commits](https://www.conventionalcommits.org/):

```plain
<type>[(scope)][!]: <description>
```

| Type                                                        | Release when it lands |
| ----------------------------------------------------------- | --------------------- |
| `feat`                                                      | minor, `2.1.0`        |
| `fix`, `perf`                                               | patch, `2.0.1`        |
| a `!` after the type, or a `BREAKING CHANGE:` footer        | major                 |
| `docs`, `style`, `refactor`, `test`, `build`, `ci`, `chore`, `revert` | none        |

For example `feat: add a --wait flag`, `fix(import): keep annotations on
recurring tasks`, or `feat!: drop the v0 database format`.

A pull request is checked in two ways, both by the *Conventional Commits*
workflow: every commit in it, and its title. The title matters because a squash
merge uses it. You can try the same check locally:

```plain
scripts/conventional-commits.sh check origin/main..HEAD
scripts/conventional-commits.sh check-title 'feat: something'
scripts/conventional-commits.sh next       # what a release would be right now
```

## Releases

Nobody tags by hand. Every push to `main` runs the *Release* workflow, which
looks at the commits since the last tag. With a `feat`, `fix`, `perf` or a
breaking change among them, it picks the next version, tags it, and
[goreleaser](https://goreleaser.com) builds and publishes the release, with
release notes made from those commits. Without one, nothing is released, so
docs and chores can pile up until the next real change.

If a release fails after it was tagged, run the workflow again from the Actions
tab and it finishes the job.

### The module is `/v2`

Go ties the major version of a module to its import path. The module is
`github.com/drewstinnett/taskpoet/v2`, so releases are `v2.x.y` and a breaking
change can't simply become `v3.0.0`: the release stops and says so. To do a
`v3`, change the module path to `/v3` first (in `go.mod` and every import).

## Tests

```plain
go test ./...
golangci-lint run
scripts/conventional-commits_test.sh
```
