FROM golang:1.24.5-alpine AS builder

RUN apk add --no-cache git


WORKDIR /app


COPY go.mod go.sum ./


RUN go mod download


COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o analyzer ./services/analyzer


FROM golang:1.24.5-alpine AS dev


RUN go install github.com/cosmtrek/air@latest

WORKDIR /app


FROM alpine:latest


RUN apk --no-cache add ca-certificates curl


RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup


WORKDIR /app


COPY --from=builder /app/analyzer .


RUN chown -R appuser:appgroup /app


USER appuser

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Run the application
CMD ["./analyzer"]