# syntax=docker/dockerfile:1.7

FROM golang:1.25 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/animal-service ./cmd/animal-service

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/animal-service /app/animal-service
COPY configs/config.yaml /app/configs/config.yaml
COPY api/openapi.yaml /app/api/openapi.yaml
EXPOSE 8080 9090
USER nonroot:nonroot
ENTRYPOINT ["/app/animal-service"]
CMD ["-config", "/app/configs/config.yaml"]
