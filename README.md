[![codecov](https://codecov.io/gh/drewstinnett/taskpoet/branch/main/graph/badge.svg?token=06C30FNUO5)](https://codecov.io/gh/drewstinnett/taskpoet)
[![Tests](https://github.com/drewstinnett/taskpoet/actions/workflows/coverage.yml/badge.svg)](https://github.com/drewstinnett/taskpoet/actions/workflows/coverage.yml)
# TaskPoet

Alternative to the awesome TaskWarrior app, with a few changes in mind:

* Switch over to Golang
* Implement impact assesment concepts from Time Management for Sysadmins
* Tweaks to command line syntax
* Pluggability to pull in tasks from other sources, such as Github or Gitlab

## Concepts

Using Tom Limoncellis Impact vs Effort chart will help us decide what tasks we
need to do. Represented in our Task structure as follows:

Impact:
* 0: Unspecified Impact (default)
* 1: Big Positive Impact (default)
* 2: Superficial Impact (default)

Effort:
* 0: Unspecified Effort (default)
* 1: Easy, Small Effort (default)
* 2: Difficult, Big Effort (default)

The lower representation of these numbers is the hot spot