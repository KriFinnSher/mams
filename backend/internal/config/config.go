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
	MongoURI  string
	MongoDB   string
	MongoLogsCollection string
	GrafanaURL string
	GitHubToken string
	KubeConfigPath string
	JWTSecret string
	JWTTTL    time.Duration
	CallbackBaseURL string
	DockerRegistry   string
	DockerUsername   string
	DockerPassword   string
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
		mongoURI := os.Getenv("MONGO_URI")
		if mongoURI == "" {
			mongoURI = "mongodb://localhost:27017"
		}
		mongoDB := os.Getenv("MONGO_DB")
		if mongoDB == "" {
			mongoDB = "mams"
		}
		mongoLogsCollection := os.Getenv("MONGO_LOGS_COLLECTION")
		if mongoLogsCollection == "" {
			mongoLogsCollection = "logs"
		}
		grafanaURL := os.Getenv("GRAFANA_URL")
		if grafanaURL == "" {
			grafanaURL = "http://localhost:3001"
		}
		gitHubToken := os.Getenv("GITHUB_TOKEN")
		kubeConfigPath := os.Getenv("KUBE_CONFIG_PATH")
		if kubeConfigPath == "" {
			kubeConfigPath = os.ExpandEnv("$HOME/.kube/config")
		}
		callbackBaseURL := os.Getenv("CALLBACK_BASE_URL")
		if callbackBaseURL == "" {
			callbackBaseURL = "http://host.docker.internal:8081"
		}
		dockerRegistry := os.Getenv("DOCKER_REGISTRY")
		dockerUsername := os.Getenv("DOCKER_USERNAME")
		dockerPassword := os.Getenv("DOCKER_PASSWORD")

		cfg = &Config{
			HTTPHost:  httpHost,
			HTTPPort:  httpPort,
			PostgresDSN: postgresDSN,
			MongoURI: mongoURI,
			MongoDB: mongoDB,
			MongoLogsCollection: mongoLogsCollection,
			GrafanaURL: grafanaURL,
			GitHubToken: gitHubToken,
			KubeConfigPath: kubeConfigPath,
			JWTSecret: jwtSecret,
			JWTTTL:    utils.ParseTTLSeconds(os.Getenv("JWT_TTL"), 3600),
			CallbackBaseURL: callbackBaseURL,
			DockerRegistry: dockerRegistry,
			DockerUsername: dockerUsername,
			DockerPassword: dockerPassword,
		}
	})

	return cfg
}

func (c *Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%s", c.HTTPHost, c.HTTPPort)
}
