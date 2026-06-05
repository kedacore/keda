SHELL := /bin/bash

##@ Format project using goimports tool
format:
	goimports -w .;

##@ Test
test-unit:  ## Run unit tests
	@printf "\033[2m→ Running unit tests...\033[0m\n"
ifdef race
	$(eval testunitargs += "-race")
endif
	$(eval testunitargs += "-cover" "./..." "-args" "-test.gocoverdir=$(PWD)/tmp/unit")
	@mkdir -p $(PWD)/tmp/unit
	@echo "go test -v" $(testunitargs); \
	go test -v $(testunitargs);
ifdef coverage
	@go tool covdata textfmt -i=$(PWD)/tmp/unit -o $(PWD)/tmp/unit.cov
endif
test: test-unit

test-integ:  ## Run integration tests
	@printf "\033[2m→ Running integration tests...\033[0m\n"
	$(eval testintegtags += "integration,core,plugins")
ifdef multinode
	$(eval testintegtags += "multinode")
endif
ifdef race
	$(eval testintegargs += "-race")
endif
	$(eval testintegargs += "-cover" "-tags=$(testintegtags)" "-timeout=1h" "./..." "-args" "-test.gocoverdir=$(PWD)/tmp/integration")
	@mkdir -p $(PWD)/tmp/integration
	@echo "go test -v" $(testintegargs); \
	go test -v $(testintegargs);
ifdef coverage
	@go tool covdata textfmt -i=$(PWD)/tmp/integration -o $(PWD)/tmp/integ.cov
endif

test-integ-core:  ## Run base integration tests
	@make test-integ testintegtags=integration,core

test-integ-plugins:  ## Run plugin integration tests
	@make test-integ testintegtags=integration,plugins

test-integ-secure: ##Run secure integration tests
	@SECURE_INTEGRATION=true make test-integ

test-integ-core-secure:  ## Run secure base integration tests
	@SECURE_INTEGRATION=true make test-integ testintegtags=integration,core

test-integ-plugins-secure:  ## Run secure plugin integration tests
	@SECURE_INTEGRATION=true make test-integ testintegtags=integration,plugins


test-bench:  ## Run benchmarks
	@printf "\033[2m→ Running benchmarks...\033[0m\n"
	go test -run=none -bench=. -benchmem ./...

coverage:  ## Print test coverage report
	@make gen-coverage
	@go tool cover -func=$(PWD)/tmp/total.cov
	@printf "\033[0m--------------------------------------------------------------------------------\n\033[0m"

coverage-html: ## Open test coverage report in browser
	@make gen-coverage
	@go tool cover -html $(PWD)/tmp/total.cov

gen-coverage:  ## Generate test coverage report
	@printf "\033[2m→ Generating test coverage report...\033[0m\n"
	@rm -rf tmp
	@mkdir tmp
	@mkdir tmp/unit
	@mkdir tmp/integration
	@make test-unit coverage=true
	@make test-integ coverage=true
	@make build-coverage

build-coverage:
	@go tool covdata textfmt -i=$(PWD)/tmp/unit,$(PWD)/tmp/integration -o $(PWD)/tmp/total.cov

##@ Development
lint:  ## Run lint on the package
	@make linters

package := "prettier"
lint.markdown:
	@printf "\033[2m→ Checking node installed...\033[0m\n"
	if type node > /dev/null 2>&1 && which node > /dev/null 2>&1 ; then \
		node -v; \
		echo -e "\033[33m Node is installed, continue...\033[0m\n"; \
	else \
		echo -e "\033[31m Please install node\033[0m\n"; \
		exit 1; \
	fi
	@printf "\033[2m→ Checking npm installed...\033[0m\n"
	if type npm > /dev/null 2>&1 && which npm > /dev/null 2>&1 ; then \
		npm -v; \
		echo -e "\033[33m NPM is installed, continue...\033[0m\n"; \
	else \
		echo -e "\033[31m Please install npm\033[0m\n"; \
		exit 1; \
	fi
	@printf "\033[2m→ Checking $(package) installed...\033[0m\n"
	if [ `npm list -g | grep -c $(package)` -eq 0 -o ! -d node_module ]; then \
		echo -e "\033[33m Installing $(package)...\033[0m"; \
		npm install -g $(package) --no-shrinkwrap; \
	fi
	@printf "\033[2m→ Running markdown lint...\033[0m\n"
	if npx $(package) --prose-wrap never --check **/*.md; [[ $$? -ne 0 ]]; then \
		echo -e "\033[32m→ Found invalid files. Want to auto-format invalid files? (y/n) \033[0m"; \
		read RESP; \
		if [[ $$RESP = "y" || $$RESP = "Y" ]]; then \
		  echo -e "\033[33m Formatting...\033[0m"; \
		  npx $(package) --prose-wrap never --write **/*.md; \
		  echo -e "\033[34m \nAll invalid files are formatted\033[0m"; \
		else \
		  echo -e "\033[33m Unfortunately you are cancelled auto fixing. But we will definitely fix it in the pipeline\033[0m"; \
		fi \
	fi


backport: ## Backport one or more commits from main into version branches
ifeq ($(origin commits), undefined)
	@echo "Missing commit(s), exiting..."
	@exit 2
endif
ifndef branches
	$(eval branches_list = '1.x')
else
	$(eval branches_list = $(shell echo $(branches) | tr ',' ' ') )
endif
	$(eval commits_list = $(shell echo $(commits) | tr ',' ' '))
	@printf "\033[2m→ Backporting commits [$(commits)]\033[0m\n"
	@{ \
		set -e -o pipefail; \
		for commit in $(commits_list); do \
			git show --pretty='%h | %s' --no-patch $$commit; \
		done; \
		echo ""; \
		for branch in $(branches_list); do \
			printf "\033[2m→ $$branch\033[0m\n"; \
			git checkout $$branch; \
			for commit in $(commits_list); do \
				git cherry-pick -x $$commit; \
			done; \
			git status --short --branch; \
			echo ""; \
		done; \
		printf "\033[2m→ Push updates to Github:\033[0m\n"; \
		for branch in $(branches_list); do \
			echo "git push --verbose origin $$branch"; \
		done; \
	}

release: ## Release a new version to Github
	$(eval branch = $(shell git rev-parse --abbrev-ref HEAD))
	$(eval current_version = $(shell cat internal/version/version.go | sed -Ee 's/const Client = "(.*)"/\1/' | tail -1))
	@printf "\033[2m→ [$(branch)] Current version: $(current_version)...\033[0m\n"
ifndef version
	@printf "\033[31m[!] Missing version argument, exiting...\033[0m\n"
	@exit 2
endif
ifeq ($(version), "")
	@printf "\033[31m[!] Empty version argument, exiting...\033[0m\n"
	@exit 2
endif
	@printf "\033[2m→ [$(branch)] Creating version $(version)...\033[0m\n"
	@{ \
		set -e -o pipefail; \
		cp internal/version/version.go internal/version/version.go.OLD && \
		cat internal/version/version.go.OLD | sed -e 's/Client = ".*"/Client = "$(version)"/' > internal/version/version.go && \
		go vet internal/version/version.go && \
		go fmt internal/version/version.go && \
		git diff --color-words internal/version/version.go | tail -n 1; \
	}
	@{ \
		set -e -o pipefail; \
		printf "\033[2m→ Commit and create Git tag? (y/n): \033[0m\c"; \
		read continue; \
		if [[ $$continue == "y" ]]; then \
			git add internal/version/version.go && \
			git commit --no-status --quiet --message "Release $(version)" && \
			git tag --annotate v$(version) --message 'Release $(version)'; \
			printf "\033[2m→ Push `git show --pretty='%h (%s)' --no-patch HEAD` to Github:\033[0m\n\n"; \
			printf "\033[1m  git push origin HEAD && git push origin v$(version)\033[0m\n\n"; \
			mv internal/version/version.go.OLD internal/version/version.go && \
			git add internal/version/version.go && \
			original_version=`cat internal/version/version.go | sed -ne 's;^const Client = "\(.*\)"$$;\1;p'` && \
			git commit --no-status --quiet --message "Update version to $$original_version"; \
			printf "\033[2m→ Version updated to [$$original_version].\033[0m\n\n"; \
		else \
			echo "Aborting..."; \
			rm internal/version/version.go.OLD; \
			exit 1; \
		fi; \
	}

godoc: ## Display documentation for the package
	@printf "\033[2m→ Generating documentation...\033[0m\n"
	@echo "* http://localhost:6060/pkg/github.com/opensearch-project/opensearch-go"
	@echo "* http://localhost:6060/pkg/github.com/opensearch-project/opensearch-go/opensearchapi"
	@echo "* http://localhost:6060/pkg/github.com/opensearch-project/opensearch-go/opensearchtransport"
	@echo "* http://localhost:6060/pkg/github.com/opensearch-project/opensearch-go/opensearchutil"
	@printf "\n"
	godoc --http=localhost:6060 --play

cluster.build:
	docker compose --project-directory .ci/opensearch build --pull;

cluster.start:
	docker compose --project-directory .ci/opensearch up -d;

cluster.stop:
	docker compose --project-directory .ci/opensearch down;

cluster.get-cert:
	@if [[ -v SECURE_INTEGRATION ]] && [[ $$SECURE_INTEGRATION == "true" ]]; then \
		docker cp $$(docker compose --project-directory .ci/opensearch ps --format '{{.Name}}'):/usr/share/opensearch/config/kirk.pem admin.pem && \
		docker cp $$(docker compose --project-directory .ci/opensearch ps --format '{{.Name}}'):/usr/share/opensearch/config/kirk-key.pem admin.key; \
	fi


cluster.clean: ## Remove unused Docker volumes and networks
	@printf "\033[2m→ Cleaning up Docker assets...\033[0m\n"
	docker volume prune --force
	docker network prune --force
	docker system prune --volumes --force

linters:
	docker run -t --rm -v $$(pwd):/app -v ~/.cache/golangci-lint/v2.7.2:/root/.cache -w /app golangci/golangci-lint:v2.7.2 golangci-lint run --timeout=5m -v

workflow: ## Run all github workflow commands here sequentially

# Lint
	make lint
# License Checker
	.github/check-license-headers.sh
# Unit Test
	make test-unit race=true
# Benchmarks Test
	make test-bench
# Integration Test
### OpenSearch
	make cluster.clean cluster.build cluster.start
	make test-integ race=true
	make cluster.stop

##@ Other
#------------------------------------------------------------------------------
help:  ## Display help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
#------------- <https://suva.sh/posts/well-documented-makefiles> --------------

.DEFAULT_GOAL := help
.PHONY: help backport cluster cluster.clean coverage  godoc lint release test test-bench test-integ test-unit linters linters.install
.SILENT: lint.markdown
