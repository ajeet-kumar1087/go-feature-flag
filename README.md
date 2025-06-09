# Feature Flag Service

A lightweight, Redis-backed feature flag service for Go applications.

## Installation

```bash
go get github.com/ajeet-kumar1087/go-feature-flag
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
    cfg := featureflag.Config{
        RedisAddr: "localhost:6379",
        Port:      8080,
    }
    
    if err := featureflag.New(cfg); err != nil {
        log.Fatal(err)
    }
}
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/flags/get` | GET | Get feature flag status |
| `/flags/create` | POST | Create a new feature flag |
| `/flags/enable` | POST | Enable a feature flag |
| `/flags/all` | GET | List all feature flags |
| `/flags/delete` | DELETE | Delete a feature flag |
| `/flags/reset` | POST | Reset all feature flags |

### Example Usage

```bash
# Get a feature flag
curl -X GET "http://localhost:8080/flags/get" \
  -H "Content-Type: application/json" \
  -d '{"key":"my-feature"}'

# Create a feature flag
curl -X POST "http://localhost:8080/flags/create" \
  -H "Content-Type: application/json" \
  -d '{"key":"my-feature","enabled":true}'
```

## Configuration

The service can be configured using the `Config` struct:

```go
type Config struct {
    RedisAddr string // Redis server address
    Port      int    // Server port
}
```

## Requirements

- Go 1.20 or higher
- Redis 6.0 or higher

## Testing
Ensure Redis is running locally before running tests:
```bash
brew services start redis  # Start Redis on MacOS
go test -v ./featureflag  # Run tests
```
## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.