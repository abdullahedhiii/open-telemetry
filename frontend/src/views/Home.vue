<script setup>
import { tracer } from '../tracing.js'
import { ref, computed, onMounted } from 'vue'
import { context, propagation, trace } from '@opentelemetry/api'

const isLoginMode = ref(true)
const loading = ref(false)
const error = ref(null)
const success = ref(null)

const loginForm = ref({
  email: '',
  password: ''
})

const registerForm = ref({
  username: '',
  email: '',
  password: '',
  confirmPassword: ''
})

const loginErrors = ref({})
const registerErrors = ref({})

const isLoginValid = computed(() => {
  return loginForm.value.email && 
         loginForm.value.password && 
         Object.keys(loginErrors.value).length === 0
})

const isRegisterValid = computed(() => {
  return registerForm.value.username && 
         registerForm.value.email && 
         registerForm.value.password && 
         registerForm.value.confirmPassword &&
         registerForm.value.password === registerForm.value.confirmPassword &&
         Object.keys(registerErrors.value).length === 0
})

function validateEmail(email) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}

function validatePassword(password) {
  return password.length >= 6
}

function validateUsername(username) {
  return username.length >= 3 && /^[a-zA-Z0-9_]+$/.test(username)
}


function validateLoginForm() {
  const span = tracer.startSpan('validate_login_form', {
    attributes: {
      'form.type': 'login',
      'validation.trigger': 'real_time'
    }
  })
  
  try {
    loginErrors.value = {}
    
    if (loginForm.value.email && !validateEmail(loginForm.value.email)) {
      loginErrors.value.email = 'Please enter a valid email address'
    }
    
    if (loginForm.value.password && !validatePassword(loginForm.value.password)) {
      loginErrors.value.password = 'Password must be at least 6 characters'
    }
    
    span.setAttributes({
      'validation.errors_count': Object.keys(loginErrors.value).length,
      'validation.is_valid': Object.keys(loginErrors.value).length === 0
    })
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}

function validateRegisterForm() {
  const span = tracer.startSpan('validate_register_form', {
    attributes: {
      'form.type': 'register',
      'validation.trigger': 'real_time'
    }
  })
  
  try {
    registerErrors.value = {}
    
    if (registerForm.value.username && !validateUsername(registerForm.value.username)) {
      registerErrors.value.username = 'Username must be at least 3 characters and contain only letters, numbers, and underscores'
    }
    
    if (registerForm.value.email && !validateEmail(registerForm.value.email)) {
      registerErrors.value.email = 'Please enter a valid email address'
    }
    
    if (registerForm.value.password && !validatePassword(registerForm.value.password)) {
      registerErrors.value.password = 'Password must be at least 6 characters'
    }
    
    if (registerForm.value.confirmPassword && registerForm.value.password !== registerForm.value.confirmPassword) {
      registerErrors.value.confirmPassword = 'Passwords do not match'
    }
    
    span.setAttributes({
      'validation.errors_count': Object.keys(registerErrors.value).length,
      'validation.is_valid': Object.keys(registerErrors.value).length === 0
    })
    
    span.setStatus({ code: 1 })
  } catch (err) {
    span.setStatus({ code: 2, message: err.message })
  } finally {
    span.end()
  }
}


async function handleLogin() {
  const span = tracer.startSpan('user_login', {
    attributes: {
      'auth.action': 'login',
      'user.email': loginForm.value.email,
      'form.type': 'login'
    }
  })
  
  try {
    loading.value = true
    error.value = null
    success.value = null
    
    const startTime = performance.now()
    
    
    validateLoginForm()
    if (!isLoginValid.value) {
      throw new Error('Please fix the form errors before submitting')
    }
    
    const apiUrl = import.meta.env.VITE_API_URL || ""
    const endpoint = `${apiUrl}/users/login`
    
    const headers = {}
    propagation.inject(context.active(), headers)
    headers['Content-Type'] = 'application/json'
    
    span.setAttributes({
      'api.endpoint': endpoint,
      'http.method': 'POST'
    })
    
    const requestData = {
      email: loginForm.value.email,
      password: loginForm.value.password
    }
    
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestData)
    })
    
    const duration = performance.now() - startTime
    
    span.setAttributes({
      'http.status_code': response.status,
      'http.response_time_ms': duration
    })
    
    const data = await response.json()
    
    if (!response.ok) {
      throw new Error(data.message || `Login failed: ${response.status}`)
    }
    
    
    if (data.token) {
      localStorage.setItem('authToken', data.token)
    }
    if (data.user) {
      localStorage.setItem('userData', JSON.stringify(data.user))
    }
    
    success.value = 'Login successful! Redirecting...'
    
    span.setAttributes({
      'auth.success': true,
      'user.authenticated': true,
      'response.has_token': !!data.token
    })
    
    
    setTimeout(() => {
      alert('Login successful! Would redirect to dashboard.')
    }, 1500)
    
    span.setStatus({ code: 1 })
    
  } catch (err) {
    error.value = err.message
    
    span.setAttributes({
      'auth.success': false,
      'error.message': err.message,
      'error.type': err.constructor.name
    })
    
    span.setStatus({ code: 2, message: err.message })
  } finally {
    loading.value = false
    span.end()
  }
}

async function handleRegister() {
  const span = tracer.startSpan('user_register', {
    attributes: {
      'auth.action': 'register',
      'user.email': registerForm.value.email,
      'user.username': registerForm.value.username,
      'form.type': 'register'
    }
  })
  
  try {
    loading.value = true
    error.value = null
    success.value = null
    
    const startTime = performance.now()
    
    
    validateRegisterForm()
    if (!isRegisterValid.value) {
      throw new Error('Please fix the form errors before submitting')
    }
    
    const apiUrl = import.meta.env.VITE_API_URL || ""
    const endpoint = `${apiUrl}/users/register`
    
    const headers = {}
    propagation.inject(context.active(), headers)
    headers['Content-Type'] = 'application/json'
    
    span.setAttributes({
      'api.endpoint': endpoint,
      'http.method': 'POST'
    })
    
    const requestData = {
      username: registerForm.value.username,
      email: registerForm.value.email,
      password: registerForm.value.password
    }
    
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestData)
    })
    
    const duration = performance.now() - startTime
    
    span.setAttributes({
      'http.status_code': response.status,
      'http.response_time_ms': duration
    })
    
    const data = await response.json()
    
    if (!response.ok) {
      throw new Error(data.message || `Registration failed: ${response.status}`)
    }
    
    success.value = 'Registration successful! You can now log in.'
    
    
    setTimeout(() => {
      isLoginMode.value = true
      registerForm.value = {
        username: '',
        email: '',
        password: '',
        confirmPassword: ''
      }
    }, 2000)
    
    span.setAttributes({
      'auth.success': true,
      'user.registered': true
    })
    
    span.setStatus({ code: 1 })
    
  } catch (err) {
    error.value = err.message
    
    span.setAttributes({
      'auth.success': false,
      'error.message': err.message,
      'error.type': err.constructor.name
    })
    
    span.setStatus({ code: 2, message: err.message })
  } finally {
    loading.value = false
    span.end()
  }
}

function toggleMode() {
  const span = tracer.startSpan('toggle_auth_mode', {
    attributes: {
      'previous.mode': isLoginMode.value ? 'login' : 'register',
      'new.mode': isLoginMode.value ? 'register' : 'login',
      'user.action': 'mode_switch'
    }
  })
  
  try {
    isLoginMode.value = !isLoginMode.value
    error.value = null
    success.value = null
    
    
    loginErrors.value = {}
    registerErrors.value = {}
    
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


onMounted(() => {
  const span = tracer.startSpan('home_page_mounted', {
    attributes: {
      'component': 'home_page',
      'lifecycle.event': 'mounted',
      'initial.mode': 'login'
    }
  })
  
  
  const token = localStorage.getItem('authToken')
  if (token) {
    span.setAttributes({
      'user.already_authenticated': true
    })
    
  }
  
  span.setStatus({ code: 1 })
  span.end()
})
</script>

<template>
  <div class="home-page">
    <div class="hero-section">
      <div class="hero-content">
        <div class="hero-text">
          <!-- <h1 class="hero-title">
            Track Your Financial Future
          </h1> -->
          <p class="hero-subtitle">
            Monitor stocks and cryptocurrencies with real-time data, advanced analytics, and personalized watchlists.
          </p>
          <!-- <div class="hero-features">
            <div class="feature-item">
              <span class="feature-icon">📈</span>
              <span>Real-time Market Data</span>
            </div>
            <div class="feature-item">
              <span class="feature-icon">⚡</span>
              <span>Lightning Fast Updates</span>
            </div>
            <div class="feature-item">
              <span class="feature-icon">🔒</span>
              <span>Secure & Private</span>
            </div>
          </div> -->
        </div>
        
        <div class="auth-section">
          <div class="auth-card">
            <div class="auth-header">
              <h2 class="auth-title">
                {{ isLoginMode ? 'Welcome Back' : 'Create Account' }}
              </h2>
              <p class="auth-subtitle">
                {{ isLoginMode ? 'Sign in to your account' : 'Join thousands of traders' }}
              </p>
            </div>
            
            <!-- Success Message -->
            <div v-if="success" class="success-message">
              <div class="success-icon">✅</div>
              <p>{{ success }}</p>
            </div>
            
            <!-- Error Message -->
            <div v-if="error" class="error-message">
              <div class="error-icon">⚠️</div>
              <p>{{ error }}</p>
            </div>
            
            <!-- Login Form -->
            <form v-if="isLoginMode" @submit.prevent="handleLogin" class="auth-form">
              <div class="form-group">
                <label for="login-email" class="form-label">Email Address</label>
                <input
                  id="login-email"
                  v-model="loginForm.email"
                  @input="validateLoginForm"
                  type="email"
                  class="form-input"
                  :class="{ 'error': loginErrors.email }"
                  placeholder="Enter your email"
                  required
                />
                <span v-if="loginErrors.email" class="field-error">{{ loginErrors.email }}</span>
              </div>
              
              <div class="form-group">
                <label for="login-password" class="form-label">Password</label>
                <input
                  id="login-password"
                  v-model="loginForm.password"
                  @input="validateLoginForm"
                  type="password"
                  class="form-input"
                  :class="{ 'error': loginErrors.password }"
                  placeholder="Enter your password"
                  required
                />
                <span v-if="loginErrors.password" class="field-error">{{ loginErrors.password }}</span>
              </div>
              
              <button
                type="submit"
                :disabled="loading || !isLoginValid"
                class="auth-button primary"
              >
                <span v-if="loading" class="loading-spinner"></span>
                {{ loading ? 'Signing In...' : 'Sign In' }}
              </button>
            </form>
            
            <!-- Register Form -->
            <form v-else @submit.prevent="handleRegister" class="auth-form">
              <div class="form-group">
                <label for="register-username" class="form-label">Username</label>
                <input
                  id="register-username"
                  v-model="registerForm.username"
                  @input="validateRegisterForm"
                  type="text"
                  class="form-input"
                  :class="{ 'error': registerErrors.username }"
                  placeholder="Choose a username"
                  required
                />
                <span v-if="registerErrors.username" class="field-error">{{ registerErrors.username }}</span>
              </div>
              
              <div class="form-group">
                <label for="register-email" class="form-label">Email Address</label>
                <input
                  id="register-email"
                  v-model="registerForm.email"
                  @input="validateRegisterForm"
                  type="email"
                  class="form-input"
                  :class="{ 'error': registerErrors.email }"
                  placeholder="Enter your email"
                  required
                />
                <span v-if="registerErrors.email" class="field-error">{{ registerErrors.email }}</span>
              </div>
              
              <div class="form-group">
                <label for="register-password" class="form-label">Password</label>
                <input
                  id="register-password"
                  v-model="registerForm.password"
                  @input="validateRegisterForm"
                  type="password"
                  class="form-input"
                  :class="{ 'error': registerErrors.password }"
                  placeholder="Create a password"
                  required
                />
                <span v-if="registerErrors.password" class="field-error">{{ registerErrors.password }}</span>
              </div>
              
              <div class="form-group">
                <label for="register-confirm-password" class="form-label">Confirm Password</label>
                <input
                  id="register-confirm-password"
                  v-model="registerForm.confirmPassword"
                  @input="validateRegisterForm"
                  type="password"
                  class="form-input"
                  :class="{ 'error': registerErrors.confirmPassword }"
                  placeholder="Confirm your password"
                  required
                />
                <span v-if="registerErrors.confirmPassword" class="field-error">{{ registerErrors.confirmPassword }}</span>
              </div>
              
              <button
                type="submit"
                :disabled="loading || !isRegisterValid"
                class="auth-button primary"
              >
                <span v-if="loading" class="loading-spinner"></span>
                {{ loading ? 'Creating Account...' : 'Create Account' }}
              </button>
            </form>
            
            <!-- Mode Toggle -->
            <div class="auth-footer">
              <p class="toggle-text">
                {{ isLoginMode ? "Don't have an account?" : "Already have an account?" }}
              </p>
              <button @click="toggleMode" class="auth-button secondary">
                {{ isLoginMode ? 'Create Account' : 'Sign In' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Features Section -->
    <div class="features-section">
      <div class="features-content">
        <h2 class="features-title">Why Choose Our Platform?</h2>
        <div class="features-grid">
          <div class="feature-card">
            <div class="feature-icon-large">📊</div>
            <h3>Real-Time Data</h3>
            <p>Get live market data for stocks and cryptocurrencies with minimal latency.</p>
          </div>
          
          <div class="feature-card">
            <div class="feature-icon-large">🎯</div>
            <h3>Smart Watchlists</h3>
            <p>Create and manage personalized watchlists to track your favorite assets.</p>
          </div>
          
          <div class="feature-card">
            <div class="feature-icon-large">📈</div>
            <h3>Advanced Analytics</h3>
            <p>Comprehensive charts and technical analysis tools for informed decisions.</p>
          </div>
          
          <div class="feature-card">
            <div class="feature-icon-large">🔔</div>
            <h3>Price Alerts</h3>
            <p>Set custom alerts and never miss important market movements.</p>
          </div>
          
          <div class="feature-card">
            <div class="feature-icon-large">📱</div>
            <h3>Mobile Ready</h3>
            <p>Access your portfolio and market data from any device, anywhere.</p>
          </div>
          
          <div class="feature-card">
            <div class="feature-icon-large">🛡️</div>
            <h3>Bank-Level Security</h3>
            <p>Your data is protected with enterprise-grade security measures.</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home-page {
  min-height: 100vh;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background: linear-gradient(135deg, #b0b3c2 0%, #d1c7db 100%);
}

.hero-section {
  min-height: 100vh;
  display: flex;
  align-items: center;
  padding: 2rem;
}

.hero-content {
  max-width: 1400px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4rem;
  align-items: center;
}

.hero-text {
  color: white;
}

.hero-title {
  font-size: 3.5rem;
  font-weight: 800;
  line-height: 1.1;
  margin: 0 0 1.5rem 0;
  letter-spacing: -0.025em;
}

.hero-subtitle {
  font-size: 1.25rem;
  line-height: 1.6;
  margin: 0 0 2rem 0;
  opacity: 0.9;
}

.hero-features {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  font-size: 1.125rem;
  font-weight: 500;
}

.feature-icon {
  font-size: 1.5rem;
}

.auth-section {
  display: flex;
  justify-content: center;
}

.auth-card {
  background: white;
  border-radius: 20px;
  padding: 2.5rem;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
  width: 100%;
  max-width: 400px;
}

.auth-header {
  text-align: center;
  margin-bottom: 2rem;
}

.auth-title {
  font-size: 2rem;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 0.5rem 0;
}

.auth-subtitle {
  color: #64748b;
  margin: 0;
  font-size: 1rem;
}

.success-message, .error-message {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 12px;
  margin-bottom: 1.5rem;
  font-weight: 500;
}

.success-message {
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
}

.error-message {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.success-icon, .error-icon {
  font-size: 1.25rem;
  flex-shrink: 0;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-weight: 600;
  color: #374151;
  font-size: 0.875rem;
}

.form-input {
  padding: 0.875rem 1rem;
  border: 2px solid #e5e7eb;
  border-radius: 12px;
  font-size: 1rem;
  transition: all 0.2s ease;
  background: #f9fafb;
}

.form-input:focus {
  outline: none;
  border-color: #667eea;
  background: white;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.form-input.error {
  border-color: #dc2626;
  background: #fef2f2;
}

.field-error {
  color: #dc2626;
  font-size: 0.875rem;
  font-weight: 500;
}

.auth-button {
  padding: 0.875rem 1.5rem;
  border-radius: 12px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: none;
}

.auth-button.primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.auth-button.primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.4);
}

.auth-button.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.auth-button.secondary {
  background: transparent;
  color: #667eea;
  border: 2px solid #667eea;
}

.auth-button.secondary:hover {
  background: #667eea;
  color: white;
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

.auth-footer {
  margin-top: 2rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.toggle-text {
  color: #64748b;
  margin: 0;
  font-size: 0.875rem;
}

.features-section {
  background: white;
  padding: 5rem 2rem;
}

.features-content {
  max-width: 1400px;
  margin: 0 auto;
}

.features-title {
  text-align: center;
  font-size: 2.5rem;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 3rem 0;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
}

.feature-card {
  background: #f8fafc;
  border-radius: 16px;
  padding: 2rem;
  text-align: center;
  border: 1px solid #e2e8f0;
  transition: all 0.3s ease;
}

.feature-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.1);
  border-color: #667eea;
}

.feature-icon-large {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.feature-card h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 1rem 0;
}

.feature-card p {
  color: #64748b;
  line-height: 1.6;
  margin: 0;
}

/* Responsive Design */
@media (max-width: 1024px) {
  .hero-content {
    grid-template-columns: 1fr;
    gap: 3rem;
    text-align: center;
  }
  
  .hero-title {
    font-size: 3rem;
  }
  
  .features-grid {
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  }
}

@media (max-width: 768px) {
  .hero-section {
    padding: 1rem;
  }
  
  .hero-title {
    font-size: 2.5rem;
  }
  
  .hero-subtitle {
    font-size: 1.125rem;
  }
  
  .auth-card {
    padding: 2rem;
  }
  
  .features-section {
    padding: 3rem 1rem;
  }
  
  .features-title {
    font-size: 2rem;
  }
  
  .features-grid {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }
  
  .feature-card {
    padding: 1.5rem;
  }
}

@media (max-width: 480px) {
  .hero-title {
    font-size: 2rem;
  }
  
  .auth-card {
    padding: 1.5rem;
  }
  
  .auth-title {
    font-size: 1.5rem;
  }
}
</style>