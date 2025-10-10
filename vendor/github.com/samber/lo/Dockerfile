
FROM golang:1.25.2

WORKDIR /go/src/github.com/samber/lo

COPY Makefile go.* ./

RUN make tools
