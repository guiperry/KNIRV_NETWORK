import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import morgan from 'morgan';
import { WebSocketServer } from 'ws';
import { createServer } from 'http';
import axios from 'axios';
import dotenv from 'dotenv';

// Load environment variables
dotenv.config();

const app = express();
const server = createServer(app);
const wss = new WebSocketServer({ server });

// Configuration
const PORT = process.env.PORT || 3000;
const KNIRVGRAPH_API = process.env.KNIRVGRAPH_API || 'http://localhost:8082';
const KNIRVCHAIN_API = process.env.KNIRVCHAIN_API || 'http://localhost:8090';
const KNIRVORACLE_API = process.env.KNIRVORACLE_API || 'http://localhost:1317';

// Middleware
app.use(helmet());
app.use(cors());
app.use(compression());
app.use(morgan('combined'));
app.use(express.json());
app.use(express.static('public'));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'KNIRVANA Testnet',
        timestamp: new Date().toISOString(),
        version: '1.0.0',
        connections: {
            graph: KNIRVGRAPH_API,
            chain: KNIRVCHAIN_API,
            oracle: KNIRVORACLE_API
        }
    });
});

// Game state endpoint
app.get('/api/game/state', async (req, res) => {
    try {
        // Get game state from KNIRVGRAPH
        const graphResponse = await axios.get(`${KNIRVGRAPH_API}/api/graph/state`, {
            timeout: 5000
        });
        
        res.json({
            status: 'success',
            gameState: graphResponse.data,
            timestamp: new Date().toISOString()
        });
    } catch (error) {
        console.error('Error fetching game state:', error);
        res.status(500).json({
            status: 'error',
            message: 'Failed to fetch game state',
            timestamp: new Date().toISOString()
        });
    }
});

// Player actions endpoint
app.post('/api/game/action', async (req, res) => {
    try {
        const { action, playerId, data } = req.body;
        
        // Submit action to KNIRVCHAIN
        const chainResponse = await axios.post(`${KNIRVCHAIN_API}/api/action`, {
            action,
            playerId,
            data,
            timestamp: new Date().toISOString()
        }, {
            timeout: 5000
        });
        
        res.json({
            status: 'success',
            actionResult: chainResponse.data,
            timestamp: new Date().toISOString()
        });
    } catch (error) {
        console.error('Error processing game action:', error);
        res.status(500).json({
            status: 'error',
            message: 'Failed to process game action',
            timestamp: new Date().toISOString()
        });
    }
});

// WebSocket connection handling
wss.on('connection', (ws, req) => {
    console.log('New WebSocket connection established');
    
    // Send welcome message
    ws.send(JSON.stringify({
        type: 'welcome',
        message: 'Connected to KNIRVANA Testnet',
        timestamp: new Date().toISOString()
    }));
    
    // Handle incoming messages
    ws.on('message', async (data) => {
        try {
            const message = JSON.parse(data.toString());
            
            switch (message.type) {
                case 'ping':
                    ws.send(JSON.stringify({
                        type: 'pong',
                        timestamp: new Date().toISOString()
                    }));
                    break;
                    
                case 'subscribe':
                    // Subscribe to game updates
                    ws.send(JSON.stringify({
                        type: 'subscribed',
                        channel: message.channel,
                        timestamp: new Date().toISOString()
                    }));
                    break;
                    
                case 'game_action':
                    // Process game action via WebSocket
                    try {
                        const actionResponse = await axios.post(`${KNIRVCHAIN_API}/api/action`, {
                            ...message.data,
                            timestamp: new Date().toISOString()
                        });
                        
                        ws.send(JSON.stringify({
                            type: 'action_result',
                            success: true,
                            data: actionResponse.data,
                            timestamp: new Date().toISOString()
                        }));
                    } catch (error) {
                        ws.send(JSON.stringify({
                            type: 'action_result',
                            success: false,
                            error: 'Action processing failed',
                            timestamp: new Date().toISOString()
                        }));
                    }
                    break;
                    
                default:
                    ws.send(JSON.stringify({
                        type: 'error',
                        message: 'Unknown message type',
                        timestamp: new Date().toISOString()
                    }));
            }
        } catch (error) {
            console.error('Error processing WebSocket message:', error);
            ws.send(JSON.stringify({
                type: 'error',
                message: 'Invalid message format',
                timestamp: new Date().toISOString()
            }));
        }
    });
    
    // Handle connection close
    ws.on('close', () => {
        console.log('WebSocket connection closed');
    });
    
    // Handle errors
    ws.on('error', (error) => {
        console.error('WebSocket error:', error);
    });
});

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
server.listen(PORT, () => {
    console.log(`🎮 KNIRVANA Testnet running on port ${PORT}`);
    console.log(`🔗 Connected to KNIRVGRAPH: ${KNIRVGRAPH_API}`);
    console.log(`⛓️  Connected to KNIRVCHAIN: ${KNIRVCHAIN_API}`);
    console.log(`🔮 Connected to KNIRVORACLE: ${KNIRVORACLE_API}`);
    console.log(`🌐 Health check: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('SIGTERM received, shutting down gracefully');
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('SIGINT received, shutting down gracefully');
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});
