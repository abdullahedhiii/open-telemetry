<script setup>
import { tracer } from '../tracing.js'
import { ref } from 'vue'
import {context,propagation,trace} from '@opentelemetry/api'

const symbols = ref(null)
const error = ref(null)

async function fetchSymbols() {
    error.value = null
    symbols.value = null
    const span = tracer.startSpan('fetchStockSymbols')
    const ctx = trace.setSpan(context.active(), span)

    try {
        const headers = {}
        propagation.inject(ctx, headers)
        headers['Content-Type'] = 'application/json'
        const apiUrl = import.meta.env.VITE_API_URL || ""
        span.setAttribute('api.url', apiUrl + '/stocks/symbols')

        const response = await fetch(`${apiUrl}/stocks/symbols`,{
            method : 'GET',
            headers : headers
        })
        span.setAttribute('http.status_code', response.status)

        if (!response.ok) throw new Error("Network response was not ok")
        const data = await response.json()
        symbols.value = JSON.stringify(data, null, 2)
        span.setStatus({ code: 1 }) 
    } catch (err) {
        error.value = err.message
        span.setStatus({ code: 2, message: err.message }) 
    } finally {
        span.end()
    }
}
</script>

<template>
    <div>
        <button @click="fetchSymbols">Fetch Crypto Symbols</button>
        <div v-if="symbols">
            <h3>Response:</h3>
            <pre>{{ symbols }}</pre>
        </div>
        <div v-if="error" style="color: red;">
            Error: {{ error }}
        </div>
    </div>
</template>