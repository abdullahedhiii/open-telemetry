<script setup>
import { tracer } from '../tracing.js'
import { ref } from 'vue'
import { context, propagation, trace } from '@opentelemetry/api'

const symbols = ref([])
const error = ref(null)
const loading = ref(false)
const watchlist = ref([])

async function fetcCoins() {
    error.value = null
    loading.value = true
    const span = tracer.startSpan('fetchCoinSymbols')
    const ctx = trace.setSpan(context.active(), span)

    try {
        const headers = {}
        propagation.inject(ctx, headers)
        headers['Content-Type'] = 'application/json'
        const apiUrl = import.meta.env.VITE_API_URL || ""
        span.setAttribute('api.url', apiUrl + '/crypto/symbols')

        const response = await fetch(`${apiUrl}/crypto/symbols`, {
            method: 'GET',
            headers: headers
        })
        span.setAttribute('http.status_code', response.status)

        if (!response.ok) throw new Error("Network response was not ok")
        const data = await response.json()
        
        // Parse the data - assuming it's an array of stock objects
        symbols.value = Array.isArray(data) ? data : data.symbols || []
        span.setStatus({ code: 1 })
    } catch (err) {
        error.value = err.message
        span.setStatus({ code: 2, message: err.message })
    } finally {
        loading.value = false
        span.end()
    }
}

function viewDetails(symbol) {
    // Handle view details functionality
    alert(`Viewing details for ${symbol}`)
    // You can replace this with navigation to a detail page or open a modal
}

function addToWatchlist(symbol) {
    if (!watchlist.value.includes(symbol)) {
        watchlist.value.push(symbol)
    }
}

function removeFromWatchlist(symbol) {
    const index = watchlist.value.indexOf(symbol)
    if (index > -1) {
        watchlist.value.splice(index, 1)
    }
}

function isInWatchlist(symbol) {
    return watchlist.value.includes(symbol)
}
</script>

<template>
    <div class="stocks-container">
        <div class="header">
            <h2>Stock Symbols</h2>
            <button 
                @click="fetchSymbols" 
                :disabled="loading"
                class="fetch-btn"
            >
                {{ loading ? 'Loading...' : 'Fetch Crypto Symbols' }}
            </button>
        </div>

        <div v-if="error" class="error-message">
            <strong>Error:</strong> {{ error }}
        </div>

        <div v-if="symbols.length > 0" class="table-container">
            <table class="stocks-table">
                <thead>
                    <tr>
                        <th>Symbol</th>
                        <th>View Details</th>
                        <th>Add to List</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="stock in symbols" :key="stock.Id ">
                        <td class="symbol-cell">
                            <div class="symbol-info">
                                <span class="symbol">{{ stock.Symbol }}</span>
                                <span v-if="stock.v" class="name">{{ stock.Name }}</span>
                            </div>
                        </td>
                        <td class="action-cell">
                            <button 
                                @click="viewDetails(stock.Id )"
                                class="view-btn"
                            >
                                View Details
                            </button>
                        </td>
                        <td class="action-cell">
                            <button 
                                v-if="!isInWatchlist(stock.Id )"
                                @click="addToWatchlist(stock.Id )"
                                class="add-btn"
                            >
                                Add to List
                            </button>
                            <button 
                                v-else
                                @click="removeFromWatchlist(stock.Id )"
                                class="remove-btn"
                            >
                                Remove
                            </button>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-if="watchlist.length > 0" class="watchlist-section">
            <h3>Your Watchlist ({{ watchlist.length }} items)</h3>
            <div class="watchlist-items">
                <span 
                    v-for="symbol in watchlist" 
                    :key="symbol" 
                    class="watchlist-item"
                >
                    {{ symbol }}
                </span>
            </div>
        </div>

        <div v-if="!loading && symbols.length === 0 && !error" class="empty-state">
            Click "Fetch Crypto Symbols" to load data
        </div>
    </div>
</template>

<style scoped>
.stocks-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 30px;
    padding-bottom: 20px;
    border-bottom: 2px solid #e0e0e0;
}

.header h2 {
    margin: 0;
    color: #333;
    font-size: 2rem;
}

.fetch-btn {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    border: none;
    padding: 12px 24px;
    border-radius: 8px;
    cursor: pointer;
    font-size: 16px;
    font-weight: 600;
    transition: all 0.3s ease;
    box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
}

.fetch-btn:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(102, 126, 234, 0.6);
}

.fetch-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
}

.error-message {
    background: #fee;
    color: #d00;
    padding: 15px;
    border-radius: 8px;
    border-left: 4px solid #d00;
    margin-bottom: 20px;
}

.table-container {
    background: white;
    border-radius: 12px;
    overflow: hidden;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1);
    margin-bottom: 30px;
}

.stocks-table {
    width: 100%;
    border-collapse: collapse;
}

.stocks-table thead {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
}

.stocks-table th {
    padding: 20px;
    text-align: left;
    font-weight: 600;
    font-size: 16px;
    letter-spacing: 0.5px;
}

.stocks-table tbody tr {
    border-bottom: 1px solid #f0f0f0;
    transition: background-color 0.2s ease;
}

.stocks-table tbody tr:hover {
    background-color: #f8f9ff;
}

.stocks-table tbody tr:last-child {
    border-bottom: none;
}

.stocks-table td {
    padding: 20px;
    vertical-align: middle;
}

.symbol-cell {
    font-weight: 600;
}

.symbol-info .symbol {
    display: block;
    font-size: 18px;
    color: #333;
    font-weight: 700;
}

.symbol-info .name {
    display: block;
    font-size: 14px;
    color: #666;
    margin-top: 4px;
    font-weight: 400;
}

.action-cell {
    text-align: center;
}

.view-btn, .add-btn, .remove-btn {
    padding: 10px 20px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 600;
    transition: all 0.3s ease;
    min-width: 120px;
}

.view-btn {
    background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
    color: white;
    box-shadow: 0 4px 15px rgba(79, 172, 254, 0.4);
}

.view-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(79, 172, 254, 0.6);
}

.add-btn {
    background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
    color: #333;
    box-shadow: 0 4px 15px rgba(168, 237, 234, 0.4);
}

.add-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(168, 237, 234, 0.6);
}

.remove-btn {
    background: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%);
    color: #333;
    box-shadow: 0 4px 15px rgba(255, 154, 158, 0.4);
}

.remove-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(255, 154, 158, 0.6);
}

.watchlist-section {
    background: white;
    padding: 25px;
    border-radius: 12px;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1);
}

.watchlist-section h3 {
    margin: 0 0 20px 0;
    color: #333;
    font-size: 1.5rem;
}

.watchlist-items {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
}

.watchlist-item {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 8px 16px;
    border-radius: 20px;
    font-size: 14px;
    font-weight: 600;
}

.empty-state {
    text-align: center;
    padding: 60px 20px;
    color: #666;
    font-size: 18px;
    background: white;
    border-radius: 12px;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1);
}

/* Responsive Design */
@media (max-width: 768px) {
    .header {
        flex-direction: column;
        gap: 15px;
        text-align: center;
    }
    
    .stocks-table {
        font-size: 14px;
    }
    
    .stocks-table th,
    .stocks-table td {
        padding: 12px 8px;
    }
    
    .view-btn, .add-btn, .remove-btn {
        min-width: 90px;
        padding: 8px 12px;
        font-size: 13px;
    }
    
    .symbol-info .symbol {
        font-size: 16px;
    }
}
</style>