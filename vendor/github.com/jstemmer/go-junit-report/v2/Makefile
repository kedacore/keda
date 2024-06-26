VERSION=$(shell git describe --match="v*")
REVISION=$(shell git rev-parse HEAD)
TIMESTAMP=$(shell date +%FT%T)

test:
	go test ./...

build/go-junit-report build/go-junit-report.exe: clean
	go build --ldflags "-s -X main.Version=$(VERSION) -X main.Revision=$(REVISION) -X main.BuildTime=$(TIMESTAMP)" -o $@

build/go-junit-report-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz: build/go-junit-report
	tar czf $@ -C build go-junit-report

build/go-junit-report-$(VERSION)-windows-amd64.zip: build/go-junit-report.exe
	zip -j $@ build/go-junit-report.exe

release: test
	$(MAKE) GOOS=linux GOARCH=amd64 build/go-junit-report-$(VERSION)-linux-amd64.tar.gz
	$(MAKE) GOOS=windows GOARCH=amd64 build/go-junit-report-$(VERSION)-windows-amd64.zip
	$(MAKE) GOOS=darwin GOARCH=amd64 build/go-junit-report-$(VERSION)-darwin-amd64.tar.gz
	$(MAKE) GOOS=darwin GOARCH=arm64 build/go-junit-report-$(VERSION)-darwin-arm64.tar.gz

clean:
	rm -f build/go-junit-report
	rm -f build/go-junit-report.exe

.PHONY: build clean release test
