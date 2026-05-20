# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o voise-agent main.go

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/voise-agent .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./voise-agent"]
