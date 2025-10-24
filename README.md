# Demo order service

## requirements

- Go 1.22+
- Docker Compose (for PostgreSQL and NATS Streaming)

## quick start

1. `docker compose up -d`
2. `go run ./cmd/service`

Use the helper publisher to push the sample order:

```
go run ./cmd/publisher -f model.json
```

Then open http://localhost:8080/ and enter the order UID.
