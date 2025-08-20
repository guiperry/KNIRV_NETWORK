"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setupSocket = void 0;
const setupSocket = (io) => {
    io.on('connection', (socket) => {
        console.log('KNIRV-NEXUS DVE Client connected:', socket.id);
        // Join DVE monitoring room
        socket.join('dve-monitoring');
        // Handle DVE node status updates
        socket.on('dve-node-update', (update) => {
            io.to('dve-monitoring').emit('dve-node-updated', update);
        });
        // Handle validation task progress updates
        socket.on('validation-task-update', (update) => {
            io.to('dve-monitoring').emit('validation-task-updated', update);
        });
        // Handle cognitive engine metrics updates
        socket.on('cognitive-engine-update', (update) => {
            io.to('dve-monitoring').emit('cognitive-engine-updated', update);
        });
        // Handle TEE security updates
        socket.on('tee-security-update', (update) => {
            io.to('dve-monitoring').emit('tee-security-updated', update);
        });
        // Handle NRN staking updates
        socket.on('nrn-staking-update', (update) => {
            io.to('dve-monitoring').emit('nrn-staking-updated', update);
        });
        // Handle security alerts
        socket.on('security-alert', (alert) => {
            io.to('dve-monitoring').emit('security-alert', alert);
        });
        // Handle system notifications
        socket.on('system-notification', (notification) => {
            io.to('dve-monitoring').emit('system-notification', notification);
        });
        // Request initial data sync
        socket.on('request-sync', () => {
            socket.emit('sync-requested', { timestamp: new Date().toISOString() });
        });
        // Handle real-time metrics subscription
        socket.on('subscribe-metrics', (metrics) => {
            metrics.forEach(metric => {
                socket.join(`metric-${metric}`);
            });
            socket.emit('subscribed-to-metrics', metrics);
        });
        // Handle real-time metrics unsubscription
        socket.on('unsubscribe-metrics', (metrics) => {
            metrics.forEach(metric => {
                socket.leave(`metric-${metric}`);
            });
            socket.emit('unsubscribed-from-metrics', metrics);
        });
        // Handle disconnect
        socket.on('disconnect', () => {
            console.log('KNIRV-NEXUS DVE Client disconnected:', socket.id);
            socket.leave('dve-monitoring');
        });
        // Send welcome message with system info
        socket.emit('system-notification', {
            type: 'info',
            title: 'Connected to KNIRV-NEXUS DVE',
            message: 'Real-time monitoring activated for DVE nodes, validation tasks, and security metrics.',
            timestamp: new Date().toISOString()
        });
        // Simulate real-time updates for demo purposes
        const simulateUpdates = () => {
            // Simulate DVE node updates
            if (Math.random() > 0.7) {
                const nodeUpdate = {
                    id: `dve-${Math.floor(Math.random() * 3) + 1}`,
                    status: Math.random() > 0.9 ? 'maintenance' : 'online',
                    cpu_usage: Math.floor(Math.random() * 100),
                    memory_usage: Math.floor(Math.random() * 100),
                    last_heartbeat: new Date().toISOString()
                };
                io.to('dve-monitoring').emit('dve-node-updated', nodeUpdate);
            }
            // Simulate validation task updates
            if (Math.random() > 0.8) {
                const taskUpdate = {
                    id: `task-${Math.floor(Math.random() * 3) + 1}`,
                    status: Math.random() > 0.7 ? 'completed' : 'running',
                    progress: Math.floor(Math.random() * 100),
                    assigned_node: `dve-${Math.floor(Math.random() * 3) + 1}`,
                    estimated_completion: new Date(Date.now() + Math.random() * 3600000).toISOString()
                };
                io.to('dve-monitoring').emit('validation-task-updated', taskUpdate);
            }
            // Simulate cognitive engine updates
            if (Math.random() > 0.9) {
                const cognitiveUpdate = {
                    status: 'active',
                    accuracy: 90 + Math.random() * 10,
                    tasks_processed: Math.floor(Math.random() * 1000) + 15000,
                    adaptation_rate: 0.8 + Math.random() * 0.2
                };
                io.to('dve-monitoring').emit('cognitive-engine-updated', cognitiveUpdate);
            }
        };
        // Start simulation updates every 5 seconds
        const simulationInterval = setInterval(simulateUpdates, 5000);
        // Clean up on disconnect
        socket.on('disconnect', () => {
            clearInterval(simulationInterval);
        });
    });
};
exports.setupSocket = setupSocket;
