FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/authsvc ./cmd/authsvc

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
USER 65532:65532
COPY --from=build /out/authsvc /authsvc
EXPOSE 50051 8080
ENTRYPOINT ["/authsvc"]
