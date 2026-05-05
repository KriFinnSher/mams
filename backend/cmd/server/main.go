package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"

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

type staticIssuer struct{}

func (staticIssuer) IssueToken(_ models.User) (string, error) {
	return "dev-token", nil
}

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	users := memoryUsers{
		users: map[string]models.User{
			"vadim": {
				Login:        "vadim",
				PasswordHash: string(hash),
			},
		},
	}
	login := authhandler.NewLoginHandler(users, staticIssuer{})

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
