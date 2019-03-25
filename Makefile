SHELL:=/bin/bash

TMP = ._tmp
COVERAGE_DIR = coverage
WORKSPACE = $(shell pwd)

all: tools vendor fmt lint install test coverage
all-ci: vendor fmt lint-report install test-report coverage

tools: # installs tools required for build (will complain about modules in this command but its ok)
	@go get -u golang.org/x/tools/cmd/cover
	# Should fail if not available locally, its better practice to fix the version
	command golangci-lint || true

lint:
	@test -z $(gofmt -s -l $GO_FILES)
	@golangci-lint run

.PHONY: test
test: ## Runs the go tests.
	@go test -v $(shell go list ./... | grep -v vendor)

.PHONY: test
test-report: ## Runs the go tests while producing a report
	@go test -v $(shell go list ./... | grep -v vendor) | tee report.json

.PHONY: fmt
fmt: ## Verifies all files have been `gofmt`ed.
	@if [[ ! -z "$(shell gofmt -s -l . | grep -v '.pb.go:' | grep -v '.twirp.go:' | grep -v vendor | tee /dev/stderr)" ]]; then \
		exit 1; \
	fi


.PHONY: coverage
coverage:  ## Runs go test with coverage.
	@rm -f coverage.txt
	@echo "mode: atomic" > coverage.txt
	@for d in $(shell go list ./... | grep -v vendor); do \
		go test -race -coverprofile=profile.out -covermode=atomic "$$d"; \
		if [ -f profile.out ]; then \
		 	tail -n +2 profile.out >> coverage.txt; \
			rm profile.out; \
		fi; \
done;

.PHONY: install
install: ## Installs the  package.
	go install -a $(shell go list ./... | grep -v vendor)

lint-report:
	set -o pipefail
	golangci-lint run --out-format tab | tee golangci-report.out

.PHONY: vendor
vendor: ## Updates the vendoring directory.
	@$(RM) go.sum
	@$(RM) -r vendor
	GO111MODULE=on go mod init || true
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod vendor
	@$(RM) Gopkg.toml Gopkg.lock