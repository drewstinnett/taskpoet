[![Go Reference](https://pkg.go.dev/badge/github.com/drewstinnett/taskpoet/v2.svg)](https://pkg.go.dev/github.com/drewstinnett/taskpoet/v2)
[![codecov](https://codecov.io/gh/drewstinnett/taskpoet/branch/main/graph/badge.svg?token=06C30FNUO5)](https://codecov.io/gh/drewstinnett/taskpoet)
[![Tests](https://github.com/drewstinnett/taskpoet/actions/workflows/coverage.yml/badge.svg)](https://github.com/drewstinnett/taskpoet/actions/workflows/coverage.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/drewstinnett/taskpoet)](https://goreportcard.com/report/github.com/drewstinnett/taskpoet)

# TaskPoet

Alternative to the awesome TaskWarrior app, with a few changes in mind:

* Switch over to Golang, in a single binary with a single database file
* Bring **everything** over from TaskWarrior: pending, waiting, completed,
  deleted and recurring tasks, projects, priorities, tags, dependencies,
  annotations and your own UDAs
* Implement impact assessment concepts from Time Management for Sysadmins
* Fun! 🎉

## Quick start

```console
# Coming from TaskWarrior? One command brings it all in
$ taskpoet import taskwarrior --from-task

$ taskpoet add --project home --priority H --due friday Plant the tomatoes
$ taskpoet add --recur weekly --due monday Water the plants
$ taskpoet                     # what should I do next?
$ taskpoet done 1b35           # any unique start of the id works
```

Upgrading from v0.x? Read [Upgrading](https://drewstinnett.github.io/taskpoet/upgrading/) first,
v2 has breaking changes.

Check out our official [Documentation](https://drewstinnett.github.io/taskpoet/)
