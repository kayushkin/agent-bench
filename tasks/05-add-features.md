# Task: Add Multiple Features to the Sample Server

You are working with a Go HTTP API server for managing items. Your job is to add several features and ensure all existing and new tests pass.

## Requirements

### 1. DELETE endpoint
Add `DELETE /items/{id}` that:
- Removes the item with the given ID
- Returns 204 No Content on success
- Returns 404 if the item doesn't exist
- Is thread-safe (uses the existing mutex)

### 2. PUT endpoint  
Add `PUT /items/{id}` that:
- Updates the name of an existing item
- Returns the updated item as JSON with 200 OK
- Returns 404 if the item doesn't exist
- Returns 400 if the request body is invalid

### 3. Search/filter
Update `GET /items` to support a `search` query parameter:
- `GET /items?search=al` should return only items whose name contains "al" (case-insensitive)
- Search should work together with existing pagination (limit/offset applied after filtering)

### 4. Item count header
Add an `X-Total-Count` header to `GET /items` responses containing the total number of items (before pagination but after search filtering).

### 5. Tests
Add tests for ALL new functionality in `main_test.go`:
- `TestDeleteItem` — delete existing item, verify 204, verify it's gone
- `TestDeleteItemNotFound` — delete non-existent item, verify 404
- `TestUpdateItem` — update existing item, verify response
- `TestUpdateItemNotFound` — update non-existent item, verify 404
- `TestSearchItems` — search with partial match, verify filtering
- `TestSearchWithPagination` — search + limit/offset combined
- `TestTotalCountHeader` — verify X-Total-Count header value

## Constraints
- Only modify `main.go` and `main_test.go`
- All existing tests must continue to pass
- All new tests must pass
- Code must compile with `go build ./...`

## Success Criteria
- `go build ./...` succeeds
- `go test ./...` passes (all existing + new tests)
