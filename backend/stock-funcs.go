package main

import (
	_ "context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	_ "go.opentelemetry.io/otel/trace"
)

const apiKey = "26AMBY8WA3V0FCMD" // Assuming this is correct and safe to hardcode for dev

func getAllStockSymbols(w http.ResponseWriter, r *http.Request) {

	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getAllStockSymbolsHandler")
	defer span.End()

	r = r.WithContext(ctx)

	Logger.InfoContext(ctx, "Handler execution started", "method", r.Method, "target", r.URL.Path)

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

	Logger.InfoContext(ctx, "External API call made", "url", apiUrl, "duration_sec", apiCallDuration, "error", err)

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

		Logger.ErrorContext(ctx, "HTTP GET to AlphaVantage failed", "error", err)
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
		Logger.ErrorContext(ctx, "AlphaVantage returned non-OK status", "status_code", response.StatusCode, "body", string(bodyBytes))
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
			Logger.ErrorContext(ctx, "CSV parsing error", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(record[6]) == "Active" {
			symbols = append(symbols, record[0])
		}
	}
	readCsvSpan.SetAttributes(attribute.Int("symbols.active_count", len(symbols)))
	readCsvSpan.SetStatus(codes.Ok, "CSV parsing complete")

	Logger.InfoContext(ctx, "CSV parsing complete", "active_symbols_count", len(symbols))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(symbols); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		Logger.ErrorContext(ctx, "Failed to encode JSON response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "Stock symbols retrieved successfully")
	span.AddEvent("Response sent")
	Logger.InfoContext(ctx, "Stock symbols retrieved and response sent", "count", len(symbols))
}

func getStockData(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getStockDataFromSymbol")
	defer span.End()

	r = r.WithContext(ctx)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	symbol := mux.Vars(r)["symbol"]
	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=compact&apikey=%s", symbol, apiKey)

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/stocks/{symbol}"),
		attribute.String("method", r.Method),
	))

	_, apiCallSpan := tracer.Start(ctx, "alphaVantage.TIME_SERIES_DAILY")
	apiCallSpan.SetAttributes(
		attribute.String("http.url", apiUrl),
		attribute.String("http.method", "GET"),
		attribute.String("api.name", "alphavantage"),
		attribute.String("api.operation", "TIME_SERIES_DAILY"),
	)
	defer apiCallSpan.End()

	apiCallStartTime := time.Now()
	response, err := http.Get(apiUrl)
	apiCallDuration := time.Since(apiCallStartTime).Seconds()

	externalAPICallDuration.Record(ctx, apiCallDuration, metric.WithAttributes(
		attribute.String("api.name", "alphavantage_api"),
		attribute.String("api.operation", "TIME_SERIES_DAILY"),
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
		bodyBytes, _ := io.ReadAll(response.Body)
		response.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		errorMsg := fmt.Sprintf("API returned non-OK status: %d, body: %s", response.StatusCode, bodyBytes)

		apiCallSpan.SetStatus(codes.Error, errorMsg)
		span.SetStatus(codes.Error, errorMsg)
		http.Error(w, errorMsg, response.StatusCode)
		return
	}
	apiCallSpan.SetStatus(codes.Ok, "API call successful")

	_, responseSpan := tracer.Start(ctx, "processAPIresponse")
	defer responseSpan.End()
	responseSpan.AddEvent("Started decoding JSON")

	var stockData map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&stockData); err != nil {
		responseSpan.SetStatus(codes.Error, fmt.Sprintf("JSON read error: %v", err))
		responseSpan.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("JSON decoding failed: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(stockData)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		span.SetStatus(codes.Ok, "Stock Data retrieved successfully")
		span.AddEvent("Response sent")
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

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/crypto/symbols"),
		attribute.String("method", r.Method),
	))

	apiUrl := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h"

	_, apiCallSpan := tracer.Start(ctx, "coingecko.LISTED_COINS")
	apiCallSpan.SetAttributes(
		attribute.String("http.url", apiUrl),
		attribute.String("http.method", "GET"),
		attribute.String("api.name", "coingecko"),
		attribute.String("api.operation", "LISTING_STATUS"),
	)
	defer apiCallSpan.End()

	apiCallStartTime := time.Now()
	response, err := http.Get(apiUrl)
	apiCallDuration := time.Since(apiCallStartTime).Seconds()

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
	err = json.NewEncoder(w).Encode(symbols)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		span.SetStatus(codes.Ok, "Crypto symbols retrieved successfully")
		span.AddEvent("Response sent")
	}

}

func getCryptoData(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getCoinDataFromSymbol")
	defer span.End()

	r = r.WithContext(ctx)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/crypto/{symbol}"),
		attribute.String("method", r.Method),
	))

	symbol := mux.Vars(r)["symbol"]
	apiUrl := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h&ids=%s", symbol)

	_, apiCallSpan := tracer.Start(ctx, "coingecko.COIN_DATA")
	apiCallSpan.SetAttributes(
		attribute.String("http.url", apiUrl),
		attribute.String("http.method", "GET"),
		attribute.String("api.name", "coingecko"),
		attribute.String("api.operation", "COIN_DATA"),
	)
	defer apiCallSpan.End()

	apiCallStartTime := time.Now()
	response, err := http.Get(apiUrl)
	apiCallDuration := time.Since(apiCallStartTime).Seconds()

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

	_, responseSpan := tracer.Start(ctx, "processAPIresponse")
	defer responseSpan.End()
	responseSpan.AddEvent("Started decoding JSON")

	var cryptoData []map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&cryptoData); err != nil {
		responseSpan.SetStatus(codes.Error, fmt.Sprintf("JSON read error: %v", err))
		responseSpan.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("JSON decoding failed: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cryptoData)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		span.SetStatus(codes.Ok, "Crypto data retrieved successfully")
		span.AddEvent("Response sent")
	}
}

func addToWatchlist(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("In add watchlist")
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "addToWatchList")
	defer span.End()

	r = r.WithContext(ctx)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/add"),
		attribute.String("method", r.Method),
	))
	// fmt.Println("http request set")
	var data struct {
		Symbol   string `json:"symbol"`
		UserId   string `json:"userId"`
		Type     string `json:"type"`
		CryptoId string `json:"cryptoId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to decoded request body: %v", err))
		span.RecordError(err)
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}
	// fmt.Printf("Request body decoded")
	_, dbCallSpan := tracer.Start(ctx, "db_call_addToList")
	dbCallSpan.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "INSERT"),
	)
	defer dbCallSpan.End()
	// fmt.Println("db call span set")
	var err error
	var dbCallDuration float64

	startTime := time.Now()
	if data.Type == "STOCK" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "STOCK", CryptoId: ""}
		err = DB.Create(&new_symbol).Error
		dbCallDuration = time.Since(startTime).Seconds()
	} else if data.Type == "CRYPTO" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "CRYPTO", CryptoId: data.CryptoId}
		err = DB.Create(&new_symbol).Error
		dbCallDuration = time.Since(startTime).Seconds()
	}

	// fmt.Println("Added to db")
	var status string
	if err != nil {
		status = "failure"
	} else {
		status = "success"
	}
	dbQueryCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/add"),
		attribute.String("status", status),
	))

	dbQueryDuration.Record(ctx, dbCallDuration, metric.WithAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "INSERT"),
		attribute.Bool("db.error", err != nil),
	))
	// fmt.Println("Db count ad duration set")
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to create watch list ite: %v", err))
		span.RecordError(err)
		http.Error(w, "Failed to create watchlist item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		// fmt.Println("Setting error")
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		// fmt.Println("Response sent")
		span.SetStatus(codes.Ok, "Watch list item added successfully")
		span.AddEvent("Response sent")
	}
}

func getWatchlist(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getWatchlistHandler")
	defer span.End()

	r = r.WithContext(ctx)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/{userId}"),
		attribute.String("method", r.Method),
	))

	userId := mux.Vars(r)["userId"]
	var watchlist []UserSymbols

	startTime := time.Now()
	_, dbCallSpan := tracer.Start(ctx, "db_call_getWatchlist")
	dbCallSpan.SetAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "SELECT"),
	)
	DB.Where("user_id = ?", userId).Find(&watchlist)
	dbCallDuration := time.Since(startTime).Seconds()
	dbCallSpan.End()

	dbQueryCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/{userId}"),
		attribute.String("status", "success"),
	))
	dbQueryDuration.Record(ctx, dbCallDuration, metric.WithAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "SELECT"),
		attribute.Bool("db.error", false),
	))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(watchlist); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "Watchlist retrieved successfully")
	span.AddEvent("Response sent")
}

func removeFromWatchlist(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "removeFromWatchlistHandler")
	defer span.End()

	r = r.WithContext(ctx)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/remove"),
		attribute.String("method", r.Method),
	))

	userId := mux.Vars(r)["userId"]
	symbol := mux.Vars(r)["symbol"]

	startTime := time.Now()
	_, dbCallSpan := tracer.Start(ctx, "db_call_removeFromWatchlist")
	dbCallSpan.SetAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "DELETE"),
	)
	result := DB.Where("user_id = ? AND symbol = ?", userId, symbol).Delete(&UserSymbols{})
	dbCallDuration := time.Since(startTime).Seconds()
	dbCallSpan.End()

	var status string
	if result.Error != nil {
		status = "failure"
	} else {
		status = "success"
	}

	dbQueryCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/remove"),
		attribute.String("status", status),
	))
	dbQueryDuration.Record(ctx, dbCallDuration, metric.WithAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "DELETE"),
		attribute.Bool("db.error", result.Error != nil),
	))

	w.Header().Set("Content-Type", "application/json")
	if result.Error != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to remove symbol from watchlist: %v", result.Error))
		span.RecordError(result.Error)
		http.Error(w, "Failed to remove symbol from watchlist", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Symbol removed from watchlist"}); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "Symbol removed from watchlist successfully")
	span.AddEvent("Response sent")
}
