<script setup>
import { tracer } from '../tracing.js'
import { ref, onMounted } from 'vue'
import { context, propagation, trace } from '@opentelemetry/api'

const symbols = ref([])
const error = ref(null)
const loading = ref(false)
const watchlist = ref([])
const searchQuery = ref('')
const filteredSymbols = ref([])

// Enhanced tracing with more detailed instrumentation
async function fetchSymbols() {
  const mainSpan = tracer.startSpan('fetchCryptoSymbols', {
    attributes: {
      'operation.type': 'data_fetch',
      'component': 'crypto_dashboard',
      'user.action': 'fetch_symbols'
    }
  })
  
  const ctx = trace.setSpan(context.active(), mainSpan)
  
  try {
    error.value = null
    loading.value = true
    
    // Add performance tracking
    const startTime = performance.now()
    mainSpan.setAttribute('fetch.start_time', startTime)
    
    const headers = {}
    propagation.inject(ctx, headers)
    headers['Content-Type'] = 'application/json'
    
    const apiUrl = import.meta.env.VITE_API_URL || ""
    const fullUrl = `${apiUrl}/crypto/symbols`
    
    mainSpan.setAttributes({
      'api.url': fullUrl,
      'http.method': 'GET',
      'api.endpoint': '/crypto/symbols'
    })
    
    // Create a child span for the actual HTTP request
    const httpSpan = tracer.startSpan('http_request', {
      parent: ctx,
      attributes: {
        'http.url': fullUrl,
        'http.method': 'GET'
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
      'http.response_time_ms': duration
    })
    
    mainSpan.setAttributes({
      'http.status_code': response.status,
      'fetch.duration_ms': duration
    })
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }
    
    // Create span for data processing
    const processingSpan = tracer.startSpan('process_response_data', {
      parent: ctx
    })
    
    const data = await response.json()
    const processedSymbols = Array.isArray(data) ? data : data.symbols || []
    console.log(processedSymbols)
    processingSpan.setAttributes({
      'data.symbols_count': processedSymbols.length,
      'data.processing_time_ms': performance.now() - endTime
    })
    
    symbols.value = processedSymbols
    filteredSymbols.value = processedSymbols
    
    mainSpan.setAttributes({
      'symbols.count': processedSymbols.length,
      'operation.success': true
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
      'operation.success': false
    })
    
    mainSpan.setStatus({ code: 2, message: err.message })
  } finally {
    loading.value = false
    mainSpan.end()
  }
}

function viewDetails(symbolId, symbolName) {
  const span = tracer.startSpan('view_symbol_details', {
    attributes: {
      'symbol.id': symbolId,
      'symbol.name': symbolName,
      'user.action': 'view_details'
    }
  })
  
  try {
    // Enhanced user interaction tracking
    span.setAttributes({
      'interaction.timestamp': Date.now(),
      'page.section': 'symbol_table'
    })
    
    window.location = `/details/crypto/${symbolId}`
    span.setAttributes({
      'operation.success': true
    })
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function addToWatchlist(symbolId) {
  const span = tracer.startSpan('add_to_watchlist', {
    attributes: {
      'symbol.id': symbolId,
      'user.action': 'add_watchlist',
      'watchlist.size_before': watchlist.value.length
    }
  })
  
  try {
    if (!watchlist.value.includes(symbolId)) {
      watchlist.value.push(symbolId)
      
      span.setAttributes({
        'watchlist.size_after': watchlist.value.length,
        'operation.success': true,
        'watchlist.action': 'added'
      })
    } else {
      span.setAttributes({
        'operation.success': false,
        'error.reason': 'symbol_already_exists'
      })
    }
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function removeFromWatchlist(symbolId) {
  const span = tracer.startSpan('remove_from_watchlist', {
    attributes: {
      'symbol.id': symbolId,
      'user.action': 'remove_watchlist',
      'watchlist.size_before': watchlist.value.length
    }
  })
  
  try {
    const index = watchlist.value.indexOf(symbolId)
    if (index > -1) {
      watchlist.value.splice(index, 1)
      
      span.setAttributes({
        'watchlist.size_after': watchlist.value.length,
        'operation.success': true,
        'watchlist.action': 'removed'
      })
    }
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function isInWatchlist(symbolId) {
  return watchlist.value.includes(symbolId)
}

function filterSymbols() {
  const span = tracer.startSpan('filter_symbols', {
    attributes: {
      'search.query': searchQuery.value,
      'symbols.total_count': symbols.value.length
    }
  })
  
  try {
    if (!searchQuery.value.trim()) {
      filteredSymbols.value = symbols.value
    } else {
      filteredSymbols.value = symbols.value.filter(stock => 
        stock.Symbol?.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
        stock.Name?.toLowerCase().includes(searchQuery.value.toLowerCase())
      )
    }
    
    span.setAttributes({
      'symbols.filtered_count': filteredSymbols.value.length,
      'filter.has_results': filteredSymbols.value.length > 0
    })
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

// Track component lifecycle
onMounted(() => {
  const span = tracer.startSpan('component_mounted', {
    attributes: {
      'component': 'crypto_dashboard',
      'lifecycle.event': 'mounted'
    }
  })
  
  span.setStatus({ code: 1 })
  span.end()
})
</script>

<template>
  <div class="crypto-dashboard">
    <div class="header-section">
      <div class="header-content">
        <div class="title-section">
          <h1 class="main-title">Cryptocurrency Symbols</h1>
          <p class="subtitle">Discover and track your favorite crypto assets</p>
        </div>
        
        <div class="actions-section">
          <div class="search-container">
            <input 
              v-model="searchQuery"
              @input="filterSymbols"
              type="text"
              placeholder="Search symbols..."
              class="search-input"
            />
          </div>
          
          <button 
            @click="fetchSymbols" 
            :disabled="loading"
            class="fetch-button"
          >
            <span v-if="loading" class="loading-spinner"></span>
            {{ loading ? 'Loading...' : 'Fetch Symbols' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="error" class="error-card">
      <div class="error-icon">⚠️</div>
      <div class="error-content">
        <h3>Something went wrong</h3>
        <p>{{ error }}</p>
      </div>
    </div>

    <div v-if="filteredSymbols.length > 0" class="table-section">
      <div class="table-header">
        <h2>Available Symbols</h2>
        <span class="symbol-count">{{ filteredSymbols.length }} symbols</span>
      </div>
      
      <div class="table-container">
        <table class="symbols-table">
          <thead>
            <tr>
              <th>Symbol</th>
              <th>Name</th>
              <th class="actions-header">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="stock in filteredSymbols" :key="stock.Id" class="symbol-row">
              <td class="symbol-cell">
                <div class="symbol-badge">{{ stock.Symbol }}</div>
              </td>
              <td class="name-cell">
                <span class="symbol-name">{{ stock.Name || 'N/A' }}</span>
              </td>
              <td class="actions-cell">
                <div class="action-buttons">
                  <button 
                    @click="viewDetails(stock.Id, stock.Symbol)"
                    class="action-button view-button"
                  >
                    View Details
                  </button>
                  
                  <button 
                    v-if="!isInWatchlist(stock.Id)"
                    @click="addToWatchlist(stock.Id)"
                    class="action-button add-button"
                  >
                    Add to List
                  </button>
                  
                  <button 
                    v-else
                    @click="removeFromWatchlist(stock.Id)"
                    class="action-button remove-button"
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
        <h2>Your Watchlist</h2>
        <span class="watchlist-count">{{ watchlist.length }} items</span>
      </div>
      
      <div class="watchlist-grid">
        <div 
          v-for="symbolId in watchlist" 
          :key="symbolId" 
          class="watchlist-item"
        >
          <span class="watchlist-symbol">{{ symbolId }}</span>
          <button 
            @click="removeFromWatchlist(symbolId)"
            class="remove-from-watchlist"
          >
            ×
          </button>
        </div>
      </div>
    </div>

    <div v-if="!loading && filteredSymbols.length === 0 && !error" class="empty-state">
      <div class="empty-icon">📊</div>
      <h3>No symbols found</h3>
      <p v-if="searchQuery">Try adjusting your search terms</p>
      <p v-else>Click "Fetch Symbols" to load cryptocurrency data</p>
    </div>
  </div>
</template>

<style scoped>
.crypto-dashboard {
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

.actions-section {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.search-container {
  position: relative;
}

.search-input {
  padding: 0.75rem 1rem;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  font-size: 1rem;
  width: 250px;
  transition: all 0.2s ease;
  background: #f8fafc;
}

.search-input:focus {
  outline: none;
  border-color: #3b82f6;
  background: white;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.fetch-button {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
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
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
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
  margin: 0;
  color: #991b1b;
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

.symbol-count {
  background: #e0e7ff;
  color: #3730a3;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
}

.table-container {
  overflow-x: auto;
}

.symbols-table {
  width: 100%;
  border-collapse: collapse;
}

.symbols-table thead {
  background: #f1f5f9;
}

.symbols-table th {
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

.symbol-row {
  transition: background-color 0.2s ease;
}

.symbol-row:hover {
  background: #f8fafc;
}

.symbols-table td {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #f1f5f9;
  vertical-align: middle;
}

.symbol-cell {
  font-weight: 600;
}

.symbol-badge {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
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

.symbol-name {
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

.watchlist-count {
  background: #fef3c7;
  color: #92400e;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
}

.watchlist-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
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
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
}

.watchlist-symbol {
  font-weight: 600;
  color: #0369a1;
}

.remove-from-watchlist {
  background: #fee2e2;
  color: #dc2626;
  border: none;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.remove-from-watchlist:hover {
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
  margin: 0;
  color: #64748b;
  font-size: 1.125rem;
}

/* Responsive Design */
@media (max-width: 1024px) {
  .header-content {
    flex-direction: column;
    align-items: stretch;
  }
  
  .actions-section {
    justify-content: space-between;
  }
  
  .search-input {
    width: 200px;
  }
}

@media (max-width: 768px) {
  .crypto-dashboard {
    padding: 1rem;
  }
  
  .header-section {
    padding: 1.5rem;
  }
  
  .main-title {
    font-size: 2rem;
  }
  
  .actions-section {
    flex-direction: column;
    gap: 1rem;
  }
  
  .search-input {
    width: 100%;
  }
  
  .action-buttons {
    flex-direction: column;
  }
  
  .symbols-table th,
  .symbols-table td {
    padding: 0.75rem;
  }
  
  .watchlist-grid {
    grid-template-columns: 1fr;
  }
}
</style>