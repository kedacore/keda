
FROM golang:1.25.3

WORKDIR /go/src/github.com/samber/lo

COPY Makefile go.* ./

RUN make tools
