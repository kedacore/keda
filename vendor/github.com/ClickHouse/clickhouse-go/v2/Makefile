CLICKHOUSE_VERSION ?= latest
CLICKHOUSE_TEST_TIMEOUT ?= 240s
CLICKHOUSE_QUORUM_INSERT ?= 1
COMPOSE_PROJECT_NAME ?= clickhouse-go

up:
	@docker ps -aqf "name=^/clickhouse$$" | xargs -r docker rm -f
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker compose up --wait --remove-orphans
down:
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker compose down

up-cluster:
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker compose -f docker-compose.cluster.yml up --force-recreate --remove-orphans

down-cluster:
	@COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker compose -f docker-compose.cluster.yml down

cli:
	docker run -it --rm --net clickhouse-go_clickhouse --link clickhouse:clickhouse-server --host clickhouse-server

test:
	@go install -race -v
	@CLICKHOUSE_VERSION=$(CLICKHOUSE_VERSION) CLICKHOUSE_QUORUM_INSERT=$(CLICKHOUSE_QUORUM_INSERT) go test -race -timeout $(CLICKHOUSE_TEST_TIMEOUT) -count=1 -v ./...

lint:
	golangci-lint run || :

contributors:
	@git log --pretty="%an <%ae>%n%cn <%ce>" | sort -u -t '<' -k 2,2 | LC_ALL=C sort | \
		grep -v "users.noreply.github.com\|GitHub <noreply@github.com>" \
		> contributors/list

staticcheck:
	staticcheck ./...

codegen: contributors
	@go run lib/column/codegen/main.go
	@go-licenser -licensor "ClickHouse, Inc."

.PHONY: contributors
