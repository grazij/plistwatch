BINARY = plistwatch
PREFIX ?= /usr/local
DIST = dist

# main.go's version const is the only place the version lives; the release
# workflow overrides this with the pushed tag.
VERSION ?= $(shell sed -n 's/^const version = "\(.*\)"[[:space:]]*$$/\1/p' main.go)

.PHONY: all build vet clean install uninstall universal dist

all: build

# Native single-arch build. Never use ./... (vendored go-plist/cmd breaks it).
build:
	go build .

vet:
	go vet .

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

# Universal (arm64 + x86_64) macOS fat binary at ./plistwatch.
universal:
	./build-macos-universal.sh

# Release artifacts in ./dist: a universal tarball and its checksum. The
# release workflow uploads whatever lands there.
dist: universal
	rm -rf $(DIST)
	mkdir -p $(DIST)
	tar -czf $(DIST)/$(BINARY)-$(VERSION)-darwin-universal.tar.gz $(BINARY)
	cd $(DIST) && shasum -a 256 *.tar.gz > SHA256SUMS
