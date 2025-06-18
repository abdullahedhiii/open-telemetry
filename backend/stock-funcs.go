package main

import (
	_ "context" // Crucial for context propagation
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"           // The main OpenTelemetry package
	"go.opentelemetry.io/otel/attribute" // For span and metric attributes
	"go.opentelemetry.io/otel/codes"     // For span status
	"go.opentelemetry.io/otel/metric"    // For metric instruments
	_ "go.opentelemetry.io/otel/trace"   // For span operations
)

const apiKey = "26AMBY8WA3V0FCMD" // Assuming this is correct and safe to hardcode for dev

func getAllStockSymbols(w http.ResponseWriter, r *http.Request) {

	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getAllStockSymbolsHandler")
	defer span.End()

	r = r.WithContext(ctx)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/stocks/symbols"),
		attribute.String("method", r.Method),
	))

	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=LISTING_STATUS&apikey=%s", apiKey)

	_, apiCallSpan := tracer.Start(ctx, "alphaVantage.LISTING_STATUS")
	apiCallSpan.SetAttributes(
		attribute.String("http.url", apiUrl),
		attribute.String("http.method", "GET"),
		attribute.String("api.name", "alphavantage"),
		attribute.String("api.operation", "LISTING_STATUS"),
	)
	defer apiCallSpan.End()

	apiCallStartTime := time.Now()
	response, err := http.Get(apiUrl)
	apiCallDuration := time.Since(apiCallStartTime).Seconds()

	externalAPICallDuration.Record(ctx, apiCallDuration, metric.WithAttributes(
		attribute.String("api.name", "alphavantage_api"),
		attribute.String("api.operation", "LISTING_STATUS"),
		attribute.Bool("api.error", err != nil),
	))

	if err != nil {
		apiCallSpan.SetStatus(codes.Error, fmt.Sprintf("HTTP GET failed: %v", err))
		apiCallSpan.RecordError(err)

		span.SetStatus(codes.Error, fmt.Sprintf("External API call failed: %v", err))
		span.RecordError(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	apiCallSpan.SetAttributes(
		attribute.Int("http.status_code", response.StatusCode),
		attribute.String("http.response_content_type", response.Header.Get("Content-Type")),
	)
	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)                          // Read body for error context
		response.Body = io.NopCloser(strings.NewReader(string(bodyBytes))) // Restore body for later
		errorMsg := fmt.Sprintf("API returned non-OK status: %d, body: %s", response.StatusCode, bodyBytes)

		apiCallSpan.SetStatus(codes.Error, errorMsg)
		span.SetStatus(codes.Error, errorMsg)
		http.Error(w, errorMsg, response.StatusCode)
		return
	}
	apiCallSpan.SetStatus(codes.Ok, "API call successful")

	reader := csv.NewReader(response.Body)
	reader.Read()
	var symbols []string

	_, readCsvSpan := tracer.Start(ctx, "processCsvResponse")
	defer readCsvSpan.End()
	readCsvSpan.AddEvent("Starting CSV parsing")

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			readCsvSpan.SetStatus(codes.Error, fmt.Sprintf("CSV read error: %v", err))
			readCsvSpan.RecordError(err)
			span.SetStatus(codes.Error, fmt.Sprintf("CSV processing failed: %v", err))
			span.RecordError(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(record[6]) == "Active" {
			symbols = append(symbols, record[0])
		}
	}
	readCsvSpan.SetAttributes(attribute.Int("symbols.active_count", len(symbols)))
	readCsvSpan.SetStatus(codes.Ok, "CSV parsing complete")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(symbols); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "Stock symbols retrieved successfully")
	span.AddEvent("Response sent")
	json.NewEncoder(w).Encode(symbols)
}

func getStockData(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["symbol"]
	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=compact&apikey=%s", symbol, apiKey)

	response, err := http.Get(apiUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var stockData map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&stockData); err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stockData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type coinData struct {
	Symbol string
	Id     string
}

func getAllCryptoSymbols(w http.ResponseWriter, r *http.Request) {

	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getAllCryptoSymbolsHandler")
	defer span.End()

	r = r.WithContext(ctx)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	fmt.Println("Registering http call")
	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/crypto/symbols"),
		attribute.String("method", r.Method),
	))
	fmt.Println("Registered http")
	apiUrl := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h"
	fmt.Println("REgistering API call")
	_, apiCallSpan := tracer.Start(ctx, "coingecko.LISTED_COINS")
	apiCallSpan.SetAttributes(
		attribute.String("http.url", apiUrl),
		attribute.String("http.method", "GET"),
		attribute.String("api.name", "alphavantage"),
		attribute.String("api.operation", "LISTING_STATUS"),
	)
	defer apiCallSpan.End()

	apiCallStartTime := time.Now()
	response, err := http.Get(apiUrl)
	apiCallDuration := time.Since(apiCallStartTime).Seconds()

	fmt.Println("REgistered API call")
	externalAPICallDuration.Record(ctx, apiCallDuration, metric.WithAttributes(
		attribute.String("api.name", "coingecko_api"),
		attribute.String("api.operation", "LISTED_COINS"),
		attribute.Bool("api.error", err != nil),
	))

	if err != nil {
		apiCallSpan.SetStatus(codes.Error, fmt.Sprintf("HTTP GET failed: %v", err))
		apiCallSpan.RecordError(err)

		span.SetStatus(codes.Error, fmt.Sprintf("External API call failed: %v", err))
		span.RecordError(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	apiCallSpan.SetAttributes(
		attribute.Int("http.status_code", response.StatusCode),
		attribute.String("http.response_content_type", response.Header.Get("Content-Type")),
	)
	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)                          // Read body for error context
		response.Body = io.NopCloser(strings.NewReader(string(bodyBytes))) // Restore body for later
		errorMsg := fmt.Sprintf("API returned non-OK status: %d, body: %s", response.StatusCode, bodyBytes)

		apiCallSpan.SetStatus(codes.Error, errorMsg)
		span.SetStatus(codes.Error, errorMsg)
		http.Error(w, errorMsg, response.StatusCode)
		return
	}
	apiCallSpan.SetStatus(codes.Ok, "API call successful")

	var cryptoData []map[string]interface{}
	_, readJSONparser := tracer.Start(ctx, "processJSONresponse")
	defer readJSONparser.End()
	readJSONparser.AddEvent("Starting JSON response parsing")
	if err := json.NewDecoder(response.Body).Decode(&cryptoData); err != nil {
		readJSONparser.SetStatus(codes.Error, fmt.Sprintf("JSON read error: %v", err))
		readJSONparser.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("crypto coins processing failed: %v", err))
		span.RecordError(err)
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}

	var symbols []coinData
	for _, coin := range cryptoData {
		symbols = append(symbols, coinData{Symbol: coin["symbol"].(string), Id: coin["id"].(string)})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(symbols); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "Crypto symbols retrieved successfully")
	span.AddEvent("Response sent")
	json.NewEncoder(w).Encode(symbols)
}

func getCryptoData(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["symbol"]
	apiUrl := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h&ids=%s", symbol)

	response, err := http.Get(apiUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var cryptoData []map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&cryptoData); err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cryptoData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func addToWatchlist(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Symbol   string `json:"symbol"`
		UserId   string `json:"userId"`
		Type     string `json:"type"`
		CryptoId string `json:"cryptoId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	var err error
	if data.Type == "STOCK" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "STOCK", CryptoId: ""}
		err = DB.Create(&new_symbol).Error
	} else if data.Type == "CRYPTO" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "CRYPTO", CryptoId: data.CryptoId}
		err = DB.Create(&new_symbol).Error
	}

	if err != nil {
		http.Error(w, "Failed to create watchlist item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getWatchlist(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["userId"]
	var watchlist []UserSymbols
	DB.Where("user_id = ?", userId).Find(&watchlist)
	json.NewEncoder(w).Encode(watchlist)
}

func removeFromWatchlist(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["userId"]
	symbol := mux.Vars(r)["symbol"]
	DB.Where("user_id = ? AND symbol = ?", userId, symbol).Delete(&UserSymbols{})
	json.NewEncoder(w).Encode(map[string]string{"message": "Symbol removed from watchlist"})
}
