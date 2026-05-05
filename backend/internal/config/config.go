package config

import (
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/mams/backend/internal/utils"
)

type Config struct {
	HTTPAddr  string
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

		httpAddr := os.Getenv("HTTP_ADDR")
		if httpAddr == "" {
			httpAddr = ":8080"
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "dev-secret"
		}

		cfg = &Config{
			HTTPAddr:  httpAddr,
			JWTSecret: jwtSecret,
			JWTTTL:    utils.ParseTTLSeconds(os.Getenv("JWT_TTL"), 3600),
		}
	})

	return cfg
}
