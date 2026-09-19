# Installation

## Download Binary

Download the appropriate binary from the [Releases](https://github.com/drewstinnett/taskpoet/releases) page

## MacOS HomeBrew

```plain
brew install drewstinnett/taskpoet/taskpoet
```

## From source

You need [Go](https://go.dev/dl/) 1.21 or newer, and `make`.

```plain
git clone https://github.com/drewstinnett/taskpoet.git
cd taskpoet
make install
```

That builds `taskpoet` and copies it to `/usr/local/bin`. To put it somewhere
else, say `~/.local/bin` so you don't need `sudo`:

```plain
make install PREFIX=$HOME/.local
```

`BINDIR=/some/dir` picks the exact directory, and `DESTDIR` is a staging root
for packagers. `make uninstall` takes it out again, use the same `PREFIX` you
installed with. `make build` just builds `bin/taskpoet`.

## Updating

```plain
taskpoet update
```

downloads the latest release, checks it against the checksums published with it,
and replaces the running program. `taskpoet update --check` only says whether
there is a newer one.

taskpoet also looks for new releases by itself, at most every six hours, and
tells you when there is one. It never installs anything unless you run `update`.
That check only happens when you are at a terminal, so scripts and cron jobs are
left alone, and it can be turned off, see [Configuration](configuration.md#update-checks).

* **Homebrew**: `taskpoet update` won't touch a Homebrew install, use
  `brew upgrade taskpoet`.
* **Installed somewhere you can't write**, like `/usr/local/bin` from a
  `make install` with `sudo`: run `sudo taskpoet update` (or use
  `make install PREFIX=$HOME/.local` and it just works).
* **Built from source** at a commit that isn't a release, `taskpoet --version`
  says something like `v2.0.0-3-gabc123` or `dev`. Those can't tell whether a
  release is newer, so they neither check nor update.
