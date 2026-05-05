package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"golang.org/x/crypto/bcrypt"

	authcore "github.com/mams/backend/internal/auth"
	"github.com/mams/backend/internal/config"
	authhandler "github.com/mams/backend/internal/handlers/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
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
	cfg := config.Get()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	users := memoryUsers{
		users: map[string]models.User{
			"vadim": {
				ID:             utils.MustUUID("11111111-1111-1111-1111-111111111111"),
				OrganizationID: utils.MustUUID("22222222-2222-2222-2222-222222222222"),
				Login:        "vadim",
				PasswordHash: string(hash),
			},
		},
	}
	issuer, err := authcore.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
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

	log.Printf("server listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen and serve: %v", err)
	}
}
