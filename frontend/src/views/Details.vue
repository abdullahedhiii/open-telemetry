<script setup>
import { tracer } from '../tracing.js'
import { ref, onMounted, computed } from 'vue'
import { context, propagation, trace } from '@opentelemetry/api'

const symbol = ref('')
const type = ref('')
const loading = ref(false)
const error = ref(null)
const detailsData = ref(null)
const priceHistory = ref([])
const newsData = ref([])
const isInWatchlist = ref(false)
const activeTab = ref('overview')

// Mock data for demonstration
const mockStockData = {
  symbol: 'AAPL',
  name: 'Apple Inc.',
  type: 'stock',
  currentPrice: 175.43,
  change: 2.15,
  changePercent: 1.24,
  marketCap: '2.75T',
  volume: '45.2M',
  peRatio: 28.5,
  dividend: 0.96,
  high52Week: 198.23,
  low52Week: 124.17,
  description: 'Apple Inc. designs, manufactures, and markets smartphones, personal computers, tablets, wearables, and accessories worldwide.',
  sector: 'Technology',
  industry: 'Consumer Electronics',
  employees: 164000,
  founded: '1976',
  headquarters: 'Cupertino, CA'
}

const mockCryptoData = {
  symbol: 'BTC',
  name: 'Bitcoin',
  type: 'crypto',
  currentPrice: 43250.75,
  change: -1250.30,
  changePercent: -2.81,
  marketCap: '847.5B',
  volume: '28.7B',
  circulatingSupply: '19.6M',
  totalSupply: '21M',
  high24h: 44500.00,
  low24h: 42800.00,
  description: 'Bitcoin is a decentralized digital currency that can be transferred on the peer-to-peer bitcoin network.',
  category: 'Cryptocurrency',
  algorithm: 'SHA-256',
  blockTime: '10 minutes'
}

// Enhanced computed properties
const formattedPrice = computed(() => {
  if (!detailsData.value) return '$0.00'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2
  }).format(detailsData.value.currentPrice)
})

const formattedChange = computed(() => {
  if (!detailsData.value) return { amount: '$0.00', percent: '0.00%', isPositive: true }
  
  const change = detailsData.value.change
  const changePercent = detailsData.value.changePercent
  const isPositive = change >= 0
  
  return {
    amount: `${isPositive ? '+' : ''}${new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2
    }).format(change)}`,
    percent: `${isPositive ? '+' : ''}${changePercent.toFixed(2)}%`,
    isPositive
  }
})

// Load details with comprehensive tracing
async function loadDetails() {
  const span = tracer.startSpan('load_symbol_details', {
    attributes: {
      'symbol': symbol.value,
      'symbol.type': type.value,
      'operation.type': 'data_fetch',
      'component': 'details_page'
    }
  })
  
  try {
    loading.value = true
    error.value = null
    
    const startTime = performance.now()
    
    // Simulate API calls for different data types
    const detailsSpan = tracer.startSpan('fetch_symbol_data', {
      parent: span,
      attributes: {
        'api.endpoint': `/api/${type.value}/${symbol.value}`,
        'data.type': 'symbol_details'
      }
    })
    
    await new Promise(resolve => setTimeout(resolve, 800))
    
    // Mock data based on type
    if (type.value === 'stock') {
      detailsData.value = { ...mockStockData, symbol: symbol.value }
    } else {
      detailsData.value = { ...mockCryptoData, symbol: symbol.value }
    }
    
    detailsSpan.setAttributes({
      'data.loaded': true,
      'symbol.name': detailsData.value.name,
      'symbol.price': detailsData.value.currentPrice
    })
    detailsSpan.setStatus({ code: 1 })
    detailsSpan.end()
    
    // Load price history
    const historySpan = tracer.startSpan('fetch_price_history', {
      parent: span,
      attributes: {
        'api.endpoint': `/api/${type.value}/${symbol.value}/history`,
        'data.type': 'price_history'
      }
    })
    
    await new Promise(resolve => setTimeout(resolve, 500))
    
    // Generate mock price history
    const basePrice = detailsData.value.currentPrice
    priceHistory.value = Array.from({ length: 30 }, (_, i) => ({
      date: new Date(Date.now() - (29 - i) * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
      price: basePrice + (Math.random() - 0.5) * basePrice * 0.1
    }))
    
    historySpan.setAttributes({
      'history.data_points': priceHistory.value.length,
      'history.date_range': '30_days'
    })
    historySpan.setStatus({ code: 1 })
    historySpan.end()
    
    // Load news data
    const newsSpan = tracer.startSpan('fetch_news_data', {
      parent: span,
      attributes: {
        'api.endpoint': `/api/news/${symbol.value}`,
        'data.type': 'news_articles'
      }
    })
    
    await new Promise(resolve => setTimeout(resolve, 300))
    
    newsData.value = [
      {
        id: 1,
        title: `${detailsData.value.name} Reports Strong Quarterly Earnings`,
        summary: 'Company exceeds analyst expectations with robust revenue growth.',
        publishedAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
        source: 'Financial Times'
      },
      {
        id: 2,
        title: `Market Analysis: ${symbol.value} Shows Bullish Momentum`,
        summary: 'Technical indicators suggest continued upward trend.',
        publishedAt: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
        source: 'MarketWatch'
      },
      {
        id: 3,
        title: `${detailsData.value.name} Announces New Product Launch`,
        summary: 'Innovation continues to drive company growth strategy.',
        publishedAt: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
        source: 'Reuters'
      }
    ]
    
    newsSpan.setAttributes({
      'news.articles_count': newsData.value.length
    })
    newsSpan.setStatus({ code: 1 })
    newsSpan.end()
    
    // Check watchlist status
    const watchlistSpan = tracer.startSpan('check_watchlist_status', {
      parent: span
    })
    
    const savedWatchlist = localStorage.getItem('userWatchlist')
    if (savedWatchlist) {
      const watchlist = JSON.parse(savedWatchlist)
      isInWatchlist.value = watchlist.some(item => item.symbol === symbol.value)
    }
    
    watchlistSpan.setAttributes({
      'watchlist.contains_symbol': isInWatchlist.value
    })
    watchlistSpan.setStatus({ code: 1 })
    watchlistSpan.end()
    
    const duration = performance.now() - startTime
    
    span.setAttributes({
      'load.duration_ms': duration,
      'data.sections_loaded': 4,
      'operation.success': true
    })
    
    span.setStatus({ code: 1 })
    
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
      watchlist = watchlist.filter(item => item.symbol !== symbol.value)
      isInWatchlist.value = false
    } else {
      // Add to watchlist
      watchlist.push({
        symbol: symbol.value,
        name: detailsData.value?.name || symbol.value,
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

function switchTab(tab) {
  const span = tracer.startSpan('switch_details_tab', {
    attributes: {
      'tab.previous': activeTab.value,
      'tab.new': tab,
      'user.action': 'tab_switch'
    }
  })
  
  activeTab.value = tab
  
  span.setAttributes({
    'operation.success': true
  })
  span.setStatus({ code: 1 })
  span.end()
}

// Initialize from URL parameters (in real app, use Vue Router)
onMounted(() => {
  const span = tracer.startSpan('details_page_mounted', {
    attributes: {
      'component': 'details_page',
      'lifecycle.event': 'mounted'
    }
  })
  
  // Mock URL parameter parsing
  const urlParams = new URLSearchParams(window.location.search)
  symbol.value = urlParams.get('symbol') || 'AAPL'
  type.value = urlParams.get('type') || 'stock'
  
  span.setAttributes({
    'url.symbol': symbol.value,
    'url.type': type.value
  })
  
  loadDetails()
  
  span.setStatus({ code: 1 })
  span.end()
})
</script>

<template>
  <div class="details-page">
    <!-- Header Section -->
    <div class="header-section">
      <div class="header-content">
        <div class="symbol-info">
          <div class="symbol-badge" :class="type">{{ symbol }}</div>
          <div class="symbol-details">
            <h1 class="symbol-name">{{ detailsData?.name || symbol }}</h1>
            <p class="symbol-type">{{ type === 'stock' ? '📈 Stock' : '₿ Cryptocurrency' }}</p>
          </div>
        </div>
        
        <div class="price-section" v-if="detailsData">
          <div class="current-price">{{ formattedPrice }}</div>
          <div class="price-change" :class="{ 'positive': formattedChange.isPositive, 'negative': !formattedChange.isPositive }">
            <span class="change-amount">{{ formattedChange.amount }}</span>
            <span class="change-percent">{{ formattedChange.percent }}</span>
          </div>
        </div>
        
        <div class="header-actions">
          <button 
            @click="toggleWatchlist"
            class="watchlist-button"
            :class="{ 'in-watchlist': isInWatchlist }"
          >
            {{ isInWatchlist ? '★ In Watchlist' : '☆ Add to Watchlist' }}
          </button>
          
          <button @click="loadDetails" :disabled="loading" class="refresh-button">
            <span v-if="loading" class="loading-spinner"></span>
            {{ loading ? 'Loading...' : 'Refresh' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="error" class="error-card">
      <div class="error-icon">⚠️</div>
      <div class="error-content">
        <h3>Unable to load details</h3>
        <p>{{ error }}</p>
        <button @click="loadDetails" class="retry-button">Try Again</button>
      </div>
    </div>

    <div v-if="loading && !detailsData" class="loading-section">
      <div class="loading-content">
        <div class="loading-spinner large"></div>
        <h3>Loading {{ symbol }} details...</h3>
        <p>Fetching latest market data and information</p>
      </div>
    </div>

    <div v-if="detailsData" class="content-section">
      <!-- Tab Navigation -->
      <div class="tab-navigation">
        <button 
          v-for="tab in ['overview', 'chart', 'news', 'fundamentals']" 
          :key="tab"
          @click="switchTab(tab)"
          class="tab-button"
          :class="{ 'active': activeTab === tab }"
        >
          {{ tab.charAt(0).toUpperCase() + tab.slice(1) }}
        </button>
      </div>

      <!-- Tab Content -->
      <div class="tab-content">
        <!-- Overview Tab -->
        <div v-if="activeTab === 'overview'" class="overview-tab">
          <div class="overview-grid">
            <div class="info-card">
              <h3>Market Data</h3>
              <div class="info-grid">
                <div class="info-item">
                  <span class="info-label">Market Cap</span>
                  <span class="info-value">{{ detailsData.marketCap }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">Volume</span>
                  <span class="info-value">{{ detailsData.volume }}</span>
                </div>
                <div class="info-item" v-if="type === 'stock'">
                  <span class="info-label">P/E Ratio</span>
                  <span class="info-value">{{ detailsData.peRatio }}</span>
                </div>
                <div class="info-item" v-if="type === 'crypto'">
                  <span class="info-label">Circulating Supply</span>
                  <span class="info-value">{{ detailsData.circulatingSupply }}</span>
                </div>
                <div class="info-item" v-if="type === 'stock'">
                  <span class="info-label">52W High</span>
                  <span class="info-value">${{ detailsData.high52Week }}</span>
                </div>
                <div class="info-item" v-if="type === 'crypto'">
                  <span class="info-label">24h High</span>
                  <span class="info-value">${{ detailsData.high24h?.toLocaleString() }}</span>
                </div>
              </div>
            </div>
            
            <div class="info-card">
              <h3>About</h3>
              <p class="description">{{ detailsData.description }}</p>
              <div class="additional-info">
                <div class="info-item" v-if="type === 'stock'">
                  <span class="info-label">Sector</span>
                  <span class="info-value">{{ detailsData.sector }}</span>
                </div>
                <div class="info-item" v-if="type === 'stock'">
                  <span class="info-label">Industry</span>
                  <span class="info-value">{{ detailsData.industry }}</span>
                </div>
                <div class="info-item" v-if="type === 'crypto'">
                  <span class="info-label">Category</span>
                  <span class="info-value">{{ detailsData.category }}</span>
                </div>
                <div class="info-item" v-if="type === 'crypto'">
                  <span class="info-label">Algorithm</span>
                  <span class="info-value">{{ detailsData.algorithm }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Chart Tab -->
        <div v-if="activeTab === 'chart'" class="chart-tab">
          <div class="chart-container">
            <h3>Price History (30 Days)</h3>
            <div class="chart-placeholder">
              <div class="chart-mock">
                <div class="chart-line">
                  <div 
                    v-for="(point, index) in priceHistory" 
                    :key="index"
                    class="chart-point"
                    :style="{ 
                      left: `${(index / (priceHistory.length - 1)) * 100}%`,
                      bottom: `${((point.price - Math.min(...priceHistory.map(p => p.price))) / 
                        (Math.max(...priceHistory.map(p => p.price)) - Math.min(...priceHistory.map(p => p.price)))) * 80 + 10}%`
                    }"
                  ></div>
                </div>
              </div>
              <p class="chart-note">Interactive chart would be implemented with a charting library like Chart.js or D3</p>
            </div>
          </div>
        </div>

        <!-- News Tab -->
        <div v-if="activeTab === 'news'" class="news-tab">
          <h3>Latest News</h3>
          <div class="news-list">
            <div v-for="article in newsData" :key="article.id" class="news-item">
              <div class="news-content">
                <h4 class="news-title">{{ article.title }}</h4>
                <p class="news-summary">{{ article.summary }}</p>
                <div class="news-meta">
                  <span class="news-source">{{ article.source }}</span>
                  <span class="news-date">{{ new Date(article.publishedAt).toLocaleDateString() }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Fundamentals Tab -->
        <div v-if="activeTab === 'fundamentals'" class="fundamentals-tab">
          <h3>{{ type === 'stock' ? 'Company Fundamentals' : 'Token Fundamentals' }}</h3>
          <div class="fundamentals-grid">
            <div class="fundamental-card" v-if="type === 'stock'">
              <h4>Financial Metrics</h4>
              <div class="metric-list">
                <div class="metric-item">
                  <span class="metric-label">P/E Ratio</span>
                  <span class="metric-value">{{ detailsData.peRatio }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">Dividend Yield</span>
                  <span class="metric-value">{{ detailsData.dividend }}%</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">Employees</span>
                  <span class="metric-value">{{ detailsData.employees?.toLocaleString() }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">Founded</span>
                  <span class="metric-value">{{ detailsData.founded }}</span>
                </div>
              </div>
            </div>
            
            <div class="fundamental-card" v-if="type === 'crypto'">
              <h4>Token Metrics</h4>
              <div class="metric-list">
                <div class="metric-item">
                  <span class="metric-label">Total Supply</span>
                  <span class="metric-value">{{ detailsData.totalSupply }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">Circulating Supply</span>
                  <span class="metric-value">{{ detailsData.circulatingSupply }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">Block Time</span>
                  <span class="metric-value">{{ detailsData.blockTime }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">Algorithm</span>
                  <span class="metric-value">{{ detailsData.algorithm }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.details-page {
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

.symbol-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.symbol-badge {
  padding: 1rem 1.5rem;
  border-radius: 12px;
  font-size: 1.5rem;
  font-weight: 700;
  color: white;
  min-width: 80px;
  text-align: center;
}

.symbol-badge.stock {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.symbol-badge.crypto {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.symbol-details {
  flex: 1;
}

.symbol-name {
  margin: 0 0 0.5rem 0;
  font-size: 2rem;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: -0.025em;
}

.symbol-type {
  margin: 0;
  font-size: 1rem;
  color: #64748b;
  font-weight: 500;
}

.price-section {
  text-align: right;
}

.current-price {
  font-size: 2.5rem;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 0.5rem;
}

.price-change {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.price-change.positive {
  color: #059669;
}

.price-change.negative {
  color: #dc2626;
}

.change-amount {
  font-size: 1.25rem;
  font-weight: 600;
}

.change-percent {
  font-size: 1rem;
  opacity: 0.8;
}

.header-actions {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.watchlist-button {
  background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 12px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 600;
  transition: all 0.2s ease;
  min-width: 160px;
}

.watchlist-button.in-watchlist {
  background: linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%);
}

.watchlist-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
}

.watchlist-button.in-watchlist:hover {
  box-shadow: 0 4px 12px rgba(251, 191, 36, 0.4);
}

.refresh-button {
  background: #f1f5f9;
  color: #475569;
  border: 1px solid #e2e8f0;
  padding: 0.75rem 1.5rem;
  border-radius: 12px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 600;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.refresh-button:hover:not(:disabled) {
  background: #e2e8f0;
}

.refresh-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top: 2px solid currentColor;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.loading-spinner.large {
  width: 48px;
  height: 48px;
  border-width: 4px;
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

.loading-section {
  background: white;
  border-radius: 16px;
  padding: 4rem 2rem;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
}

.loading-content h3 {
  margin: 1rem 0 0.5rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #1e293b;
}

.loading-content p {
  margin: 0;
  color: #64748b;
  font-size: 1.125rem;
}

.content-section {
  background: white;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
}

.tab-navigation {
  display: flex;
  border-bottom: 1px solid #e2e8f0;
  background: #f8fafc;
}

.tab-button {
  background: transparent;
  border: none;
  padding: 1rem 2rem;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 600;
  color: #64748b;
  transition: all 0.2s ease;
  border-bottom: 3px solid transparent;
}

.tab-button:hover {
  background: #f1f5f9;
  color: #1e293b;
}

.tab-button.active {
  color: #6366f1;
  border-bottom-color: #6366f1;
  background: white;
}

.tab-content {
  padding: 2rem;
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 2rem;
}

.info-card {
  background: #f8fafc;
  border-radius: 12px;
  padding: 1.5rem;
  border: 1px solid #e2e8f0;
}

.info-card h3 {
  margin: 0 0 1rem 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: #1e293b;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid #e2e8f0;
}

.info-item:last-child {
  border-bottom: none;
}

.info-label {
  font-weight: 500;
  color: #64748b;
}

.info-value {
  font-weight: 600;
  color: #1e293b;
}

.description {
  color: #64748b;
  line-height: 1.6;
  margin-bottom: 1rem;
}

.additional-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.chart-container {
  text-align: center;
}

.chart-container h3 {
  margin: 0 0 2rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #1e293b;
}

.chart-placeholder {
  background: #f8fafc;
  border-radius: 12px;
  padding: 2rem;
  border: 1px solid #e2e8f0;
  min-height: 400px;
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.chart-mock {
  width: 100%;
  height: 300px;
  position: relative;
  background: linear-gradient(to right, #f1f5f9 0%, #f1f5f9 100%);
  border-radius: 8px;
  overflow: hidden;
}

.chart-line {
  position: relative;
  width: 100%;
  height: 100%;
}

.chart-point {
  position: absolute;
  width: 4px;
  height: 4px;
  background: #6366f1;
  border-radius: 50%;
  transform: translate(-50%, 50%);
}

.chart-note {
  margin-top: 1rem;
  color: #64748b;
  font-style: italic;
}

.news-tab h3 {
  margin: 0 0 1.5rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #1e293b;
}

.news-list {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.news-item {
  background: #f8fafc;
  border-radius: 12px;
  padding: 1.5rem;
  border: 1px solid #e2e8f0;
  transition: all 0.2s ease;
}

.news-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.news-title {
  margin: 0 0 0.5rem 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: #1e293b;
  line-height: 1.4;
}

.news-summary {
  margin: 0 0 1rem 0;
  color: #64748b;
  line-height: 1.5;
}

.news-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.875rem;
  color: #64748b;
}

.news-source {
  font-weight: 500;
}

.fundamentals-tab h3 {
  margin: 0 0 1.5rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: #1e293b;
}

.fundamentals-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 2rem;
}

.fundamental-card {
  background: #f8fafc;
  border-radius: 12px;
  padding: 1.5rem;
  border: 1px solid #e2e8f0;
}

.fundamental-card h4 {
  margin: 0 0 1rem 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: #1e293b;
}

.metric-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.metric-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid #e2e8f0;
}

.metric-item:last-child {
  border-bottom: none;
}

.metric-label {
  font-weight: 500;
  color: #64748b;
}

.metric-value {
  font-weight: 600;
  color: #1e293b;
}

/* Responsive Design */
@media (max-width: 1024px) {
  .header-content {
    flex-direction: column;
    align-items: stretch;
    gap: 1.5rem;
  }
  
  .symbol-info {
    justify-content: center;
  }
  
  .price-section {
    text-align: center;
  }
  
  .header-actions {
    flex-direction: row;
    justify-content: center;
  }
  
  .overview-grid {
    grid-template-columns: 1fr;
  }
  
  .fundamentals-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .details-page {
    padding: 1rem;
  }
  
  .header-section {
    padding: 1.5rem;
  }
  
  .symbol-name {
    font-size: 1.5rem;
  }
  
  .current-price {
    font-size: 2rem;
  }
  
  .tab-navigation {
    overflow-x: auto;
  }
  
  .tab-button {
    white-space: nowrap;
    padding: 1rem 1.5rem;
  }
  
  .tab-content {
    padding: 1.5rem;
  }
  
  .info-grid {
    grid-template-columns: 1fr;
  }
  
  .additional-info {
    grid-template-columns: 1fr;
  }
  
  .chart-mock {
    height: 200px;
  }
}
</style>