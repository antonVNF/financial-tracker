FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/finance-tracker ./cmd/api

FROM debian:bookworm-slim
WORKDIR /root/
COPY --from=builder /app/finance-tracker .
COPY --from=builder /app/.env .env
EXPOSE 8080
CMD ["./finance-tracker"]