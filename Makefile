BINARY = plistwatch
VERSION ?= 2025.09.24+grazij.4
# A literal + in a URL path is ambiguous enough that GitHub's redirects
# mishandle it; %2B is not. The git tag itself keeps the literal +.
TAG_PATH = v$(subst +,%2B,$(VERSION))
PREFIX ?= /usr/local

# GitHub coordinates the `formula` target builds the tarball URL from. Override
# on the command line if your layout differs:
#   make formula GITHUB_USER=alice GITHUB_REPO=plistwatch
GITHUB_USER ?= grazij
GITHUB_REPO ?= plistwatch

.PHONY: all build vet clean install uninstall universal formula

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

# Bump the Homebrew formula to v$(VERSION): compute the SHA256 of the tagged
# tarball, rewrite Formula/plistwatch.rb, then commit + push it *here*. Assumes
# the tag has already been pushed to GitHub.
#
# The formula is only authored in this repo. Copying it into grazij/homebrew-tap
# is a manual step — nothing here reads, writes or pushes to the tap.
formula:
	@set -e; \
	echo "==> $(BINARY) $(VERSION) — bumping Homebrew formula"; \
	tarball="https://github.com/$(GITHUB_USER)/$(GITHUB_REPO)/archive/refs/tags/$(TAG_PATH).tar.gz"; \
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
	sed -i.bak -E "s|^(  version )\".*\"|\1\"$(VERSION)\"|" Formula/plistwatch.rb; \
	sed -i.bak -E "s|^(  sha256 )\".*\"|\1\"$$sha\"|" Formula/plistwatch.rb; \
	rm -f Formula/plistwatch.rb.bak; \
	grep -q "^  version \"$(VERSION)\"$$" Formula/plistwatch.rb || \
		{ echo "error: version line not rewritten in Formula/plistwatch.rb" >&2; exit 1; }; \
	if git diff --quiet -- Formula/plistwatch.rb; then \
		echo "==> Formula/plistwatch.rb already at $(VERSION); nothing to commit"; \
		exit 0; \
	fi; \
	git add Formula/plistwatch.rb; \
	git commit -m "chore(formula): bump to v$(VERSION)"; \
	git push origin main; \
	echo "==> Formula/plistwatch.rb updated and pushed"; \
	echo "    next (by hand): copy it into the grazij/homebrew-tap checkout,"; \
	echo "    chmod 644 the copy, commit and push there."
