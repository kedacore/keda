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
tests: ## Run all tests and requires a running rabbitmq-server. Use GO_TEST_FLAGS to add extra flags to go test
	go test -race -v -tags integration $(GO_TEST_FLAGS)

.PHONY: tests-docker
tests-docker: rabbitmq-server
	RABBITMQ_RABBITMQCTL_PATH="DOCKER:$(CONTAINER_NAME)" go test -race -v -tags integration $(GO_TEST_FLAGS)
	$(MAKE) stop-rabbitmq-server

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

certs:
	./certs.sh

.PHONY: certs-rm
certs-rm:
	rm -r ./certs/

.PHONY: rabbitmq-server-tls
rabbitmq-server-tls: | certs ## Start a RabbitMQ server using Docker. Container name can be customised with CONTAINER_NAME=some-rabbit
	docker run --detach --rm --name $(CONTAINER_NAME) \
		--publish 5672:5672 --publish 5671:5671 --publish 15672:15672 \
		--mount type=bind,src=./certs/server,dst=/certs \
		--mount type=bind,src=./certs/ca/cacert.pem,dst=/certs/cacert.pem,readonly \
		--mount type=bind,src=./rabbitmq-confs/tls/90-tls.conf,dst=/etc/rabbitmq/conf.d/90-tls.conf \
		--pull always rabbitmq:3-management
