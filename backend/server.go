package main

// GET /portfolio/analysis
// Fetches prices for all assets in the user's watchlist, computes portfolio metrics such as total value, percentage allocation per asset, and profit/loss if historical prices are stored. Makes multiple parallel API calls to fetch prices. Uses Alpha Vantage, Yahoo Finance, and CoinGecko. This endpoint is memory and CPU intensive if user has many assets.
// GET /markets/correlation
// Calculates correlation between selected symbols (stocks or crypto) using historical time-series data. Fetch historical data from Alpha Vantage (daily) and CoinGecko (crypto). Then compute Pearson or Spearman correlation between each pair of assets. Parsing and processing large historical data makes this CPU and memory intensive.
// POST /alerts/setup
// Allows users to set up alerts for specific symbols. Store alerts in DB with symbol, threshold (greater than or less than), and target price. Also save user_id, direction, and created_at.
// GET /alerts
// Returns all active alerts for the user. You may optionally implement this if you want a frontend to show active rules.
// GET /crypto/arbitrage
// Compares real-time prices of cryptocurrencies across multiple exchanges like Binance, Coinbase, Kraken, etc., using CoinGecko API. Computes and returns price difference and potential arbitrage opportunities. Useful for creating multi-API trace spans and detecting slowness in any one source.
// GET /economic/indicators
// Fetches global economic data like inflation, interest rates, GDP, etc. Use public APIs such as FRED, OECD, or mock an external slow API. Parsing nested or bulky data formats can create high-latency spans. This is a good endpoint to simulate long-running calls.

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	fmt.Println("Starting..")
	initDB()
	router.HandleFunc("/stocks/symbols", func(w http.ResponseWriter, r *http.Request) {
		getAllStockSymbols(w, r)
	}).Methods("GET")

	router.HandleFunc("/stocks/{symbol}", func(w http.ResponseWriter, r *http.Request) {
		getStockData(w, r)
	}).Methods("GET")

	router.HandleFunc("/crypto/symbols", func(w http.ResponseWriter, r *http.Request) {
		getAllCryptoSymbols(w, r)
	}).Methods("GET")

	router.HandleFunc("/crypto/{symbol}", func(w http.ResponseWriter, r *http.Request) {
		getCryptoData(w, r)
	}).Methods("GET")

	router.HandleFunc("/watchlist/add", func(w http.ResponseWriter, r *http.Request) {
		addToWatchlist(w, r)
	}).Methods("POST")

	router.HandleFunc("/watchlist/{userId}", func(w http.ResponseWriter, r *http.Request) {
		getWatchlist(w, r)
	}).Methods("GET")

	router.HandleFunc("/watchlist/remove/{userId}/{symbol}", func(w http.ResponseWriter, r *http.Request) {
		removeFromWatchlist(w, r)
	}).Methods("POST")

	router.HandleFunc("/portfolio/analysis/{userId}", func(w http.ResponseWriter, r *http.Request) {
		getPortfolioAnalysis(w, r)
	}).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", router))
}
