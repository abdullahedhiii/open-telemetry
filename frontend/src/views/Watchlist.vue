<script setup>
import { tracer } from '../tracing.js'
import { ref, onMounted, computed } from 'vue'
import { context, propagation, trace } from '@opentelemetry/api'

const watchlistItems = ref([])
const loading = ref(false)
const error = ref(null)
const searchQuery = ref('')
const sortBy = ref('symbol')
const sortOrder = ref('asc')
const selectedItems = ref([])
const showBulkActions = ref(false)

// Enhanced computed properties with tracing
const filteredAndSortedWatchlist = computed(() => {
  const span = tracer.startSpan('filter_and_sort_watchlist', {
    attributes: {
      'search.query': searchQuery.value,
      'sort.by': sortBy.value,
      'sort.order': sortOrder.value,
      'watchlist.total_count': watchlistItems.value.length
    }
  })
  
  try {
    let filtered = watchlistItems.value
    
    // Filter by search query
    if (searchQuery.value.trim()) {
      filtered = watchlistItems.value.filter(item => 
        item.symbol?.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
        item.name?.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
        item.type?.toLowerCase().includes(searchQuery.value.toLowerCase())
      )
    }
    
    // Sort the results
    filtered.sort((a, b) => {
      let aValue, bValue
      
      switch (sortBy.value) {
        case 'symbol':
          aValue = a.symbol || ''
          bValue = b.symbol || ''
          break
        case 'name':
          aValue = a.name || ''
          bValue = b.name || ''
          break
        case 'type':
          aValue = a.type || ''
          bValue = b.type || ''
          break
        case 'dateAdded':
          aValue = new Date(a.dateAdded || 0)
          bValue = new Date(b.dateAdded || 0)
          break
        default:
          aValue = a.symbol || ''
          bValue = b.symbol || ''
      }
      
      if (sortBy.value === 'dateAdded') {
        return sortOrder.value === 'asc' ? aValue - bValue : bValue - aValue
      } else {
        aValue = aValue.toString().toLowerCase()
        bValue = bValue.toString().toLowerCase()
        return sortOrder.value === 'asc' ? aValue.localeCompare(bValue) : bValue.localeCompare(aValue)
      }
    })
    
    span.setAttributes({
      'watchlist.filtered_count': filtered.length,
      'filter.has_results': filtered.length > 0,
      'operation.success': true
    })
    
    span.setStatus({ code: 1 })
    return filtered
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
    return watchlistItems.value
  } finally {
    span.end()
  }
})

const stockItems = computed(() => filteredAndSortedWatchlist.value.filter(item => item.type === 'stock'))
const cryptoItems = computed(() => filteredAndSortedWatchlist.value.filter(item => item.type === 'crypto'))

// Load watchlist data with enhanced tracing
async function loadWatchlist() {
  const span = tracer.startSpan('load_watchlist_data', {
    attributes: {
      'operation.type': 'data_load',
      'component': 'watchlist_page',
      'data.source': 'localStorage_and_api'
    }
  })
  
  try {
    loading.value = true
    error.value = null
    
    const startTime = performance.now()
    
    // Load from localStorage first
    const localStorageSpan = tracer.startSpan('load_from_localStorage', {
      parent: span
    })
    
    const savedWatchlist = localStorage.getItem('userWatchlist')
    let localItems = []
    
    if (savedWatchlist) {
      localItems = JSON.parse(savedWatchlist)
      localStorageSpan.setAttributes({
        'localStorage.items_found': localItems.length,
        'localStorage.data_size_bytes': savedWatchlist.length
      })
    }
    
    localStorageSpan.setStatus({ code: 1 })
    localStorageSpan.end()
    
    // Simulate API call to get additional details
    const apiSpan = tracer.startSpan('fetch_watchlist_details', {
      parent: span,
      attributes: {
        'api.endpoint': '/watchlist/details',
        'items.count': localItems.length
      }
    })
    
    // Mock API call - in real app, this would fetch current prices, etc.
    await new Promise(resolve => setTimeout(resolve, 500))
    
    const enrichedItems = localItems.map(item => ({
      ...item,
      currentPrice: (Math.random() * 1000 + 10).toFixed(2),
      change: ((Math.random() - 0.5) * 20).toFixed(2),
      changePercent: ((Math.random() - 0.5) * 10).toFixed(2),
      lastUpdated: new Date().toISOString(),
      dateAdded: item.dateAdded || new Date().toISOString()
    }))
    
    watchlistItems.value = enrichedItems
    
    const duration = performance.now() - startTime
    
    apiSpan.setAttributes({
      'api.response_time_ms': duration,
      'items.enriched_count': enrichedItems.length,
      'operation.success': true
    })
    
    span.setAttributes({
      'watchlist.total_items': enrichedItems.length,
      'watchlist.stock_items': enrichedItems.filter(i => i.type === 'stock').length,
      'watchlist.crypto_items': enrichedItems.filter(i => i.type === 'crypto').length,
      'load.duration_ms': duration,
      'operation.success': true
    })
    
    apiSpan.setStatus({ code: 1 })
    span.setStatus({ code: 1 })
    
    apiSpan.end()
    
  } catch (err) {
    error.value = err.message
    
    span.setAttributes({
      'error.message': err.message,
      'error.type': err.constructor.name,
      'operation.success': false
    })
    
    span.setStatus({ code: 2, message: err.message })
  } finally {
    loading.value = false
    span.end()
  }
}

function removeFromWatchlist(symbol) {
  const span = tracer.startSpan('remove_from_watchlist', {
    attributes: {
      'symbol': symbol,
      'user.action': 'remove_single_item',
      'watchlist.size_before': watchlistItems.value.length
    }
  })
  
  try {
    const index = watchlistItems.value.findIndex(item => item.symbol === symbol)
    if (index > -1) {
      const removedItem = watchlistItems.value.splice(index, 1)[0]
      
      // Update localStorage
      localStorage.setItem('userWatchlist', JSON.stringify(
        watchlistItems.value.map(item => ({
          symbol: item.symbol,
          name: item.name,
          type: item.type,
          dateAdded: item.dateAdded
        }))
      ))
      
      span.setAttributes({
        'watchlist.size_after': watchlistItems.value.length,
        'removed.item_type': removedItem.type,
        'removed.item_name': removedItem.name,
        'operation.success': true
      })
    }
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function toggleItemSelection(symbol) {
  const span = tracer.startSpan('toggle_item_selection', {
    attributes: {
      'symbol': symbol,
      'selection.count_before': selectedItems.value.length
    }
  })
  
  try {
    const index = selectedItems.value.indexOf(symbol)
    if (index > -1) {
      selectedItems.value.splice(index, 1)
    } else {
      selectedItems.value.push(symbol)
    }
    
    showBulkActions.value = selectedItems.value.length > 0
    
    span.setAttributes({
      'selection.count_after': selectedItems.value.length,
      'selection.action': index > -1 ? 'deselected' : 'selected',
      'bulk_actions.visible': showBulkActions.value
    })
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function removeSelectedItems() {
  const span = tracer.startSpan('remove_selected_items', {
    attributes: {
      'selected.count': selectedItems.value.length,
      'user.action': 'bulk_remove',
      'watchlist.size_before': watchlistItems.value.length
    }
  })
  
  try {
    const removedCount = selectedItems.value.length
    watchlistItems.value = watchlistItems.value.filter(
      item => !selectedItems.value.includes(item.symbol)
    )
    
    // Update localStorage
    localStorage.setItem('userWatchlist', JSON.stringify(
      watchlistItems.value.map(item => ({
        symbol: item.symbol,
        name: item.name,
        type: item.type,
        dateAdded: item.dateAdded
      }))
    ))
    
    selectedItems.value = []
    showBulkActions.value = false
    
    span.setAttributes({
      'removed.count': removedCount,
      'watchlist.size_after': watchlistItems.value.length,
      'operation.success': true
    })
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function clearAllItems() {
  const span = tracer.startSpan('clear_all_watchlist', {
    attributes: {
      'watchlist.size_before': watchlistItems.value.length,
      'user.action': 'clear_all'
    }
  })
  
  try {
    const clearedCount = watchlistItems.value.length
    watchlistItems.value = []
    selectedItems.value = []
    showBulkActions.value = false
    
    localStorage.removeItem('userWatchlist')
    
    span.setAttributes({
      'cleared.count': clearedCount,
      'operation.success': true
    })
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function viewDetails(symbol, type) {
  const span = tracer.startSpan('navigate_to_details', {
    attributes: {
      'symbol': symbol,
      'item.type': type,
      'user.action': 'view_details',
      'navigation.source': 'watchlist_page'
    }
  })
  
  try {
    // In a real app, this would use Vue Router
    const detailsUrl = `/details?symbol=${symbol}&type=${type}`
    
    span.setAttributes({
      'navigation.url': detailsUrl,
      'operation.success': true
    })
    
    alert(`Navigating to details for ${symbol} (${type})`)
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function updateSort(field) {
  const span = tracer.startSpan('update_watchlist_sort', {
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

// Component lifecycle
onMounted(() => {
  const span = tracer.startSpan('watchlist_page_mounted', {
    attributes: {
      'component': 'watchlist_page',
      'lifecycle.event': 'mounted',
      'page.url': window.location.href
    }
  })
  
  loadWatchlist()
  
  span.setStatus({ code: 1 })
  span.end()
})
</script>

<template>
  <div class="watchlist-page">
    <div class="header-section">
      <div class="header-content">
        <div class="title-section">
          <h1 class="main-title">My Watchlist</h1>
          <p class="subtitle">Track your favorite stocks and cryptocurrencies</p>
        </div>
        
        <div class="controls-section">
          <div class="search-container">
            <input 
              v-model="searchQuery"
              type="text"
              placeholder="Search watchlist..."
              class="search-input"
            />
          </div>
          
          <div class="sort-controls">
            <select v-model="sortBy" class="sort-select">
              <option value="symbol">Sort by Symbol</option>
              <option value="name">Sort by Name</option>
              <option value="type">Sort by Type</option>
              <option value="dateAdded">Sort by Date Added</option>
            </select>
            <button 
              @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'"
              class="sort-order-btn"
              :class="{ 'desc': sortOrder === 'desc' }"
            >
              {{ sortOrder === 'asc' ? '↑' : '↓' }}
            </button>
          </div>
          
          <button @click="loadWatchlist" :disabled="loading" class="refresh-button">
            <span v-if="loading" class="loading-spinner"></span>
            {{ loading ? 'Loading...' : 'Refresh' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Bulk Actions Bar -->
    <div v-if="showBulkActions" class="bulk-actions-bar">
      <div class="bulk-actions-content">
        <span class="selected-count">{{ selectedItems.length }} items selected</span>
        <div class="bulk-actions">
          <button @click="removeSelectedItems" class="bulk-remove-btn">
            Remove Selected
          </button>
          <button @click="selectedItems = []; showBulkActions = false" class="cancel-selection-btn">
            Cancel
          </button>
        </div>
      </div>
    </div>

    <div v-if="error" class="error-card">
      <div class="error-icon">⚠️</div>
      <div class="error-content">
        <h3>Unable to load watchlist</h3>
        <p>{{ error }}</p>
        <button @click="loadWatchlist" class="retry-button">Try Again</button>
      </div>
    </div>

    <!-- Summary Cards -->
    <div v-if="!loading && watchlistItems.length > 0" class="summary-section">
      <div class="summary-cards">
        <div class="summary-card">
          <div class="summary-icon">📊</div>
          <div class="summary-content">
            <h3>Total Items</h3>
            <p class="summary-number">{{ watchlistItems.length }}</p>
          </div>
        </div>
        
        <div class="summary-card">
          <div class="summary-icon">📈</div>
          <div class="summary-content">
            <h3>Stocks</h3>
            <p class="summary-number">{{ stockItems.length }}</p>
          </div>
        </div>
        
        <div class="summary-card">
          <div class="summary-icon">₿</div>
          <div class="summary-content">
            <h3>Crypto</h3>
            <p class="summary-number">{{ cryptoItems.length }}</p>
          </div>
        </div>
        
        <div class="summary-card danger" v-if="watchlistItems.length > 0">
          <div class="summary-icon">🗑️</div>
          <div class="summary-content">
            <h3>Clear All</h3>
            <button @click="clearAllItems" class="clear-all-btn">Clear</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Watchlist Table -->
    <div v-if="filteredAndSortedWatchlist.length > 0" class="table-section">
      <div class="table-header">
        <h2>Watchlist Items</h2>
        <div class="table-stats">
          <span class="item-count">{{ filteredAndSortedWatchlist.length }} of {{ watchlistItems.length }} items</span>
          <span v-if="searchQuery" class="search-indicator">Filtered by: "{{ searchQuery }}"</span>
        </div>
      </div>
      
      <div class="table-container">
        <table class="watchlist-table">
          <thead>
            <tr>
              <th class="checkbox-header">
                <input 
                  type="checkbox" 
                  @change="selectedItems = $event.target.checked ? filteredAndSortedWatchlist.map(i => i.symbol) : []; showBulkActions = selectedItems.length > 0"
                  :checked="selectedItems.length === filteredAndSortedWatchlist.length && filteredAndSortedWatchlist.length > 0"
                />
              </th>
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
                  Name
                  <span v-if="sortBy === 'name'" class="sort-indicator">
                    {{ sortOrder === 'asc' ? '↑' : '↓' }}
                  </span>
                </div>
              </th>
              <th @click="updateSort('type')" class="sortable-header">
                <div class="header-content">
                  Type
                  <span v-if="sortBy === 'type'" class="sort-indicator">
                    {{ sortOrder === 'asc' ? '↑' : '↓' }}
                  </span>
                </div>
              </th>
              <th>Current Price</th>
              <th>Change</th>
              <th @click="updateSort('dateAdded')" class="sortable-header">
                <div class="header-content">
                  Date Added
                  <span v-if="sortBy === 'dateAdded'" class="sort-indicator">
                    {{ sortOrder === 'asc' ? '↑' : '↓' }}
                  </span>
                </div>
              </th>
              <th class="actions-header">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredAndSortedWatchlist" :key="item.symbol" class="watchlist-row">
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  :checked="selectedItems.includes(item.symbol)"
                  @change="toggleItemSelection(item.symbol)"
                />
              </td>
              <td class="symbol-cell">
                <div class="symbol-badge" :class="item.type">{{ item.symbol }}</div>
              </td>
              <td class="name-cell">
                <span class="item-name">{{ item.name || 'N/A' }}</span>
              </td>
              <td class="type-cell">
                <span class="type-badge" :class="item.type">
                  {{ item.type === 'stock' ? '📈 Stock' : '₿ Crypto' }}
                </span>
              </td>
              <td class="price-cell">
                <span class="current-price">${{ item.currentPrice }}</span>
              </td>
              <td class="change-cell">
                <div class="change-info" :class="{ 'positive': parseFloat(item.change) > 0, 'negative': parseFloat(item.change) < 0 }">
                  <span class="change-amount">{{ parseFloat(item.change) > 0 ? '+' : '' }}${{ item.change }}</span>
                  <span class="change-percent">({{ parseFloat(item.changePercent) > 0 ? '+' : '' }}{{ item.changePercent }}%)</span>
                </div>
              </td>
              <td class="date-cell">
                <span class="date-added">{{ new Date(item.dateAdded).toLocaleDateString() }}</span>
              </td>
              <td class="actions-cell">
                <div class="action-buttons">
                  <button 
                    @click="viewDetails(item.symbol, item.type)"
                    class="action-button view-button"
                    :title="`View details for ${item.symbol}`"
                  >
                    View Details
                  </button>
                  
                  <button 
                    @click="removeFromWatchlist(item.symbol)"
                    class="action-button remove-button"
                    :title="`Remove ${item.symbol} from watchlist`"
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

    <!-- Empty State -->
    <div v-if="!loading && filteredAndSortedWatchlist.length === 0 && !error" class="empty-state">
      <div class="empty-icon">📋</div>
      <h3>{{ searchQuery ? 'No matching items found' : 'Your watchlist is empty' }}</h3>
      <p v-if="searchQuery">Try adjusting your search terms or clear the search filter</p>
      <p v-else>Start adding stocks and cryptocurrencies to track them here</p>
      <div class="empty-actions">
        <button v-if="searchQuery" @click="searchQuery = ''" class="clear-search-btn">Clear Search</button>
        <button v-if="!watchlistItems.length" @click="$router?.push?.('/') || alert('Navigate to dashboard')" class="go-dashboard-btn">Go to Dashboard</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.watchlist-page {
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
  border-color: #8b5cf6;
  background: white;
  box-shadow: 0 0 0 3px rgba(139, 92, 246, 0.1);
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

.sort-order-btn {
  background: #f1f5f9;
  border: 2px solid #e2e8f0;
  border-radius: 8px;
  width: 40px;
  height: 40px;
  cursor: pointer;
  font-size: 1.2rem;
  transition: all 0.2s ease;
}

.sort-order-btn.desc {
  background: #8b5cf6;
  color: white;
  border-color: #8b5cf6;
}

.refresh-button {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
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
  min-width: 120px;
  justify-content: center;
}

.refresh-button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.4);
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

.bulk-actions-bar {
  background: #fef3c7;
  border: 1px solid #fbbf24;
  border-radius: 12px;
  padding: 1rem 2rem;
  margin-bottom: 2rem;
  display: flex;
  justify-content: center;
}

.bulk-actions-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  max-width: 600px;
}

.selected-count {
  font-weight: 600;
  color: #92400e;
}

.bulk-actions {
  display: flex;
  gap: 1rem;
}

.bulk-remove-btn {
  background: #dc2626;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 500;
}

.cancel-selection-btn {
  background: #6b7280;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 500;
}

.summary-section {
  margin-bottom: 2rem;
}

.summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.summary-card {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  gap: 1rem;
}

.summary-card.danger {
  border-color: #fecaca;
  background: #fef2f2;
}

.summary-icon {
  font-size: 2rem;
  flex-shrink: 0;
}

.summary-content h3 {
  margin: 0 0 0.5rem 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.summary-number {
  margin: 0;
  font-size: 2rem;
  font-weight: 700;
  color: #1e293b;
}

.clear-all-btn {
  background: #dc2626;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
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

.table-stats {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.item-count {
  background: #e0e7ff;
  color: #3730a3;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
}

.watchlist-table {
  width: 100%;
  border-collapse: collapse;
}

.watchlist-table thead {
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
  color: #8b5cf6;
  font-weight: bold;
}

.watchlist-table th {
  padding: 1rem 1.5rem;
  text-align: left;
  font-weight: 600;
  font-size: 0.875rem;
  color: #475569;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid #e2e8f0;
}

.checkbox-header {
  width: 50px;
  text-align: center;
}

.actions-header {
  text-align: center;
}

.watchlist-row {
  transition: background-color 0.2s ease;
}

.watchlist-row:hover {
  background: #f8fafc;
}

.watchlist-table td {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #f1f5f9;
  vertical-align: middle;
}

.checkbox-cell {
  text-align: center;
}

.symbol-badge {
  padding: 0.5rem 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 700;
  display: inline-block;
  min-width: 60px;
  text-align: center;
  color: white;
}

.symbol-badge.stock {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.symbol-badge.crypto {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.type-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 500;
}

.type-badge.stock {
  background: #dcfce7;
  color: #166534;
}

.type-badge.crypto {
  background: #fef3c7;
  color: #92400e;
}

.current-price {
  font-weight: 600;
  color: #1e293b;
  font-size: 1rem;
}

.change-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.change-info.positive {
  color: #059669;
}

.change-info.negative {
  color: #dc2626;
}

.change-amount {
  font-weight: 600;
}

.change-percent {
  font-size: 0.875rem;
  opacity: 0.8;
}

.date-added {
  color: #64748b;
  font-size: 0.875rem;
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

.empty-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  margin-top: 1.5rem;
}

.clear-search-btn, .go-dashboard-btn {
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

.go-dashboard-btn {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
  color: white;
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
  .watchlist-page {
    padding: 1rem;
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
  
  .summary-cards {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .watchlist-table {
    font-size: 0.875rem;
  }
  
  .watchlist-table th,
  .watchlist-table td {
    padding: 0.75rem 0.5rem;
  }
  
  .action-buttons {
    flex-direction: column;
  }
}
</style>