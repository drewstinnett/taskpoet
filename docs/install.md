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
