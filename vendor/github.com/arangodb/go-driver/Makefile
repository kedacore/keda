PROJECT := go-driver
SCRIPTDIR := $(shell pwd)

CURR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
ROOTDIR:=$(CURR)

GOVERSION ?= 1.24.9
GOTOOLCHAIN ?= auto
GOIMAGE ?= golang:$(GOVERSION)
GOV2IMAGE ?= $(GOIMAGE)
ALPINE_IMAGE ?= alpine:3.21
TMPDIR := ${SCRIPTDIR}/.tmp

DOCKER_CMD:=docker run

GOBUILDTAGS:=$(TAGS)
GOBUILDTAGSOPT=-tags "$(GOBUILDTAGS)"

ARANGODB ?= arangodb/enterprise:latest
STARTER ?= arangodb/arangodb-starter:latest

ifdef VERBOSE
	TESTVERBOSEOPTIONS := -v
endif

CGO_ENABLED=0
ifdef RACE
	TESTVERBOSEOPTIONS += -race
	CGO_ENABLED=1
endif

ifndef AF_ENABLED
	AF_ENABLED := "false"
endif

ifndef VST_ENABLED
	VST_ENABLED := "false"
endif

TESTV2PARALLEL ?= 1

ORGPATH := github.com/arangodb
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)

SOURCES_EXCLUDE:=vendor
SOURCES := $(shell find "$(ROOTDIR)" $(foreach SOURCE,$(SOURCES_EXCLUDE),-not -path '$(ROOTDIR)/$(SOURCE)/*') -name '*.go')

# Test variables

ifndef TESTCONTAINER
	TESTCONTAINER := $(PROJECT)-test
endif
ifndef DBCONTAINER
	DBCONTAINER := $(TESTCONTAINER)-db
endif 

ifeq ("$(TEST_AUTH)", "none")
	ARANGOENV := -e ARANGO_NO_AUTH=1
	TEST_AUTHENTICATION :=
	TESTS := $(REPOPATH) $(REPOPATH)/test
else ifeq ("$(TEST_AUTH)", "rootpw")
	ARANGOENV := -e ARANGO_ROOT_PASSWORD=rootpw
	TEST_AUTHENTICATION := basic:root:rootpw
	GOBUILDTAGS += auth
	TESTS := $(REPOPATH)/test
else ifeq ("$(TEST_AUTH)", "jwt")
	ARANGOENV := -e ARANGO_ROOT_PASSWORD=rootpw 
	TEST_AUTHENTICATION := jwt:root:rootpw
	GOBUILDTAGS += auth
	TESTS := $(REPOPATH)/test
	JWTSECRET := testing
	JWTSECRETFILE := "${TMPDIR}/${TESTCONTAINER}-jwtsecret"
	ARANGOVOL := -v "$(JWTSECRETFILE):/jwtsecret"
	ARANGOARGS := --server.jwt-secret=/jwtsecret
endif

TEST_NET := --net=host

# By default we run tests against single endpoint to avoid problems with data propagation in Cluster mode
# e.g. when we create a document in one endpoint, it may not be visible in another endpoint for a while
TEST_ENDPOINTS := http://localhost:7001

TESTS := $(REPOPATH)/test
ifeq ("$(TEST_AUTH)", "rootpw")
	CLUSTERENV := JWTSECRET=testing
	TEST_JWTSECRET := testing
	TEST_AUTHENTICATION := basic:root:
endif
ifeq ("$(TEST_AUTH)", "jwt")
	CLUSTERENV := JWTSECRET=testing
	TEST_JWTSECRET := testing
	TEST_AUTHENTICATION := jwt:root:
endif
ifeq ("$(TEST_AUTH)", "jwtsuper")
	CLUSTERENV := JWTSECRET=testing
	TEST_JWTSECRET := testing
	TEST_AUTHENTICATION := super:testing
endif
ifeq ("$(TEST_SSL)", "auto")
	CLUSTERENV := SSL=auto $(CLUSTERENV)
	TEST_ENDPOINTS = https://localhost:7001
endif

ifeq ("$(TEST_CONNECTION)", "vst")
	TESTS := $(REPOPATH)/test
ifndef TEST_CONTENT_TYPE
	TEST_CONTENT_TYPE := vpack
endif
endif

ifeq ("$(TEST_BENCHMARK)", "true")
	TAGS := -bench=. -benchmem -run=notests -cpu=1,2,4
	TESTS := $(REPOPATH)/test
ifndef TESTOPTIONS
	TESTOPTIONS := $(TAGS)
endif
endif

ifdef TEST_ENDPOINTS_OVERRIDE
	TEST_ENDPOINTS := $(TEST_ENDPOINTS_OVERRIDE)
endif

ifdef TEST_NET_OVERRIDE
	TEST_NET := $(TEST_NET_OVERRIDE)
endif

ifdef ENABLE_VST11
	VST11_SINGLE_TESTS := run-tests-single-vst-1.1
	VST11_RESILIENTSINGLE_TESTS := run-tests-resilientsingle-vst-1.1
	VST11_CLUSTER_TESTS := run-tests-cluster-vst-1.1
endif

TEST_RESOURCES_VOLUME :=
ifdef TEST_RESOURCES
	TEST_RESOURCES_VOLUME := -v ${TEST_RESOURCES}:/tmp/resources
endif

ifeq ("$(DEBUG)", "true")
	GOIMAGE := go-driver-tests:debug
	DOCKER_DEBUG_ARGS := --security-opt=seccomp:unconfined -e GOTOOLCHAIN=$(GOTOOLCHAIN)
	DEBUG_PORT := 2345

	DOCKER_RUN_CMD := $(DOCKER_DEBUG_ARGS) $(GOIMAGE) /go/bin/dlv --listen=:$(DEBUG_PORT) --headless=true --api-version=2 --accept-multiclient exec /test_debug.test -- $(TESTOPTIONS)
	DOCKER_V2_RUN_CMD := $(DOCKER_RUN_CMD)
else
    DOCKER_RUN_CMD := $(GOIMAGE) go test -timeout 120m $(GOBUILDTAGSOPT) $(TESTOPTIONS) $(TESTVERBOSEOPTIONS) $(TESTS)
    DOCKER_V2_RUN_CMD := $(GOV2IMAGE) go test -timeout 120m $(GOBUILDTAGSOPT) $(TESTOPTIONS) $(TESTVERBOSEOPTIONS) -parallel $(TESTV2PARALLEL) ./tests
endif

ifeq ("$(ADD_TIMESTAMP)", "true")
	ADD_TIMESTAMP :=| go run ./test/timestamp_output/timestamp_output.go 
endif

.PHONY: all build clean linter run-tests vulncheck

all: build

build: __dir_setup $(SOURCES)
	go build -v $(REPOPATH) $(REPOPATH)/http $(REPOPATH)/vst $(REPOPATH)/agency $(REPOPATH)/jwt

clean: 
	@rm -rf "${TMPDIR}"

.PHONY: changelog
changelog:
	@$(DOCKER_CMD) --rm \
		-e CHANGELOG_GITHUB_TOKEN=$(shell cat ~/.arangodb/github-token) \
		-v "${ROOTDIR}":/usr/local/src/your-app \
		ferrarimarco/github-changelog-generator \
		--user arangodb \
		--project go-driver \
		--no-author \
		--unreleased-label "Master"

run-tests: run-unit-tests run-tests-single run-tests-cluster
ifeq ("$(AF_ENABLED)", "true")
	make run-tests-resilientsingle
endif

# The below rule exists only for backward compatibility.
run-tests-http: run-unit-tests

run-unit-tests: run-v2-unit-tests
	@$(DOCKER_CMD) \
		--rm \
		-v "${ROOTDIR}":/usr/code \
		-e CGO_ENABLED=$(CGO_ENABLED) \
		-e GOTOOLCHAIN=$(GOTOOLCHAIN) \
		-w /usr/code/ \
		$(GOIMAGE) \
		go test $(TESTOPTIONS) $(REPOPATH) $(REPOPATH)/http $(REPOPATH)/agency $(REPOPATH)/vst/protocol

run-v2-unit-tests:
	@$(DOCKER_CMD) \
		--rm \
		-v "${ROOTDIR}"/v2:/usr/code \
		-e CGO_ENABLED=$(CGO_ENABLED) \
		-e GOTOOLCHAIN=$(GOTOOLCHAIN) \
		-w /usr/code/ \
		$(GOIMAGE) \
		go test $(TESTOPTIONS) $(REPOPATH)/v2/connection $(REPOPATH)/v2/arangodb/...

# Single server tests 
run-tests-single: run-tests-single-json run-tests-single-vpack
ifeq ("$(VST_ENABLED)", "true")
	make run-tests-single-vst-1.0 $(VST11_SINGLE_TESTS)
endif

run-tests-single-json: run-tests-single-json-with-auth run-tests-single-json-no-auth run-tests-single-json-jwt-super run-tests-single-json-ssl

run-tests-single-vpack: run-tests-single-vpack-with-auth run-tests-single-vpack-no-auth run-tests-single-vpack-ssl

run-tests-single-vst-1.0: run-tests-single-vst-1.0-with-auth run-tests-single-vst-1.0-no-auth run-tests-single-vst-1.0-ssl

run-tests-single-vst-1.1: run-tests-single-vst-1.1-with-auth run-tests-single-vst-1.1-jwt-auth run-tests-single-vst-1.1-no-auth run-tests-single-vst-1.1-ssl run-tests-single-vst-1.1-jwt-ssl

run-tests-single-json-no-auth:
	@echo "Single server, HTTP+JSON, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-no-auth:
	@echo "Single server, HTTP+Velocypack, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-single-vst-1.0-no-auth:
	@echo "Single server, Velocystream 1.0, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-single-vst-1.1-no-auth:
	@echo "Single server, Velocystream 1.1, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-json-with-auth:
	@echo "Single server, HTTP+JSON, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-json-http2-with-auth:
	@echo "Single server, HTTP+JSON, with authentication"
	@${MAKE} TEST_MODE="single" TAGS="http2" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-with-auth:
	@echo "Single server, HTTP+Velocypack, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-single-vst-1.0-with-auth:
	@echo "Single server, Velocystream 1.0, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-single-vst-1.1-with-auth:
	@echo "Single server, Velocystream 1.1, with authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-vst-1.1-jwt-auth:
	@echo "Single server, Velocystream 1.1, JWT authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="jwt" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-json-jwt-super:
	@echo "Single server, HTTP+JSON, JWT super authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="jwtsuper" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-json-ssl:
	@echo "Single server, HTTP+JSON, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="json" __run_tests

run-tests-single-vpack-ssl:
	@echo "Single server, HTTP+Velocypack, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-single-vst-1.0-ssl:
	@echo "Single server, Velocystream 1.0, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-single-vst-1.1-ssl:
	@echo "Single server, Velocystream 1.1, with authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-single-vst-1.1-jwt-ssl:
	@echo "Single server, Velocystream 1.1, JWT authentication, SSL"
	@${MAKE} TEST_MODE="single" TEST_AUTH="jwt" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

# ResilientSingle server tests 
run-tests-resilientsingle: run-tests-resilientsingle-json run-tests-resilientsingle-vpack
ifeq ("$(VST_ENABLED)", "true")
	make run-tests-resilientsingle-vst-1.0 $(VST11_RESILIENTSINGLE_TESTS)
endif

run-tests-resilientsingle-json: run-tests-resilientsingle-json-with-auth run-tests-resilientsingle-json-no-auth

run-tests-resilientsingle-vpack: run-tests-resilientsingle-vpack-with-auth run-tests-resilientsingle-vpack-no-auth

run-tests-resilientsingle-vst-1.0: run-tests-resilientsingle-vst-1.0-with-auth run-tests-resilientsingle-vst-1.0-no-auth

run-tests-resilientsingle-vst-1.1: run-tests-resilientsingle-vst-1.1-with-auth run-tests-resilientsingle-vst-1.1-jwt-auth run-tests-resilientsingle-vst-1.1-no-auth

run-tests-resilientsingle-json-no-auth:
	@echo "Resilient Single server, HTTP+JSON, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-resilientsingle-vpack-no-auth:
	@echo "Resilient Single server, HTTP+Velocypack, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-resilientsingle-vst-1.0-no-auth:
	@echo "Resilient Single server, Velocystream 1.0, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-resilientsingle-vst-1.1-no-auth:
	@echo "Resilient Single server, Velocystream 1.1, no authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-resilientsingle-json-with-auth:
	@echo "Resilient Single server, HTTP+JSON, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-resilientsingle-vpack-with-auth:
	@echo "Resilient Single server, HTTP+Velocypack, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-resilientsingle-vst-1.0-with-auth:
	@echo "Resilient Single server, Velocystream 1.0, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-resilientsingle-vst-1.1-with-auth:
	@echo "Resilient Single server, Velocystream 1.1, with authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-resilientsingle-vst-1.1-jwt-auth:
	@echo "Resilient Single server, Velocystream 1.1, JWT authentication"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="jwt" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

# Cluster mode tests
run-tests-cluster: run-tests-cluster-json run-tests-cluster-vpack
ifeq ("$(VST_ENABLED)", "true")
	make run-tests-cluster-vst-1.0 $(VST11_CLUSTER_TESTS)
endif


run-tests-cluster-json: run-tests-cluster-json-no-auth run-tests-cluster-json-with-auth run-tests-cluster-json-ssl

run-tests-cluster-vpack: run-tests-cluster-vpack-no-auth run-tests-cluster-vpack-with-auth run-tests-cluster-vpack-ssl

run-tests-cluster-vst-1.0: run-tests-cluster-vst-1.0-no-auth run-tests-cluster-vst-1.0-with-auth run-tests-cluster-vst-1.0-ssl

run-tests-cluster-vst-1.1: run-tests-cluster-vst-1.1-no-auth run-tests-cluster-vst-1.1-with-auth run-tests-cluster-vst-1.1-ssl

run-tests-cluster-json-no-auth: 
	@echo "Cluster server, JSON, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-no-auth:
	@echo "Cluster server, Velocypack, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-vst-1.0-no-auth: 
	@echo "Cluster server, Velocystream 1.0, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-cluster-vst-1.1-no-auth:
	@echo "Cluster server, Velocystream 1.1, no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-cluster-json-with-auth:
	@echo "Cluster server, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-json-jwt-super:
	@echo "Cluster server, HTTP+JSON, JWT super authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="jwtsuper" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-with-auth:
	@echo "Cluster server, Velocypack, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-vst-1.0-with-auth: 
	@echo "Cluster server, Velocystream 1.0, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-cluster-vst-1.1-with-auth: 
	@echo "Cluster server, Velocystream 1.1, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

run-tests-cluster-json-ssl: 
	@echo "Cluster server, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="json" __run_tests

run-tests-cluster-vpack-ssl: 
	@echo "Cluster server, Velocypack, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONTENT_TYPE="vpack" __run_tests

run-tests-cluster-vst-1.0-ssl:
	@echo "Cluster server, Velocystream 1.0, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.0" __run_tests

run-tests-cluster-vst-1.1-ssl: 
	@echo "Cluster server, Velocystream 1.1, SSL, with authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="rootpw" TEST_SSL="auto" TEST_CONNECTION="vst" TEST_CVERSION="1.1" __run_tests

ON_FAILURE_PARAMS = \
	TESTCONTAINER=$(TESTCONTAINER) \
	TEST_MODE=$(TEST_MODE) \
	TEST_SSL=$(TEST_SSL) \
	TEST_AUTH=$(TEST_AUTH) \
	TEST_CONNECTION=$(TEST_CONNECTION) \
	TEST_CONTENT_TYPE=$(TEST_CONTENT_TYPE) \
	TEST_CVERSION=$(TEST_CVERSION) \
	TEST_JWTSECRET=$(TEST_JWTSECRET) \
	DUMP_AGENCY_ON_FAILURE=$(DUMP_AGENCY_ON_FAILURE)


COMMON_DOCKER_CMD_PARAMS = \
	--name=$(TESTCONTAINER) \
	$(TEST_NET) \
	-e TEST_ENDPOINTS=$(TEST_ENDPOINTS) \
	-e TEST_NOT_WAIT_UNTIL_READY=$(TEST_NOT_WAIT_UNTIL_READY) \
	-e TEST_AUTHENTICATION=$(TEST_AUTHENTICATION) \
	-e TEST_JWTSECRET=$(TEST_JWTSECRET) \
	-e TEST_MODE=$(TEST_MODE) \
	-e TEST_MODE_K8S=$(TEST_MODE_K8S) \
	-e TEST_BACKUP_REMOTE_REPO=$(TEST_BACKUP_REMOTE_REPO) \
	-e TEST_BACKUP_REMOTE_CONFIG='$(TEST_BACKUP_REMOTE_CONFIG)' \
	-e TEST_DEBUG='$(TEST_DEBUG)' \
	-e TEST_ENABLE_SHUTDOWN=$(TEST_ENABLE_SHUTDOWN) \
	-e ENABLE_DATABASE_EXTRA_FEATURES=$(ENABLE_DATABASE_EXTRA_FEATURES) \
	-e GODEBUG=tls13=1 \
	-e CGO_ENABLED=$(CGO_ENABLED) \
	-e GOTOOLCHAIN=$(GOTOOLCHAIN)


# Internal test tasks
__run_tests: __test_debug__ __test_prepare __test_go_test __test_cleanup


DOCKER_V1_CMD_PARAMS=\
	$(COMMON_DOCKER_CMD_PARAMS) \
	-e TEST_CONNECTION=$(TEST_CONNECTION) \
	-e TEST_CVERSION=$(TEST_CVERSION) \
	-e TEST_CONTENT_TYPE=$(TEST_CONTENT_TYPE) \
	-e TEST_PPROF=$(TEST_PPROF) \
	-e TEST_REQUEST_LOG=$(TEST_REQUEST_LOG) \
	-e TEST_DISALLOW_UNKNOWN_FIELDS=$(TEST_DISALLOW_UNKNOWN_FIELDS) \
	-v "${ROOTDIR}":/usr/code ${TEST_RESOURCES_VOLUME} \
	-w /usr/code/

__test_go_test:
	($(DOCKER_CMD) $(DOCKER_V1_CMD_PARAMS) $(DOCKER_RUN_CMD) $(ADD_TIMESTAMP)) && echo "success!" \
	|| ( $(ON_FAILURE_PARAMS) MAJOR_VERSION=1 . ./test/on_failure.sh)

			
# Internal test tasks
__run_v2_tests: __test_v2_debug__ __test_prepare __test_v2_go_test __test_cleanup

DOCKER_CMD_V2_PARAMS=\
	$(COMMON_DOCKER_CMD_PARAMS) \
	-v "${ROOTDIR}":/usr/code:ro ${TEST_RESOURCES_VOLUME} \
	-w /usr/code/v2/

__test_v2_go_test:
	($(DOCKER_CMD) $(DOCKER_CMD_V2_PARAMS) $(DOCKER_V2_RUN_CMD) $(ADD_TIMESTAMP)) && echo "success!" \
	|| ($(ON_FAILURE_PARAMS) MAJOR_VERSION=2 . ./test/on_failure.sh)

__test_debug__:
ifeq ("$(DEBUG)", "true")
	@docker build -f Dockerfile.debug --build-arg GOVERSION=$(GOVERSION) --build-arg GOTOOLCHAIN=$(GOTOOLCHAIN) --build-arg "TESTS_DIRECTORY=./test" -t $(GOIMAGE) .
endif

__test_v2_debug__:
ifeq ("$(DEBUG)", "true")
	@docker build -f Dockerfile.debug --build-arg GOVERSION=$(GOVERSION) --build-arg GOTOOLCHAIN=$(GOTOOLCHAIN) --build-arg "TESTS_DIRECTORY=./tests" --build-arg "TESTS_ROOT_PATH=v2" -t $(GOIMAGE) .
endif

__dir_setup:
	@mkdir -p "${TMPDIR}"
	@echo "${TMPDIR}"

__test_prepare: __dir_setup
ifdef TEST_ENDPOINTS_OVERRIDE
	@-docker rm -f -v $(TESTCONTAINER) &> /dev/null
	@sleep 3
else
ifdef JWTSECRET 
	echo "$JWTSECRET" > "${JWTSECRETFILE}"
endif
	@-docker rm -f -v $(TESTCONTAINER) &> /dev/null
	@TESTCONTAINER=$(TESTCONTAINER) ARANGODB=$(ARANGODB) ALPINE_IMAGE=$(ALPINE_IMAGE) ENABLE_BACKUP=$(ENABLE_BACKUP) \
	  ARANGO_LICENSE_KEY=$(ARANGO_LICENSE_KEY) STARTER=$(STARTER) STARTERMODE=$(TEST_MODE) TMPDIR="${TMPDIR}" \
	  ENABLE_DATABASE_EXTRA_FEATURES=$(ENABLE_DATABASE_EXTRA_FEATURES) DEBUG_PORT=$(DEBUG_PORT) $(CLUSTERENV) DOCKER_NETWORK=${TEST_NET} "${ROOTDIR}/test/cluster.sh" start
endif

__test_cleanup:
ifdef TESTCONTAINER
	@TESTCONTAINERS=$$(docker ps -a -q --filter="name=$(TESTCONTAINER)")
	@if [ -n "$$TESTCONTAINERS" ]; then docker rm -f -v $$(docker ps -a -q --filter="name=$(TESTCONTAINER)"); fi
endif
ifndef TEST_ENDPOINTS_OVERRIDE
	@TESTCONTAINER=$(TESTCONTAINER) ARANGODB=$(ARANGODB) ALPINE_IMAGE=$(ALPINE_IMAGE) STARTER=$(STARTER) STARTERMODE=$(TEST_MODE) DOCKER_NETWORK=${TEST_NET} "${ROOTDIR}/test/cluster.sh" cleanup
else
	@-docker rm -f -v $(TESTCONTAINER) &> /dev/null
endif
	@sleep 3

# Benchmarks
run-benchmarks-single-json-no-auth: 
	@echo "Benchmarks: Single server, JSON no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_tests

run-benchmarks-single-vpack-no-auth: 
	@echo "Benchmarks: Single server, Velocypack, no authentication"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" TEST_CONTENT_TYPE="vpack" TEST_BENCHMARK="true" __run_tests

# V1 Cluster benchmarks
run-benchmarks-cluster-json-no-auth: 
	@echo "V1 Benchmarks: Cluster server, JSON no authentication"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_tests

run-benchmarks-remote-with-auth:
	@echo "V1 Benchmarks: Remote server with authentication"
	@echo "Usage: TEST_ENDPOINTS_OVERRIDE=https://host:port TEST_AUTHENTICATION=basic:user:pass make run-benchmarks-remote-with-auth"
ifndef TEST_AUTHENTICATION
	@echo "ERROR: TEST_AUTHENTICATION is not set. Please export it first."
	@echo "Example: export TEST_AUTHENTICATION='basic:root:password'"
	@exit 1
endif
ifndef TEST_ENDPOINTS_OVERRIDE
	@echo "ERROR: TEST_ENDPOINTS_OVERRIDE is not set. Please export it first."
	@echo "Example: export TEST_ENDPOINTS_OVERRIDE='https://host:port'"
	@exit 1
endif
	@${MAKE} TEST_MODE="cluster" TEST_MODE_K8S="k8s" TEST_NOT_WAIT_UNTIL_READY="1" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_tests
	

# V2 Cluster benchmarks
run-benchmarks-v2-cluster-json-no-auth: 
	@echo "V2 Benchmarks: Cluster server, JSON no authentication (from benchmark_tests/)"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_v2_tests

run-benchmarks-v2-cluster-json-ssl: 
	@echo "V2 Benchmarks: Cluster server, JSON with SSL"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" TEST_SSL="auto" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_v2_tests

run-benchmarks-v2-remote-with-auth:
	@echo "V2 Benchmarks: Remote server with authentication"
	@echo "Usage: TEST_ENDPOINTS_OVERRIDE=https://host:port TEST_AUTHENTICATION=basic:user:pass make run-benchmarks-v2-remote-with-auth"
ifndef TEST_AUTHENTICATION
	@echo "ERROR: TEST_AUTHENTICATION is not set. Please export it first."
	@echo "Example: export TEST_AUTHENTICATION='basic:root:password'"
	@exit 1
endif
ifndef TEST_ENDPOINTS_OVERRIDE
	@echo "ERROR: TEST_ENDPOINTS_OVERRIDE is not set. Please export it first."
	@echo "Example: export TEST_ENDPOINTS_OVERRIDE='https://host:port'"
	@exit 1
endif
	@${MAKE} TEST_MODE="cluster" TEST_MODE_K8S="k8s" TEST_NOT_WAIT_UNTIL_READY="1" TEST_CONTENT_TYPE="json" TEST_BENCHMARK="true" __run_v2_tests

## Lint

.PHONY: tools
tools: __dir_setup
	@echo ">> Fetching golangci-lint linter"
	@GOBIN=$(TMPDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.1
	@echo ">> Fetching goimports"
	@GOBIN=$(TMPDIR)/bin go install golang.org/x/tools/cmd/goimports@v0.38.0
	@echo ">> Fetching license check"
	@GOBIN=$(TMPDIR)/bin go install github.com/google/addlicense@v1.0.0
	@echo ">> Fetching govulncheck"
	@GOBIN=$(TMPDIR)/bin go install golang.org/x/vuln/cmd/govulncheck@v1.1.3
	@echo ">> Fetching github-release"
	@GOBIN=$(TMPDIR)/bin go install github.com/github-release/github-release@v0.10.0

.PHONY: license
license:
	@echo ">> Ensuring license of files"
	@$(TMPDIR)/bin/addlicense -f "$(ROOTDIR)/HEADER" $(SOURCES)

.PHONY: license-verify
license-verify:
	@echo ">> Verify license of files"
	@$(TMPDIR)/bin/addlicense -f "$(ROOTDIR)/HEADER" -check $(SOURCES)

.PHONY: fmt
fmt:
	@echo ">> Ensuring style of files"
	@$(TMPDIR)/bin/goimports -w $(SOURCES)

.PHONY: fmt-verify
fmt-verify: license-verify
	@echo ">> Verify files style"
	@if [ X"$$($(TMPDIR)/bin/goimports -l $(SOURCES) | wc -l)" != X"0" ]; then echo ">> Style errors"; $(TMPDIR)/bin/goimports -l $(SOURCES); exit 1; fi

.PHONY: linter
linter: fmt-verify
	@echo ">> Running linter"
	@$(TMPDIR)/bin/golangci-lint run ./...

.PHONY: vulncheck
vulncheck:
	$(TMPDIR)/bin/govulncheck ./...

# V2

v2-%:
	@(cd "$(ROOTDIR)/v2"; make)

run-v2-tests: run-v2-tests-single run-v2-tests-cluster
ifeq ("$(AF_ENABLED)", "true")
	make run-v2-tests-resilientsingle
endif

run-v2-tests-cluster: run-v2-tests-cluster-with-basic-auth run-v2-tests-cluster-without-ssl run-v2-tests-cluster-without-auth run-v2-tests-cluster-with-jwt-auth

run-v2-tests-cluster-with-basic-auth:
	@echo "Cluster server, with basic authentication, v2"
	@${MAKE} TEST_MODE="cluster" TEST_SSL="auto" TEST_AUTH="rootpw" __run_v2_tests

run-v2-tests-cluster-with-jwt-auth:
	@echo "Cluster server, with JWT authentication, v2"
	@${MAKE} TEST_MODE="cluster" TEST_SSL="auto" TEST_AUTH="jwt" __run_v2_tests

run-v2-tests-cluster-without-auth:
	@echo "Cluster server, without authentication, v2"
	@${MAKE} TEST_MODE="cluster" TEST_SSL="auto" TEST_AUTH="none" __run_v2_tests

run-v2-tests-cluster-without-ssl:
	@echo "Cluster server, without authentication and SSL, v2"
	@${MAKE} TEST_MODE="cluster" TEST_AUTH="none" __run_v2_tests

run-v2-tests-single: run-v2-tests-single-without-auth run-v2-tests-single-with-auth

run-v2-tests-single-without-auth:
	@echo "Single server, without authentication, v2"
	@${MAKE} TEST_MODE="single" TEST_AUTH="none" __run_v2_tests

run-v2-tests-single-with-auth:
	@echo "Single server, with authentication, v2"
	@${MAKE} TEST_MODE="single" TEST_SSL="auto" TEST_AUTH="rootpw" __run_v2_tests

run-v2-tests-resilientsingle: run-v2-tests-resilientsingle-with-auth

run-v2-tests-resilientsingle-with-auth:
	@echo "Resilient Single, with authentication, v2"
	@${MAKE} TEST_MODE="resilientsingle" TEST_AUTH="rootpw" TESTV2PARALLEL=1 __run_v2_tests

GH_RELEASE := $(TMPDIR)/bin/github-release
RELEASE := $(SCRIPTDIR)/tools/release
V2_VERSION := ./v2/version/VERSION

release-patch:
	go run $(RELEASE) -type=patch -github-release=$(GH_RELEASE)

release-minor:
	go run $(RELEASE) -type=minor -github-release=$(GH_RELEASE)

release-major:
	go run $(RELEASE) -type=major -github-release=$(GH_RELEASE)

release-v2-patch:
	go run $(RELEASE) -type=patch -github-release=$(GH_RELEASE) -versionfile=$(V2_VERSION)

release-v2-minor:
	go run $(RELEASE) -type=minor -github-release=$(GH_RELEASE) -versionfile=$(V2_VERSION)

release-v2-major:
	go run $(RELEASE) -type=major -github-release=$(GH_RELEASE) -versionfile=$(V2_VERSION)
