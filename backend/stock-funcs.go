package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const apiKey = "26AMBY8WA3V0FCMD"

// gets all the symbols from the alphavantage api ,sends back only the symbols that are active
func getAllStockSymbols(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := startSpan(r.Context(), "getAllStockSymbols")
	defer span.End()

	span.SetAttributes(
		attribute.String("api.name", "alphavantage"),
		attribute.String("request.type", "stock_symbols"),
	)

	// Child span for API call
	_, apiSpan := startChildSpan(ctx, "alphavantage.listing_status")
	defer apiSpan.End()

	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=LISTING_STATUS&apikey=%s", apiKey)
	apiSpan.AddEvent("making API request", trace.WithAttributes(
		attribute.String("url", apiUrl),
	))

	response, err := http.Get(apiUrl)
	if err != nil {
		apiSpan.SetStatus(codes.Error, "failed to fetch stock symbols")
		apiSpan.RecordError(err)
		recordError(ctx, "getAllStockSymbols", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		recordApiRequest(ctx, "/stocks/symbols", time.Since(start), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	// Child span for CSV processing
	_, csvSpan := startChildSpan(ctx, "process_csv_response")
	defer csvSpan.End()

	reader := csv.NewReader(response.Body)
	reader.Read() // Skip header
	var symbols []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			csvSpan.RecordError(err)
			recordError(ctx, "getAllStockSymbols_csvRead", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			recordApiRequest(ctx, "/stocks/symbols", time.Since(start), http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(record[6]) == "Active" {
			symbols = append(symbols, record[0])
		}
	}

	csvSpan.SetAttributes(attribute.Int("symbols.count", len(symbols)))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(symbols); err != nil {
		span.RecordError(err)
		recordError(ctx, "getAllStockSymbols_jsonEncode", err)
		recordApiRequest(ctx, "/stocks/symbols", time.Since(start), http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	recordApiRequest(ctx, "/stocks/symbols", duration, http.StatusOK)

	span.AddEvent("request completed", trace.WithAttributes(
		attribute.Int("response.symbols", len(symbols)),
		attribute.Int64("response.time_ms", duration.Milliseconds()),
	))
}

func getStockData(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := startSpan(r.Context(), "getStockData")
	defer span.End()

	symbol := mux.Vars(r)["symbol"]
	span.SetAttributes(
		attribute.String("stock.symbol", symbol),
		attribute.String("api.name", "alphavantage"),
	)

	// Child span for API call
	_, apiSpan := startChildSpan(ctx, "alphavantage.time_series")
	defer apiSpan.End()

	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=compact&apikey=%s", symbol, apiKey)
	apiSpan.AddEvent("making API request", trace.WithAttributes(
		attribute.String("url", apiUrl),
	))

	response, err := http.Get(apiUrl)
	if err != nil {
		apiSpan.SetStatus(codes.Error, "failed to fetch stock data")
		apiSpan.RecordError(err)
		recordError(ctx, "getStockData", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		recordApiRequest(ctx, "/stocks/data", time.Since(start), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var stockData map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&stockData); err != nil {
		apiSpan.RecordError(err)
		recordError(ctx, "getStockData_jsonDecode", err)
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		recordApiRequest(ctx, "/stocks/data", time.Since(start), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stockData); err != nil {
		span.RecordError(err)
		recordError(ctx, "getStockData_jsonEncode", err)
		recordApiRequest(ctx, "/stocks/data", time.Since(start), http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	recordApiRequest(ctx, "/stocks/data", duration, http.StatusOK)

	span.AddEvent("request completed", trace.WithAttributes(
		attribute.String("symbol", symbol),
		attribute.Int64("response.time_ms", duration.Milliseconds()),
	))
}

type coinData struct {
	Symbol string
	Id     string
}

func getAllCryptoSymbols(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := startSpan(r.Context(), "getAllCryptoSymbols")
	defer span.End()

	span.SetAttributes(
		attribute.String("api.name", "coingecko"),
		attribute.String("request.type", "crypto_symbols"),
	)

	// Child span for API call
	_, apiSpan := startChildSpan(ctx, "coingecko.markets")
	defer apiSpan.End()

	apiUrl := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h"
	apiSpan.AddEvent("making API request", trace.WithAttributes(
		attribute.String("url", apiUrl),
	))

	response, err := http.Get(apiUrl)
	if err != nil {
		apiSpan.SetStatus(codes.Error, "failed to fetch crypto symbols")
		apiSpan.RecordError(err)
		recordError(ctx, "getAllCryptoSymbols", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		recordApiRequest(ctx, "/crypto/symbols", time.Since(start), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var cryptoData []map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&cryptoData); err != nil {
		apiSpan.RecordError(err)
		recordError(ctx, "getAllCryptoSymbols_jsonDecode", err)
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		recordApiRequest(ctx, "/crypto/symbols", time.Since(start), http.StatusInternalServerError)
		return
	}

	var symbols []coinData
	for _, coin := range cryptoData {
		symbols = append(symbols, coinData{Symbol: coin["symbol"].(string), Id: coin["id"].(string)})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(symbols); err != nil {
		span.RecordError(err)
		recordError(ctx, "getAllCryptoSymbols_jsonEncode", err)
		recordApiRequest(ctx, "/crypto/symbols", time.Since(start), http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	recordApiRequest(ctx, "/crypto/symbols", duration, http.StatusOK)

	span.AddEvent("request completed", trace.WithAttributes(
		attribute.Int("response.symbols", len(symbols)),
		attribute.Int64("response.time_ms", duration.Milliseconds()),
	))
}

func getCryptoData(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := startSpan(r.Context(), "getCryptoData")
	defer span.End()

	symbol := mux.Vars(r)["symbol"]
	span.SetAttributes(
		attribute.String("crypto.symbol", symbol),
		attribute.String("api.name", "coingecko"),
	)

	// Child span for API call
	_, apiSpan := startChildSpan(ctx, "coingecko.markets")
	defer apiSpan.End()

	apiUrl := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h&ids=%s", symbol)
	apiSpan.AddEvent("making API request", trace.WithAttributes(
		attribute.String("url", apiUrl),
	))

	response, err := http.Get(apiUrl)
	if err != nil {
		apiSpan.SetStatus(codes.Error, "failed to fetch crypto data")
		apiSpan.RecordError(err)
		recordError(ctx, "getCryptoData", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		recordApiRequest(ctx, "/crypto/data", time.Since(start), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var cryptoData []map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&cryptoData); err != nil {
		apiSpan.RecordError(err)
		recordError(ctx, "getCryptoData_jsonDecode", err)
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		recordApiRequest(ctx, "/crypto/data", time.Since(start), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cryptoData); err != nil {
		span.RecordError(err)
		recordError(ctx, "getCryptoData_jsonEncode", err)
		recordApiRequest(ctx, "/crypto/data", time.Since(start), http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	recordApiRequest(ctx, "/crypto/data", duration, http.StatusOK)

	span.AddEvent("request completed", trace.WithAttributes(
		attribute.String("symbol", symbol),
		attribute.Int64("response.time_ms", duration.Milliseconds()),
	))
}

func randomCheck() {
	fmt.Println("HELLOO")
}
func addToWatchlist(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := startSpan(r.Context(), "addToWatchlist")
	defer span.End()

	// Child span for request parsing
	_, parseSpan := startChildSpan(ctx, "parse_request")
	defer parseSpan.End()

	var data struct {
		Symbol   string `json:"symbol"`
		UserId   string `json:"userId"`
		Type     string `json:"type"`
		CryptoId string `json:"cryptoId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		parseSpan.RecordError(err)
		recordError(ctx, "addToWatchlist_decode", err)
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		recordApiRequest(ctx, "/watchlist/add", time.Since(start), http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("symbol", data.Symbol),
		attribute.String("userId", data.UserId),
		attribute.String("type", data.Type),
	)

	// Child span for database operation
	dbCtx, dbSpan := instrumentDBCall(ctx, "create_watchlist_item")
	defer dbSpan.End()

	dbStart := time.Now()
	var err error
	if data.Type == "STOCK" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "STOCK", CryptoId: ""}
		err = DB.WithContext(dbCtx).Create(&new_symbol).Error
	} else if data.Type == "CRYPTO" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "CRYPTO", CryptoId: data.CryptoId}
		err = DB.WithContext(dbCtx).Create(&new_symbol).Error
	}

	if err != nil {
		dbSpan.RecordError(err)
		recordError(ctx, "addToWatchlist_dbCreate", err)
		http.Error(w, "Failed to create watchlist item", http.StatusInternalServerError)
		recordApiRequest(ctx, "/watchlist/add", time.Since(start), http.StatusInternalServerError)
		return
	}

	dbDuration := time.Since(dbStart)
	recordDBOperation(ctx, "create_watchlist_item", dbDuration, nil)
	watchlistAddCounter.Add(ctx, 1)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		span.RecordError(err)
		recordError(ctx, "addToWatchlist_jsonEncode", err)
		return
	}

	duration := time.Since(start)
	recordApiRequest(ctx, "/watchlist/add", duration, http.StatusOK)

	span.AddEvent("request completed", trace.WithAttributes(
		attribute.String("symbol", data.Symbol),
		attribute.String("type", data.Type),
		attribute.Int64("response.time_ms", duration.Milliseconds()),
	))
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

// func getPortfolioAnalysis(w http.ResponseWriter, r *http.Request) {
// 	userId := mux.Vars(r)["userId"]
// 	var watchlist []UserSymbols
// 	DB.Where("user_id = ?", userId).Find(&watchlist)
// 	// Fetches prices for all assets in the user's watchlist, computes portfolio metrics such as total value, percentage allocation per asset, and profit/loss if historical prices are stored. Makes multiple parallel API calls to fetch prices. Uses Alpha Vantage, Yahoo Finance, and CoinGecko. This endpoint is memory and CPU intensive if user has many assets.
// 	type portfolioAnalysis struct {
// 		Symbol               string
// 		Type                 string
// 		TotalValue           float64
// 		PercentageAllocation float64
// 		ProfitLoss           float64
// 	}

// 	allData := []portfolioAnalysis{}
// 	for _, symbol := range watchlist {
// 		if symbol.Type == "STOCK" {
// 			apiVantageUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=compact&apikey=%s", symbol.Symbol, apiKey)
// 			response, err := http.Get(apiVantageUrl)
// 			if err != nil {
// 				http.Error(w, err.Error(), http.StatusInternalServerError)
// 				return
// 			}
// 			defer response.Body.Close()
// 			var stockData map[string]interface{}
// 			err = json.NewDecoder(response.Body).Decode(&stockData)
// 			if err != nil {
// 				http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
// 				return
// 			}
// 			fmt.Println(stockData)

// 			// allData = append(allData, portfolioAnalysis{Symbol: symbol.Symbol, TotalValue: stockData["Meta Data"]["2. Symbol"].(string), PercentageAllocation: stockData["Meta Data"]["2. Symbol"].(string), ProfitLoss: stockData["Meta Data"]["2. Symbol"].(string)})
// 		} else if symbol.Type == "CRYPTO" {
// 			coinURL := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h&ids=%s", symbol.CryptoId)
// 			response, err := http.Get(coinURL)

// 			if err != nil {
// 				http.Error(w, err.Error(), http.StatusInternalServerError)
// 				return
// 			}
// 			defer response.Body.Close()
// 			var coinData []map[string]interface{}
// 			err = json.NewDecoder(response.Body).Decode(&coinData)
// 			if err != nil {
// 				http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
// 				return
// 			}
// 			fmt.Println(coinData)
// 			pp := portfolioAnalysis{Symbol: symbol.Symbol, Type: symbol.Type, TotalValue: coinData[0]["current_price"].(float64), PercentageAllocation: coinData[0]["price_change_percentage_24h"].(float64), ProfitLoss: coinData[0]["market_cap"].(float64)}
// 			allData = append(allData, pp)
// 			fmt.Println(coinData[0]["current_price"], coinData[0]["price_change_percentage_24h"], coinData[0]["market_cap_change_24h"])

// 		}
// 	}
// 	fmt.Println(allData)
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(allData)
// }
