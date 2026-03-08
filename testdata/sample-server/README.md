# Sample Server

A simple Go HTTP API server for agent benchmark testing.

## Endpoints

- `GET /items` - List all items (supports ?limit=N&offset=N pagination)
- `POST /items` - Create a new item
- `GET /items/{id}` - Get a specific item by ID

## Running

```bash
go run main.go
```

Server runs on `:8080`

## Testing

```bash
go test ./...
```
