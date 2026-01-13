# Network Monitor Integration

This document describes the integration of the network-monitor functionality into the KNIRV WebGUI for root administrator use.

## Overview

The Network Monitor provides comprehensive real-time monitoring of the KNIRV network infrastructure, including:

- **Service Health Monitoring**: Track status, response times, and availability of all KNIRV services
- **System Metrics**: CPU, memory, disk, and IPFS storage monitoring
- **Log Aggregation**: Centralized log viewing and searching across all services
- **Network Status**: Overall network health and service coordination

## Architecture

### Components

1. **Go Network Monitor Service** (`network-monitor/`)
   - Standalone Go application providing monitoring backend
   - REST API for status, services, metrics, and logs
   - Can run in headless mode or with GUI

2. **WebGUI API Routes** (`services/webgui/src/pages/api/network-monitor/`)
   - Next.js API routes that proxy requests to the Go service
   - Root admin authentication enforcement
   - Fallback data when service is unavailable

3. **WebGUI Frontend** (`services/webgui/src/pages/network-monitor.js`)
   - React-based monitoring dashboard
   - Real-time data updates every 5 seconds
   - Interactive modals for detailed views
   - Root admin access control

### Data Flow

```
┌─────────────┐         ┌──────────────┐         ┌─────────────────┐
│  WebGUI     │ ──────> │ Next.js API  │ ──────> │ Go Network      │
│  Frontend   │ <────── │ Routes       │ <────── │ Monitor Service │
└─────────────┘         └──────────────┘         └─────────────────┘
                               │
                               │ (Authentication)
                               ▼
                        ┌──────────────┐
                        │ RoleContext  │
                        │ (Root Only)  │
                        └──────────────┘
```

## Setup

### 1. Start the Network Monitor Service

```bash
cd network-monitor

# Headless mode (for production)
./bin/knirv-network-monitor --headless --port 8090 --network production

# Or with GUI (for development)
./bin/knirv-network-monitor --network production
```

### 2. Configure WebGUI Environment

Copy the example environment file:

```bash
cd services/webgui
cp .env.example .env.local
```

Edit `.env.local` to configure:

```bash
# Network Monitor URL
NETWORK_MONITOR_URL=http://localhost:8090

# Network Name
NETWORK_NAME=KNIRV Production Network
```

### 3. Start WebGUI

```bash
cd services/webgui
npm install
npm run dev
```

### 4. Access as Root Admin

1. Navigate to http://localhost:3000
2. Authenticate with Root administrator credentials
3. Access Network Monitor from the navigation menu

## API Endpoints

### Status
```
GET /api/network-monitor/status
```
Returns overall network status and service health.

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
    "last_update": "2024-01-15T10:30:00Z",
    "services": { ... }
  }
}
```

### Services
```
GET /api/network-monitor/services
```
Returns detailed information about all monitored services.

### Metrics
```
GET /api/network-monitor/metrics
```
Returns system metrics (CPU, memory, disk, IPFS).

**Response:**
```json
{
  "success": true,
  "data": {
    "system": {
      "cpu": {
        "usage_percent": 45.0,
        "load_average": [1.2, 1.1, 1.0]
      },
      "memory": {
        "total_bytes": 17179869184,
        "used_bytes": 10737418240,
        "usage_percent": 62.5
      },
      ...
    }
  }
}
```

### Logs
```
GET /api/network-monitor/logs?limit=100
```
Returns recent log entries from all services.

### Configuration
```
GET /api/network-monitor/config
```
Returns current monitoring configuration.

## Access Control

The Network Monitor is **restricted to Root administrators only**.

### Role Hierarchy

- **Root**: Full access to all features including Network Monitor
- **Bootnode**: No access to Network Monitor
- **Dev**: No access to Network Monitor
- **General**: No access to Network Monitor

### Implementation

Access control is enforced at two levels:

1. **API Level** (`services/webgui/src/pages/api/network-monitor/*.js`):
   ```javascript
   const userRole = req.headers['x-knirv-role'] || 'General';
   if (userRole !== 'Root') {
     return res.status(403).json({
       success: false,
       error: 'Forbidden: Root admin access required'
     });
   }
   ```

2. **UI Level** (`services/webgui/src/pages/network-monitor.js`):
   ```javascript
   const { role } = useRole();

   if (role !== 'Root') {
     return <AccessDenied />;
   }
   ```

## Features

### Network Status Overview

- **Overall Status**: Healthy, degraded, or critical
- **Service Count**: Number of services up/down
- **Last Update**: Timestamp of last status check
- **Network Name**: Active network being monitored

### Service Monitoring

- **Health Checks**: Regular HTTP health endpoint polling
- **Response Times**: Track service latency
- **Status Indicators**: Up, down, degraded, unknown
- **Critical Flags**: Mark critical services
- **Interactive Details**: Click any service for detailed information

### System Metrics

- **CPU Usage**: Overall and per-core utilization
- **Memory Usage**: Total, used, available, swap
- **Disk Usage**: Total, used, available, I/O stats
- **IPFS Metrics**: Storage, peers, pinned objects
- **Network Stats**: Bytes sent/received, connections

### Log Aggregation

- **Recent Logs**: Last 20 entries across all services
- **Log Levels**: Info, warn, error, debug
- **Service Filtering**: View logs by service
- **Timestamp**: Precise event timing
- **Full Logs Modal**: View complete log history

### Quick Actions

- **Refresh Data**: Manual data refresh
- **Export Report**: Generate monitoring report
- **Configure Alerts**: Set up alert rules
- **Settings**: Configure monitoring parameters

## Configuration

### Network Monitor (`network-monitor/config/config.yaml`)

```yaml
active_network: production

networks:
  production:
    name: "KNIRV Production Network"
    services:
      - name: "knirvoracle"
        url: "http://localhost:8083"
        health_endpoint: "/health"
        check_interval: 30s
        timeout: 10s
        critical: true
      - name: "knirvchain"
        url: "http://localhost:8080"
        health_endpoint: "/health"
        check_interval: 30s
        timeout: 10s
        critical: true
      # ... more services

monitoring:
  enabled: true
  check_interval: 30s
  metrics_retention: 720h  # 30 days

logging:
  enabled: true
  level: "info"
  retention: 168h  # 7 days
```

## Troubleshooting

### Network Monitor Service Not Accessible

**Issue**: WebGUI cannot connect to network monitor service

**Solutions**:
1. Verify service is running: `ps aux | grep knirv-network-monitor`
2. Check service URL in `.env.local`
3. Ensure port 8090 is not blocked by firewall
4. Check service logs for errors

### Access Denied

**Issue**: User cannot access Network Monitor page

**Solutions**:
1. Verify user role is set to "Root"
2. Check `localStorage` for `knirv_user_role`
3. Re-authenticate if needed
4. Clear browser cache and retry

### No Data Displayed

**Issue**: Network Monitor shows empty or "No data" messages

**Solutions**:
1. Check network monitor service is running and healthy
2. Verify services are configured in `config.yaml`
3. Check API endpoints are responding: `curl http://localhost:8090/api/status`
4. Review browser console for API errors

### Stale Data

**Issue**: Data not updating in real-time

**Solutions**:
1. Verify auto-refresh interval (default: 5 seconds)
2. Check browser network tab for failed requests
3. Try manual refresh using "Refresh Data" button
4. Restart network monitor service if needed

## Development

### Running in Development Mode

```bash
# Terminal 1: Start Network Monitor
cd network-monitor
go run cmd/monitor/main.go --headless --port 8090 --debug

# Terminal 2: Start WebGUI
cd services/webgui
npm run dev
```

### Adding New Monitored Services

1. Update `network-monitor/config/config.yaml`:
   ```yaml
   services:
     - name: "new-service"
       url: "http://localhost:PORT"
       health_endpoint: "/health"
       check_interval: 30s
       timeout: 10s
       critical: false
   ```

2. Restart network monitor service

3. New service will appear automatically in WebGUI

### Customizing Metrics

To add custom metrics:

1. Update `network-monitor/internal/metrics/collector.go`
2. Add new metric collection function
3. Update `SystemMetrics` struct
4. Update WebGUI to display new metrics

## Production Deployment

### Running Network Monitor as a Service

Create systemd service file:

```ini
[Unit]
Description=KNIRV Network Monitor
After=network.target

[Service]
Type=simple
User=knirv
WorkingDirectory=/opt/knirv/network-monitor
ExecStart=/opt/knirv/network-monitor/bin/knirv-network-monitor --headless --port 8090 --network production
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable knirv-network-monitor
sudo systemctl start knirv-network-monitor
```

### Security Considerations

1. **Authentication**: Always enforce Root admin authentication
2. **HTTPS**: Use HTTPS in production
3. **Firewall**: Restrict network monitor port to localhost or internal network
4. **Secrets**: Never expose sensitive data in logs or metrics
5. **Rate Limiting**: Implement rate limiting on API endpoints

## Future Enhancements

- [ ] Alert system with email/SMS notifications
- [ ] Historical data charts and graphs
- [ ] Custom dashboards per network
- [ ] Service dependency mapping
- [ ] Automated remediation actions
- [ ] Integration with Prometheus/Grafana
- [ ] Mobile-responsive design improvements
- [ ] Export to PDF/CSV reports
- [ ] Real-time WebSocket updates
- [ ] Multi-network monitoring from single dashboard

## Support

For issues or questions:
- GitHub Issues: https://github.com/knirv/network/issues
- Documentation: https://docs.knirv.network
- Discord: https://discord.gg/knirv
