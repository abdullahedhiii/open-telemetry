FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o backend

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/backend .

RUN mkdir -p /fluentd/log
VOLUME ["/fluentd/log"]
EXPOSE 8000
CMD ["/app/backend"]