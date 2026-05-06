package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	authcore "github.com/mams/backend/internal/auth"
	"github.com/mams/backend/internal/bootstrap"
	"github.com/mams/backend/internal/config"
	authhandler "github.com/mams/backend/internal/handlers/auth"
	serviceshandler "github.com/mams/backend/internal/handlers/services"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/migrator"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
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

	migrationsDir := migrator.ResolveMigrationsDir()
	if err := migrator.Up(context.Background(), pool, migrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	if err := bootstrap.SeedAdmin(context.Background(), pool); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	users := postgresrepo.NewUserRepository(pool)
	services := postgresrepo.NewServiceRepositoryPool(pool)

	issuer, err := authcore.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		log.Fatalf("create jwt issuer: %v", err)
	}
	validator, err := authmw.NewJWTValidator(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("create jwt validator: %v", err)
	}
	login := authhandler.NewLoginHandler(users, issuer)
	servicesH := serviceshandler.NewHandler(services)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		login.Post(w, r)
	})
	mux.Handle("/api/auth/me", authmw.RequireAuth(validator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		login.Me(w, r)
	})))
	mux.Handle("/api/services", authmw.RequireAuth(validator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		servicesH.List(w, r)
	})))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	addr := cfg.HTTPAddr()
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	done := make(chan struct{})
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
		close(done)
	}()

	log.Printf("server listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen and serve: %v", err)
	}
	<-done
}
