# Go Feature Flag Examples

This directory contains complete, runnable examples demonstrating different aspects of the Go Feature Flag library.

## Running the Examples

### Prerequisites

- Go 1.19 or higher
- For Redis examples: Redis server running on localhost:6379
- For PostgreSQL examples: PostgreSQL server with feature flags database

### Quick Start

Use the provided script to run examples easily:

```bash
# Run basic usage example
./examples/run-example.sh basic

# Run Redis production example (requires Redis)
docker run -d -p 6379:6379 redis:alpine
./examples/run-example.sh redis

# Run web service example
./examples/run-example.sh web
# Then visit: http://localhost:8080
```

Or run them manually:

1. **Basic Usage** - Simple in-memory example:
   ```bash
   cd examples/basic-usage
   go run main.go
   ```

2. **Production Redis** - Redis with caching and metrics:
   ```bash
   # Start Redis first
   docker run -d -p 6379:6379 redis:alpine
   
   # Run example
   cd examples/redis-production
   go run main.go
   ```

3. **Web Service** - HTTP service with feature flags:
   ```bash
   cd examples/web-service
   go run main.go
   
   # In another terminal, test the endpoints:
   curl http://localhost:8080
   curl http://localhost:8080/api/features
   curl http://localhost:8080/health
   curl http://localhost:8080/metrics
   ```

## Example Files

| Directory | Description | Use Case |
|-----------|-------------|----------|
| `basic-usage/` | Simple feature flag operations | Learning the basics |
| `redis-production/` | Production setup with Redis | Production deployment |
| `web-service/` | HTTP service integration | Web applications |
| `config.yaml` | Configuration file example | File-based configuration |

## Configuration Examples

### Environment Variables
```bash
export FEATUREFLAG_STORAGE_TYPE=redis
export FEATUREFLAG_REDIS_ADDR=localhost:6379
export FEATUREFLAG_CACHE_ENABLED=true
export FEATUREFLAG_CACHE_TTL=10m
export FEATUREFLAG_LOGGING_ENABLED=true
export FEATUREFLAG_LOGGING_LEVEL=info

go run basic_usage.go
```

### Docker Compose Setup
```yaml
version: '3.8'
services:
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
  
  app:
    build: .
    environment:
      - FEATUREFLAG_STORAGE_TYPE=redis
      - FEATUREFLAG_REDIS_ADDR=redis:6379
    depends_on:
      - redis
```

## Performance Testing

Run performance tests to see the impact of caching:

```bash
# Without cache
go run -ldflags="-X main.cacheEnabled=false" redis_production.go

# With cache (default)
go run redis_production.go
```

## Troubleshooting

### Redis Connection Issues
```bash
# Check if Redis is running
redis-cli ping

# Check Redis logs
docker logs <redis-container-id>
```

### PostgreSQL Setup
```sql
-- Create database
CREATE DATABASE featureflags;

-- Create table (run the schema from db/postgres_schema.sql)
\i db/postgres_schema.sql
```

## Next Steps

After running these examples:

1. **Integrate into your application** - Use the patterns shown in `web_service.go`
2. **Configure for production** - Use Redis or PostgreSQL storage
3. **Enable observability** - Set up logging and metrics collection
4. **Monitor performance** - Use the metrics endpoint to track cache hit rates

For more advanced usage, see the main README.md and the architecture documentation in `featureflag/README.md`.