import { propagation, context, trace } from '@opentelemetry/api'

export async function logFrontendEvent({ event, type, metadata, span }) {
  const headers = {}
  const ctx = span ? trace.setSpan(context.active(), span) : context.active()
 
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
}
