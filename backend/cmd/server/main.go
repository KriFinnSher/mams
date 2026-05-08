package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	authcore "github.com/mams/backend/internal/auth"
	"github.com/mams/backend/internal/bootstrap"
	"github.com/mams/backend/internal/config"
	"github.com/mams/backend/internal/githubclient"
	authhandler "github.com/mams/backend/internal/handlers/auth"
	contractshandler "github.com/mams/backend/internal/handlers/contracts"
	logshandler "github.com/mams/backend/internal/handlers/logs"
	metricshandler "github.com/mams/backend/internal/handlers/metrics"
	releaseshandler "github.com/mams/backend/internal/handlers/releases"
	serviceshandler "github.com/mams/backend/internal/handlers/services"
	"github.com/mams/backend/internal/kubeclient"
	"github.com/mams/backend/internal/logx"
	authmw "github.com/mams/backend/internal/middleware/auth"
	rbacmw "github.com/mams/backend/internal/middleware/rbac"
	"github.com/mams/backend/internal/migrator"
	mongorepo "github.com/mams/backend/internal/repository/mongo"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/ws"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		_ = mongoClient.Disconnect(context.Background())
	}()

	migrationsDir := migrator.ResolveMigrationsDir()
	if err := migrator.Up(context.Background(), pool, migrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	if err := bootstrap.SeedAll(context.Background(), pool); err != nil {
		log.Fatalf("seed data: %v", err)
	}

	users := postgresrepo.NewUserRepository(pool)
	orgs := postgresrepo.NewOrganizationRepository(pool)
	services := postgresrepo.NewServiceRepositoryPool(pool)
	access := postgresrepo.NewServiceAccessRepositoryPool(pool)
	profile := postgresrepo.NewProfileReader(users, services)
	logsRepo := mongorepo.NewLogsRepositoryCollection(mongoClient.Database(cfg.MongoDB).Collection(cfg.MongoLogsCollection))
	releasesRepo := postgresrepo.NewReleaseRepositoryPool(pool)
	ghClient := githubclient.New(http.DefaultClient, cfg.GitHubToken)

	issuer, err := authcore.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		log.Fatalf("create jwt issuer: %v", err)
	}
	validator, err := authmw.NewJWTValidator(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("create jwt validator: %v", err)
	}
	logger := logx.New(slog.Default())
	hub := ws.NewHub()
	login := authhandler.NewLoginHandler(profile, issuer, logger)
	servicesH := serviceshandler.NewHandler(services, logger)
	logsH := logshandler.NewHandler(logsRepo, hub, logger)
	metricsH := metricshandler.NewHandler(services, cfg.GrafanaURL)
	contractsH := contractshandler.NewHandler(services, ghClient)
	if cfg.KubeConfigPath == "" {
		log.Fatal("KUBE_CONFIG_PATH is required")
	}
	restCfg, err := clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
	if err != nil {
		log.Fatalf("load kubeconfig: %v", err)
	}
	kubeSet, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		log.Fatalf("create kubernetes client: %v", err)
	}
	kube := kubeclient.NewWithDocker(kubeSet, cfg.DockerRegistry, cfg.DockerUsername, cfg.DockerPassword)
	releasesH := releaseshandler.NewHandler(services, orgs, releasesRepo, ghClient, kube, cfg.CallbackBaseURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		login.Post(w, r)
	})
	protected := http.NewServeMux()
	protected.Handle("/api/auth/me", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		login.Me(w, r)
	}))
	protected.Handle("/api/services", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			servicesH.List(w, r)
		case http.MethodPost:
			servicesH.Create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	protected.Handle("/api/services/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			servicesH.Get(w, r)
		case http.MethodPut:
			servicesH.UpdateInfo(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	protected.Handle("/api/services/{id}/settings", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			servicesH.GetSettings(w, r)
		case http.MethodPut:
			servicesH.UpdateSettings(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	protected.Handle("/api/services/{id}/logs", rbacmw.RequireLogsAccess(services, access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		logsH.Get(w, r)
	})))
	protected.Handle("/api/services/{id}/logs/stream", rbacmw.RequireLogsAccess(services, access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		logsH.Stream(w, r)
	})))
	protected.Handle("/api/services/{id}/metrics", rbacmw.RequireMetricsAccess(services, access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		metricsH.Get(w, r)
	})))
	protected.Handle("/api/services/{id}/contracts", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		contractsH.Get(w, r)
	}))
	protected.Handle("/api/services/{id}/releases", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		releasesH.Get(w, r)
	}))
	protected.Handle("/api/services/{id}/deploy", rbacmw.RequireReleaseManageAccess(services, access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		releasesH.Deploy(w, r)
	})))
	protected.Handle("/api/services/{id}/rollback", rbacmw.RequireReleaseManageAccess(services, access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		releasesH.Rollback(w, r)
	})))
	protected.Handle("/api/services/{id}/rollback/candidates", rbacmw.RequireReleaseManageAccess(services, access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		releasesH.RollbackCandidates(w, r)
	})))
	mux.Handle("/api/", authmw.RequireAuth(validator, protected))

	mux.HandleFunc("/api/internal/services/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		logsH.Ingest(w, r)
	})
	mux.HandleFunc("/api/internal/releases/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		releasesH.UpdateFromCI(w, r)
	})
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
