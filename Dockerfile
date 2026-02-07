FROM golang:latest AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ .
RUN go build -o bucketd .

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/bucketd .
COPY config.prod.yaml .

EXPOSE 8080

CMD ["./bucketd"]