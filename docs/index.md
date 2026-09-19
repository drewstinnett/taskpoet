# Overview

[![Go Reference](https://pkg.go.dev/badge/github.com/drewstinnett/taskpoet/v2.svg)](https://pkg.go.dev/github.com/drewstinnett/taskpoet/v2)
[![codecov](https://codecov.io/gh/drewstinnett/taskpoet/branch/main/graph/badge.svg?token=06C30FNUO5)](https://codecov.io/gh/drewstinnett/taskpoet)
[![Tests](https://github.com/drewstinnett/taskpoet/actions/workflows/coverage.yml/badge.svg)](https://github.com/drewstinnett/taskpoet/actions/workflows/coverage.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/drewstinnett/taskpoet)](https://goreportcard.com/report/github.com/drewstinnett/taskpoet)

TaskPoet is a [TaskWarrior](https://taskwarrior.org/) inspired task manager. We
aim to be easy to use from the CLI, but also pretty, focusing on TUI elements
from the amazing [Charm.sh](https://charm.sh/) folks.

Key reasons why this exists:

* Focus on aesthetics, even though this is a CLI app.
* Move over from TaskWarrior without losing anything, see
  [Coming from TaskWarrior](taskwarrior.md).
* Recurring tasks that work the way you expect, see [Recurring tasks](recurring.md).
* Implement Tom Limoncellis Impact vs Effort chart, as a core component.
* Everything lives in one [bbolt](https://github.com/etcd-io/bbolt) database file,
  see [Configuration](configuration.md).
* It's fun to make! 😁

*We are Poets here, not Warriors.*
