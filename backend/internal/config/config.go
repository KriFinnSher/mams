package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/mams/backend/internal/utils"
)

type Config struct {
	HTTPHost  string
	HTTPPort  string
	PostgresDSN string
	JWTSecret string
	JWTTTL    time.Duration
}

var (
	cfg  *Config
	once sync.Once
)

func Get() *Config {
	once.Do(func() {
		_ = godotenv.Load()

		httpHost := os.Getenv("HTTP_HOST")
		if httpHost == "" {
			httpHost = "0.0.0.0"
		}

		httpPort := os.Getenv("HTTP_PORT")
		if httpPort == "" {
			httpPort = "8080"
		}

		postgresDSN := os.Getenv("POSTGRES_DSN")
		if postgresDSN == "" {
			postgresDSN = "postgres://postgres:postgres@localhost:5432/mams?sslmode=disable"
		}

		jwtSecret := os.Getenv("JWT_SECRET")

		cfg = &Config{
			HTTPHost:  httpHost,
			HTTPPort:  httpPort,
			PostgresDSN: postgresDSN,
			JWTSecret: jwtSecret,
			JWTTTL:    utils.ParseTTLSeconds(os.Getenv("JWT_TTL"), 3600),
		}
	})

	return cfg
}

func (c *Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%s", c.HTTPHost, c.HTTPPort)
}
