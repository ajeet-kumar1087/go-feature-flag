package featureflag

import (
	"context"
	"database/sql"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{
		db: db,
	}
}

// Get flag from persistent store
func (ps *PostgresStore) Get(ctx context.Context, key string) (*FeatureFlag, error) {
	var flag FeatureFlag
	query := "SELECT * FROM feature_flags WHERE key = $1"
	row := ps.db.QueryRow(query, key)
	err := row.Scan(&flag.Key, &flag.Enabled, &flag.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No flag found for the given key
		}
		return nil, err // Other error
	}
	return &flag, nil
}

// Set flag to persistent store
func (p *PostgresStore) Set(ctx context.Context, flag FeatureFlag) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO feature_flags (key, enabled) 
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET enabled = EXCLUDED.enabled, description = EXCLUDED.enabled
	`, flag.Key, flag.Enabled, flag.Description)
	return err
}

// Get all flags from persistent store
func (ps *PostgresStore) GetAll() ([]FeatureFlag, error) {
	query := "SELECT * FROM feature_flags"
	rows, err := ps.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []FeatureFlag
	for rows.Next() {
		var flag FeatureFlag
		if err := rows.Scan(&flag.Key, &flag.Enabled, &flag.Description); err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return flags, nil
}

// Delete flag from persistent store
func (ps *PostgresStore) Delete(ctx context.Context, key string) error {
	_, err := ps.db.ExecContext(ctx, "DELETE FROM feature_flags WHERE key = $1", key)
	return err
}
