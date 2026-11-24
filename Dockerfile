# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/monitor .

# Production Stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/monitor /app/monitor

EXPOSE 8080
USER nobody:nobody

ENTRYPOINT ["/app/monitor"]
