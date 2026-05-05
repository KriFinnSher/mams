package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	authcore "github.com/mams/backend/internal/auth"
	"github.com/mams/backend/internal/config"
	authhandler "github.com/mams/backend/internal/handlers/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Get()
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	pool, err := pgxpool.New(context.Background(), cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	users := postgresrepo.NewUserRepository(pool)

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

	addr := cfg.HTTPAddr()
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen and serve: %v", err)
	}
}
