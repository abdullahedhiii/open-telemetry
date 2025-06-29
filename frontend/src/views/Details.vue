<script setup>
import { tracer } from '../tracing.js'
import { ref, onMounted, computed } from 'vue'
import { context, propagation, trace } from '@opentelemetry/api'

const symbol = ref('')
const type = ref('')
const loading = ref(false)
const error = ref(null)
const detailsData = ref(null)
const isInWatchlist = ref(false)

// Computed properties for data type detection
const dataType = computed(() => {
  return type.value === 'stocks' ? 'stocks' : 'crypto'
})

const stockData = computed(() => {
  if (dataType.value === 'stocks' && detailsData.value && Array.isArray(detailsData.value)) {
    return detailsData.value[0] || null
  }
  return dataType.value === 'stocks' ? detailsData.value : null
})

const cryptoData = computed(() => {
  if (dataType.value === 'crypto' && detailsData.value && Array.isArray(detailsData.value)) {
    return detailsData.value[0] || null
  }
  return dataType.value === 'crypto' ? detailsData.value : null
})

// Computed properties for stock data
const currentPrice = computed(() => {
  if (!stockData.value?.timeSeries) return '0.00'
  const dates = Object.keys(stockData.value.timeSeries).sort().reverse()
  const latestDate = dates[0]
  return stockData.value.timeSeries[latestDate]?.close || '0.00'
})

const dayHigh = computed(() => {
  if (!stockData.value?.timeSeries) return '0.00'
  const dates = Object.keys(stockData.value.timeSeries).sort().reverse()
  const latestDate = dates[0]
  return stockData.value.timeSeries[latestDate]?.high || '0.00'
})

const dayLow = computed(() => {
  if (!stockData.value?.timeSeries) return '0.00'
  const dates = Object.keys(stockData.value.timeSeries).sort().reverse()
  const latestDate = dates[0]
  return stockData.value.timeSeries[latestDate]?.low || '0.00'
})

const currentVolume = computed(() => {
  if (!stockData.value?.timeSeries) return '0'
  const dates = Object.keys(stockData.value.timeSeries).sort().reverse()
  const latestDate = dates[0]
  return stockData.value.timeSeries[latestDate]?.volume || '0'
})

const limitedTimeSeriesData = computed(() => {
  if (!stockData.value?.timeSeries) return {}
  const entries = Object.entries(stockData.value.timeSeries)
  const sortedEntries = entries.sort(([a], [b]) => new Date(b) - new Date(a))
  const limitedEntries = sortedEntries.slice(0, 10) // Show last 10 days
  return Object.fromEntries(limitedEntries)
})

// Utility functions
function formatNumber(num) {
  if (!num) return '0'
  const number = parseFloat(num)
  if (number >= 1e9) {
    return (number / 1e9).toFixed(2) + 'B'
  } else if (number >= 1e6) {
    return (number / 1e6).toFixed(2) + 'M'
  } else if (number >= 1e3) {
    return (number / 1e3).toFixed(2) + 'K'
  }
  return number.toLocaleString()
}

function formatDate(dateString) {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

async function fetchData() {
  const span = tracer.startSpan('load_symbol_details', {
    attributes: {
      'symbol': symbol.value,
      'symbol.type': type.value,
      'operation.type': 'data_fetch',
      'component': 'details_page'
    }
  });

  try {
   
    loading.value = true;
    error.value = null;

    const detailsSpan = tracer.startSpan('fetch_symbol_data', { parent: span });
    
    const response = await fetch(`${import.meta.env.VITE_API_URL}/${type.value}/${symbol.value}`);

    if (!response.ok) {
      throw new Error(`Failed to fetch details: ${response.statusText}`);
    }

    const responseData = await response.json();
    detailsData.value = responseData;
    
    // Debug log to see the structure
    console.log('API Response:', responseData);
    console.log('Is Array:', Array.isArray(responseData));
    if (Array.isArray(responseData)) {
      console.log('First item:', responseData[0]);
    }
    
    detailsSpan.setStatus({ code: 1 });
    detailsSpan.end();

    // Check if symbol is in watchlist
    checkWatchlistStatus();
    
  } catch (err) {
    error.value = err.message;
    span.setStatus({ code: 2, message: err.message });
  } finally {
    loading.value = false;
    span.end();
  }
}

function checkWatchlistStatus() {
  try {
    const savedWatchlist = localStorage.getItem('userWatchlist')
    let watchlist = savedWatchlist ? JSON.parse(savedWatchlist) : []
    isInWatchlist.value = watchlist.some(item => item.symbol === symbol.value && item.type === type.value)
  } catch (err) {
    console.error('Error checking watchlist status:', err)
    isInWatchlist.value = false
  }
}

function toggleWatchlist() {
  const span = tracer.startSpan('toggle_watchlist_from_details', {
    attributes: {
      'symbol': symbol.value,
      'symbol.type': type.value,
      'current.in_watchlist': isInWatchlist.value,
      'user.action': isInWatchlist.value ? 'remove_from_watchlist' : 'add_to_watchlist'
    }
  })
  
  try {
    const savedWatchlist = localStorage.getItem('userWatchlist')
    let watchlist = savedWatchlist ? JSON.parse(savedWatchlist) : []
    
    if (isInWatchlist.value) {
      // Remove from watchlist
      watchlist = watchlist.filter(item => !(item.symbol === symbol.value && item.type === type.value))
      isInWatchlist.value = false
    } else {
      // Add to watchlist
      const itemName = type.value === 'crypto' 
        ? (cryptoData.value?.name || symbol.value)
        : (stockData.value?.metaData?.symbol || stockData.value?.name || symbol.value);
        
      watchlist.push({
        symbol: symbol.value,
        name: itemName,
        type: type.value,
        dateAdded: new Date().toISOString()
      })
      isInWatchlist.value = true
    }
    
    localStorage.setItem('userWatchlist', JSON.stringify(watchlist))
    
    span.setAttributes({
      'watchlist.new_status': isInWatchlist.value,
      'watchlist.total_items': watchlist.length,
      'operation.success': true
    })
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

onMounted(() => {
  const span = tracer.startSpan('details_page_mounted', {
    attributes: {
      'component': 'details_page',
      'lifecycle.event': 'mounted'
    }
  })
  
  // Get symbol and type from URL path or query params

    type.value = window.location.pathname.includes('/stocks') ? 'stocks' : 'crypto'
    symbol.value = window.location.pathname.split('/').pop() 
  
  
  span.setAttributes({
    'url.symbol': symbol.value,
    'url.type': type.value
  })
  
  // Load data on mount
  fetchData()
  
  span.setStatus({ code: 1 })
  span.end()
})
</script>

<template>
  <div class="app-container">
    <div class="container">
      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <div class="spinner"></div>
        <p>Loading data...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-container">
        <h2>Error Loading Data</h2>
        <p>{{ error }}</p>
        <button @click="fetchData" class="retry-btn">Retry</button>
      </div>

      <!-- Stock Data Display -->
      <div v-else-if="dataType === 'stocks' && stockData" class="content">
        <!-- Header -->
        <div class="header-card">
          <div class="header-content">
            <h1>{{ stockData.metaData?.symbol || symbol }}</h1>
            <p class="subtitle">{{ stockData.metaData?.information || 'Stock Information' }}</p>
            <button @click="toggleWatchlist" class="watchlist-btn" :class="{ active: isInWatchlist }">
              {{ isInWatchlist ? 'Remove from Watchlist' : 'Add to Watchlist' }}
            </button>
          </div>
          <div class="meta-grid">
            <div class="meta-item">
              <span class="label">Last Refreshed</span>
              <span class="value">{{ stockData.metaData?.lastRefreshed || 'N/A' }}</span>
            </div>
            <div class="meta-item">
              <span class="label">Output Size</span>
              <span class="value">{{ stockData.metaData?.outputSize || 'N/A' }}</span>
            </div>
            <div class="meta-item">
              <span class="label">Time Zone</span>
              <span class="value">{{ stockData.metaData?.timeZone || 'N/A' }}</span>
            </div>
            <div class="meta-item">
              <span class="label">Current Price</span>
              <span class="value price">${{ parseFloat(currentPrice).toFixed(2) }}</span>
            </div>
          </div>
        </div>

        <!-- Recent Performance -->
        <div class="performance-card">
          <h2>Recent Performance</h2>
          <div class="performance-grid">
            <div class="performance-item current">
              <span class="perf-label">Current Price</span>
              <span class="perf-value">${{ parseFloat(currentPrice).toFixed(2) }}</span>
            </div>
            <div class="performance-item high">
              <span class="perf-label">Day High</span>
              <span class="perf-value">${{ parseFloat(dayHigh).toFixed(2) }}</span>
            </div>
            <div class="performance-item low">
              <span class="perf-label">Day Low</span>
              <span class="perf-value">${{ parseFloat(dayLow).toFixed(2) }}</span>
            </div>
            <div class="performance-item volume">
              <span class="perf-label">Volume</span>
              <span class="perf-value">{{ formatNumber(currentVolume) }}</span>
            </div>
          </div>
        </div>

        <!-- Historical Data Table -->
        <div class="table-card">
          <h2>Historical Data</h2>
          <div class="table-container">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Open</th>
                  <th>High</th>
                  <th>Low</th>
                  <th>Close</th>
                  <th>Volume</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(data, date) in limitedTimeSeriesData" :key="date">
                  <td class="date-cell">{{ formatDate(date) }}</td>
                  <td class="price-cell">${{ parseFloat(data.open).toFixed(2) }}</td>
                  <td class="price-cell high-price">${{ parseFloat(data.high).toFixed(2) }}</td>
                  <td class="price-cell low-price">${{ parseFloat(data.low).toFixed(2) }}</td>
                  <td class="price-cell">${{ parseFloat(data.close).toFixed(2) }}</td>
                  <td class="volume-cell">{{ formatNumber(data.volume) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Crypto Data Display -->
      <div v-else-if="dataType === 'crypto' && cryptoData" class="content">
        <!-- Crypto Header -->
        <div class="header-card crypto-header">
          <div class="crypto-title">
            <img v-if="cryptoData.image" :src="cryptoData.image" :alt="cryptoData.name" class="crypto-icon">
            <div>
              <h1>{{ cryptoData.name || symbol }} {{ cryptoData.symbol ? `(${cryptoData.symbol.toUpperCase()})` : '' }}</h1>
              <p class="subtitle" v-if="cryptoData.market_cap_rank">Rank #{{ cryptoData.market_cap_rank }}</p>
            </div>
          </div>
          <div class="crypto-actions">
            <button @click="toggleWatchlist" class="watchlist-btn" :class="{ active: isInWatchlist }">
              {{ isInWatchlist ? 'Remove from Watchlist' : 'Add to Watchlist' }}
            </button>
          </div>
          <div class="crypto-price">
            <span class="current-price">${{ formatNumber(cryptoData.current_price) }}</span>
            <span v-if="cryptoData.price_change_percentage_24h !== undefined" 
                  :class="['price-change', cryptoData.price_change_percentage_24h >= 0 ? 'positive' : 'negative']">
              {{ cryptoData.price_change_percentage_24h >= 0 ? '+' : '' }}{{ cryptoData.price_change_percentage_24h.toFixed(2) }}%
            </span>
          </div>
        </div>

        <!-- Crypto Stats -->
        <div class="crypto-stats">
          <h2>Market Statistics</h2>
          <div class="stats-grid">
            <div class="stat-item" v-if="cryptoData.market_cap">
              <span class="stat-label">Market Cap</span>
              <span class="stat-value">${{ formatNumber(cryptoData.market_cap) }}</span>
            </div>
            <div class="stat-item" v-if="cryptoData.total_volume">
              <span class="stat-label">24h Volume</span>
              <span class="stat-value">${{ formatNumber(cryptoData.total_volume) }}</span>
            </div>
            <div class="stat-item" v-if="cryptoData.high_24h">
              <span class="stat-label">24h High</span>
              <span class="stat-value">${{ formatNumber(cryptoData.high_24h) }}</span>
            </div>
            <div class="stat-item" v-if="cryptoData.low_24h">
              <span class="stat-label">24h Low</span>
              <span class="stat-value">${{ formatNumber(cryptoData.low_24h) }}</span>
            </div>
            <div class="stat-item" v-if="cryptoData.ath">
              <span class="stat-label">All Time High</span>
              <span class="stat-value">${{ formatNumber(cryptoData.ath) }}</span>
            </div>
            <div class="stat-item" v-if="cryptoData.circulating_supply">
              <span class="stat-label">Circulating Supply</span>
              <span class="stat-value">{{ formatNumber(cryptoData.circulating_supply) }}</span>
            </div>
          </div>
        </div>

        <!-- Additional Crypto Info -->
        <div class="crypto-info">
          <h2>Additional Information</h2>
          <div class="info-grid">
            <div class="info-item" v-if="cryptoData.ath_date">
              <span class="info-label">All Time High Date</span>
              <span class="info-value">{{ formatDate(cryptoData.ath_date) }}</span>
            </div>
            <div class="info-item" v-if="cryptoData.atl">
              <span class="info-label">All Time Low</span>
              <span class="info-value">${{ formatNumber(cryptoData.atl) }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">Max Supply</span>
              <span class="info-value">{{ cryptoData.max_supply ? formatNumber(cryptoData.max_supply) : 'N/A' }}</span>
            </div>
            <div class="info-item" v-if="cryptoData.last_updated">
              <span class="info-label">Last Updated</span>
              <span class="info-value">{{ formatDate(cryptoData.last_updated) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- No Data State -->
      <div v-else-if="!loading && !error" class="no-data-container">
        <h2>No Data Available</h2>
        <p>Unable to load {{ dataType }} data for {{ symbol }}.</p>
        <button @click="fetchData" class="retry-btn">Try Again</button>
      </div>
    </div>
  </div>
</template>
<style scoped>
.app-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 400px;
  color: #4a5568;
}

.spinner {
  width: 50px;
  height: 50px;
  border: 4px solid #e2e8f0;
  border-top: 4px solid #3182ce;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error-container {
  background: #fed7d7;
  border: 1px solid #feb2b2;
  border-radius: 8px;
  padding: 40px;
  text-align: center;
  margin: 20px 0;
}

.error-container h2 {
  color: #c53030;
  margin-bottom: 10px;
}

.error-container p {
  color: #742a2a;
  margin-bottom: 20px;
}

.retry-btn {
  background: #e53e3e;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 5px;
  cursor: pointer;
  font-size: 14px;
}

.retry-btn:hover {
  background: #c53030;
}

.content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  padding: 30px;
  border-left: 4px solid #3182ce;
}

.header-card h1 {
  font-size: 2.5rem;
  font-weight: bold;
  color: #2d3748;
  margin-bottom: 8px;
}

.subtitle {
  color: #718096;
  font-size: 1.1rem;
  margin-bottom: 20px;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-top: 20px;
}

.meta-item {
  text-align: center;
  padding: 15px;
  background: #f7fafc;
  border-radius: 8px;
}

.label {
  display: block;
  font-size: 0.9rem;
  color: #718096;
  margin-bottom: 5px;
}

.value {
  display: block;
  font-weight: 600;
  color: #2d3748;
  font-size: 1.1rem;
}

.value.price {
  color: #38a169;
  font-size: 1.3rem;
}

.performance-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  padding: 30px;
}

.performance-card h2 {
  font-size: 1.5rem;
  font-weight: 600;
  color: #2d3748;
  margin-bottom: 20px;
}

.performance-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.performance-item {
  display: flex;
  flex-direction: column;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
}

.performance-item.current {
  background: #ebf8ff;
  border: 1px solid #bee3f8;
}

.performance-item.high {
  background: #f0fff4;
  border: 1px solid #c6f6d5;
}

.performance-item.low {
  background: #fff5f5;
  border: 1px solid #fed7d7;
}

.performance-item.volume {
  background: #f7fafc;
  border: 1px solid #e2e8f0;
}

.perf-label {
  font-size: 0.9rem;
  color: #718096;
  margin-bottom: 8px;
}

.perf-value {
  font-size: 1.3rem;
  font-weight: bold;
}

.performance-item.current .perf-value {
  color: #3182ce;
}

.performance-item.high .perf-value {
  color: #38a169;
}

.performance-item.low .perf-value {
  color: #e53e3e;
}

.performance-item.volume .perf-value {
  color: #4a5568;
}

.table-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  padding: 30px;
}

.table-card h2 {
  font-size: 1.5rem;
  font-weight: 600;
  color: #2d3748;
  margin-bottom: 20px;
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.data-table th {
  background: #f7fafc;
  padding: 12px;
  text-align: left;
  font-weight: 600;
  color: #4a5568;
  border-bottom: 2px solid #e2e8f0;
}

.data-table th:not(:first-child) {
  text-align: right;
}

.data-table td {
  padding: 12px;
  border-bottom: 1px solid #e2e8f0;
}

.data-table tr:hover {
  background: #f7fafc;
}

.date-cell {
  font-weight: 600;
  color: #2d3748;
}

.price-cell {
  text-align: right;
  font-weight: 500;
  color: #4a5568;
}

.high-price {
  color: #38a169;
}

.low-price {
  color: #e53e3e;
}

.volume-cell {
  text-align: right;
  color: #718096;
}

/* Crypto specific styles */
.crypto-header {
  border-left-color: #f6ad55;
}

.crypto-title {
  display: flex;
  align-items: center;
  gap: 15px;
  margin-bottom: 20px;
}

.crypto-icon {
  width: 50px;
  height: 50px;
  border-radius: 50%;
}

.crypto-price {
  display: flex;
  align-items: center;
  gap: 15px;
  flex-wrap: wrap;
}

.current-price {
  font-size: 2rem;
  font-weight: bold;
  color: #2d3748;
}

.price-change {
  font-size: 1.2rem;
  font-weight: 600;
  padding: 5px 12px;
  border-radius: 20px;
}

.price-change.positive {
  background: #c6f6d5;
  color: #2f855a;
}

.price-change.negative {
  background: #fed7d7;
  color: #c53030;
}

.crypto-stats {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  padding: 30px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  padding: 20px;
  background: #f7fafc;
  border-radius: 8px;
  text-align: center;
}

.stat-label {
  font-size: 0.9rem;
  color: #718096;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 1.2rem;
  font-weight: bold;
  color: #2d3748;
}

.crypto-info {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  padding: 30px;
}

.crypto-info h2 {
  font-size: 1.5rem;
  font-weight: 600;
  color: #2d3748;
  margin-bottom: 20px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  background: #f7fafc;
  border-radius: 8px;
}

.info-label {
  font-weight: 500;
  color: #4a5568;
}

.info-value {
  font-weight: 600;
  color: #2d3748;
}

@media (max-width: 768px) {
  .container {
    padding: 10px;
  }
  
  .header-card h1 {
    font-size: 2rem;
  }
  
  .meta-grid,
  .performance-grid,
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .current-price {
    font-size: 1.5rem;
  }
  
  .crypto-title {
    flex-direction: column;
    text-align: center;
  }
  
  .crypto-price {
    justify-content: center;
  }
  
  .info-item {
    flex-direction: column;
    text-align: center;
    gap: 10px;
  }
}
</style>