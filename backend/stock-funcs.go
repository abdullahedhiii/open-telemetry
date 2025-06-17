package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const apiKey = "26AMBY8WA3V0FCMD"

// gets all the symbols from the alphavantage api ,sends back only the symbols that are active
func getAllStockSymbols(w http.ResponseWriter, r *http.Request) {
	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=LISTING_STATUS&apikey=%s", apiKey)
	fmt.Println("API URL request sent")
	response, err := http.Get(apiUrl)
	fmt.Println("API response", response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	reader := csv.NewReader(response.Body)
	reader.Read()
	var symbols []string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if strings.TrimSpace(record[6]) == "Active" {
			symbols = append(symbols, record[0])
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(symbols)

}

func getStockData(w http.ResponseWriter, r *http.Request) {
	symbol := mux.Vars(r)["symbol"]
	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=compact&apikey=%s", symbol, apiKey)
	fmt.Println("API URL request sent", symbol, apiKey)
	response, err := http.Get(apiUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	//send the data of stock as json
	var stockData map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&stockData)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stockData)
}

type coinData struct {
	Symbol string
	Id     string
}

func getAllCryptoSymbols(w http.ResponseWriter, r *http.Request) {
	apiUrl := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=24h"
	response, err := http.Get(apiUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var cryptoData []map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&cryptoData)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}

	var symbols []coinData
	for _, coin := range cryptoData {
		symbols = append(symbols, coinData{Symbol: coin["symbol"].(string), Id: coin["id"].(string)})
	}

	w.Header().Set("Content-Type", "application/json")
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
	err = json.NewDecoder(response.Body).Decode(&cryptoData)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(cryptoData)
}

func addToWatchlist(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Adding to watchlist")

	var data struct {
		Symbol   string `json:"symbol"`
		UserId   string `json:"userId"`
		Type     string `json:"type"`
		CryptoId string `json:"cryptoId"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	fmt.Println("Data decoded", data)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusInternalServerError)
		return
	}
	if data.Type == "STOCK" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "STOCK", CryptoId: ""}
		DB.Create(&new_symbol)
	} else if data.Type == "CRYPTO" {
		new_symbol := UserSymbols{Symbol: data.Symbol, UserId: data.UserId, Type: "CRYPTO", CryptoId: data.CryptoId}
		DB.Create(&new_symbol)
	}
	fmt.Println("Data added to watchlist", data)
	json.NewEncoder(w).Encode(data)
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
