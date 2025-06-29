<script setup>
import { tracer } from '../tracing.js'
import { ref, onMounted, computed } from 'vue'
import { context, propagation, trace } from '@opentelemetry/api'

const symbols = ref([])
const error = ref(null)
const loading = ref(false)
const watchlist = ref([])
const searchQuery = ref('')
const sortBy = ref('symbol')
const sortOrder = ref('asc')

const filteredAndSortedSymbols = computed(() => {
  const span = tracer.startSpan('filter_and_sort_stocks', {
    attributes: {
      'search.query': searchQuery.value,
      'sort.by': sortBy.value,
      'sort.order': sortOrder.value,
      'stocks.total_count': symbols.value.length
    }
  })
  
  try {
    let filtered = symbols.value
    
    // Filter by search query
    if (searchQuery.value.trim()) {
      filtered = symbols.value.filter(stock => 
        stock.Symbol?.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
        stock.Name?.toLowerCase().includes(searchQuery.value.toLowerCase())
      )
    }
    
    // Sort the results
    filtered.sort((a, b) => {
      let aValue = sortBy.value === 'symbol' ? a.Symbol : a.Name || ''
      let bValue = sortBy.value === 'symbol' ? b.Symbol : b.Name || ''
      
      aValue = aValue.toLowerCase()
      bValue = bValue.toLowerCase()
      
      if (sortOrder.value === 'asc') {
        return aValue.localeCompare(bValue)
      } else {
        return bValue.localeCompare(aValue)
      }
    })
    
    span.setAttributes({
      'stocks.filtered_count': filtered.length,
      'filter.has_results': filtered.length > 0,
      'operation.success': true
    })
    
    span.setStatus({ code: 1 })
    return filtered
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
    return symbols.value
  } finally {
    span.end()
  }
})

async function fetchSymbols() {
  const mainSpan = tracer.startSpan('fetchStockSymbols', {
    attributes: {
      'operation.type': 'data_fetch',
      'component': 'stock_dashboard',
      'user.action': 'fetch_symbols',
      'data.source': 'stocks_api'
    }
  })
  
  const ctx = trace.setSpan(context.active(), mainSpan)
  
  try {
    error.value = null
    loading.value = true
    
    // Performance tracking
    const startTime = performance.now()
    mainSpan.setAttribute('fetch.start_time', startTime)
    
    const headers = {}
    propagation.inject(ctx, headers)
    headers['Content-Type'] = 'application/json'
    
    const apiUrl = import.meta.env.VITE_API_URL || ""
    const fullUrl = `${apiUrl}/stocks/symbols`
    console.log(apiUrl)
    mainSpan.setAttributes({
      'api.url': fullUrl,
      'http.method': 'GET',
      'api.endpoint': '/stocks/symbols',
      'api.version': 'v1'
    })
    
    // Create child span for HTTP request
    const httpSpan = tracer.startSpan('http_request_stocks', {
      parent: ctx,
      attributes: {
        'http.url': fullUrl,
        'http.method': 'GET',
        'http.user_agent': navigator.userAgent
      }
    })
    
    const response = await fetch(fullUrl, {
      method: 'GET',
      headers: headers
    })
    
    const endTime = performance.now()
    const duration = endTime - startTime
    
    httpSpan.setAttributes({
      'http.status_code': response.status,
      'http.response_time_ms': duration,
      'http.response_size_bytes': response.headers.get('content-length') || 0
    })
    
    mainSpan.setAttributes({
      'http.status_code': response.status,
      'fetch.duration_ms': duration
    })
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }
    
    // Data processing span
    const processingSpan = tracer.startSpan('process_stock_data', {
      parent: ctx,
      attributes: {
        'processing.type': 'json_parse_and_validate'
      }
    })
    
    const data = await response.json()
    const processedSymbols = Array.isArray(data) ? data : data.symbols || []
    
    // Validate data structure
    const validSymbols = processedSymbols.filter(stock => stock.Symbol)
    
    processingSpan.setAttributes({
      'data.symbols_count': processedSymbols.length,
      'data.valid_symbols_count': validSymbols.length,
      'data.invalid_symbols_count': processedSymbols.length - validSymbols.length,
      'data.processing_time_ms': performance.now() - endTime
    })
    
    symbols.value = validSymbols
    
    mainSpan.setAttributes({
      'symbols.count': validSymbols.length,
      'operation.success': true,
      'data.quality_score': validSymbols.length / processedSymbols.length
    })
    
    httpSpan.setStatus({ code: 1 })
    processingSpan.setStatus({ code: 1 })
    mainSpan.setStatus({ code: 1 })
    
    httpSpan.end()
    processingSpan.end()
    
  } catch (err) {
    error.value = err.message
    
    mainSpan.setAttributes({
      'error.message': err.message,
      'error.type': err.constructor.name,
      'error.stack': err.stack,
      'operation.success': false
    })
    
    mainSpan.setStatus({ code: 2, message: err.message })
  } finally {
    loading.value = false
    mainSpan.setAttribute('fetch.end_time', performance.now())
    mainSpan.end()
  }
}

function viewDetails(symbol, name) {
  const span = tracer.startSpan('view_stock_details', {
    attributes: {
      'stock.symbol': symbol,
      'stock.name': name,
      'user.action': 'view_details',
      'interaction.type': 'button_click'
    }
  })
  
  try {
    span.setAttributes({
      'interaction.timestamp': Date.now(),
      'page.section': 'stock_table',
      'user.intent': 'view_stock_information'
    })
    
    // Enhanced user interaction - could be replaced with actual navigation
    window.location = '/details/stocks/' + symbol
    
    span.setAttributes({
      'operation.success': true,
      'modal.opened': true
    })
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setAttributes({
      'error.message': err.message,
      'operation.success': false
    })
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function addToWatchlist(symbol) {
  const span = tracer.startSpan('add_stock_to_watchlist', {
    attributes: {
      'stock.symbol': symbol,
      'user.action': 'add_watchlist',
      'watchlist.size_before': watchlist.value.length,
      'interaction.type': 'add_item'
    }
  })
  
  try {
    if (!watchlist.value.includes(symbol)) {
      watchlist.value.push(symbol)
      
      span.setAttributes({
        'watchlist.size_after': watchlist.value.length,
        'operation.success': true,
        'watchlist.action': 'added',
        'watchlist.growth': 1
      })
      
      // Track watchlist analytics
      const analyticsSpan = tracer.startSpan('watchlist_analytics', {
        parent: span,
        attributes: {
          'analytics.event': 'watchlist_item_added',
          'analytics.item_type': 'stock',
          'analytics.list_size': watchlist.value.length
        }
      })
      analyticsSpan.setStatus({ code: 1 })
      analyticsSpan.end()
      
    } else {
      span.setAttributes({
        'operation.success': false,
        'error.reason': 'symbol_already_exists',
        'watchlist.duplicate_attempt': true
      })
    }
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setAttributes({
      'error.message': err.message,
      'operation.success': false
    })
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function removeFromWatchlist(symbol) {
  const span = tracer.startSpan('remove_stock_from_watchlist', {
    attributes: {
      'stock.symbol': symbol,
      'user.action': 'remove_watchlist',
      'watchlist.size_before': watchlist.value.length,
      'interaction.type': 'remove_item'
    }
  })
  
  try {
    const index = watchlist.value.indexOf(symbol)
    if (index > -1) {
      watchlist.value.splice(index, 1)
      
      span.setAttributes({
        'watchlist.size_after': watchlist.value.length,
        'operation.success': true,
        'watchlist.action': 'removed',
        'watchlist.reduction': 1,
        'removed.index': index
      })
    } else {
      span.setAttributes({
        'operation.success': false,
        'error.reason': 'symbol_not_found'
      })
    }
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setAttributes({
      'error.message': err.message,
      'operation.success': false
    })
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function isInWatchlist(symbol) {
  return watchlist.value.includes(symbol)
}

function updateSort(field) {
  const span = tracer.startSpan('update_sort_order', {
    attributes: {
      'sort.field': field,
      'sort.previous_field': sortBy.value,
      'sort.previous_order': sortOrder.value
    }
  })
  
  try {
    if (sortBy.value === field) {
      sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortBy.value = field
      sortOrder.value = 'asc'
    }
    
    span.setAttributes({
      'sort.new_field': sortBy.value,
      'sort.new_order': sortOrder.value,
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
  const span = tracer.startSpan('stock_dashboard_mounted', {
    attributes: {
      'component': 'stock_dashboard',
      'lifecycle.event': 'mounted',
      'page.url': window.location.href,
      'user.agent': navigator.userAgent
    }
  })
  
  span.setStatus({ code: 1 })
  span.end()
})
</script>

<template>
  <div class="stock-dashboard">
    <div class="header-section">
      <div class="header-content">
        <div class="title-section">
          <h1 class="main-title">Stock Market Symbols</h1>
          <p class="subtitle">Track and monitor your favorite stock symbols</p>
        </div>
        
        <div class="controls-section">
          <div class="search-container">
            <input 
              v-model="searchQuery"
              type="text"
              placeholder="Search stocks..."
              class="search-input"
            />
          </div>
          
          <div class="sort-controls">
            <select v-model="sortBy" class="sort-select">
              <option value="symbol">Sort by Symbol</option>
              <option value="name">Sort by Name</option>
            </select>
            <button 
              @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'"
              class="sort-order-btn"
              :class="{ 'desc': sortOrder === 'desc' }"
            >
              {{ sortOrder === 'asc' ? '↑' : '↓' }}
            </button>
          </div>
          
          <button 
            @click="fetchSymbols" 
            :disabled="loading"
            class="fetch-button"
          >
            <span v-if="loading" class="loading-spinner"></span>
            {{ loading ? 'Loading...' : 'Fetch Stocks' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="error" class="error-card">
      <div class="error-icon">⚠️</div>
      <div class="error-content">
        <h3>Unable to fetch stock data</h3>
        <p>{{ error }}</p>
        <button @click="fetchSymbols" class="retry-button">Try Again</button>
      </div>
    </div>

    <div v-if="filteredAndSortedSymbols.length > 0" class="table-section">
      <div class="table-header">
        <h2>Stock Symbols</h2>
        <div class="table-stats">
          <span class="symbol-count">{{ filteredAndSortedSymbols.length }} of {{ symbols.length }} symbols</span>
          <span v-if="searchQuery" class="search-indicator">Filtered by: "{{ searchQuery }}"</span>
        </div>
      </div>
      
      <div class="table-container">
        <table class="stocks-table">
          <thead>
            <tr>
              <th @click="updateSort('symbol')" class="sortable-header">
                <div class="header-content">
                  Symbol
                  <span v-if="sortBy === 'symbol'" class="sort-indicator">
                    {{ sortOrder === 'asc' ? '↑' : '↓' }}
                  </span>
                </div>
              </th>
              <th @click="updateSort('name')" class="sortable-header">
                <div class="header-content">
                  Company Name
                  <span v-if="sortBy === 'name'" class="sort-indicator">
                    {{ sortOrder === 'asc' ? '↑' : '↓' }}
                  </span>
                </div>
              </th>
              <th class="actions-header">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="stock in filteredAndSortedSymbols" :key="stock.Symbol" class="stock-row">
              <td class="symbol-cell">
                <div class="symbol-badge">{{ stock.Symbol }}</div>
              </td>
              <td class="name-cell">
                <span class="company-name">{{ stock.Name || 'N/A' }}</span>
              </td>
              <td class="actions-cell">
                <div class="action-buttons">
                  <button 
                    @click="viewDetails(stock.Symbol, stock.Name)"
                    class="action-button view-button"
                    :title="`View details for ${stock.Symbol}`"
                  >
                    View Details
                  </button>
                  
                  <button 
                    v-if="!isInWatchlist(stock.Symbol)"
                    @click="addToWatchlist(stock.Symbol)"
                    class="action-button add-button"
                    :title="`Add ${stock.Symbol} to watchlist`"
                  >
                    Add to List
                  </button>
                  
                  <button 
                    v-else
                    @click="removeFromWatchlist(stock.Symbol)"
                    class="action-button remove-button"
                    :title="`Remove ${stock.Symbol} from watchlist`"
                  >
                    Remove
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="watchlist.length > 0" class="watchlist-section">
      <div class="watchlist-header">
        <h2>Your Stock Watchlist</h2>
        <div class="watchlist-stats">
          <span class="watchlist-count">{{ watchlist.length }} stocks</span>
          <button @click="watchlist = []" class="clear-watchlist-btn">Clear All</button>
        </div>
      </div>
      
      <div class="watchlist-grid">
        <div 
          v-for="symbol in watchlist" 
          :key="symbol" 
          class="watchlist-item"
        >
          <div class="watchlist-symbol">{{ symbol }}</div>
          <div class="watchlist-actions">
            <button 
              @click="viewDetails(symbol)"
              class="watchlist-view-btn"
              :title="`View ${symbol} details`"
            >
              👁️
            </button>
            <button 
              @click="removeFromWatchlist(symbol)"
              class="watchlist-remove-btn"
              :title="`Remove ${symbol} from watchlist`"
            >
              ×
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="!loading && filteredAndSortedSymbols.length === 0 && !error" class="empty-state">
      <div class="empty-icon">📈</div>
      <h3>{{ searchQuery ? 'No matching stocks found' : 'No stock data available' }}</h3>
      <p v-if="searchQuery">Try adjusting your search terms or clear the search filter</p>
      <p v-else>Click "Fetch Stocks" to load stock market data</p>
      <div class="empty-actions">
        <button v-if="searchQuery" @click="searchQuery = ''" class="clear-search-btn">Clear Search</button>
        <button v-if="!symbols.length" @click="fetchSymbols" class="fetch-data-btn">Fetch Stock Data</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.stock-dashboard {
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  min-height: 100vh;
}

.header-section {
  background: white;
  border-radius: 16px;
  padding: 2rem;
  margin-bottom: 2rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 2rem;
}

.title-section {
  flex: 1;
}

.main-title {
  margin: 0 0 0.5rem 0;
  font-size: 2.5rem;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: -0.025em;
}

.subtitle {
  margin: 0;
  font-size: 1.125rem;
  color: #64748b;
  font-weight: 400;
}

.controls-section {
  display: flex;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
}

.search-container {
  position: relative;
}

.search-input {
  padding: 0.75rem 1rem;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  font-size: 1rem;
  width: 200px;
  transition: all 0.2s ease;
  background: #f8fafc;
}

.search-input:focus {
  outline: none;
  border-color: #10b981;
  background: white;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

.sort-controls {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.sort-select {
  padding: 0.75rem;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  font-size: 0.875rem;
  background: #f8fafc;
  cursor: pointer;
}

.sort-select:focus {
  outline: none;
  border-color: #10b981;
  background: white;
}

.sort-order-btn {
  background: #f1f5f9;
  border: 2px solid #e2e8f0;
  border-radius: 8px;
  width: 40px;
  height: 40px;
  cursor: pointer;
  font-size: 1.2rem;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sort-order-btn:hover {
  background: #e2e8f0;
}

.sort-order-btn.desc {
  background: #10b981;
  color: white;
  border-color: #10b981;
}

.fetch-button {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 12px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 600;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 140px;
  justify-content: center;
}

.fetch-button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
}

.fetch-button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top: 2px solid white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error-card {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 12px;
  padding: 1.5rem;
  margin-bottom: 2rem;
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.error-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.error-content h3 {
  margin: 0 0 0.5rem 0;
  color: #dc2626;
  font-size: 1.125rem;
  font-weight: 600;
}

.error-content p {
  margin: 0 0 1rem 0;
  color: #991b1b;
}

.retry-button {
  background: #dc2626;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
}

.retry-button:hover {
  background: #b91c1c;
}

.table-section {
  background: white;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
  margin-bottom: 2rem;
}

.table-header {
  padding: 1.5rem 2rem;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f8fafc;
}

.table-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #1e293b;
}

.table-stats {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.symbol-count {
  background: #dcfce7;
  color: #166534;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
}

.search-indicator {
  background: #dbeafe;
  color: #1e40af;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
}

.table-container {
  overflow-x: auto;
}

.stocks-table {
  width: 100%;
  border-collapse: collapse;
}

.stocks-table thead {
  background: #f1f5f9;
}

.sortable-header {
  cursor: pointer;
  user-select: none;
  transition: background-color 0.2s ease;
}

.sortable-header:hover {
  background: #e2e8f0;
}

.header-content {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sort-indicator {
  font-size: 0.875rem;
  color: #10b981;
  font-weight: bold;
}

.stocks-table th {
  padding: 1rem 1.5rem;
  text-align: left;
  font-weight: 600;
  font-size: 0.875rem;
  color: #475569;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid #e2e8f0;
}

.actions-header {
  text-align: center;
}

.stock-row {
  transition: background-color 0.2s ease;
}

.stock-row:hover {
  background: #f8fafc;
}

.stocks-table td {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #f1f5f9;
  vertical-align: middle;
}

.symbol-cell {
  font-weight: 600;
}

.symbol-badge {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  color: white;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 700;
  display: inline-block;
  min-width: 60px;
  text-align: center;
}

.name-cell {
  color: #64748b;
  font-size: 0.875rem;
}

.company-name {
  font-weight: 500;
}

.actions-cell {
  text-align: center;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
  justify-content: center;
}

.action-button {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
  min-width: 100px;
}

.view-button {
  background: #e0f2fe;
  color: #0369a1;
  border: 1px solid #bae6fd;
}

.view-button:hover {
  background: #0369a1;
  color: white;
  transform: translateY(-1px);
}

.add-button {
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
}

.add-button:hover {
  background: #166534;
  color: white;
  transform: translateY(-1px);
}

.remove-button {
  background: #fee2e2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.remove-button:hover {
  background: #dc2626;
  color: white;
  transform: translateY(-1px);
}

.watchlist-section {
  background: white;
  border-radius: 16px;
  padding: 2rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
  margin-bottom: 2rem;
}

.watchlist-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.watchlist-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #1e293b;
}

.watchlist-stats {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.watchlist-count {
  background: #fef3c7;
  color: #92400e;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
}

.clear-watchlist-btn {
  background: #fee2e2;
  color: #dc2626;
  border: 1px solid #fecaca;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
}

.clear-watchlist-btn:hover {
  background: #dc2626;
  color: white;
}

.watchlist-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 1rem;
}

.watchlist-item {
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 1px solid #bae6fd;
  border-radius: 12px;
  padding: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  transition: all 0.2s ease;
}

.watchlist-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.15);
}

.watchlist-symbol {
  font-weight: 600;
  color: #0369a1;
  font-size: 1rem;
}

.watchlist-actions {
  display: flex;
  gap: 0.5rem;
}

.watchlist-view-btn, .watchlist-remove-btn {
  background: transparent;
  border: none;
  border-radius: 50%;
  width: 32px;
  height: 32px;
  cursor: pointer;
  font-size: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.watchlist-view-btn {
  background: #e0f2fe;
  color: #0369a1;
}

.watchlist-view-btn:hover {
  background: #0369a1;
  color: white;
}

.watchlist-remove-btn {
  background: #fee2e2;
  color: #dc2626;
  font-weight: bold;
}

.watchlist-remove-btn:hover {
  background: #dc2626;
  color: white;
}

.empty-state {
  text-align: center;
  padding: 4rem 2rem;
  background: white;
  border-radius: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.empty-state h3 {
  margin: 0 0 0.5rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #1e293b;
}

.empty-state p {
  margin: 0 0 1.5rem 0;
  color: #64748b;
  font-size: 1.125rem;
}

.empty-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
}

.clear-search-btn, .fetch-data-btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 600;
  transition: all 0.2s ease;
}

.clear-search-btn {
  background: #f1f5f9;
  color: #475569;
  border: 1px solid #e2e8f0;
}

.clear-search-btn:hover {
  background: #e2e8f0;
}

.fetch-data-btn {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  color: white;
}

.fetch-data-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
}

/* Responsive Design */
@media (max-width: 1024px) {
  .header-content {
    flex-direction: column;
    align-items: stretch;
  }
  
  .controls-section {
    justify-content: space-between;
  }
}

@media (max-width: 768px) {
  .stock-dashboard {
    padding: 1rem;
  }
  
  .header-section {
    padding: 1.5rem;
  }
  
  .main-title {
    font-size: 2rem;
  }
  
  .controls-section {
    flex-direction: column;
    gap: 1rem;
  }
  
  .search-input {
    width: 100%;
  }
  
  .action-buttons {
    flex-direction: column;
  }
  
  .stocks-table th,
  .stocks-table td {
    padding: 0.75rem;
  }
  
  .watchlist-grid {
    grid-template-columns: 1fr;
  }
  
  .table-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }
  
  .table-stats {
    justify-content: center;
  }
}
</style>