package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

func TestNewPostgresStore(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.PostgresConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "postgres configuration cannot be nil",
		},
		{
			name: "invalid config - empty host",
			config: &config.PostgresConfig{
				Host:     "",
				Port:     5432,
				Database: "test",
				Username: "user",
			},
			expectError: true,
			errorMsg:    "PostgreSQL host cannot be empty",
		},
		{
			name: "invalid config - invalid port",
			config: &config.PostgresConfig{
				Host:     "localhost",
				Port:     0,
				Database: "test",
				Username: "user",
			},
			expectError: true,
			errorMsg:    "PostgreSQL port must be between 1 and 65535",
		},
		{
			name: "invalid config - empty database",
			config: &config.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "",
				Username: "user",
			},
			expectError: true,
			errorMsg:    "PostgreSQL database name cannot be empty",
		},
		{
			name: "invalid config - empty username",
			config: &config.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "test",
				Username: "",
			},
			expectError: true,
			errorMsg:    "PostgreSQL username cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if store != nil {
					t.Errorf("expected nil store but got %v", store)
				}
				// Check if error message contains expected text
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if store == nil {
					t.Errorf("expected store but got nil")
				}
			}

			// Clean up if store was created
			if store != nil {
				store.Close()
			}
		})
	}
}

func TestPostgresStore_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		key         string
		setupMock   func()
		expectError bool
		errorType   error
		expected    *core.FeatureFlag
	}{
		{
			name:        "empty key",
			key:         "",
			setupMock:   func() {},
			expectError: true,
			errorType:   nil, // Will check error message instead
		},
		{
			name: "flag not found",
			key:  "nonexistent",
			setupMock: func() {
				mock.ExpectQuery(`SELECT key, enabled, description, created_at, updated_at, metadata FROM feature_flags WHERE key = \$1`).
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
			errorType:   core.ErrFlagNotFound,
		},
		{
			name: "database error",
			key:  "test-flag",
			setupMock: func() {
				mock.ExpectQuery(`SELECT key, enabled, description, created_at, updated_at, metadata FROM feature_flags WHERE key = \$1`).
					WithArgs("test-flag").
					WillReturnError(errors.New("database connection failed"))
			},
			expectError: true,
		},
		{
			name: "successful get with minimal data",
			key:  "test-flag",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"key", "enabled", "description", "created_at", "updated_at", "metadata"}).
					AddRow("test-flag", true, nil, time.Now(), time.Now(), nil)
				mock.ExpectQuery(`SELECT key, enabled, description, created_at, updated_at, metadata FROM feature_flags WHERE key = \$1`).
					WithArgs("test-flag").
					WillReturnRows(rows)
			},
			expectError: false,
			expected: &core.FeatureFlag{
				Key:     "test-flag",
				Enabled: true,
			},
		},
		{
			name: "successful get with full data",
			key:  "test-flag",
			setupMock: func() {
				metadata := map[string]string{"env": "test", "team": "backend"}
				metadataJSON, _ := json.Marshal(metadata)
				now := time.Now()
				rows := sqlmock.NewRows([]string{"key", "enabled", "description", "created_at", "updated_at", "metadata"}).
					AddRow("test-flag", true, "Test flag description", now, now, string(metadataJSON))
				mock.ExpectQuery(`SELECT key, enabled, description, created_at, updated_at, metadata FROM feature_flags WHERE key = \$1`).
					WithArgs("test-flag").
					WillReturnRows(rows)
			},
			expectError: false,
			expected: &core.FeatureFlag{
				Key:         "test-flag",
				Enabled:     true,
				Description: "Test flag description",
				Metadata:    map[string]string{"env": "test", "team": "backend"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := store.Get(ctx, tt.key)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil {
					var ffErr *core.FeatureFlagError
					if errors.As(err, &ffErr) && !errors.Is(ffErr.Err, tt.errorType) {
						t.Errorf("expected error type %v, got %v", tt.errorType, ffErr.Err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result == nil {
					t.Errorf("expected result but got nil")
					return
				}
				if result.Key != tt.expected.Key {
					t.Errorf("expected key %s, got %s", tt.expected.Key, result.Key)
				}
				if result.Enabled != tt.expected.Enabled {
					t.Errorf("expected enabled %v, got %v", tt.expected.Enabled, result.Enabled)
				}
				if result.Description != tt.expected.Description {
					t.Errorf("expected description %s, got %s", tt.expected.Description, result.Description)
				}
				if !mapsEqual(result.Metadata, tt.expected.Metadata) {
					t.Errorf("expected metadata %v, got %v", tt.expected.Metadata, result.Metadata)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestPostgresStore_Set(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		flag        core.FeatureFlag
		setupMock   func()
		expectError bool
	}{
		{
			name: "invalid flag",
			flag: core.FeatureFlag{
				Key: "", // Invalid empty key
			},
			setupMock:   func() {},
			expectError: true,
		},
		{
			name: "database error",
			flag: core.FeatureFlag{
				Key:     "test-flag",
				Enabled: true,
			},
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO feature_flags`).
					WillReturnError(errors.New("database connection failed"))
			},
			expectError: true,
		},
		{
			name: "successful set with minimal data",
			flag: core.FeatureFlag{
				Key:     "test-flag",
				Enabled: true,
			},
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO feature_flags \(key, enabled, description, created_at, updated_at, metadata\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) ON CONFLICT \(key\) DO UPDATE SET enabled = EXCLUDED\.enabled, description = EXCLUDED\.description, updated_at = NOW\(\), metadata = EXCLUDED\.metadata`).
					WithArgs("test-flag", true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name: "successful set with full data",
			flag: core.FeatureFlag{
				Key:         "test-flag",
				Enabled:     true,
				Description: "Test description",
				Metadata:    map[string]string{"env": "test"},
			},
			setupMock: func() {
				mock.ExpectExec(`INSERT INTO feature_flags \(key, enabled, description, created_at, updated_at, metadata\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6\) ON CONFLICT \(key\) DO UPDATE SET enabled = EXCLUDED\.enabled, description = EXCLUDED\.description, updated_at = NOW\(\), metadata = EXCLUDED\.metadata`).
					WithArgs("test-flag", true, "Test description", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := store.Set(ctx, tt.flag)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestPostgresStore_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		key         string
		setupMock   func()
		expectError bool
		errorType   error
	}{
		{
			name:        "empty key",
			key:         "",
			setupMock:   func() {},
			expectError: true,
		},
		{
			name: "flag not found",
			key:  "nonexistent",
			setupMock: func() {
				mock.ExpectExec(`DELETE FROM feature_flags WHERE key = \$1`).
					WithArgs("nonexistent").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectError: true,
			errorType:   core.ErrFlagNotFound,
		},
		{
			name: "database error",
			key:  "test-flag",
			setupMock: func() {
				mock.ExpectExec(`DELETE FROM feature_flags WHERE key = \$1`).
					WithArgs("test-flag").
					WillReturnError(errors.New("database connection failed"))
			},
			expectError: true,
		},
		{
			name: "successful delete",
			key:  "test-flag",
			setupMock: func() {
				mock.ExpectExec(`DELETE FROM feature_flags WHERE key = \$1`).
					WithArgs("test-flag").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := store.Delete(ctx, tt.key)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil {
					var ffErr *core.FeatureFlagError
					if errors.As(err, &ffErr) && !errors.Is(ffErr.Err, tt.errorType) {
						t.Errorf("expected error type %v, got %v", tt.errorType, ffErr.Err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestPostgresStore_GetAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		setupMock   func()
		expectError bool
		expected    []core.FeatureFlag
	}{
		{
			name: "database error",
			setupMock: func() {
				mock.ExpectQuery(`SELECT key, enabled, description, created_at, updated_at, metadata FROM feature_flags ORDER BY key`).
					WillReturnError(errors.New("database connection failed"))
			},
			expectError: true,
		},
		{
			name: "empty result",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"key", "enabled", "description", "created_at", "updated_at", "metadata"})
				mock.ExpectQuery(`SELECT key, enabled, description, created_at, updated_at, metadata FROM feature_flags ORDER BY key`).
					WillReturnRows(rows)
			},
			expectError: false,
			expected:    []core.FeatureFlag{},
		},
		{
			name: "successful get all",
			setupMock: func() {
				now := time.Now()
				metadata1, _ := json.Marshal(map[string]string{"env": "test"})
				rows := sqlmock.NewRows([]string{"key", "enabled", "description", "created_at", "updated_at", "metadata"}).
					AddRow("flag1", true, "First flag", now, now, string(metadata1)).
					AddRow("flag2", false, nil, now, now, nil)
				mock.ExpectQuery(`SELECT key, enabled, description, created_at, updated_at, metadata FROM feature_flags ORDER BY key`).
					WillReturnRows(rows)
			},
			expectError: false,
			expected: []core.FeatureFlag{
				{
					Key:         "flag1",
					Enabled:     true,
					Description: "First flag",
					Metadata:    map[string]string{"env": "test"},
				},
				{
					Key:     "flag2",
					Enabled: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := store.GetAll(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if len(result) != len(tt.expected) {
					t.Errorf("expected %d flags, got %d", len(tt.expected), len(result))
					return
				}
				for i, expected := range tt.expected {
					if i >= len(result) {
						t.Errorf("missing flag at index %d", i)
						continue
					}
					actual := result[i]
					if actual.Key != expected.Key {
						t.Errorf("flag %d: expected key %s, got %s", i, expected.Key, actual.Key)
					}
					if actual.Enabled != expected.Enabled {
						t.Errorf("flag %d: expected enabled %v, got %v", i, expected.Enabled, actual.Enabled)
					}
					if actual.Description != expected.Description {
						t.Errorf("flag %d: expected description %s, got %s", i, expected.Description, actual.Description)
					}
					if !mapsEqual(actual.Metadata, expected.Metadata) {
						t.Errorf("flag %d: expected metadata %v, got %v", i, expected.Metadata, actual.Metadata)
					}
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestPostgresStore_HealthCheck(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		setupMock   func()
		expectError bool
	}{
		{
			name: "successful health check",
			setupMock: func() {
				mock.ExpectPing()
			},
			expectError: false,
		},
		{
			name: "failed health check",
			setupMock: func() {
				mock.ExpectPing().WillReturnError(errors.New("connection failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := store.HealthCheck(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestPostgresStore_Close(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	store := &PostgresStore{db: db}

	// Mock expects close to be called
	mock.ExpectClose()

	err = store.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
