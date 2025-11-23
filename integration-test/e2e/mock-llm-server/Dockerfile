FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY main.go .

# Build the mock server
RUN go build -o mock-llm-server main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/mock-llm-server .

# Expose the default port
EXPOSE 8080

CMD ["./mock-llm-server"]
