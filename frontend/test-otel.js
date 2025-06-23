// test-otel-collector.js - Direct test of OTEL collector endpoint
// Run this in your browser console or as a separate test

const testOTLPCollector = async () => {
  const endpoint = 'http://localhost:4318/v1/traces'; // Adjust if needed
  
  // Sample OTLP trace data
  const testTrace = {
    resourceSpans: [
      {
        resource: {
          attributes: [
            {
              key: "service.name",
              value: { stringValue: "test-frontend" }
            },
            {
              key: "service.version", 
              value: { stringValue: "1.0.0" }
            }
          ]
        },
        scopeSpans: [
          {
            scope: {
              name: "test-tracer",
              version: "1.0.0"
            },
            spans: [
              {
                traceId: "12345678901234567890123456789012",
                spanId: "1234567890123456",
                name: "test_span",
                kind: 1,
                startTimeUnixNano: String(Date.now() * 1000000),
                endTimeUnixNano: String((Date.now() + 1000) * 1000000),
                attributes: [
                  {
                    key: "test.type",
                    value: { stringValue: "collector_test" }
                  },
                  {
                    key: "test.timestamp",
                    value: { stringValue: new Date().toISOString() }
                  }
                ],
                status: {
                  code: 1 // STATUS_CODE_OK
                }
              }
            ]
          }
        ]
      }
    ]
  };

  console.log('🧪 Testing OTLP collector endpoint:', endpoint);
  console.log('📤 Sending test trace:', testTrace);

  try {
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      body: JSON.stringify(testTrace)
    });

    console.log('📝 Response status:', response.status);
    console.log('📝 Response headers:', [...response.headers.entries()]);

    if (response.ok) {
      console.log('✅ OTLP collector is working!');
      const responseText = await response.text();
      console.log('📄 Response body:', responseText);
    } else {
      console.error('❌ OTLP collector error:', response.status, response.statusText);
      const errorText = await response.text();
      console.error('📄 Error response:', errorText);
    }

  } catch (error) {
    console.error('❌ Network error connecting to OTLP collector:', error);
    
    if (error.name === 'TypeError' && error.message.includes('CORS')) {
      console.error('🚫 CORS error - check your OTLP collector CORS configuration');
    }
  }
};

// Test the collector immediately
testOTLPCollector();

// Also make it available globally for manual testing
if (typeof window !== 'undefined') {
  window.testOTLPCollector = testOTLPCollector;
  console.log('🪟 testOTLPCollector() function available globally');
}

// Additional diagnostic function
const checkOTLPEndpoint = async () => {
  const endpoints = [
    'http://localhost:4318/v1/traces',
    'http://localhost:4317/v1/traces', // gRPC port, might not work with HTTP
    'http://127.0.0.1:4318/v1/traces',
    'http://otel-collector:4318/v1/traces' // Docker container name
  ];

  for (const endpoint of endpoints) {
    console.log(`🔍 Checking endpoint: ${endpoint}`);
    try {
      const response = await fetch(endpoint, {
        method: 'OPTIONS', // Preflight request
        headers: {
          'Access-Control-Request-Method': 'POST',
          'Access-Control-Request-Headers': 'Content-Type'
        }
      });
      console.log(`✅ ${endpoint} - Status: ${response.status}`);
    } catch (error) {
      console.log(`❌ ${endpoint} - Error: ${error.message}`);
    }
  }
};

if (typeof window !== 'undefined') {
  window.checkOTLPEndpoint = checkOTLPEndpoint;
}