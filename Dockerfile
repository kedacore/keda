FROM golang:1.12-alpine3.9 as build-env

RUN apk update && \
    apk add --no-cache make ca-certificates && \
    update-ca-certificates

WORKDIR $GOPATH/src/github.com/Azure/Kore
COPY . .

RUN make build && mv dist/kore /kore

FROM scratch

COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /kore /kore

ENTRYPOINT [ "/kore" ]
