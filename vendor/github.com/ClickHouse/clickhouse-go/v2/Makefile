CLICKHOUSE_VERSION ?= latest
CLICKHOUSE_TEST_TIMEOUT ?= 240s
CLICKHOUSE_QUORUM_INSERT ?= 1

up:
	@docker compose up -d
down:
	@docker compose down

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
