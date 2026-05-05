FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mams-server ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/mams-server /app/mams-server
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8081

ENTRYPOINT ["/app/mams-server"]
