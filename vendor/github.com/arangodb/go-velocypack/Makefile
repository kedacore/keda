PROJECT := go-velocypack
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)

TESTOPTIONS := 
ifdef VERBOSE
	TESTOPTIONS := -v
endif

ORGPATH := github.com/arangodb
REPONAME := $(PROJECT)
REPOPATH := $(ORGPATH)/$(REPONAME)

SOURCES := $(shell find . -name '*.go')

.PHONY: all build clean run-tests show-coverage

all: build

build: $(SOURCES)
	go build -v github.com/arangodb/go-velocypack 

# All unit tests
run-tests:
	@go get github.com/stretchr/testify/assert
	@go test -v $(REPOPATH)/test/runtime 
	@go test $(TESTOPTIONS) $(REPOPATH) 
	@go test -cover -coverpkg $(REPOPATH) -coverprofile=coverage.out $(TESTOPTIONS) $(REPOPATH)/test

# All benchmarks
run-benchmarks:
	@go get github.com/stretchr/testify/assert
	@go test $(TESTOPTIONS) -bench=. -run=notests -cpu=1,2,4 $(REPOPATH)/test

# All benchmarks using local profiling
run-benchmarks-prof: $(GOBUILDDIR)
	@go get github.com/stretchr/testify/assert
	@go test $(TESTOPTIONS) -bench=. -run=notests -cpu=1,2,4 -cpuprofile=cpu.out $(REPOPATH)/test
	@echo Now profile using: go tool pprof test.test cpu.out

# All unit tests using local Go tools
run-tests-local: $(GOBUILDDIR)
	@go get github.com/stretchr/testify/assert
	@go test -v $(REPOPATH)/test/runtime
	@go test $(TESTOPTIONS) $(REPOPATH) 
	@go test -cover -coverpkg $(REPOPATH) -coverprofile=coverage.out $(TESTOPTIONS) $(REPOPATH)/test

# All (except large memory) unit tests using local Go tools
run-tests-local-nolarge: $(GOBUILDDIR)
	@go get github.com/stretchr/testify/assert
	@go test -tags nolarge -v $(REPOPATH)/test/runtime
	@go test -tags nolarge $(TESTOPTIONS) $(REPOPATH) 
	@go test -tags nolarge -cover -coverpkg $(REPOPATH) -coverprofile=coverage.out $(TESTOPTIONS) $(REPOPATH)/test

show-coverage: run-tests
	go tool cover -html coverage.out 
