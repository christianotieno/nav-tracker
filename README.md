# Navigation Tracker

A Go web service for tracking visitor navigation events and providing analytics.

## Features

- **Event Ingestion**: Record visitor navigation events via REST API
- **Analytics**: Get distinct visitor counts and detailed statistics  
- **Performance**: High-throughput, concurrent request handling
- **Memory Management**: Automatic cleanup and resource optimization
- **Monitoring**: Health checks, metrics, and system monitoring

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

```bash
# Clone and setup
git clone git@github.com:christianotieno/nav-tracker.git
cd nav-tracker
go mod tidy

# Run the service
go run .
```

### Using Makefile

```bash
# Show available commands
make help

# Run the service
make run

# Build the application  
make build

# Run tests
make test

# Build and run with Docker
make docker-run
```

## API Endpoints

### Core Endpoints

#### Record Navigation Event

```bash
POST /api/v1/ingest
Content-Type: application/json

{
  "visitor_id": "user123",
  "url": "https://example.com/home"
}
```

#### Get Visitor Statistics

```bash
GET /api/v1/stats?url=https://example.com/home

# Response
{
  "success": true,
  "data": {
    "url": "https://example.com/home",
    "distinct_visitors": 42,
    "total_page_views": 156,
    "last_updated": "2024-01-01T12:00:00Z"
  }
}
```

### Additional Endpoints

- `GET /api/v1/top-urls` - Get top URLs by visitor count
- `GET /api/v1/top-visitors?url=<url>` - Get top visitors for URL
- `GET /api/v1/system-stats` - Get system metrics
- `GET /api/v1/health` - Health check
- `GET /docs` - API documentation

### Legacy Endpoints (Backward Compatibility)

- `POST /ingest` → `POST /api/v1/ingest`
- `GET /stats?url=<url>` → `GET /api/v1/stats?url=<url>`
- `GET /health` → `GET /api/v1/health`

## Configuration

The service can be configured through environment variables:

| Option | Default | Description |
|--------|---------|-------------|
| `Port` | `8080` | Server port |
| `MaxMemoryUsage` | `100MB` | Memory limit before cleanup |
| `CleanupInterval` | `5m` | Cleanup frequency |
| `MaxURLs` | `10000` | Maximum URLs to track |
| `EnableMetrics` | `true` | Enable performance metrics |

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run tests manually
go test ./...
```

## Performance

- **Event Recording**: >10,000 events/second
- **Statistics Retrieval**: >50,000 reads/second
- **Memory Efficient**: Automatic cleanup and optimization
- **Thread-Safe**: Concurrent operations with minimal contention

## Docker

```bash
# Build and run with Docker
make docker-run

# Or manually
docker build -t nav-tracker .
docker run -p 8080:8080 nav-tracker
```

## Development

```bash
# Set up development environment
make deps

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```
