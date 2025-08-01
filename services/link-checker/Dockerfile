FROM golang:1.24.5-alpine AS builder


RUN apk add --no-cache git


WORKDIR /app


COPY go.mod go.sum ./


RUN go mod download


COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o link-checker ./services/link-checker


FROM alpine:latest


RUN apk --no-cache add ca-certificates


RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup


WORKDIR /app


COPY --from=builder /app/link-checker .


RUN chown -R appuser:appgroup /app


USER appuser


EXPOSE 8082


HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1


CMD ["./link-checker"]