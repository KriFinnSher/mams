package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	authcore "github.com/mams/backend/internal/auth"
	authhandler "github.com/mams/backend/internal/handlers/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
)

type memoryUsers struct {
	users map[string]models.User
}

func (m memoryUsers) GetByLogin(_ context.Context, login string) (models.User, error) {
	u, ok := m.users[login]
	if !ok {
		return models.User{}, postgresrepo.ErrUserNotFound
	}
	return u, nil
}

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret"
	}
	jwtTTL := parseTTLSeconds(os.Getenv("JWT_TTL"), 3600)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	users := memoryUsers{
		users: map[string]models.User{
			"vadim": {
				ID:             mustUUID("11111111-1111-1111-1111-111111111111"),
				OrganizationID: mustUUID("22222222-2222-2222-2222-222222222222"),
				Login:        "vadim",
				PasswordHash: string(hash),
			},
		},
	}
	issuer, err := authcore.NewJWTIssuer(jwtSecret, jwtTTL)
	if err != nil {
		log.Fatalf("create jwt issuer: %v", err)
	}
	login := authhandler.NewLoginHandler(users, issuer)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		login.Post(w, r)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen and serve: %v", err)
	}
}

func parseTTLSeconds(raw string, fallback int64) time.Duration {
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || sec <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(sec) * time.Second
}

func mustUUID(v string) uuid.UUID {
	id, err := uuid.Parse(v)
	if err != nil {
		panic(err)
	}
	return id
}
