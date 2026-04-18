FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./cmd/api-gateway

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api-gateway /app/api-gateway
COPY api/openapi.yaml /app/api/openapi.yaml
COPY configs/config.yaml /app/configs/config.yaml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api-gateway"]
