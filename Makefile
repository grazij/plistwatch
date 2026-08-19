BINARY = plistwatch
PREFIX ?= /usr/local

.PHONY: all build vet clean install uninstall universal

all: build

# Native single-arch build. Never use ./... (vendored go-plist/cmd breaks it).
build:
	go build .

vet:
	go vet .

clean:
	rm -f $(BINARY)

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

# Universal (arm64 + x86_64) macOS fat binary at ./plistwatch.
universal:
	./build-macos-universal.sh
