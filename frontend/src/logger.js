import { propagation, context, trace } from '@opentelemetry/api'

export async function logFrontendEvent({ event, type, metadata, span }) {
  const headers = {}
  const ctx = span ? trace.setSpan(context.active(), span) : context.active()
 
  console.log(span ? 'yes' : 'no')
  propagation.inject(ctx, headers)
  headers['Content-Type'] = 'application/json'

  console.log("Sending request to backend -post log")

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

  console.log("Request sent to backend -post log")
}
