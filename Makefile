# Build and install taskpoet from source.
#
#   make install                       # to /usr/local/bin
#   make install PREFIX=$HOME/.local   # to ~/.local/bin, no sudo needed
#   make install BINDIR=$HOME/bin      # to exactly this directory
#   make install DESTDIR=/tmp/pkg      # staging root for packagers
#
# The version is stamped in by the Makefile, so Go's own VCS stamping (which
# needs a working git) is turned off. Releases are built by goreleaser, see
# .goreleaser.yaml.

BINARY  := taskpoet
PKG     := github.com/drewstinnett/taskpoet/v2

PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
DESTDIR ?=

# What 'taskpoet --version' says. Falls back to "dev" without git or tags.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG)/cmd/taskpoet/cmd.version=$(VERSION)

.PHONY: build install uninstall clean

build:
	go build -trimpath -buildvcs=false -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/taskpoet

install: build
	install -d '$(DESTDIR)$(BINDIR)'
	install -m 0755 bin/$(BINARY) '$(DESTDIR)$(BINDIR)/$(BINARY)'
	@echo "Installed $(BINARY) $(VERSION) to $(DESTDIR)$(BINDIR)/$(BINARY)"

uninstall:
	rm -f '$(DESTDIR)$(BINDIR)/$(BINARY)'

clean:
	rm -rf bin
