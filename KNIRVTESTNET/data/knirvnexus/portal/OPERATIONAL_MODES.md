# KNIRV-NEXUS Operational Modes

## Overview

KNIRV-NEXUS supports two distinct operational modes to accommodate different deployment scenarios and administrative needs.

## Mode Comparison

| Feature | Headless Mode (Default) | GUI Mode (-gui flag) |
|---------|------------------------|----------------------|
| **Target Use** | Production deployment | Local administration |
| **Authentication** | JWT required | No authentication |
| **Web Interface** | None (API only) | Built-in admin dashboard |
| **Network Access** | All interfaces (0.0.0.0) | Localhost only (127.0.0.1) |
| **Security** | Full RBAC + audit logging | Local access only |
| **Resource Usage** | Minimal | Higher (includes web server) |
| **Real-time Updates** | SSE via KNIRVGATEWAY | WebSocket direct |
| **Deployment** | Kubernetes clusters | Local workstations |

## Headless Mode (Production)

### Activation
```bash
# Default behavior (no flags needed)
./dve-manager
./validation-core

# Explicit headless mode
./dve-manager --headless
./validation-core --headless
```

### Characteristics
- **API-Only Access**: No web interface, REST APIs only
- **Full Authentication**: JWT tokens required for all operations
- **Production Optimized**: Minimal resource footprint
- **Network Accessible**: Binds to all interfaces for cluster access
- **Secure by Default**: All security features enabled
- **Kubernetes Ready**: Health checks, metrics, proper logging

### Use Cases
- Production Kubernetes deployments
- Cloud-based validator nodes
- Remote server deployments
- Automated CI/CD environments
- High-availability clusters

### Security Features
- JWT-based authentication
- Role-based access control (RBAC)
- Comprehensive audit logging
- TLS encryption
- Network policies compliance

## GUI Mode (Local Admin)

### Activation
```bash
# Enable GUI mode
./dve-manager -gui
./validation-core -gui

# Alternative syntax
./dve-manager --gui
./validation-core --gui
```

### Characteristics
- **Built-in Web Interface**: Admin dashboard at http://localhost:9080
- **No Authentication**: Direct access (admin environment assumed)
- **Local Access Only**: Binds to localhost (127.0.0.1)
- **Real-time Updates**: WebSocket connections for live data
- **Debug Information**: Extended logging and diagnostics
- **Development Tools**: API testing, configuration management

### Use Cases
- Local development and testing
- System administration and debugging
- Configuration management
- Performance monitoring
- Troubleshooting and diagnostics
- Training and demonstrations

### GUI Features
- **Node Management**: Visual DVE node administration
- **Task Monitoring**: Real-time validation task tracking
- **System Health**: Live system status and metrics
- **Configuration**: Service configuration management
- **Logs Viewer**: Real-time log streaming
- **API Testing**: Built-in API testing tools

## Implementation Architecture

### CLI Flag Handling
```go
func main() {
    var (
        guiMode = flag.Bool("gui", false, "Enable GUI mode")
        port    = flag.Int("port", 8080, "Service port")
        guiPort = flag.Int("gui-port", 9080, "GUI port")
    )
    flag.Parse()

    config := getConfig(*guiMode)
    service := initializeService(config)
    
    if *guiMode {
        startGUIServer(*guiPort, service)
    }
    
    service.Start()
}
```

### Security Configuration
```go
type Config struct {
    Mode            string
    AuthRequired    bool
    BindAddress     string
    TLSEnabled      bool
    AuditLogging    bool
    GUIEnabled      bool
}

func getConfig(guiMode bool) Config {
    if guiMode {
        return Config{
            Mode:         "gui",
            AuthRequired: false,
            BindAddress:  "127.0.0.1",
            TLSEnabled:   false,
            AuditLogging: false,
            GUIEnabled:   true,
        }
    }
    return Config{
        Mode:         "headless",
        AuthRequired: true,
        BindAddress:  "0.0.0.0",
        TLSEnabled:   true,
        AuditLogging: true,
        GUIEnabled:   false,
    }
}
```

## Deployment Examples

### Production (Headless)
```yaml
# Kubernetes deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dve-manager
spec:
  template:
    spec:
      containers:
      - name: dve-manager
        image: knirv/dve-manager:latest
        command: ["./dve-manager"]  # Default headless
        ports:
        - containerPort: 8080
        env:
        - name: KNIRV_AUTH_REQUIRED
          value: "true"
```

### Local Development (GUI)
```bash
# Docker with GUI mode
docker run -p 8080:8080 -p 9080:9080 \
  knirv/dve-manager:latest ./dve-manager -gui

# Access GUI: http://localhost:9080
# Access API: http://localhost:8080
```

### Local Binary (GUI)
```bash
# Start with GUI
./dve-manager -gui &
./validation-core -gui &

# Access dashboards
open http://localhost:9080  # DVE Manager GUI
open http://localhost:9081  # Validation Core GUI
```

## Port Allocation

### Default Ports
| Service | Headless Mode | GUI Mode (API) | GUI Mode (Web) |
|---------|---------------|----------------|----------------|
| DVE Manager | 8080 | 8080 | 9080 |
| Validation Core | 8081 | 8081 | 9081 |

### Custom Ports
```bash
# Custom ports
./dve-manager -gui --port 8090 --gui-port 9090
./validation-core -gui --port 8091 --gui-port 9091
```

## Security Considerations

### Headless Mode Security
- **Network Exposure**: Accessible from cluster networks
- **Authentication**: JWT tokens required
- **Authorization**: Full RBAC enforcement
- **Audit Trail**: All operations logged
- **TLS**: Encrypted communications

### GUI Mode Security
- **Local Only**: Not accessible from external networks
- **Admin Environment**: Assumes secure admin workstation
- **No Authentication**: Direct access for convenience
- **Development Use**: Not intended for production
- **Firewall**: Should be blocked at network level

## Best Practices

### Production Deployment
1. Always use headless mode in production
2. Deploy behind KNIRVGATEWAY for unified access
3. Enable all security features
4. Monitor with Prometheus/Grafana
5. Use Kubernetes for orchestration

### Development Workflow
1. Use GUI mode for local development
2. Test APIs directly through GUI interface
3. Monitor real-time updates via WebSocket
4. Debug issues with extended logging
5. Switch to headless mode for integration testing

### Administration
1. Use GUI mode for system administration
2. Access local instances for troubleshooting
3. Monitor system health through GUI dashboard
4. Configure services through web interface
5. View logs in real-time for debugging

## Migration Impact

The operational modes feature enhances the KNIRV-NEXUS migration by:

1. **Flexibility**: Supports both production and development scenarios
2. **Admin Convenience**: Local GUI for administration without authentication
3. **Security**: Appropriate security model for each use case
4. **Development Speed**: Faster development with immediate visual feedback
5. **Production Readiness**: Optimized headless mode for production deployment
