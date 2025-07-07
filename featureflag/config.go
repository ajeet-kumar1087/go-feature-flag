package featureflag

import (
	"fmt"
)

// Config holds the configuration for the feature flag service
type Config struct {
	redis    *RedisAddr
	postgres *PostgresAddr
}

type RedisAddr struct {
	Addr     string
	Port     int
	Password string
	Database int
}

// SetRedis sets the Redis address and port in the Config.
func (r *RedisAddr) SetRedis(addr string, port int, password string) {
	r.Addr = addr
	r.Port = port
	r.Password = password
	r.Database = 0 // Default database index
	if r.Database < 0 {
		r.Database = 0 // Ensure database index is non-negative
	}

}

func (r *RedisAddr) GetRedisFormatted() string {
	return fmt.Sprintf("%s:%d", r.Addr, r.Port)
}

// PostgresAddr holds the address and port for the Postgres database.
type PostgresAddr struct {
	Addr string
	Port int
}

// SetPostgres sets the Postgres address and port in the Config.
func (p *PostgresAddr) SetPostgres(addr string, port int) {
	p.Addr = addr
	p.Port = port
}

// NewConfig initializes a new Config with Redis and Postgres addresses.
func NewConfig(p *PostgresAddr, r *RedisAddr) *Config {
	return &Config{
		redis:    r,
		postgres: p,
	}
}

// Handler returns an http.Handler with all feature flag routes set up.
// func Handler(cfg Config) http.Handler {
// 	store := NewCachedStore(
// 		NewRedisStore(cfg.redis.Addr, cfg.redis.Port),
// 		NewPostgresStore(cfg.postgres.Addr, cfg.postgres.Port),
// 	)
// 	return SetupRoutes(store)

// }
