package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"webhook-ingestion-service/internal/httpapi"
	"webhook-ingestion-service/internal/store/postgres"
	"webhook-ingestion-service/internal/task"
)

func main() {
	dbURL := os.Getenv("DB_URL")
	secret := os.Getenv("WEBHOOK_SECRET")
	if dbURL == "" {
		log.Fatal("DB_URL is required")
	}
	if secret == "" {
		log.Fatal("WEBHOOK_SECRET is required")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		log.Fatalf("db ping: %v", err)
	}
	cancel()

	repo := postgres.NewEventRepo(db)
	svc := task.NewService(repo)

	// Worker deps
	workerDeps := task.WorkerDeps{
		Repo:      repo,
		Processor: task.NoopProcessor{}, // podmień później na real processor
		Backoff:   task.DefaultBackoff(),
		Now:       func() time.Time { return time.Now().UTC() },
		// RNG optional
	}

	// Root context cancelled on SIGINT/SIGTERM
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start background worker
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		task.RunWorker(rootCtx, workerDeps, task.DefaultWorkerConfig(), log.Default())
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler())
	mux.HandleFunc("/readyz", httpapi.ReadyzHandler(db))
	mux.HandleFunc("/webhooks/provider", httpapi.WebhookProviderHandler(secret, nil, svc))
	mux.HandleFunc("/events/", httpapi.GetEventHandler(svc))
	mux.HandleFunc("/process/once", httpapi.ProcessOnceHandler(workerDeps))

	handler := httpapi.WithRequestID(log.Default())(
		httpapi.Logging(log.Default())(
			mux,
		),
	)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-rootCtx.Done()
	log.Printf("shutdown signal received")

	// Stop accepting new requests; wait for in-flight with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	// Wait for worker to stop (it stops because rootCtx is cancelled)
	wg.Wait()
	log.Printf("bye")
}

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
