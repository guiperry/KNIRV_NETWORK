# ASIC Device Scanner

## Key Features

**1. Multi-Protocol Detection**
- **Port Scanning**: Checks common ASIC miner ports (SSH 22, Telnet 23, HTTP 80/8080, HTTPS 443, CGMiner API 4028/4029, Stratum 3333)
- **SSH Authentication**: Attempts login with provided credentials using `golang.org/x/crypto/ssh`
- **Telnet Support**: Detects and attempts login to legacy devices
- **Web Interface**: Detects OpenWrt LuCI, Antminer, and other common mining firmware interfaces

**2. ASIC-Specific Signatures**
- Detects CGMiner/BMMiner API ports (4028/4029)
- Identifies device types (Antminer, Avalon, Whatsminer, etc.) via HTTP banners
- Recognizes OpenWrt firmware signatures

**3. Concurrent Scanning**
- Uses goroutines with semaphore-controlled concurrency (default 50 concurrent scans)
- Fast TCP connect scanning with configurable timeouts

## Required Dependencies

```bash
go get golang.org/x/crypto/ssh
```

## Usage Example

```go
config := ScannerConfig{
    Subnet:          "192.168.1.0/24",
    Username:        "root",      // OpenWrt default
    Password:        "yourpassword",
    ConcurrentScans: 100,
    Timeout:         2 * time.Second,
}
```

## Important Security Considerations

⚠️ **Only run this on networks you own or have explicit permission to scan.** Unauthorized scanning may violate laws or terms of service.

1. **Host Key Verification**: The example uses `InsecureIgnoreHostKey()` for simplicity. In production, verify host keys properly.
2. **Rate Limiting**: Adjust `ConcurrentScans` to avoid overwhelming your network.
3. **Credential Security**: Never hardcode passwords in production code; use environment variables or secure vaults.

## Advanced: CGMiner API Integration

If you want to query the CGMiner API directly after discovery:

```go
func queryCGMinerAPI(ip string, port int) (map[string]interface{}, error) {
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    // CGMiner API command
    cmd := `{"command":"summary"}`
    fmt.Fprintf(conn, "%s", cmd)
    
    reader := bufio.NewReader(conn)
    response, _ := reader.ReadString('\x00')
    
    var result map[string]interface{}
    json.Unmarshal([]byte(response), &result)
    return result, nil
}
```

This approach provides comprehensive detection of ASIC miners by combining network discovery with protocol-specific authentication and signature detection .