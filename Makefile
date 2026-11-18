GO      ?= go
GOFMT   ?= gofmt
# The golangci-lint on PATH can be a release too old to read the export data
# this toolchain writes, and then it cannot type-check the module, so the gate
# builds its own at a pinned version.
GOLINT_VERSION := v2.14.0
GOLINT  := bin/golangci-lint

.DEFAULT_GOAL := help

.PHONY: help vet lint lint-fix fmt-check-tests generate-check test-race check FORCE

# The steps of check run in the order written and one Go job at a time, also
# under make -j.
.NOTPARALLEL:

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "\nUsage:\n  make <target>\n\nTargets:\n"} \
	/^[a-zA-Z0-9_.-]+:.*## / { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ── Vet ────────────────────────────────────────────────────────────────────────

vet: ## Run go vet
	$(GO) vet ./...

# ── Lint / Format ──────────────────────────────────────────────────────────────

# The Go toolchain that builds and loads this module and the linter's pinned
# version, recorded in a file whose mtime moves only when one of them does. Go
# selects the toolchain from go.mod's go line and the installed one, so a
# change in either changes it. A tool built by an older Go cannot read the
# export data a newer compiler writes, so the linter depends on this file and
# is rebuilt when it moves, and only then.
GOLINT_STAMP := bin/.golangci-lint-stamp

$(GOLINT_STAMP): FORCE
	@mkdir -p $(@D)
	@v="$$($(GO) env GOVERSION) $(GOLINT_VERSION)"; [ "$$(cat $@ 2>/dev/null)" = "$$v" ] || echo "$$v" > $@

FORCE:

# The install resolves the linter's own module, never this one, and GOFLAGS is
# emptied so a flag from the environment (-mod=vendor) cannot reach it. go
# install can leave an up-to-date binary's mtime alone, hence the touch. The
# first build needs network.
$(GOLINT): $(GOLINT_STAMP)
	GOBIN=$(CURDIR)/bin GOFLAGS= $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLINT_VERSION)
	touch $@

lint: $(GOLINT) ## Run the pinned golangci-lint (all configured linters, gofmt and goimports on non-test files)
	$(GOLINT) run ./...

lint-fix: $(GOLINT) ## Run golangci-lint with auto-fix
	$(GOLINT) run --fix ./...

# .golangci.yml sets run.tests: false, so the linter's gofmt never sees a
# _test.go file. This covers them: tracked test files only, so an untracked
# scratch file cannot fail the gate. The first line fails the step where git
# cannot list files, which the pipeline below would report as nothing to format.
fmt-check-tests: ## Verify gofmt -s on tracked _test.go files (read-only)
	@git rev-parse --is-inside-work-tree >/dev/null
	@unformatted=$$(git ls-files -z -- '*_test.go' | xargs -0 -r $(GOFMT) -s -l); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt -s needed on:"; echo "$$unformatted"; exit 1; \
	fi

# ── Generated code ─────────────────────────────────────────────────────────────

# Compares the file before and after regenerating it, so the result does not
# depend on what is staged. A failure leaves the regenerated file in place: it
# is the fix, to review and commit.
generate-check: ## Regenerate shorthand.go and fail if it changed
	@before=$$(cksum shorthand.go); \
	$(GO) generate ./... || exit 1; \
	if [ "$$before" != "$$(cksum shorthand.go)" ]; then \
		echo "shorthand.go was out of date and has been regenerated: review and commit it"; exit 1; \
	fi

# ── Test ───────────────────────────────────────────────────────────────────────

# The command of the CI workflow's "Run tests" step, without -v.
test-race: ## Run the tests with the race detector and coverage
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out -count=1 ./...

# ── Composite ──────────────────────────────────────────────────────────────────

check: vet lint fmt-check-tests generate-check test-race ## Full pre-commit check: vet + lint + gofmt -s on test files + generated code up to date + test -race
	@echo ""
	@echo "All checks passed"
