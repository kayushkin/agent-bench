# Add health endpoint

Add a GET /health endpoint to the HTTP server that returns JSON:

```json
{"status": "ok", "version": "1.0.0"}
```

## Acceptance Criteria
- Returns HTTP 200 with Content-Type application/json
- Response body matches the format above
- Add a test for the endpoint
- Do not modify any existing endpoints or tests
