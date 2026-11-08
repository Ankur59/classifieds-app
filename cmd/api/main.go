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

	"github.com/ankur59/classifieds-app/internal/config"
	"github.com/ankur59/classifieds-app/internal/config/handlers"
	"github.com/ankur59/classifieds-app/internal/db"
	"github.com/ankur59/classifieds-app/internal/middleware"
	"github.com/ankur59/classifieds-app/internal/repository"
)

// shutdownTimeout bounds how long in-flight requests get to finish on SIGINT/SIGTERM.
const shutdownTimeout = 15 * time.Second

// main is the entry point for the classifieds API server.
func main() {
	// Load PORT and DB_URL from .env; exits if either is missing.
	cfg := config.MustLoad()
	conn, err := db.Connect(cfg.DB_URL)
	if err != nil {
		log.Fatalf("main.db.connect: %v", err)
	}
	defer conn.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.Healthz)

	students := handlers.NewStudentHandler(repository.NewStudentRepository(conn))
	mux.HandleFunc("POST /students", students.Create)
	mux.HandleFunc("GET /students", students.List)
	mux.HandleFunc("GET /students/{id}", students.Get)
	mux.HandleFunc("PUT /students/{id}", students.Update)
	mux.HandleFunc("DELETE /students/{id}", students.Delete)

	server := &http.Server{
		Addr:         ":" + cfg.PORT,
		Handler:      middleware.Logging(middleware.Recover(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server listening at Port: %s", cfg.PORT)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		// ListenAndServe only returns on failure here (e.g. port already in use).
		log.Fatalf("unable to start server: %v", err)
	case <-ctx.Done():
		stop() // a second Ctrl+C kills the process immediately
		log.Printf("shutdown signal received, draining requests (max %s)", shutdownTimeout)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server error: %v", err)
	}
	log.Print("server stopped")
}
