# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/billing-service ./cmd/billing-service

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/billing-service /app/billing-service
COPY configs/config.yaml /app/configs/config.yaml
COPY api/openapi.yaml /app/api/openapi.yaml
USER nonroot:nonroot
EXPOSE 8080 9090
ENTRYPOINT ["/app/billing-service", "-config", "/app/configs/config.yaml"]
