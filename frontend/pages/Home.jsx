import React, { useState, useEffect } from 'react';
import { Container, Typography, Button, Box, CircularProgress, Alert } from '@mui/material';
import { SpanStatusCode } from '@opentelemetry/api';
import { flushTraces, getTracer } from '../tracing'; // Import from clean tracing setup

const Home = () => {
    const [loading, setLoading] = useState(false);
    const [symbols, setSymbols] = useState(null);
    const [error, setError] = useState(null);
    
    const handleCheckBackend = async () => {
        console.log('🚀 Starting backend check...');
        
        const tracer = getTracer('home-component');
        const span = tracer.startSpan('check_backend_operation', {
            attributes: {
                'ui.component': 'Home',
                'operation.name': 'check_backend',
                'user.action': 'button_click',
                'timestamp': new Date().toISOString()
            }
        });
        
        console.log('✨ Backend check span created:', {
            traceId: span.spanContext().traceId,
            spanId: span.spanContext().spanId
        });
        
        setLoading(true);
        setError(null);
        setSymbols(null);

        try {
            const apiUrl = `${import.meta.env.VITE_API_URL}/stocks/symbols`;
            console.log('🌐 Fetching from:', apiUrl);
            
            span.setAttributes({
                'http.method': 'GET',
                'http.url': apiUrl,
                'http.request.started': new Date().toISOString()
            });
            
            const startTime = Date.now();
            const response = await fetch(apiUrl, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const duration = Date.now() - startTime;
            
            console.log('📝 Response received:', {
                status: response.status,
                statusText: response.statusText,
                duration: `${duration}ms`
            });

            span.setAttributes({
                'http.status_code': response.status,
                'http.status_text': response.statusText,
                'http.response.duration_ms': duration,
                'http.response.received': new Date().toISOString()
            });

            if (!response.ok) {
                const errorMsg = `HTTP ${response.status}: ${response.statusText}`;
                span.setStatus({ 
                    code: SpanStatusCode.ERROR, 
                    message: errorMsg 
                });
                throw new Error(errorMsg);
            }

            const data = await response.json();
            const dataSize = JSON.stringify(data).length;
            
            console.log('📊 Data received:', {
                type: typeof data,
                isArray: Array.isArray(data),
                length: Array.isArray(data) ? data.length : 'N/A',
                size: `${dataSize} bytes`
            });
            
            setSymbols(data);

            span.setAttributes({
                'response.success': true,
                'response.data.type': typeof data,
                'response.data.is_array': Array.isArray(data),
                'response.data.length': Array.isArray(data) ? data.length : 0,
                'response.data.size_bytes': dataSize
            });
            
            span.setStatus({ code: SpanStatusCode.OK });
            span.addEvent('data_processed', {
                'processing.timestamp': new Date().toISOString(),
                'processing.items': Array.isArray(data) ? data.length : 0
            });

        } catch (err) {
            console.error('❌ Backend check failed:', err);
            setError(`Failed to fetch symbols: ${err.message}`);

            span.recordException(err);
            span.setAttributes({
                'error.occurred': true,
                'error.name': err.name,
                'error.message': err.message,
                'response.success': false
            });
            
            span.setStatus({ 
                code: SpanStatusCode.ERROR, 
                message: err.message 
            });

        } finally {
            span.addEvent('operation_completed', {
                'completion.timestamp': new Date().toISOString(),
                'completion.success': !error
            });
            
            span.end();
            console.log('✅ Backend check span ended');
            
            setLoading(false);
            
            // Force flush after operation
            setTimeout(async () => {
                console.log('🚀 Flushing traces after backend check...');
                await flushTraces();
            }, 500);
        }
    };

    const handleManualFlush = async () => {
        console.log('🔧 Manual flush requested...');
        const success = await flushTraces();
        if (success) {
            console.log('✅ Manual flush completed successfully');
        } else {
            console.error('❌ Manual flush failed');
        }
    };

    const handleRunTest = () => {
        console.log('🧪 Running manual test...');
        if (window.opentelemetry && window.opentelemetry.runTest) {
            window.opentelemetry.runTest();
        } else {
            console.warn('⚠️ Test function not available');
        }
    };

    useEffect(() => {
        console.log('🏠 Home component mounted');
        
        // Check if OpenTelemetry is properly initialized
        if (window.opentelemetry) {
            console.log('✅ OpenTelemetry is available:', {
                endpoint: window.opentelemetry.endpoint,
                resourceAttributes: window.opentelemetry.resourceAttributes
            });
        } else {
            console.warn('⚠️ OpenTelemetry debug tools not found');
        }
        
        // Create a simple mount test span
        const tracer = getTracer('component-lifecycle');
        const mountSpan = tracer.startSpan('component_mounted', {
            attributes: {
                'component.name': 'Home',
                'lifecycle.event': 'mount',
                'mount.timestamp': new Date().toISOString()
            }
        });
        
        console.log('🏠 Component mount span created');
        
        setTimeout(() => {
            mountSpan.addEvent('mount_complete');
            mountSpan.end();
            console.log('🏠 Component mount span completed');
        }, 100);
        
    }, []);

    return (
        <Container maxWidth="md" sx={{ mt: 4 }}>
            <Box textAlign="center" mb={4}>
                <Typography variant="h3" component="h1" gutterBottom>
                    Stock & Crypto Demo
                </Typography>
                <Typography variant="h6" color="text.secondary">
                    Welcome! Explore real-time stock and crypto data with OpenTelemetry tracing.
                </Typography>
            </Box>

            <Box textAlign="center" mb={4}>
                <Button
                    variant="contained"
                    size="large"
                    onClick={handleCheckBackend}
                    disabled={loading}
                    sx={{ minWidth: 200, mr: 2, mb: 1 }}
                >
                    {loading ? <CircularProgress size={24} color="inherit" /> : 'Check Backend'}
                </Button>
                
                <Button
                    variant="outlined"
                    size="large"
                    onClick={handleManualFlush}
                    sx={{ minWidth: 150, mr: 2, mb: 1 }}
                >
                    Flush Traces
                </Button>
                
                <Button
                    variant="outlined"
                    size="large"
                    onClick={handleRunTest}
                    sx={{ minWidth: 120, mb: 1 }}
                >
                    Run Test
                </Button>
            </Box>

            {error && (
                <Alert severity="error" sx={{ mb: 2 }}>
                    {error}
                </Alert>
            )}

            {symbols && (
                <Box>
                    <Typography variant="h6" gutterBottom>
                        Backend Response:
                    </Typography>
                    <Box
                        component="pre"
                        sx={{
                            backgroundColor: 'grey.100',
                            p: 2,
                            borderRadius: 1,
                            overflow: 'auto',
                            fontSize: '0.875rem',
                            maxHeight: '400px'
                        }}
                    >
                        {JSON.stringify(symbols, null, 2)}
                    </Box>
                </Box>
            )}
            
            <Box sx={{ mt: 4, p: 2, backgroundColor: 'info.light', borderRadius: 1 }}>
                <Typography variant="body2" color="text.secondary">
                    🔍 Open browser console to see OpenTelemetry logs and trace information
                </Typography>
            </Box>
        </Container>
    );
};

export default Home;