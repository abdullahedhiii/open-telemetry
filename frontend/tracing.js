// import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
// import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
// import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
// import { trace } from '@opentelemetry/api';

// console.log('🔧 Starting Clean OpenTelemetry Setup...');

// const resourceAttributes = {
//   'service.name': import.meta.env.VITE_FRONTEND_SERVICE_NAME || 'frontend-service',
//   'service.version': '1.0.0',
//   'deployment.environment': 'development',
// };

// // Do NOT pass resource here!
// const provider = new WebTracerProvider();

// const otlpEndpoint = import.meta.env.VITE_OTEL_COLLECTOR_URL || 'http://localhost:4318/v1/traces';

// console.log('🌐 Using OTLP endpoint:', otlpEndpoint);

// const otlpExporter = new OTLPTraceExporter({
//   url: otlpEndpoint,
//   headers: {
//     'Content-Type': 'application/json',
//   },
//   timeoutMillis: 10000,
// });

// const otlpProcessor = new BatchSpanProcessor(otlpExporter, {
//   maxExportBatchSize: 5,
//   scheduledDelayMillis: 1000,
//   exportTimeoutMillis: 5000,
//   maxQueueSize: 100,
// });

// provider.addSpanProcessor(otlpProcessor);
// provider.register();

// console.log('OpenTelemetry initialized successfully');


// export const flushTraces = async () => {
//   console.log(' Flushing traces...');
//   try {
//     await provider.forceFlush();
//     console.log('Traces flushed successfully');
//     return true;
//   } catch (error) {
//     console.error('Error flushing traces:', error);
//     return false;
//   }
// };

// export const getTracer = (name = 'default-tracer') => {
//   return trace.getTracer(name);
// };

// export const runTest = () => {
//   console.log('Running OpenTelemetry test...');
  
//   const tracer = getTracer('test-tracer');
//   const span = tracer.startSpan('initialization_test', {
//     attributes: {
//       'test.type': 'initialization',
//       'test.timestamp': new Date().toISOString(),
//       'test.environment': import.meta.env.MODE || 'unknown'
//     }
//   });
  
//   console.log('🧪 Test span created:', {
//     traceId: span.spanContext().traceId,
//     spanId: span.spanContext().spanId
//   });
  
//   span.addEvent('test_started', {
//     'event.timestamp': new Date().toISOString()
//   });
  
//   setTimeout(() => {
//     span.setAttributes({
//       'test.status': 'completed',
//       'test.duration_ms': 1000
//     });
    
//     span.addEvent('test_completed');
//     span.end();
    
//     console.log('🧪 Test span completed');
    
//     // Flush after test
//     setTimeout(() => flushTraces(), 100);
//   }, 1000);
  
//   return span;
// };

// if (import.meta.env.DEV) {
//   setTimeout(() => {
//     runTest();
//   }, 2000);
// }

// if (import.meta.env.DEV && typeof window !== 'undefined') {
//   window.opentelemetry = {
//     provider,
//     flushTraces,
//     runTest,
//     getTracer,
//     endpoint: otlpEndpoint,
//     resourceAttributes
//   };
//   console.log('🪟 OpenTelemetry debug tools available at window.opentelemetry');
// }

// // Export provider for advanced usage
// export { provider };

// tracing.js
import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { trace } from '@opentelemetry/api';

console.log('🔧 OpenTelemetry tracing initializing...');

export const initTracing = async () => {
  const { ZoneContextManager } = await import('@opentelemetry/context-zone');
  const { Resource } = await import('@opentelemetry/resources');
  const {
    SEMRESATTRS_SERVICE_NAME,
    SEMRESATTRS_SERVICE_VERSION,
  } = await import('@opentelemetry/semantic-conventions');

  const serviceName = import.meta.env.VITE_FRONTEND_SERVICE_NAME || 'frontend';
  const exporterUrl = import.meta.env.VITE_OTEL_COLLECTOR_URL || 'http://localhost:4318/v1/traces';

  const provider = new WebTracerProvider({
    resource: new Resource({
      [SEMRESATTRS_SERVICE_NAME]: serviceName,
      [SEMRESATTRS_SERVICE_VERSION]: '1.0.0',
    }),
  });

  const exporter = new OTLPTraceExporter({ url: exporterUrl });
  provider.addSpanProcessor(new BatchSpanProcessor(exporter));

  const contextManager = new ZoneContextManager();
  provider.register({ contextManager });

  console.log('✅ Tracing initialized for', serviceName);

  return provider;
};
