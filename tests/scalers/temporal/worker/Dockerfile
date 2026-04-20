FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -o worker .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/worker /worker
ENTRYPOINT ["/worker"]
