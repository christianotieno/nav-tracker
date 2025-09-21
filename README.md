# Navigation Tracker

A simple Go web service for tracking visitor navigation events and providing distinct visitor statistics.

## API Endpoints

- **POST** `/ingest` - Record navigation events
- **GET** `/stats?url=<url>` - Get distinct visitor count
- **GET** `/health` - Health check

## Usage

```bash
# Install dependencies
go mod tidy

# Run the service
go run .

# Run tests
go test
```

The service runs on port 8080.

## Example

```bash
# Record an event
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"visitor_id": "user123", "url": "https://example.com/home"}'

# Get statistics
curl "http://localhost:8080/stats?url=https://example.com/home"
```

## Features

- Thread-safe in-memory storage
- Concurrent request handling
- No external dependencies (except gorilla/mux)
