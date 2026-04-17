FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/matching-service ./cmd/matching-service

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=builder /out/matching-service /matching-service
COPY --from=builder /src/configs /configs

EXPOSE 8080 9090
USER nonroot:nonroot
ENTRYPOINT ["/matching-service"]

