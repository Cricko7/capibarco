FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/notification-service ./cmd/notification-service

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/notification-service /app/notification-service
COPY --from=builder /src/configs/config.yaml /app/config.yaml
EXPOSE 8080 9090
ENTRYPOINT ["/app/notification-service", "-config", "/app/config.yaml"]
