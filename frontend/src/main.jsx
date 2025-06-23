import { initTracing } from '../tracing.js';
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'

// FrontendTracer(); 

initTracing().then(() =>
  {createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>
)});

