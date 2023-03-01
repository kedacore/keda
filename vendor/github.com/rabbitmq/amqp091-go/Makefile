.DEFAULT_GOAL := list

# Insert a comment starting with '##' after a target, and it will be printed by 'make' and 'make list'
.PHONY: list
list: ## list Makefile targets
	@echo "The most used targets: \n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: check-fmt
check-fmt: ## Ensure code is formatted
	gofmt -l -d . 	# For the sake of debugging
	test -z "$$(gofmt -l .)"

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: tests
tests: ## Run all tests and requires a running rabbitmq-server
	go test -cpu 1,2 -race -v -tags integration

.PHONY: check
check:
	golangci-lint run ./...

CONTAINER_NAME ?= amqp091-go-rabbitmq

.PHONY: rabbitmq-server
rabbitmq-server: ## Start a RabbitMQ server using Docker. Container name can be customised with CONTAINER_NAME=some-rabbit
	docker run --detach --rm --name $(CONTAINER_NAME) \
		--publish 5672:5672 --publish 15672:15672 \
		--pull always rabbitmq:3-management

.PHONY: stop-rabbitmq-server
stop-rabbitmq-server: ## Stop a RabbitMQ server using Docker. Container name can be customised with CONTAINER_NAME=some-rabbit
	docker stop $(CONTAINER_NAME)
