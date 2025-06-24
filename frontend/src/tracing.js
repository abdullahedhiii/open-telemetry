import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { SimpleSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { getWebAutoInstrumentations } from '@opentelemetry/auto-instrumentations-web';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import {Resource} from '@opentelemetry/resources';

const provider = new WebTracerProvider({
  resource: Resource.default().merge(new Resource({
    'service.name': 'stock-tracker-frontend',
  })),
});

const exporter = new OTLPTraceExporter({
  url: 'http://localhost:4318/v1/traces',
});

provider.addSpanProcessor(new SimpleSpanProcessor(exporter));
provider.register({
  contextManager: new ZoneContextManager(),
});

registerInstrumentations({
  instrumentations: [
    getWebAutoInstrumentations(),
  ],
});

// provider.resource = {
//   'service.name': 'stock-tracker-frontend',
// };
export const tracer = provider.getTracer('stock-tracker-frontend');
console.log('Tracing service started');