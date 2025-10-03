package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/smontalbano/microservice/handlers"
)

func main() {
	l := log.New(os.Stdout, "product-api", log.LstdFlags)

	// create the handlers
	ph := handlers.NewProducts(l)

	// create a new serve mux and register the handlers
	sm := mux.NewRouter()

	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/", ph.GetProducts)

	putRouter := sm.Methods(http.MethodPut).Subrouter()
	putRouter.HandleFunc("/{id:[0-9]+}", ph.UpdateProducts)
	putRouter.Use(ph.MiddlewareProductValidation)

	postRouter := sm.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/", ph.AddProduct)
	postRouter.Use(ph.MiddlewareProductValidation)

	// create a new server
	s := &http.Server{
		Addr:         ":9911",
		Handler:      sm,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	l.Println("Starting server on port 9911")

	// start the server
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Only fatal if it's not a graceful shutdown
			l.Fatalf("Could not listen on :9911 %v\n", err)
		}
	}()

	// Trap interrupt or terminate signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigChan
	l.Println("Received terminate, graceful shutdown", sig)

	// Create deadline to wait for ongoing requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := s.Shutdown(ctx); err != nil {
		l.Printf("Graceful shutdown did not complete in 30s: %v\n", err)
		if err := s.Close(); err != nil {
			l.Printf("Error forcing close: %v\n", err)
		}
	}

	l.Println("Server shut down gracefully")
}
