import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import morgan from 'morgan';
import axios from 'axios';
import dotenv from 'dotenv';

// Load environment variables
dotenv.config();

const app = express();

// Configuration
const PORT = process.env.BRIDGE_PORT || 8088;
const KNIRVORACLE_API = process.env.KNIRVORACLE_API || 'http://localhost:1317';
const KNIRVCHAIN_API = process.env.KNIRVCHAIN_API || 'http://localhost:8090';

// XION Network Configuration
const XION_RPC_ENDPOINT = process.env.XION_RPC_ENDPOINT || 'https://rpc.xion-testnet-1.burnt.com:443';
const XION_CHAIN_ID = process.env.XION_CHAIN_ID || 'xion-testnet-1';

// Bridge state
interface BridgeTransaction {
    id: string;
    sourceChain: string;
    targetChain: string;
    amount: string;
    token: string;
    sender: string;
    recipient: string;
    status: 'pending' | 'confirmed' | 'failed';
    timestamp: string;
    txHash?: string;
}

const bridgeTransactions: Map<string, BridgeTransaction> = new Map();

// Middleware
app.use(helmet());
app.use(cors());
app.use(compression());
app.use(morgan('combined'));
app.use(express.json());

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'XION Bridge Testnet',
        timestamp: new Date().toISOString(),
        version: '1.0.0',
        connections: {
            oracle: KNIRVORACLE_API,
            chain: KNIRVCHAIN_API,
            xion: XION_RPC_ENDPOINT
        },
        bridgeStats: {
            totalTransactions: bridgeTransactions.size,
            pendingTransactions: Array.from(bridgeTransactions.values()).filter(tx => tx.status === 'pending').length
        }
    });
});

// Bridge status endpoint
app.get('/api/bridge/status', (req, res) => {
    res.json({
        status: 'operational',
        supportedChains: ['knirv', 'xion'],
        supportedTokens: ['NRN', 'XION'],
        fees: {
            'knirv-to-xion': '0.001',
            'xion-to-knirv': '0.001'
        },
        timestamp: new Date().toISOString()
    });
});

// Get bridge transaction
app.get('/api/bridge/transaction/:id', (req, res) => {
    const { id } = req.params;
    const transaction = bridgeTransactions.get(id);
    
    if (!transaction) {
        return res.status(404).json({
            status: 'error',
            message: 'Transaction not found',
            timestamp: new Date().toISOString()
        });
    }
    
    res.json({
        status: 'success',
        transaction,
        timestamp: new Date().toISOString()
    });
});

// List bridge transactions
app.get('/api/bridge/transactions', (req, res) => {
    const { address, status, limit = '10' } = req.query;
    let transactions = Array.from(bridgeTransactions.values());
    
    // Filter by address
    if (address) {
        transactions = transactions.filter(tx => 
            tx.sender === address || tx.recipient === address
        );
    }
    
    // Filter by status
    if (status) {
        transactions = transactions.filter(tx => tx.status === status);
    }
    
    // Sort by timestamp (newest first)
    transactions.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
    
    // Limit results
    const limitNum = parseInt(limit as string, 10);
    transactions = transactions.slice(0, limitNum);
    
    res.json({
        status: 'success',
        transactions,
        total: bridgeTransactions.size,
        timestamp: new Date().toISOString()
    });
});

// Initiate bridge transaction
app.post('/api/bridge/transfer', async (req, res) => {
    try {
        const { sourceChain, targetChain, amount, token, sender, recipient } = req.body;
        
        // Validate input
        if (!sourceChain || !targetChain || !amount || !token || !sender || !recipient) {
            return res.status(400).json({
                status: 'error',
                message: 'Missing required fields',
                timestamp: new Date().toISOString()
            });
        }
        
        // Generate transaction ID
        const txId = `bridge_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        
        // Create bridge transaction
        const bridgeTx: BridgeTransaction = {
            id: txId,
            sourceChain,
            targetChain,
            amount,
            token,
            sender,
            recipient,
            status: 'pending',
            timestamp: new Date().toISOString()
        };
        
        // Store transaction
        bridgeTransactions.set(txId, bridgeTx);
        
        // Process bridge transaction based on source chain
        if (sourceChain === 'knirv') {
            await processKnirvToXionTransfer(bridgeTx);
        } else if (sourceChain === 'xion') {
            await processXionToKnirvTransfer(bridgeTx);
        } else {
            bridgeTx.status = 'failed';
            bridgeTransactions.set(txId, bridgeTx);
            return res.status(400).json({
                status: 'error',
                message: 'Unsupported source chain',
                timestamp: new Date().toISOString()
            });
        }
        
        res.json({
            status: 'success',
            transactionId: txId,
            bridgeTransaction: bridgeTx,
            timestamp: new Date().toISOString()
        });
        
    } catch (error) {
        console.error('Error initiating bridge transfer:', error);
        res.status(500).json({
            status: 'error',
            message: 'Failed to initiate bridge transfer',
            timestamp: new Date().toISOString()
        });
    }
});

// Process KNIRV to XION transfer
async function processKnirvToXionTransfer(bridgeTx: BridgeTransaction): Promise<void> {
    try {
        console.log(`Processing KNIRV to XION transfer: ${bridgeTx.id}`);
        
        // Lock tokens on KNIRV chain
        const lockResponse = await axios.post(`${KNIRVCHAIN_API}/api/bridge/lock`, {
            amount: bridgeTx.amount,
            token: bridgeTx.token,
            sender: bridgeTx.sender,
            bridgeId: bridgeTx.id
        });
        
        if (lockResponse.data.success) {
            bridgeTx.txHash = lockResponse.data.txHash;
            
            // Simulate XION chain minting (in real implementation, this would interact with XION)
            setTimeout(() => {
                bridgeTx.status = 'confirmed';
                bridgeTransactions.set(bridgeTx.id, bridgeTx);
                console.log(`Bridge transfer confirmed: ${bridgeTx.id}`);
            }, 5000);
        } else {
            bridgeTx.status = 'failed';
            bridgeTransactions.set(bridgeTx.id, bridgeTx);
        }
        
    } catch (error) {
        console.error(`Error processing KNIRV to XION transfer ${bridgeTx.id}:`, error);
        bridgeTx.status = 'failed';
        bridgeTransactions.set(bridgeTx.id, bridgeTx);
    }
}

// Process XION to KNIRV transfer
async function processXionToKnirvTransfer(bridgeTx: BridgeTransaction): Promise<void> {
    try {
        console.log(`Processing XION to KNIRV transfer: ${bridgeTx.id}`);
        
        // Simulate XION chain burning (in real implementation, this would interact with XION)
        // Then mint on KNIRV chain
        const mintResponse = await axios.post(`${KNIRVCHAIN_API}/api/bridge/mint`, {
            amount: bridgeTx.amount,
            token: bridgeTx.token,
            recipient: bridgeTx.recipient,
            bridgeId: bridgeTx.id
        });
        
        if (mintResponse.data.success) {
            bridgeTx.txHash = mintResponse.data.txHash;
            bridgeTx.status = 'confirmed';
        } else {
            bridgeTx.status = 'failed';
        }
        
        bridgeTransactions.set(bridgeTx.id, bridgeTx);
        
    } catch (error) {
        console.error(`Error processing XION to KNIRV transfer ${bridgeTx.id}:`, error);
        bridgeTx.status = 'failed';
        bridgeTransactions.set(bridgeTx.id, bridgeTx);
    }
}

// Error handling middleware
app.use((error: any, req: express.Request, res: express.Response, next: express.NextFunction) => {
    console.error('Express error:', error);
    res.status(500).json({
        status: 'error',
        message: 'Internal server error',
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`🌉 XION Bridge Testnet running on port ${PORT}`);
    console.log(`🔮 Connected to KNIRVORACLE: ${KNIRVORACLE_API}`);
    console.log(`⛓️  Connected to KNIRVCHAIN: ${KNIRVCHAIN_API}`);
    console.log(`🔗 Connected to XION: ${XION_RPC_ENDPOINT}`);
    console.log(`🌐 Health check: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('SIGTERM received, shutting down gracefully');
    process.exit(0);
});

process.on('SIGINT', () => {
    console.log('SIGINT received, shutting down gracefully');
    process.exit(0);
});
