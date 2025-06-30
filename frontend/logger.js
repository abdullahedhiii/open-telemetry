import { propagation, context, trace } from '@opentelemetry/api'

export async function logFrontendEvent({ event,type, metadata }) {
  const userData = JSON.parse(localStorage.getItem('userData') || '{}')
  const span = trace.getActiveSpan() || trace.getTracer('app').startSpan('frontend_log_event')

  const ctx = trace.setSpan(context.active(), span)
  const headers = {}

  propagation.inject(ctx, headers)

  headers['Content-Type'] = 'application/json'

  await fetch(import.meta.env.VITE_API_URL + '/log-event', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      type,
      event,
      timestamp: Date.now(),
      metadata
    })
  })

  span.end()
}
