package config

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	RPC          string         `env:"RPC" env-required:"true"`
	FallbackRPCs []string       `env:"FALLBACK_RPCS" env-default:"https://rpc.ankr.com/eth,https://eth.api.onfinality.io/public"`
	TokenAddress common.Address `env:"TOKEN_ADDRESS" env-required:"true"`
	DB           struct {
		Host     string        `env:"DB_HOST" env-required:"true"`
		Port     int           `env:"DB_PORT" env-required:"true"`
		User     string        `env:"DB_USER" env-required:"true"`
		Password string        `env:"DB_PASSWORD" env-required:"true"`
		Name     string        `env:"DB_NAME" env-required:"true"`
		SSLMode  string        `env:"DB_SSL_MODE" env-default:"disable"`
		MaxConns int           `env:"DB_MAX_CONNS" env-default:"10"`
		Timeout  time.Duration `env:"DB_TIMEOUT" env-default:"5s"`
	}
}

// DSN returns database connection string
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Name, c.DB.SSLMode)
}

func LoadConfig() (Config, error) {
	var cfg Config

	// Try different possible locations for .env file
	envFiles := []string{
		".env",
		"../.env",
		"../../.env",
	}

	var readErr error
	for _, file := range envFiles {
		err := cleanenv.ReadConfig(file, &cfg)
		if err == nil {
			return cfg, nil
		}
		readErr = err
	}

	// If no .env file found, try environment variables
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to read config from env files (%v) and environment: %v", readErr, err)
	}

	return cfg, nil
}
