BIN      := bin
STATE    := $(CURDIR)/.state
PKGS     := ./...
GO       ?= go

.PHONY: help
help: ## show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## build the published programs
	$(GO) build -o $(BIN)/team-server ./cmd/team-server
	$(GO) build -o $(BIN)/beacon-stub ./cmd/beacon
	@echo "built: $(BIN)/team-server $(BIN)/beacon-stub"

.PHONY: lab-beacon
lab-beacon: ## build the working Beacon (not published)
	$(GO) build -ldflags="-s -w" -o $(BIN)/beacon ./lab/beacon-impl

.PHONY: strip-beacon
strip-beacon: lab-beacon ## report the stripped size of the working Beacon
	@ls -lh $(BIN)/beacon

.PHONY: test
test: ## run the tests with the race detector
	$(GO) test -race -count=1 $(PKGS)

.PHONY: cover
cover: ## run the tests and report coverage
	$(GO) test -race -count=1 -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -func=coverage.out | tail -1

.PHONY: ste
ste: ## check the documents against the mechanical rules of ASD-STE100 Issue 9
	$(GO) run ./tools/stelint README.md docs/*.md CODE_OF_CONDUCT.md

.PHONY: vet
vet: ## run go vet
	$(GO) vet $(PKGS) ./lab/...

.PHONY: fmt
fmt: ## format the code
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## fail when the code is not formatted
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "not formatted:"; echo "$$out"; exit 1; fi

.PHONY: lint
lint: ## run staticcheck and gosec when they are installed
	@command -v staticcheck >/dev/null 2>&1 \
		&& staticcheck $(PKGS) \
		|| echo "staticcheck not installed, skipped"
	@command -v gosec >/dev/null 2>&1 \
		&& gosec -quiet -exclude-dir=lab ./... \
		|| echo "gosec not installed, skipped"

.PHONY: vuln
vuln: ## check the dependencies for known vulnerabilities
	@command -v govulncheck >/dev/null 2>&1 \
		&& govulncheck $(PKGS) \
		|| echo "govulncheck not installed, skipped"

.PHONY: tidy
tidy: ## tidy the module files
	$(GO) mod tidy

.PHONY: check
check: fmt-check vet test ste ## run every check that CI runs

.PHONY: run-server
run-server: build ## run the Team Server on 127.0.0.1:8443
	$(BIN)/team-server -addr 127.0.0.1:8443 -state $(STATE)

.PHONY: show-secret
show-secret: ## print the shared secret that the beacon needs
	@cat $(STATE)/c2.key

.PHONY: clean
clean: ## remove build output
	rm -rf $(BIN) coverage.out
