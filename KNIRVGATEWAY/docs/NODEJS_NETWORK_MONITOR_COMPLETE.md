# Pure Node.js Network Monitor Implementation

## Overview

The Network Monitor functionality has been **completely implemented in Node.js** and is integrated directly into the KNIRV WebGUI. **No Go service is required!**

This is a standalone, pure JavaScript implementation that provides:
- ✅ Service health monitoring
- ✅ System metrics collection (CPU, memory, disk, IPFS)
- ✅ Log aggregation and management
- ✅ Real-time dashboard with auto-refresh
- ✅ Root admin access control

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│              Pure Node.js Implementation                 │
└──────────────────────────────────────────────────────────┘

┌─────────────────┐
│   WebGUI UI     │  (React/Next.js Frontend)
│ network-monitor │
└────────┬────────┘
         │
         ▼
┌────────────────────────┐
│   API Routes           │  (Next.js API)
│ /api/network-monitor/* │
└────────┬───────────────┘
         │
         ▼
┌──────────────────────────────────────┐
│   Network Monitor Library            │
│   (Pure Node.js)                     │
│   ┌──────────────────────────────┐   │
│   │ NetworkMonitor (main)        │   │
│   │ ├─ ServiceMonitor           │   │
│   │ ├─ MetricsCollector         │   │
│   │ ├─ LogManager               │   │
│   │ └─ Config                   │   │
│   └──────────────────────────────┘   │
└──────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────┐
│   Target Services                    │
│   ├─ IPFS                           │
│   ├─ KNIRVORACLE                    │
│   ├─ KNIRVCHAIN                     │
│   ├─ KNIRVGRAPH                     │
│   ├─ KNIRVNEXUS                     │
│   └─ KNIRVROUTER                    │
└──────────────────────────────────────┘
```

## File Structure

```
services/webgui/
├── src/
│   ├── lib/
│   │   └── network-monitor/           # Pure Node.js Implementation
│   │       ├── index.js              # Main NetworkMonitor class
│   │       ├── config.js             # Configuration management
│   │       ├── service-monitor.js    # Service health checking
│   │       ├── metrics-collector.js  # System metrics collection
│   │       └── log-manager.js        # Log aggregation
│   │
│   └── pages/
│       ├── api/
│       │   └── network-monitor/      # API Routes
│       │       ├── status.js         # Network status
│       │       ├── services.js       # Service list
│       │       ├── metrics.js        # System metrics
│       │       ├── logs.js           # Log entries
│       │       └── config.js         # Configuration
│       │
│       ├── network-monitor.js        # UI Dashboard
│       └── network-monitor.module.css
│
├── .env.example                      # Environment template
└── package.json                      # Dependencies
```

## Features

### 1. Service Health Monitoring
- **Automatic health checks** every 30 seconds
- **HTTP endpoint polling** with configurable timeout
- **Status tracking**: up, down, degraded, unknown
- **Response time measurement**
- **Uptime percentage calculation**
- **Critical service flagging**

### 2. System Metrics Collection
- **CPU Usage**: Overall and per-core utilization
- **Memory**: Total, used, available, usage percentage
- **Disk**: Space usage and availability
- **IPFS**: Storage, peer count, repo stats
- **Real-time collection** every 30 seconds

### 3. Log Management
- **Centralized log aggregation** from all services
- **Log levels**: info, warn, error, debug
- **Automatic retention** policy (7 days default)
- **Search and filtering** by service, level, message
- **In-memory storage** with configurable limits

### 4. Configuration
- **Environment-based** service URLs
- **Multiple network support** (production, testnet)
- **Flexible service definitions**
- **Configurable check intervals**

## Installation & Setup

### 1. Install Dependencies

```bash
cd services/webgui
npm install axios
```

The network monitor uses only standard Node.js modules and axios (already a dependency).

### 2. Configure Environment

Create `.env.local` from the example:

```bash
cp .env.example .env.local
```

Edit `.env.local` to configure your services:

```bash
# Active Network
ACTIVE_NETWORK=production

# Service URLs - Production
IPFS_URL=http://localhost:5001
KNIRVORACLE_URL=http://localhost:8083
KNIRVCHAIN_URL=http://localhost:8080
KNIRVGRAPH_URL=http://localhost:8081
KNIRVNEXUS_URL=http://localhost:8082
KNIRVROUTER_URL=http://localhost:5001

# Service URLs - Testnet
IPFS_TESTNET_URL=http://localhost:5001
KNIRVORACLE_TESTNET_URL=http://localhost:1317

# Monitoring Configuration
LOG_LEVEL=info
```

### 3. Start WebGUI

```bash
npm run dev
```

The network monitor will automatically start when the WebGUI starts!

### 4. Access Network Monitor

1. Navigate to http://localhost:3000
2. Authenticate as **Root administrator**
3. Click "Network Monitor" in the navigation menu

## Configuration

### Service Configuration

Edit `src/lib/network-monitor/config.js` to add/modify services:

```javascript
services: [
  {
    name: 'service-name',
    url: process.env.SERVICE_URL || 'http://localhost:PORT',
    healthEndpoint: '/health',           // Health check endpoint
    metricsEndpoint: '/metrics',         // Optional metrics endpoint
    checkInterval: 30000,                // Check every 30 seconds
    timeout: 10000,                      // 10 second timeout
    critical: true,                      // Is this service critical?
    type: 'cosmos'                       // Service type (cosmos, ipfs, p2p)
  }
]
```

### Monitoring Settings

Adjust monitoring behavior in `config.js`:

```javascript
monitoring: {
  enabled: true,
  checkInterval: 30000,                  // 30 seconds
  metricsRetention: 30 * 24 * 60 * 60 * 1000,  // 30 days
  maxLogEntries: 1000,
  collectSystemMetrics: true
}
```

### Logging Settings

Configure log management:

```javascript
logging: {
  enabled: true,
  level: 'info',                         // info, warn, error, debug
  retention: 7 * 24 * 60 * 60 * 1000,   // 7 days
  maxEntries: 10000                      // Max logs in memory
}
```

## Usage

### For Root Administrators

#### View Network Status
- Overall network health (healthy, degraded, critical)
- Service counts (up/down/total)
- Last update timestamp

#### Monitor Services
- List of all monitored services with status
- Click any service for detailed information
- Response times and error messages
- Uptime percentages

#### Check System Metrics
- CPU usage (overall and per-core)
- Memory usage and availability
- Disk space usage
- IPFS storage and peer count
- Click "View Detailed Metrics" for comprehensive stats

#### Review Logs
- Recent log entries from all services
- Filter by service and log level
- Search by message content
- Click "View All Logs" for full history

#### Quick Actions
- **Refresh Data**: Manual refresh (auto-refreshes every 5 seconds)
- **Export Report**: Download monitoring report
- **Configure Alerts**: Set up alert rules
- **Settings**: Adjust monitoring parameters

### API Endpoints

All endpoints require Root admin authentication (`x-knirv-role: Root` header).

#### GET /api/network-monitor/status
Returns network status and all service statuses.

**Response:**
```json
{
  "success": true,
  "data": {
    "name": "KNIRV Production Network",
    "overall_status": "healthy",
    "services_up": 6,
    "services_down": 0,
    "services_total": 6,
    "last_update": "2024-12-07T10:30:00Z",
    "services": {
      "knirvchain": {
        "name": "knirvchain",
        "url": "http://localhost:8080",
        "status": "up",
        "lastCheck": "2024-12-07T10:30:00Z",
        "responseTime": 45,
        "error": null,
        "critical": true,
        "uptime": 99.8
      }
      // ... other services
    }
  }
}
```

#### GET /api/network-monitor/services
Returns detailed information about all services.

#### GET /api/network-monitor/metrics
Returns system metrics (CPU, memory, disk, IPFS).

**Response:**
```json
{
  "success": true,
  "data": {
    "system": {
      "cpu": {
        "usage_percent": 45.2,
        "load_average": [1.2, 1.1, 1.0],
        "core_usage": { "cpu0": 40.0, "cpu1": 50.0 },
        "cores": 8
      },
      "memory": {
        "total_bytes": 17179869184,
        "used_bytes": 10737418240,
        "available_bytes": 6442450944,
        "usage_percent": 62.5
      }
      // ... disk, ipfs, network
    }
  }
}
```

#### GET /api/network-monitor/logs?limit=100
Returns recent log entries.

#### GET /api/network-monitor/config
Returns current configuration.

## Removing the Go Network Monitor

Since everything is now implemented in Node.js, you can safely remove the Go network-monitor:

### 1. Stop the Go Service (if running)

```bash
# If running as a process
pkill knirv-network-monitor

# If running as systemd service
sudo systemctl stop knirv-network-monitor
sudo systemctl disable knirv-network-monitor
```

### 2. Remove Go Files (Optional)

```bash
# Archive first (recommended)
cd KNIRVGATEWAY
tar -czf network-monitor-go-backup.tar.gz network-monitor/
mv network-monitor-go-backup.tar.gz ~/backups/

# Then remove
rm -rf network-monitor/
```

### 3. Clean Up Environment

Remove any Go-specific environment variables from `.env`:

```bash
# These are no longer needed
# NETWORK_MONITOR_URL=http://localhost:8090
```

### 4. Update Documentation

Remove references to the Go service from:
- README files
- Deployment scripts
- System service files
- Docker compose files

## Benefits of Pure Node.js Implementation

### ✅ Simplified Architecture
- No separate Go service to run
- No additional ports to manage
- Fewer moving parts

### ✅ Better Integration
- Native Next.js/React integration
- Shared codebase with WebGUI
- Consistent error handling

### ✅ Easier Development
- Single language (JavaScript)
- Hot reloading in dev mode
- Easier debugging

### ✅ Reduced Dependencies
- No Go runtime required
- No cross-language communication
- Fewer deployment steps

### ✅ Cost Effective
- Lower resource usage
- Simpler infrastructure
- Easier to scale

## Advanced Usage

### Custom Service Types

Add new service types by updating the configuration:

```javascript
{
  name: 'custom-service',
  url: 'http://localhost:9000',
  healthEndpoint: '/custom/health',
  type: 'custom',
  critical: false
}
```

### Adding Custom Metrics

Extend `MetricsCollector` to collect additional metrics:

```javascript
// In metrics-collector.js
async getCustomMetrics() {
  // Your custom metrics collection logic
  return {
    custom_metric_1: value1,
    custom_metric_2: value2
  };
}
```

### Custom Log Processing

Extend `LogManager` for custom log processing:

```javascript
// In log-manager.js
processLog(entry) {
  // Custom processing logic
  // e.g., send to external service
  return entry;
}
```

## Troubleshooting

### No Services Showing

**Issue**: Network monitor shows no services

**Solution**:
1. Check service URLs in `.env.local`
2. Verify services are running
3. Check browser console for errors
4. Review server logs: `npm run dev`

### Metrics Not Updating

**Issue**: System metrics show 0 or stale data

**Solution**:
1. Check if services are accessible
2. Verify IPFS is running (for IPFS metrics)
3. Check OS compatibility (some metrics require Unix-like systems)
4. Review console for errors

### Access Denied

**Issue**: Cannot access Network Monitor

**Solution**:
1. Verify you're logged in as Root administrator
2. Check role in browser: `localStorage.getItem('knirv_user_role')`
3. Clear cache and re-authenticate
4. Enable demo mode in development: `localStorage.setItem('knirv_demo_mode', 'true')`

### High Memory Usage

**Issue**: WebGUI using too much memory

**Solution**:
1. Reduce `maxLogEntries` in config
2. Shorten log retention period
3. Increase check intervals
4. Clear old logs more frequently

## Performance Tips

### Optimize Check Intervals

Adjust based on your needs:

```javascript
// Less frequent checks = lower CPU usage
checkInterval: 60000,  // 1 minute instead of 30 seconds
```

### Reduce Log Storage

```javascript
logging: {
  maxEntries: 1000,    // Store fewer logs
  retention: 24 * 60 * 60 * 1000  // 1 day instead of 7
}
```

### Disable Non-Critical Services

Comment out or remove services you don't need:

```javascript
// services: [
//   {
//     name: 'non-critical-service',
//     ...
//   }
// ]
```

## Production Deployment

### Environment Variables

Set production values in `.env.production`:

```bash
NODE_ENV=production
ACTIVE_NETWORK=production

# Production service URLs
KNIRVORACLE_URL=https://oracle.knirv.network
KNIRVCHAIN_URL=https://chain.knirv.network
# ... etc
```

### Build for Production

```bash
npm run build
npm start
```

### Docker Deployment

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build
EXPOSE 3000
CMD ["npm", "start"]
```

### Monitoring the Monitor

Set up external monitoring for the WebGUI:
- Uptime monitoring (e.g., UptimeRobot)
- Log aggregation (e.g., LogRocket)
- Error tracking (e.g., Sentry)
- Performance monitoring (e.g., New Relic)

## Migration Checklist

- [x] Install Node.js dependencies
- [x] Configure environment variables
- [x] Test Network Monitor in development
- [x] Verify all services are detected
- [x] Check metrics collection
- [x] Review logs functionality
- [x] Test as Root administrator
- [ ] Stop Go network-monitor service
- [ ] Remove Go service files
- [ ] Update documentation
- [ ] Deploy to production
- [ ] Monitor performance

## Support & Troubleshooting

For issues or questions:
- Check this documentation
- Review browser console for errors
- Check Next.js server logs
- Verify service configurations
- Test individual API endpoints

## Future Enhancements

Potential improvements:
- [ ] WebSocket real-time updates
- [ ] Alert system with notifications
- [ ] Historical data charts
- [ ] Custom dashboards
- [ ] Export to PDF/CSV
- [ ] Integration with external monitoring tools
- [ ] Service dependency mapping
- [ ] Automated remediation
- [ ] Multi-network switching UI
- [ ] Mobile app integration

---

**Implementation**: Pure Node.js
**Status**: ✅ Complete and Production Ready
**Go Dependency**: ❌ None - Go service can be removed!
**Access Level**: Root Admin Only
