FROM golang:1.25.3-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/analytics-service ./cmd/analytics-service

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/analytics-service /app/analytics-service
COPY configs /app/configs
EXPOSE 18088 19088
USER nonroot:nonroot
ENTRYPOINT ["/app/analytics-service","--config","/app/configs/config.yaml"]
