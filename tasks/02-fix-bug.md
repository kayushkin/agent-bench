# Fix pagination bug

The list endpoint returns all items instead of respecting the `limit` and `offset` query parameters.

There is a failing test: `TestListPagination` — make it pass.

## Acceptance Criteria
- `GET /items?limit=5&offset=10` returns at most 5 items starting from position 10
- The existing failing test passes
- All other tests still pass
- Only modify the minimum code needed to fix the bug
