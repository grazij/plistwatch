BINARY = plistwatch
VERSION ?= 2025.9.24.1
PREFIX ?= /usr/local

# Homebrew tap location and GitHub coordinates used by the `formula` target.
# Override on the command line if your layout differs:
#   make formula TAP_DIR=../my-tap GITHUB_USER=alice GITHUB_REPO=plistwatch
TAP_DIR ?= ../homebrew-tap
GITHUB_USER ?= grazij
GITHUB_REPO ?= plistwatch

.PHONY: all build vet clean install uninstall universal formula formula-verify

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

# Bump the Homebrew formula to v$(VERSION): compute the SHA256 of the
# tagged tarball, rewrite Formula/plistwatch.rb, commit + push here, then
# mirror to TAP_DIR and commit + push there. Assumes the tag has already
# been pushed to GitHub.
formula:
	@set -e; \
	echo "==> $(BINARY) $(VERSION) — bumping Homebrew formula"; \
	if [ ! -d "$(TAP_DIR)/Formula" ]; then \
		echo "error: $(TAP_DIR)/Formula not found (override with TAP_DIR=...)" >&2; \
		exit 1; \
	fi; \
	tarball="https://github.com/$(GITHUB_USER)/$(GITHUB_REPO)/archive/refs/tags/v$(VERSION).tar.gz"; \
	echo "    fetching $$tarball"; \
	tmp=$$(mktemp); \
	curl -fsSL "$$tarball" -o "$$tmp"; \
	sha=$$(shasum -a 256 "$$tmp" | awk '{print $$1}'); \
	rm -f "$$tmp"; \
	if [ -z "$$sha" ]; then \
		echo "error: empty SHA — is tag v$(VERSION) pushed to $(GITHUB_USER)/$(GITHUB_REPO)?" >&2; \
		exit 1; \
	fi; \
	echo "    sha256: $$sha"; \
	sed -i.bak -E "s|^(  url )\".*\"|\1\"$$tarball\"|" Formula/plistwatch.rb; \
	sed -i.bak -E "s|^(  sha256 )\".*\"|\1\"$$sha\"|" Formula/plistwatch.rb; \
	rm -f Formula/plistwatch.rb.bak; \
	git add Formula/plistwatch.rb; \
	git commit -m "chore(formula): bump to v$(VERSION)"; \
	git push origin main; \
	cp Formula/plistwatch.rb "$(TAP_DIR)/Formula/plistwatch.rb"; \
	cd "$(TAP_DIR)" && \
		git add Formula/plistwatch.rb && \
		git diff --cached --stat && \
		git commit -m "$(BINARY) $(VERSION)" && \
		git push origin main; \
	echo "==> formula published to $(GITHUB_USER)/homebrew-tap"; \
	echo "    sanity check: make formula-verify"

# First-time / sanity-check install via the published tap.
formula-verify:
	brew untap $(GITHUB_USER)/tap 2>/dev/null || true
	brew tap $(GITHUB_USER)/tap
	brew install $(GITHUB_USER)/tap/$(BINARY)
	$(BINARY) --version
	brew uninstall $(BINARY)
