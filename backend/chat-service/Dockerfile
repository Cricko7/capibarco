FROM golang:1.25-alpine AS builder

WORKDIR /src
ENV PATH="/go/bin:${PATH}"
RUN apk add --no-cache ca-certificates git
RUN go install github.com/bufbuild/buf/cmd/buf@v1.53.0 \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.8 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN buf generate
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/chat-service ./cmd/chat-service

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/chat-service /app/chat-service
COPY --from=builder /src/configs /app/configs
COPY --from=builder /src/api /app/api

EXPOSE 8080 9090
USER nonroot:nonroot
ENTRYPOINT ["/app/chat-service"]
