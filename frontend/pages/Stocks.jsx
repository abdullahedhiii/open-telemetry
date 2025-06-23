import React, { useEffect, useState } from 'react';
import {Box, Typography, Table,TableBody,TableCell,TableContainer,TableHead,TableRow,Paper,Button,Link,} from '@mui/material';


export default function Stocks() {

const [watchList, setWatchList] = useState([]);
const [stockSymbols,setSymbols] = useState(null)
const handleAddToWatchList = (symbol) => {
    if (!watchList.includes(symbol)) {
        setWatchList([...watchList, symbol]);
    }
};

useEffect(() => {
  const fetchSymbols = async () => {    
    try {
        const response = await fetch(`${import.meta.env.VITE_API_URL}/stocks/symbols`);
        if (!response.ok) throw new Error('Failed to fetch stock symbols');
        const data = await response.json();
        setSymbols(data);
    } catch (error) {
        console.error('Error fetching stock symbols:', error);
    }
  }  
  fetchSymbols()
},[]);

return (
    <Box sx={{ p: 3 }}>
        <Typography variant="h4" gutterBottom>
            Stock Symbols
        </Typography>
        <TableContainer
            component={Paper}
            sx={{ maxHeight: 400, overflow: 'auto' }}
        >
            <Table stickyHeader>
                <TableHead>
                    <TableRow>
                        <TableCell>Symbol</TableCell>
                        <TableCell>View Data</TableCell>
                        <TableCell>Add to Watch List</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {stockSymbols && stockSymbols.map((stock) => (
                        <TableRow key={stock.symbol}>
                            <TableCell>{stock.symbol}</TableCell>
                            <TableCell>
                                <Link href={`/stocks/${stock.symbol}`} underline="hover">
                                    View Data
                                </Link>
                            </TableCell>
                            <TableCell>
                                <Button
                                    variant="contained"
                                    size="small"
                                    disabled={watchList.includes(stock.symbol)}
                                    onClick={() => handleAddToWatchList(stock.symbol)}
                                >
                                    {watchList.includes(stock.symbol)
                                        ? 'Added'
                                        : 'Add to Watch List'}
                                </Button>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </TableContainer>
    </Box>
);
}