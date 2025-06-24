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
	"go.opentelemetry.io/otel/propagation"
	_ "go.opentelemetry.io/otel/trace"
)

const apiKey = "26AMBY8WA3V0FCMD"

func getAllStockSymbols(w http.ResponseWriter, r *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(ctx, "getAllStockSymbolsHandler")
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
		bodyBytes, _ := io.ReadAll(response.Body)
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
	// Log handler entry
	Logger.InfoContext(ctx, "Handler execution started", "method", r.Method, "target", r.URL.Path)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	symbol := mux.Vars(r)["symbol"]
	if symbol == "" {
		span.SetStatus(codes.Error, "Missing stock symbol in request")
		Logger.ErrorContext(ctx, "Missing stock symbol in request path", "path", r.URL.Path)
		http.Error(w, "Stock symbol is required", http.StatusBadRequest)
		return
	}
	Logger.InfoContext(ctx, "Retrieving stock data for symbol", "symbol", symbol)

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
		attribute.String("stock_symbol", symbol),
	)
	defer apiCallSpan.End()

	apiCallStartTime := time.Now()
	response, err := http.Get(apiUrl)
	apiCallDuration := time.Since(apiCallStartTime).Seconds()

	// Log external API call completion
	Logger.InfoContext(ctx, "External API call completed", "url", apiUrl, "duration_sec", apiCallDuration, "error_present", err != nil)

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
		// Log error for HTTP GET
		Logger.ErrorContext(ctx, "HTTP GET to AlphaVantage for stock data failed", "error", err, "api_url", apiUrl, "symbol", symbol)

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
		Logger.ErrorContext(ctx, "AlphaVantage returned non-OK status for stock data",
			"status_code", response.StatusCode,
			"response_body", string(bodyBytes),
			"api_url", apiUrl,
			"symbol", symbol)
		http.Error(w, errorMsg, response.StatusCode)
		return
	}
	apiCallSpan.SetStatus(codes.Ok, "API call successful")
	Logger.InfoContext(ctx, "AlphaVantage API call successful for stock data", "api_url", apiUrl, "status_code", response.StatusCode, "symbol", symbol)

	_, responseSpan := tracer.Start(ctx, "processAPIresponse")
	defer responseSpan.End()
	responseSpan.AddEvent("Started decoding JSON")
	Logger.InfoContext(ctx, "Starting JSON decoding for stock data response", "symbol", symbol)

	var stockData map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&stockData); err != nil {
		responseSpan.SetStatus(codes.Error, fmt.Sprintf("JSON read error: %v", err))
		responseSpan.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("JSON decoding failed: %v", err))
		span.RecordError(err)
		// Log error for JSON decoding
		Logger.ErrorContext(ctx, "Failed to decode JSON response for stock data", "error", err, "symbol", symbol)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	Logger.InfoContext(ctx, "JSON decoding complete for stock data response", "symbol", symbol)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(stockData)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		// Log error for JSON encoding
		Logger.ErrorContext(ctx, "Failed to encode JSON response for stock data", "error", err, "symbol", symbol)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		span.SetStatus(codes.Ok, "Stock Data retrieved successfully")
		span.AddEvent("Response sent")
		Logger.InfoContext(ctx, "Stock data retrieved and response sent", "symbol", symbol)
	}
}

type coinData struct {
	Symbol string
	Id     string
}

func getAllCryptoSymbols(w http.ResponseWriter, r *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(ctx, "getAllCryptoSymbolsHandler")
	defer span.End()

	r = r.WithContext(ctx)

	// Log handler entry
	Logger.InfoContext(ctx, "Handler execution started", "method", r.Method, "target", r.URL.Path)

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

	// Log external API call completion
	Logger.InfoContext(ctx, "External API call completed", "url", apiUrl, "duration_sec", apiCallDuration, "error_present", err != nil)

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

		// Log error for HTTP GET
		Logger.ErrorContext(ctx, "HTTP GET to Coingecko for crypto symbols failed", "error", err, "api_url", apiUrl)
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
		// Log error for non-OK status
		Logger.ErrorContext(ctx, "Coingecko returned non-OK status for crypto symbols",
			"status_code", response.StatusCode,
			"response_body", string(bodyBytes),
			"api_url", apiUrl)
		http.Error(w, errorMsg, response.StatusCode)
		return
	}
	apiCallSpan.SetStatus(codes.Ok, "API call successful")
	Logger.InfoContext(ctx, "Coingecko API call successful for crypto symbols", "api_url", apiUrl, "status_code", response.StatusCode)

	var cryptoData []map[string]interface{}
	_, readJSONparser := tracer.Start(ctx, "processJSONresponse")
	defer readJSONparser.End()
	readJSONparser.AddEvent("Starting JSON response parsing")
	Logger.InfoContext(ctx, "Starting JSON decoding for crypto symbols response")

	if err := json.NewDecoder(response.Body).Decode(&cryptoData); err != nil {
		readJSONparser.SetStatus(codes.Error, fmt.Sprintf("JSON read error: %v", err))
		readJSONparser.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("crypto coins processing failed: %v", err))
		span.RecordError(err)
		// Log error for JSON decoding
		Logger.ErrorContext(ctx, "Failed to decode JSON response for crypto symbols", "error", err, "api_url", apiUrl)
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}
	Logger.InfoContext(ctx, "JSON decoding complete for crypto symbols response")

	var symbols []coinData
	for _, coin := range cryptoData {
		if symbol, ok := coin["symbol"].(string); ok {
			if id, ok := coin["id"].(string); ok {
				symbols = append(symbols, coinData{Symbol: symbol, Id: id})
			} else {
				Logger.WarnContext(ctx, "Coin data missing 'id' field, skipping coin", "coin_symbol", symbol)
			}
		} else {
			Logger.WarnContext(ctx, "Coin data missing 'symbol' field, skipping record")
		}
	}
	Logger.InfoContext(ctx, "Processed crypto data", "total_coins", len(cryptoData), "valid_symbols_extracted", len(symbols))

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(symbols)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		// Log error for JSON encoding
		Logger.ErrorContext(ctx, "Failed to encode JSON response for crypto symbols", "error", err, "symbols_count", len(symbols))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		span.SetStatus(codes.Ok, "Crypto symbols retrieved successfully")
		span.AddEvent("Response sent")
		Logger.InfoContext(ctx, "Crypto symbols retrieved and response sent", "count", len(symbols))
	}

}

func getCryptoData(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getCoinDataFromSymbol")
	defer span.End()

	r = r.WithContext(ctx)

	// Log handler entry
	Logger.InfoContext(ctx, "Handler execution started", "method", r.Method, "target", r.URL.Path)

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
	if symbol == "" {
		span.SetStatus(codes.Error, "Missing crypto symbol in request")
		Logger.ErrorContext(ctx, "Missing crypto symbol in request path", "path", r.URL.Path)
		http.Error(w, "Crypto symbol is required", http.StatusBadRequest)
		return
	}
	Logger.InfoContext(ctx, "Retrieving crypto data for symbol", "symbol", symbol)

	apiUrl := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h&ids=%s", symbol)

	_, apiCallSpan := tracer.Start(ctx, "coingecko.COIN_DATA")
	apiCallSpan.SetAttributes(
		attribute.String("http.url", apiUrl),
		attribute.String("http.method", "GET"),
		attribute.String("api.name", "coingecko"),
		attribute.String("api.operation", "COIN_DATA"),
		attribute.String("crypto_symbol", symbol),
	)
	defer apiCallSpan.End()

	apiCallStartTime := time.Now()
	response, err := http.Get(apiUrl)
	apiCallDuration := time.Since(apiCallStartTime).Seconds()

	// Log external API call completion
	Logger.InfoContext(ctx, "External API call completed", "url", apiUrl, "duration_sec", apiCallDuration, "error_present", err != nil)

	externalAPICallDuration.Record(ctx, apiCallDuration, metric.WithAttributes(
		attribute.String("api.name", "coingecko_api"),
		attribute.String("api.operation", "LISTED_COINS"), // This might be COIN_DATA, depending on what metric name makes sense
		attribute.Bool("api.error", err != nil),
	))

	if err != nil {
		apiCallSpan.SetStatus(codes.Error, fmt.Sprintf("HTTP GET failed: %v", err))
		apiCallSpan.RecordError(err)

		span.SetStatus(codes.Error, fmt.Sprintf("External API call failed: %v", err))
		span.RecordError(err)

		// Log error for HTTP GET
		Logger.ErrorContext(ctx, "HTTP GET to Coingecko for crypto data failed", "error", err, "api_url", apiUrl, "symbol", symbol)
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
		// Log error for non-OK status
		Logger.ErrorContext(ctx, "Coingecko returned non-OK status for crypto data",
			"status_code", response.StatusCode,
			"response_body", string(bodyBytes),
			"api_url", apiUrl,
			"symbol", symbol)
		http.Error(w, errorMsg, response.StatusCode)
		return
	}
	apiCallSpan.SetStatus(codes.Ok, "API call successful")
	Logger.InfoContext(ctx, "Coingecko API call successful for crypto data", "api_url", apiUrl, "status_code", response.StatusCode, "symbol", symbol)

	_, responseSpan := tracer.Start(ctx, "processAPIresponse")
	defer responseSpan.End()
	responseSpan.AddEvent("Started decoding JSON")
	Logger.InfoContext(ctx, "Starting JSON decoding for crypto data response", "symbol", symbol)

	var cryptoData []map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&cryptoData); err != nil {
		responseSpan.SetStatus(codes.Error, fmt.Sprintf("JSON read error: %v", err))
		responseSpan.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("JSON decoding failed: %v", err))
		span.RecordError(err)
		// Log error for JSON decoding
		Logger.ErrorContext(ctx, "Failed to decode JSON response for crypto data", "error", err, "symbol", symbol)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	Logger.InfoContext(ctx, "JSON decoding complete for crypto data response", "symbol", symbol, "data_items_count", len(cryptoData))

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(cryptoData)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		// Log error for JSON encoding
		Logger.ErrorContext(ctx, "Failed to encode JSON response for crypto data", "error", err, "symbol", symbol)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		span.SetStatus(codes.Ok, "Crypto data retrieved successfully")
		span.AddEvent("Response sent")
		Logger.InfoContext(ctx, "Crypto data retrieved and response sent", "symbol", symbol)
	}
}

func addToWatchlist(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "addToWatchList")
	defer span.End()

	r = r.WithContext(ctx)

	// Log handler entry
	Logger.InfoContext(ctx, "Handler execution started", "method", r.Method, "target", r.URL.Path)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.target", r.URL.Path),
	)
	span.AddEvent("Handler execution started")

	httpRequestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/add"),
		attribute.String("method", r.Method),
	))

	var data struct {
		Symbol   string `json:"symbol"`
		UserId   string `json:"userId"`
		Type     string `json:"type"`
		CryptoId string `json:"cryptoId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to decode request body: %v", err))
		span.RecordError(err)
		// Log error for decoding request body
		Logger.ErrorContext(ctx, "Failed to decode request body for add to watchlist", "error", err)
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}
	Logger.InfoContext(ctx, "Request body decoded for watchlist add", "symbol", data.Symbol, "type", data.Type, "userId", data.UserId)

	_, dbCallSpan := tracer.Start(ctx, "db_call_addToList")
	dbCallSpan.SetAttributes(
		attribute.String("http.method", "GET"), // This should probably be "POST" or "INSERT" for database ops
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("user_id", data.UserId),
		attribute.String("symbol_to_add", data.Symbol),
		attribute.String("item_type", data.Type),
	)
	defer dbCallSpan.End()

	var err error
	var dbCallDuration float64
	startTime := time.Now()

	// Assuming UserSymbols and DB are defined elsewhere in main package
	if data.Type == "STOCK" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "STOCK", CryptoId: ""}
		Logger.InfoContext(ctx, "Attempting to add stock to watchlist", "symbol", data.Symbol, "userId", data.UserId)
		err = DB.Create(&new_symbol).Error
		dbCallDuration = time.Since(startTime).Seconds()
	} else if data.Type == "CRYPTO" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "CRYPTO", CryptoId: data.CryptoId}
		Logger.InfoContext(ctx, "Attempting to add crypto to watchlist", "symbol", data.Symbol, "cryptoId", data.CryptoId, "userId", data.UserId)
		err = DB.Create(&new_symbol).Error
		dbCallDuration = time.Since(startTime).Seconds()
	} else {
		span.SetStatus(codes.Error, fmt.Sprintf("Invalid item type for watchlist: %s", data.Type))
		Logger.ErrorContext(ctx, "Invalid item type for watchlist", "type", data.Type)
		http.Error(w, "Invalid item type", http.StatusBadRequest)
		return
	}

	var status string
	if err != nil {
		status = "failure"
		dbCallSpan.SetStatus(codes.Error, fmt.Sprintf("Database insert failed: %v", err))
		dbCallSpan.RecordError(err)
		Logger.ErrorContext(ctx, "Failed to add item to watchlist in database", "error", err, "symbol", data.Symbol, "userId", data.UserId, "type", data.Type)

	} else {
		status = "success"
		dbCallSpan.SetStatus(codes.Ok, "Database insert successful")
		Logger.InfoContext(ctx, "Successfully added item to watchlist in database", "symbol", data.Symbol, "userId", data.UserId, "type", data.Type)
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

	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to create watch list item: %v", err))
		span.RecordError(err)
		http.Error(w, "Failed to create watchlist item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		// Log error for JSON encoding
		Logger.ErrorContext(ctx, "Failed to encode JSON response for add to watchlist", "error", err, "symbol", data.Symbol, "userId", data.UserId)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		span.SetStatus(codes.Ok, "Watch list item added successfully")
		span.AddEvent("Response sent")
		Logger.InfoContext(ctx, "Watchlist item added and response sent", "symbol", data.Symbol, "userId", data.UserId)
	}
}

func getWatchlist(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "getWatchlistHandler")
	defer span.End()

	r = r.WithContext(ctx)

	// Log handler entry
	Logger.InfoContext(ctx, "Handler execution started", "method", r.Method, "target", r.URL.Path)

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
	if userId == "" {
		span.SetStatus(codes.Error, "Missing userId in request for watchlist retrieval")
		Logger.ErrorContext(ctx, "Missing userId in request path for watchlist", "path", r.URL.Path)
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	Logger.InfoContext(ctx, "Retrieving watchlist for user", "userId", userId)

	var watchlist []UserSymbols

	startTime := time.Now()
	_, dbCallSpan := tracer.Start(ctx, "db_call_getWatchlist")
	dbCallSpan.SetAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("user_id", userId),
	)
	// Assuming DB is defined and connected
	err := DB.Where("user_id = ?", userId).Find(&watchlist).Error
	dbCallDuration := time.Since(startTime).Seconds()
	dbCallSpan.End()

	var status string
	if err != nil {
		status = "failure"
		dbCallSpan.SetStatus(codes.Error, fmt.Sprintf("Database select failed: %v", err))
		dbCallSpan.RecordError(err)
		Logger.ErrorContext(ctx, "Failed to retrieve watchlist from database", "error", err, "userId", userId)
	} else {
		status = "success"
		dbCallSpan.SetStatus(codes.Ok, "Database select successful")
		Logger.InfoContext(ctx, "Successfully retrieved watchlist from database", "userId", userId, "items_count", len(watchlist))
	}

	dbQueryCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", "/watchlist/{userId}"),
		attribute.String("status", status),
	))
	dbQueryDuration.Record(ctx, dbCallDuration, metric.WithAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "SELECT"),
		attribute.Bool("db.error", err != nil), // Reflect actual error status
	))

	if err != nil { // Re-check err for HTTP response
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to retrieve watchlist: %v", err))
		span.RecordError(err)
		http.Error(w, "Failed to retrieve watchlist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(watchlist); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		// Log error for JSON encoding
		Logger.ErrorContext(ctx, "Failed to encode JSON response for watchlist", "error", err, "userId", userId, "watchlist_items_count", len(watchlist))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "Watchlist retrieved successfully")
	span.AddEvent("Response sent")
	Logger.InfoContext(ctx, "Watchlist retrieved and response sent", "userId", userId, "count", len(watchlist))
}

func removeFromWatchlist(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("stock-tracker-app-tracer")
	ctx, span := tracer.Start(r.Context(), "removeFromWatchlistHandler")
	defer span.End()

	r = r.WithContext(ctx)

	// Log handler entry
	Logger.InfoContext(ctx, "Handler execution started", "method", r.Method, "target", r.URL.Path)

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

	if userId == "" || symbol == "" {
		span.SetStatus(codes.Error, "Missing userId or symbol in request for watchlist removal")
		Logger.ErrorContext(ctx, "Missing parameters for watchlist removal", "userId", userId, "symbol", symbol, "path", r.URL.Path)
		http.Error(w, "User ID and Symbol are required", http.StatusBadRequest)
		return
	}
	Logger.InfoContext(ctx, "Attempting to remove item from watchlist", "userId", userId, "symbol", symbol)

	startTime := time.Now()
	_, dbCallSpan := tracer.Start(ctx, "db_call_removeFromWatchlist")
	dbCallSpan.SetAttributes(
		attribute.String("db.table", "UserSymbols"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("user_id", userId),
		attribute.String("symbol_to_remove", symbol),
	)
	// Assuming DB is defined and connected
	result := DB.Where("user_id = ? AND symbol = ?", userId, symbol).Delete(&UserSymbols{})
	dbCallDuration := time.Since(startTime).Seconds()
	dbCallSpan.End()

	var status string
	if result.Error != nil {
		status = "failure"
		dbCallSpan.SetStatus(codes.Error, fmt.Sprintf("Database delete failed: %v", result.Error))
		dbCallSpan.RecordError(result.Error)
		Logger.ErrorContext(ctx, "Failed to remove item from watchlist in database", "error", result.Error, "userId", userId, "symbol", symbol)
	} else {
		status = "success"
		dbCallSpan.SetStatus(codes.Ok, "Database delete successful")
		Logger.InfoContext(ctx, "Successfully removed item from watchlist in database", "userId", userId, "symbol", symbol, "rows_affected", result.RowsAffected)
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

	if result.RowsAffected == 0 {
		Logger.WarnContext(ctx, "Attempted to remove non-existent item from watchlist", "userId", userId, "symbol", symbol)
		// Consider returning 404 Not Found if no rows were affected by a delete
		http.Error(w, "Symbol not found in watchlist or already removed", http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Symbol removed from watchlist"}); err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to encode JSON response: %v", err))
		span.RecordError(err)
		// Log error for JSON encoding
		Logger.ErrorContext(ctx, "Failed to encode JSON response for watchlist removal", "error", err, "userId", userId, "symbol", symbol)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "Symbol removed from watchlist successfully")
	span.AddEvent("Response sent")
	Logger.InfoContext(ctx, "Symbol removed from watchlist and response sent", "userId", userId, "symbol", symbol)
}
