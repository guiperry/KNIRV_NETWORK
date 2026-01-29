package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ASICMiner represents a discovered mining device
type ASICMiner struct {
	IPAddress   string            `json:"ip_address"`
	Hostname    string            `json:"hostname"`
	MACAddress  string            `json:"mac_address,omitempty"`
	DeviceType  string            `json:"device_type,omitempty"`
	OpenPorts   []int             `json:"open_ports"`
	Services    map[string]string `json:"services"` // e.g., "ssh": "OpenSSH_8.2", "http": "Antminer"
	WebTitle    string            `json:"web_title,omitempty"`
	AuthMethods []string          `json:"auth_methods"`
	Accessible  bool              `json:"accessible"` // Successfully authenticated
}

// ScannerConfig holds configuration for the network scan
type ScannerConfig struct {
	Subnet          string        // CIDR notation, e.g., "192.168.1.0/24"
	Timeout         time.Duration // Connection timeout
	ConcurrentScans int           // Number of concurrent workers
	Username        string        // Known username for auth testing
	Password        string        // Known password for auth testing
	Ports           []int         // Ports to scan
}

// Common ASIC miner ports
var defaultPorts = []int{
	22,    // SSH
	23,    // Telnet (older devices)
	80,    // HTTP (web interface)
	443,   // HTTPS (secure web)
	8080,  // Alternative HTTP
	4028,  // CGMiner API
	4029,  // BMMiner API (Antminer)
	3333,  // Mining stratum (indicates miner)
	3334,  // Alternative stratum
	22,    // SSH
}

// Common ASIC miner signatures
var minerSignatures = []string{
	"antminer",
	"bmminer",
	"cgminer",
	"avalon",
	"whatsminer",
	"innosilicon",
	"goldshell",
}

func main() {
	config := ScannerConfig{
		Subnet:          "192.168.1.0/24", // Change to your subnet
		Timeout:         2 * time.Second,
		ConcurrentScans: 50,
		Username:        "root",     // Default OpenWrt username
		Password:        "admin",    // Change to your known password
		Ports:           defaultPorts,
	}

	fmt.Println("Starting ASIC Miner Network Scanner...")
	fmt.Printf("Scanning subnet: %s\n", config.Subnet)

	miners := scanNetwork(config)
	
	fmt.Printf("\nFound %d potential ASIC miners:\n", len(miners))
	for _, miner := range miners {
		printMinerInfo(miner)
	}
}

func scanNetwork(config ScannerConfig) []ASICMiner {
	ip, ipnet, err := net.ParseCIDR(config.Subnet)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.ConcurrentScans)
	results := make(chan ASICMiner, 100)
	var mu sync.Mutex
	var miners []ASICMiner

	// Generate all IPs in subnet
	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		ips = append(ips, ip.String())
	}

	// Scan each IP
	for _, ipStr := range ips {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire
		
		go func(ip string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release

			if miner := scanHost(ip, config); miner != nil {
				results <- *miner
			}
		}(ipStr)
	}

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	for miner := range results {
		mu.Lock()
		miners = append(miners, miner)
		mu.Unlock()
	}

	return miners
}

func scanHost(ip string, config ScannerConfig) *ASICMiner {
	miner := &ASICMiner{
		IPAddress:   ip,
		Services:    make(map[string]string),
		AuthMethods: []string{},
	}

	// Fast port scan
	openPorts := scanPorts(ip, config.Ports, config.Timeout)
	if len(openPorts) == 0 {
		return nil
	}
	miner.OpenPorts = openPorts

	// Check for miner-specific signatures
	for _, port := range openPorts {
		switch port {
		case 22:
			if checkSSH(ip, config, miner) {
				miner.Services["ssh"] = "OpenSSH"
			}
		case 23:
			if checkTelnet(ip, config, miner) {
				miner.Services["telnet"] = "Telnet"
			}
		case 80, 8080:
			checkHTTP(ip, port, config, miner, false)
		case 443:
			checkHTTP(ip, port, config, miner, true)
		case 4028, 4029:
			miner.Services["cgminer-api"] = fmt.Sprintf("Port %d", port)
			miner.DeviceType = guessDeviceType(port)
		}
	}

	// Determine if this looks like a miner
	if isMinerDevice(miner) {
		return miner
	}
	return nil
}

func scanPorts(ip string, ports []int, timeout time.Duration) []int {
	var openPorts []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			address := fmt.Sprintf("%s:%d", ip, p)
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err != nil {
				return
			}
			conn.Close()
			mu.Lock()
			openPorts = append(openPorts, p)
			mu.Unlock()
		}(port)
	}
	wg.Wait()
	return openPorts
}

func checkSSH(ip string, config ScannerConfig, miner *ASICMiner) bool {
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         config.Timeout,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), sshConfig)
	if err != nil {
		// Check if it's an SSH server even if auth failed
		if strings.Contains(err.Error(), "unable to authenticate") {
			miner.AuthMethods = append(miner.AuthMethods, "ssh-password")
			return true
		}
		return false
	}
	defer client.Close()

	miner.Accessible = true
	miner.AuthMethods = append(miner.AuthMethods, "ssh-success")
	
	// Try to get hostname
	session, err := client.NewSession()
	if err == nil {
		output, _ := session.Output("hostname")
		if len(output) > 0 {
			miner.Hostname = strings.TrimSpace(string(output))
		}
		session.Close()
	}

	return true
}

func checkTelnet(ip string, config ScannerConfig, miner *ASICMiner) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:23", ip), config.Timeout)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Set deadlines
	conn.SetReadDeadline(time.Now().Add(config.Timeout))
	conn.SetWriteDeadline(time.Now().Add(config.Timeout))

	// Read banner
	reader := bufio.NewReader(conn)
	banner, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}

	if strings.Contains(strings.ToLower(banner), "login") {
		miner.AuthMethods = append(miner.AuthMethods, "telnet")
		
		// Try to login
		fmt.Fprintf(conn, "%s\n", config.Username)
		time.Sleep(100 * time.Millisecond)
		fmt.Fprintf(conn, "%s\n", config.Password)
		time.Sleep(100 * time.Millisecond)

		// Try to read prompt
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		response, _ := reader.ReadString('$')
		if strings.Contains(response, "$") || strings.Contains(response, "#") {
			miner.Accessible = true
		}
	}

	return true
}

func checkHTTP(ip string, port int, config ScannerConfig, miner *ASICMiner, isTLS bool) {
	scheme := "http"
	if isTLS {
		scheme = "https"
	}
	
	url := fmt.Sprintf("%s://%s:%d/", scheme, ip, port)
	
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   config.Timeout,
			ResponseHeaderTimeout: config.Timeout,
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Read body for signatures
	bodyBytes := make([]byte, 2048)
	n, _ := io.ReadFull(resp.Body, bodyBytes)
	body := string(bodyBytes[:n])
	bodyLower := strings.ToLower(body)

	// Check for miner signatures
	for _, sig := range minerSignatures {
		if strings.Contains(bodyLower, sig) {
			miner.DeviceType = sig
			miner.Services[scheme] = "ASIC Web Interface"
			break
		}
	}

	// Extract title
	if title := extractTitle(body); title != "" {
		miner.WebTitle = title
	}

	// Check for authentication mechanisms
	if resp.StatusCode == 401 {
		miner.AuthMethods = append(miner.AuthMethods, fmt.Sprintf("%s-basic-auth", scheme))
	} else if strings.Contains(bodyLower, "login") || strings.Contains(bodyLower, "password") {
		miner.AuthMethods = append(miner.AuthMethods, fmt.Sprintf("%s-form-auth", scheme))
	}

	// Try form-based authentication (common on OpenWrt/Antminer)
	if strings.Contains(bodyLower, "openwrt") || strings.Contains(bodyLower, "luci") {
		miner.DeviceType = "OpenWrt"
		tryOpenWrtLogin(ip, port, scheme, config, miner)
	}
}

func tryOpenWrtLogin(ip string, port int, scheme string, config ScannerConfig, miner *ASICMiner) {
	loginURL := fmt.Sprintf("%s://%s:%d/cgi-bin/luci", scheme, ip, port)
	
	data := url.Values{}
	data.Set("luci_username", config.Username)
	data.Set("luci_password", config.Password)

	client := &http.Client{
		Timeout: config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	resp, err := client.PostForm(loginURL, data)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Check if login succeeded (usually redirects to admin page with session cookie)
	if resp.StatusCode == 302 || resp.StatusCode == 303 {
		location := resp.Header.Get("Location")
		if strings.Contains(location, "admin") || strings.Contains(location, "overview") {
			miner.Accessible = true
			miner.AuthMethods = append(miner.AuthMethods, "openwrt-web-success")
		}
	}
}

func isMinerDevice(miner *ASICMiner) bool {
	// Check for mining-specific ports
	hasMiningPort := false
	for _, port := range miner.OpenPorts {
		if port == 4028 || port == 4029 || port == 3333 || port == 3334 {
			hasMiningPort = true
			break
		}
	}

	// Check for device type identification
	hasMinerSignature := miner.DeviceType != ""

	// Check for web interfaces common on miners
	hasWebInterface := false
	for svc := range miner.Services {
		if svc == "http" || svc == "https" {
			hasWebInterface = true
			break
		}
	}

	return hasMiningPort || hasMinerSignature || (hasWebInterface && len(miner.OpenPorts) <= 5)
}

func guessDeviceType(port int) string {
	switch port {
	case 4028:
		return "CGMiner"
	case 4029:
		return "BMMiner/Antminer"
	default:
		return "Unknown Miner"
	}
}

func extractTitle(html string) string {
	start := strings.Index(strings.ToLower(html), "<title>")
	if start == -1 {
		return ""
	}
	start += 7
	end := strings.Index(strings.ToLower(html)[start:], "</title>")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(html[start : start+end])
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func printMinerInfo(miner ASICMiner) {
	data, _ := json.MarshalIndent(miner, "", "  ")
	fmt.Println(string(data))
	fmt.Println("---")
}
