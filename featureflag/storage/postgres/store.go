package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

// PostgresStore implements the Store interface using PostgreSQL as the backend
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store with the given configuration
func NewStore(config *config.PostgresConfig) (*PostgresStore, error) {
	if config == nil {
		return nil, core.NewError("init", "", fmt.Errorf("postgres configuration cannot be nil"))
	}

	if err := config.Validate(); err != nil {
		return nil, core.NewError("init", "", err)
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s",
		config.Host, config.Port, config.Username, config.Database)

	if config.Password != "" {
		connStr += fmt.Sprintf(" password=%s", config.Password)
	}

	if config.SSLMode != "" {
		connStr += fmt.Sprintf(" sslmode=%s", config.SSLMode)
	} else {
		connStr += " sslmode=prefer"
	}

	// Open database connection with connection pooling
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, core.NewError("init", "", fmt.Errorf("failed to open database connection: %w", err))
	}

	// Configure connection pool for high performance
	// Optimized for concurrent access patterns
	db.SetMaxOpenConns(50)                  // Increased for higher concurrency
	db.SetMaxIdleConns(10)                  // More idle connections for faster reuse
	db.SetConnMaxLifetime(10 * time.Minute) // Longer lifetime to reduce connection churn
	db.SetConnMaxIdleTime(2 * time.Minute)  // Reasonable idle time

	store := &PostgresStore{
		db: db,
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := store.HealthCheck(ctx); err != nil {
		db.Close()
		return nil, core.NewError("init", "", fmt.Errorf("failed to connect to PostgreSQL: %w", err))
	}

	// Initialize database schema
	if err := store.initSchema(ctx); err != nil {
		db.Close()
		return nil, core.NewError("init", "", fmt.Errorf("failed to initialize database schema: %w", err))
	}

	return store, nil
}

// initSchema creates the feature_flags table if it doesn't exist
func (p *PostgresStore) initSchema(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS feature_flags (
			key VARCHAR(255) PRIMARY KEY,
			enabled BOOLEAN NOT NULL DEFAULT false,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			metadata JSONB
		);

		CREATE INDEX IF NOT EXISTS idx_feature_flags_enabled ON feature_flags(enabled);
		CREATE INDEX IF NOT EXISTS idx_feature_flags_created_at ON feature_flags(created_at);
		CREATE INDEX IF NOT EXISTS idx_feature_flags_updated_at ON feature_flags(updated_at);
		CREATE INDEX IF NOT EXISTS idx_feature_flags_metadata ON feature_flags USING GIN(metadata);

		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ language 'plpgsql';

		DROP TRIGGER IF EXISTS update_feature_flags_updated_at ON feature_flags;
		CREATE TRIGGER update_feature_flags_updated_at
			BEFORE UPDATE ON feature_flags
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := p.db.ExecContext(ctx, query)
	return err
}

// Get retrieves a feature flag by key
func (p *PostgresStore) Get(ctx context.Context, key string) (*core.FeatureFlag, error) {
	if key == "" {
		return nil, core.NewError("get", key, fmt.Errorf("key cannot be empty"))
	}

	query := `
		SELECT key, enabled, description, created_at, updated_at, metadata
		FROM feature_flags
		WHERE key = $1
	`

	var flag core.FeatureFlag
	var description sql.NullString
	var metadataJSON sql.NullString

	err := p.db.QueryRowContext(ctx, query, key).Scan(
		&flag.Key,
		&flag.Enabled,
		&description,
		&flag.CreatedAt,
		&flag.UpdatedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.NewError("get", key, core.ErrFlagNotFound)
		}
		return nil, core.NewError("get", key, fmt.Errorf("database query failed: %w", err))
	}

	// Handle nullable fields
	if description.Valid {
		flag.Description = description.String
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &flag.Metadata); err != nil {
			return nil, core.NewError("get", key, fmt.Errorf("failed to deserialize metadata: %w", err))
		}
	}

	return &flag, nil
}

// Set creates or updates a feature flag
func (p *PostgresStore) Set(ctx context.Context, flag core.FeatureFlag) error {
	if err := flag.Validate(); err != nil {
		return core.NewError("set", flag.Key, err)
	}

	// Serialize metadata to JSON
	var metadataJSON sql.NullString
	if len(flag.Metadata) > 0 {
		data, err := json.Marshal(flag.Metadata)
		if err != nil {
			return core.NewError("set", flag.Key, fmt.Errorf("failed to serialize metadata: %w", err))
		}
		metadataJSON = sql.NullString{String: string(data), Valid: true}
	}

	// Use UPSERT (INSERT ... ON CONFLICT) to handle both create and update
	query := `
		INSERT INTO feature_flags (key, enabled, description, created_at, updated_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			description = EXCLUDED.description,
			updated_at = NOW(),
			metadata = EXCLUDED.metadata
	`

	// Set timestamps if not already set
	now := time.Now()
	if flag.CreatedAt.IsZero() {
		flag.CreatedAt = now
	}
	flag.UpdatedAt = now

	var description sql.NullString
	if flag.Description != "" {
		description = sql.NullString{String: flag.Description, Valid: true}
	}

	_, err := p.db.ExecContext(ctx, query,
		flag.Key,
		flag.Enabled,
		description,
		flag.CreatedAt,
		flag.UpdatedAt,
		metadataJSON,
	)

	if err != nil {
		return core.NewError("set", flag.Key, fmt.Errorf("database insert/update failed: %w", err))
	}

	return nil
}

// Delete removes a feature flag
func (p *PostgresStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return core.NewError("delete", key, fmt.Errorf("key cannot be empty"))
	}

	query := `DELETE FROM feature_flags WHERE key = $1`

	result, err := p.db.ExecContext(ctx, query, key)
	if err != nil {
		return core.NewError("delete", key, fmt.Errorf("database delete failed: %w", err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return core.NewError("delete", key, fmt.Errorf("failed to get rows affected: %w", err))
	}

	if rowsAffected == 0 {
		return core.NewError("delete", key, core.ErrFlagNotFound)
	}

	return nil
}

// GetAll retrieves all feature flags
func (p *PostgresStore) GetAll(ctx context.Context) ([]core.FeatureFlag, error) {
	query := `
		SELECT key, enabled, description, created_at, updated_at, metadata
		FROM feature_flags
		ORDER BY key
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, core.NewError("getall", "", fmt.Errorf("database query failed: %w", err))
	}
	defer rows.Close()

	var flags []core.FeatureFlag

	for rows.Next() {
		var flag core.FeatureFlag
		var description sql.NullString
		var metadataJSON sql.NullString

		err := rows.Scan(
			&flag.Key,
			&flag.Enabled,
			&description,
			&flag.CreatedAt,
			&flag.UpdatedAt,
			&metadataJSON,
		)

		if err != nil {
			return nil, core.NewError("getall", "", fmt.Errorf("failed to scan row: %w", err))
		}

		// Handle nullable fields
		if description.Valid {
			flag.Description = description.String
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &flag.Metadata); err != nil {
				return nil, core.NewError("getall", "", fmt.Errorf("failed to deserialize metadata for key %s: %w", flag.Key, err))
			}
		}

		flags = append(flags, flag)
	}

	if err := rows.Err(); err != nil {
		return nil, core.NewError("getall", "", fmt.Errorf("row iteration failed: %w", err))
	}

	return flags, nil
}

// HealthCheck verifies store connectivity
func (p *PostgresStore) HealthCheck(ctx context.Context) error {
	if err := p.db.PingContext(ctx); err != nil {
		return core.NewError("healthcheck", "", fmt.Errorf("database ping failed: %w", err))
	}
	return nil
}

// Close cleanly shuts down the store
func (p *PostgresStore) Close() error {
	if err := p.db.Close(); err != nil {
		return core.NewError("close", "", fmt.Errorf("failed to close database connection: %w", err))
	}
	return nil
}
