# PostgreSQL Store

The PostgreSQL store provides persistent storage for feature flags using PostgreSQL as the backend database. It offers excellent performance, ACID compliance, and supports complex queries through the JSONB metadata field.

## Features

- **ACID Compliance**: Full transaction support with PostgreSQL
- **Connection Pooling**: Efficient connection management with configurable pool settings
- **Prepared Statements**: Optimized query performance and SQL injection protection
- **JSONB Metadata**: Rich metadata support with indexing capabilities
- **Automatic Schema Management**: Creates tables and indexes automatically
- **Graceful Error Handling**: Comprehensive error handling with detailed messages
- **Health Checks**: Built-in connectivity monitoring

## Configuration

### Basic Configuration

```go
config := &featureflag.PostgresConfig{
    Host:     "localhost",
    Port:     5432,
    Database: "featureflags",
    Username: "postgres",
    Password: "password",
    SSLMode:  "prefer", // or "disable", "require", "verify-ca", "verify-full"
}

store, err := featureflag.NewPostgresStore(config)
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

### Environment Variables

You can also configure the PostgreSQL store using environment variables:

```bash
export FEATUREFLAG_STORAGE_TYPE=postgres
export FEATUREFLAG_POSTGRES_HOST=localhost
export FEATUREFLAG_POSTGRES_PORT=5432
export FEATUREFLAG_POSTGRES_DATABASE=featureflags
export FEATUREFLAG_POSTGRES_USERNAME=postgres
export FEATUREFLAG_POSTGRES_PASSWORD=password
export FEATUREFLAG_POSTGRES_SSLMODE=prefer
```

### Configuration File

```yaml
storage:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    database: featureflags
    username: postgres
    password: password
    ssl_mode: prefer
```

## Database Schema

The PostgreSQL store automatically creates the following schema:

```sql
CREATE TABLE feature_flags (
    key VARCHAR(255) PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB
);

-- Indexes for performance
CREATE INDEX idx_feature_flags_enabled ON feature_flags(enabled);
CREATE INDEX idx_feature_flags_created_at ON feature_flags(created_at);
CREATE INDEX idx_feature_flags_updated_at ON feature_flags(updated_at);
CREATE INDEX idx_feature_flags_metadata ON feature_flags USING GIN(metadata);

-- Automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_feature_flags_updated_at
    BEFORE UPDATE ON feature_flags
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

## Usage Examples

### Basic CRUD Operations

```go
ctx := context.Background()

// Create a feature flag
flag := featureflag.FeatureFlag{
    Key:         "new-feature",
    Enabled:     true,
    Description: "Enable new feature for all users",
    Metadata: map[string]string{
        "team":        "backend",
        "environment": "production",
        "rollout":     "100%",
    },
}

err := store.Set(ctx, flag)
if err != nil {
    log.Fatal(err)
}

// Retrieve a feature flag
retrievedFlag, err := store.Get(ctx, "new-feature")
if err != nil {
    log.Fatal(err)
}

// Update a feature flag
retrievedFlag.Enabled = false
retrievedFlag.Metadata["rollout"] = "0%"
err = store.Set(ctx, *retrievedFlag)
if err != nil {
    log.Fatal(err)
}

// Delete a feature flag
err = store.Delete(ctx, "new-feature")
if err != nil {
    log.Fatal(err)
}

// Get all feature flags
allFlags, err := store.GetAll(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Health Monitoring

```go
// Check database connectivity
err := store.HealthCheck(ctx)
if err != nil {
    log.Printf("Database health check failed: %v", err)
    // Handle unhealthy database
}
```

### Advanced Metadata Queries

Since metadata is stored as JSONB, you can perform complex queries directly on the database:

```sql
-- Find all flags for a specific team
SELECT * FROM feature_flags WHERE metadata->>'team' = 'backend';

-- Find all enabled flags in production
SELECT * FROM feature_flags 
WHERE enabled = true AND metadata->>'environment' = 'production';

-- Find flags with rollout percentage greater than 50%
SELECT * FROM feature_flags 
WHERE (metadata->>'rollout')::text LIKE '%0%' 
   OR (metadata->>'rollout')::text LIKE '%5%'
   OR (metadata->>'rollout')::text LIKE '%6%'
   OR (metadata->>'rollout')::text LIKE '%7%'
   OR (metadata->>'rollout')::text LIKE '%8%'
   OR (metadata->>'rollout')::text LIKE '%9%'
   OR metadata->>'rollout' = '100%';
```

## Performance Considerations

### Connection Pooling

The PostgreSQL store is configured with optimized connection pool settings:

- **Max Open Connections**: 25
- **Max Idle Connections**: 5
- **Connection Max Lifetime**: 5 minutes
- **Connection Max Idle Time**: 1 minute

### Indexing

The store creates several indexes for optimal performance:

- Primary key index on `key` (automatic)
- Index on `enabled` for filtering enabled/disabled flags
- Indexes on `created_at` and `updated_at` for time-based queries
- GIN index on `metadata` for JSONB queries

### Query Optimization

- Uses prepared statements for all operations
- UPSERT operations with `ON CONFLICT` for efficient updates
- Batch operations where possible
- Proper NULL handling for optional fields

## Error Handling

The PostgreSQL store provides detailed error information:

```go
_, err := store.Get(ctx, "nonexistent-flag")
if err != nil {
    var ffErr *featureflag.FeatureFlagError
    if errors.As(err, &ffErr) {
        if errors.Is(ffErr.Err, featureflag.ErrFlagNotFound) {
            // Handle flag not found
        } else {
            // Handle other database errors
            log.Printf("Database error: %v", ffErr.Err)
        }
    }
}
```

## Security Considerations

### SSL/TLS Configuration

Always use SSL in production:

```go
config := &featureflag.PostgresConfig{
    Host:     "your-postgres-host",
    Port:     5432,
    Database: "featureflags",
    Username: "postgres",
    Password: "secure-password",
    SSLMode:  "require", // or "verify-ca", "verify-full"
}
```

### Connection Security

- Use strong passwords
- Limit database user permissions to only what's needed
- Use connection pooling to prevent connection exhaustion
- Monitor connection usage and performance

### Data Protection

- Enable PostgreSQL logging for audit trails
- Use database-level encryption for sensitive data
- Implement proper backup and recovery procedures
- Consider using read replicas for high-availability setups

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```
   failed to connect to PostgreSQL: database ping failed: dial tcp: connect: connection refused
   ```
   - Check if PostgreSQL is running
   - Verify host and port configuration
   - Check firewall settings

2. **Authentication Failed**
   ```
   failed to connect to PostgreSQL: database ping failed: pq: password authentication failed
   ```
   - Verify username and password
   - Check PostgreSQL `pg_hba.conf` configuration
   - Ensure user has necessary permissions

3. **Database Does Not Exist**
   ```
   failed to connect to PostgreSQL: database ping failed: pq: database "featureflags" does not exist
   ```
   - Create the database first: `CREATE DATABASE featureflags;`
   - Verify database name in configuration

4. **SSL Issues**
   ```
   failed to connect to PostgreSQL: database ping failed: pq: SSL is not enabled on the server
   ```
   - Set `SSLMode: "disable"` for local development
   - Enable SSL on PostgreSQL server for production

### Performance Issues

1. **Slow Queries**
   - Check if indexes are being used: `EXPLAIN ANALYZE SELECT ...`
   - Monitor connection pool usage
   - Consider increasing connection pool size

2. **High Memory Usage**
   - Reduce connection pool size
   - Check for connection leaks
   - Monitor PostgreSQL memory usage

### Monitoring

Enable PostgreSQL logging to monitor the store:

```sql
-- Enable query logging
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET log_min_duration_statement = 100; -- Log queries > 100ms

-- Reload configuration
SELECT pg_reload_conf();
```

## Migration from Other Stores

### From Memory Store

```go
// Export from memory store
memoryStore := featureflag.NewStore()
flags, err := memoryStore.GetAll(ctx)
if err != nil {
    log.Fatal(err)
}

// Import to PostgreSQL store
postgresStore, err := featureflag.NewPostgresStore(config)
if err != nil {
    log.Fatal(err)
}

for _, flag := range flags {
    err := postgresStore.Set(ctx, flag)
    if err != nil {
        log.Printf("Failed to migrate flag %s: %v", flag.Key, err)
    }
}
```

### From Redis Store

Similar process as memory store - export all flags and import them into PostgreSQL.

## Best Practices

1. **Use Connection Pooling**: Always configure appropriate connection pool settings
2. **Monitor Performance**: Set up monitoring for query performance and connection usage
3. **Handle Errors Gracefully**: Implement proper error handling and fallback mechanisms
4. **Use Transactions**: For complex operations, consider using database transactions
5. **Regular Backups**: Implement regular backup procedures for your feature flag data
6. **Index Optimization**: Monitor query patterns and add custom indexes if needed
7. **Security**: Always use SSL in production and follow PostgreSQL security best practices