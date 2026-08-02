BINARY = plistwatch
VERSION ?= 2025.09.24+grazij.4
# A literal + in a URL path is ambiguous enough that GitHub's redirects
# mishandle it; %2B is not. The git tag itself keeps the literal +.
TAG_PATH = v$(subst +,%2B,$(VERSION))
PREFIX ?= /usr/local

# Homebrew tap location and GitHub coordinates used by the `formula` target.
# Override on the command line if your layout differs:
#   make formula TAP_DIR=../my-tap GITHUB_USER=alice GITHUB_REPO=plistwatch
TAP_DIR ?= ../homebrew-tap
GITHUB_USER ?= grazij
GITHUB_REPO ?= plistwatch

# The tap push is the last step of `formula` and the one that has actually failed
# in practice (transient "Connection closed by ... port 22" from GitHub), leaving
# the commit stranded locally. Retry it.
PUSH_RETRIES ?= 3
PUSH_RETRY_DELAY ?= 5

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
	git add Formula/plistwatch.rb; \
	git commit -m "chore(formula): bump to v$(VERSION)"; \
	git push origin main; \
	cp Formula/plistwatch.rb "$(TAP_DIR)/Formula/plistwatch.rb"; \
	chmod 644 "$(TAP_DIR)/Formula/plistwatch.rb"; \
	cd "$(TAP_DIR)"; \
	git add Formula/plistwatch.rb; \
	git diff --cached --stat; \
	git commit -m "$(BINARY) $(VERSION)"; \
	attempt=1; \
	until git push origin main; do \
		if [ "$$attempt" -ge "$(PUSH_RETRIES)" ]; then \
			echo "error: tap push failed after $(PUSH_RETRIES) attempts; the commit is" >&2; \
			echo "       local in $(TAP_DIR). If the tap is behind its remote, run" >&2; \
			echo "       git -C $(TAP_DIR) pull --rebase origin main and push again." >&2; \
			exit 1; \
		fi; \
		echo "    tap push failed, retrying in $(PUSH_RETRY_DELAY)s ($$attempt/$(PUSH_RETRIES))"; \
		sleep $(PUSH_RETRY_DELAY); \
		attempt=$$((attempt + 1)); \
	done; \
	echo "==> formula published to $(GITHUB_USER)/homebrew-tap"; \
	echo "    sanity check: make formula-verify"

# First-time / sanity-check install via the published tap.
#
# Don't untap to refresh: `brew untap` refuses while any formula from the tap is
# installed (duti lives in the same tap), and `|| true` used to swallow that,
# leaving a stale clone that silently installed the previous release. Fetch the
# tap clone directly instead, and assert the installed version.
formula-verify:
	@set -e; \
	brew tap $(GITHUB_USER)/tap; \
	tap_dir=$$(brew --repository $(GITHUB_USER)/tap); \
	git -C "$$tap_dir" fetch origin main; \
	git -C "$$tap_dir" reset --hard origin/main; \
	chmod 644 "$$tap_dir/Formula/$(BINARY).rb"; \
	brew install $(GITHUB_USER)/tap/$(BINARY) || \
		brew upgrade $(GITHUB_USER)/tap/$(BINARY); \
	got=$$($(BINARY) --version); \
	echo "==> $$got"; \
	[ "$$got" = "$(BINARY) $(VERSION)" ] || \
		{ echo "error: expected '$(BINARY) $(VERSION)'" >&2; exit 1; }; \
	brew uninstall $(BINARY)
