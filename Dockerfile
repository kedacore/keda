FROM golang:1.13-alpine3.10 as build-env

RUN apk update && \
    apk add --no-cache make ca-certificates git mercurial && \
    update-ca-certificates

WORKDIR $GOPATH/src/github.com/kedacore/keda
COPY . .

RUN make build && mv dist/keda /keda

RUN mkdir -p /tmp/empty

FROM scratch

COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /keda /keda
# make sure /tmp exists
COPY --from=build-env /tmp/empty /tmp

ENTRYPOINT [ "/keda" ]
