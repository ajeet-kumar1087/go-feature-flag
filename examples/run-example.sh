#!/bin/bash

# Script to run feature flag examples

set -e

case "$1" in
    "basic"|"basic-usage")
        echo "🚀 Running basic usage example..."
        cd basic-usage && go run main.go
        ;;
    "redis"|"redis-production")
        echo "🚀 Running Redis production example..."
        echo "📋 Make sure Redis is running: docker run -d -p 6379:6379 redis:alpine"
        cd redis-production && go run main.go
        ;;
    "web"|"web-service")
        echo "🚀 Running web service example..."
        echo "📋 Visit http://localhost:8080 after it starts"
        cd web-service && go run main.go
        ;;
    *)
        echo "Usage: $0 {basic|redis|web}"
        echo ""
        echo "Examples:"
        echo "  $0 basic     - Run basic usage example"
        echo "  $0 redis     - Run Redis production example"
        echo "  $0 web       - Run web service example"
        echo ""
        echo "Available examples:"
        echo "  basic-usage/     - Simple feature flag operations"
        echo "  redis-production/ - Production setup with Redis"
        echo "  web-service/     - HTTP service integration"
        exit 1
        ;;
esac