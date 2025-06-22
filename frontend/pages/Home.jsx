import React, { useState } from 'react';
import { Container, Typography, Button, Box, CircularProgress, Alert } from '@mui/material';

// Home Page Component
const Home = () => {
    const [loading, setLoading] = useState(false);
    const [symbols, setSymbols] = useState(null);
    const [error, setError] = useState(null);

    const handleCheckBackend = async () => {
        setLoading(true);
        setError(null);
        setSymbols(null);
        try {
            const response = await fetch(`${import.meta.env.VITE_API_URL}/stocks/symbols`);
            if (!response.ok) throw new Error('Backend error');
            const data = await response.json();
            setSymbols(data);
            console.log(data)
        } catch (err) {
            console.log(err)
            setError('Failed to fetch symbols from backend.');
        } finally {
            setLoading(false);
        }
    };

    return (
        <Container maxWidth="sm" sx={{ mt: 8, textAlign: 'center' }}>
            <Typography variant="h3" gutterBottom>
                Stock & Crypto Demo
            </Typography>
            <Typography variant="h6" color="text.secondary" gutterBottom>
                Welcome! Explore real-time stock and crypto data.
            </Typography>
            <Box sx={{ mt: 4 }}>
                <Button
                    variant="contained"
                    color="primary"
                    size="large"
                    onClick={handleCheckBackend}
                    disabled={loading}
                >
                    {loading ? <CircularProgress size={24} color="inherit" /> : 'Check Backend'}
                </Button>
            </Box>
            <Box sx={{ mt: 4 }}>
                {error && <Alert severity="error">{error}</Alert>}
                {symbols && (
                    <Alert severity="success">
                        <strong>Backend Response:</strong>
                        <pre style={{ textAlign: 'left', margin: 0 }}>{JSON.stringify(symbols, null, 2)}</pre>
                    </Alert>
                )}
            </Box>
        </Container>
    );
};

export default Home;