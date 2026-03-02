# Extract storage interface

The handler package directly uses a concrete `SQLiteStore` struct. Refactor it to use an interface so we can swap storage backends.

## Acceptance Criteria
- Define a `Store` interface with the methods the handlers use
- Handlers accept the interface, not the concrete type
- SQLiteStore implements the interface
- All existing tests pass
- No new dependencies
- Do not change the API behavior
