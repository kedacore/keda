SOURCE_FILES?=$$(go list ./... | grep -v /vendor/)
TEST_PATTERN?=.
TEST_OPTIONS?=
VERSION?=$$(cat VERSION)
LINTER?=$$(which golangci-lint)
LINTER_VERSION=2.5.0

ifeq ($(OS),Windows_NT)
	LINTER_FILE=golangci-lint-$(LINTER_VERSION)-windows-amd64.zip
	LINTER_UNPACK= >| app.zip; unzip -j app.zip -d $$GOPATH/bin; rm app.zip
else ifeq ($(OS), Darwin)
	LINTER_FILE=golangci-lint-$(LINTER_VERSION)-darwin-amd64.tar.gz
	LINTER_UNPACK= | tar xzf - -C $$GOPATH/bin --wildcards --strip 1 "**/golangci-lint"
else
	LINTER_FILE=golangci-lint-$(LINTER_VERSION)-linux-amd64.tar.gz
	LINTER_UNPACK= | tar xzf - -C $$GOPATH/bin --wildcards --strip 1 "**/golangci-lint"
endif

setup:
	go install github.com/pierrre/gotestcover@latest
	go install golang.org/x/tools/cmd/cover@latest
	go install github.com/robertkrimen/godocdown/godocdown@latest
	go mod download

generate: ## Generate README.md
	godocdown >| README.md

test: generate test_and_cover_report lint

test_and_cover_report:
	gotestcover $(TEST_OPTIONS) -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m

cover: test ## Run all the tests and opens the coverage report
	go tool cover -html=coverage.txt

fmt: ## gofmt and goimports all go files
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

lint: ## Run all the linters
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v$(LINTER_VERSION) golangci-lint run

ci: test_and_cover_report ## Run all the tests but no linters - use https://golangci.com integration instead

build:
	go build

release: ## Release new version
	git tag | grep -q $(VERSION) && echo This version was released! Increase VERSION! || git tag $(VERSION) && git push origin $(VERSION) && git tag v$(VERSION) && git push origin v$(VERSION)

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := build
