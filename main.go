package main

import (
	"log"
	"net/http"
	"time"

	"github.com/ankur59/classifieds-app/internal/config"
	"github.com/ankur59/classifieds-app/internal/config/handlers"
)

func main() {
	cfg := config.MustLoad()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.Healthz)

	server := http.Server{
		Addr:         ":" + cfg.PORT,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	err := server.ListenAndServe()
	log.Printf("Server listening at Port: %s", cfg.PORT)
	if err != nil {
		log.Fatal("unable to start server")
	}
}
