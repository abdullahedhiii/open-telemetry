package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

func main() {
	router := mux.NewRouter()
	fmt.Println("Starting..")

	shutdown, err := initTelemetry()
	if err != nil {
		log.Fatal("Failed to initialize telemetry:", err)
	}
	defer shutdown()
	fmt.Println("Telemetry initialized")

	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		log.Fatal("Failed to start runtime metrics:", err)
	}
	fmt.Println("Runtime metrics started")

	//note: this is for auto instrumentation of routes
	// router.Use(otelmux.Middleware("stock-tracker"))

	// router.Use(OpenTelemetryMetricsMiddleware())

	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	defer CloseDB()
	fmt.Println("Database initialized")

	router.HandleFunc("/users/login", loginUser).Methods("POST")
	router.HandleFunc("/users/register", registerUser).Methods("POST")
	router.HandleFunc("/stocks/symbols", getAllStockSymbols).Methods("GET")
	router.HandleFunc("/stocks/{symbol}", getStockData).Methods("GET")
	router.HandleFunc("/crypto/symbols", getAllCryptoSymbols).Methods("GET")
	router.HandleFunc("/crypto/{symbol}", getCryptoData).Methods("GET")
	router.HandleFunc("/watchlist/add", addToWatchlist).Methods("POST")
	router.HandleFunc("/watchlist/{userId}", getWatchlist).Methods("GET")
	router.HandleFunc("/watchlist/remove/{userId}/{type}/{symbol}", removeFromWatchlist).Methods("POST")
	router.HandleFunc("/log-event", logFrontendEvent).Methods("POST")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"traceparent", "tracestate", "Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)
	server := &http.Server{
		Addr:    ":8000",
		Handler: handler,
	}

	go func() {
		fmt.Println("Server starting on :8000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server exited")
}
