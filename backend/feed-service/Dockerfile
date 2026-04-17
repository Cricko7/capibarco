FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/feed-service ./cmd/feed-service

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/feed-service /feed-service
EXPOSE 8080 8081
USER nonroot:nonroot
ENTRYPOINT ["/feed-service"]
