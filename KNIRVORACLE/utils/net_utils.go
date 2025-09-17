package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// IPInfoResponse represents the JSON response from ipinfo.io
type IPInfoResponse struct {
	IP            string `json:"ip"`
	ASN           string `json:"asn"`
	ASName        string `json:"as_name"`
	ASDomain      string `json:"as_domain"`
	CountryCode   string `json:"country_code"`
	Country       string `json:"country"`
	ContinentCode string `json:"continent_code"`
	Continent     string `json:"continent"`
}

// Global variable to store the last successful IPInfo response
// This will be used by fetchAndStorePublicIPInfo in main.go
var LastIPInfoResponse *IPInfoResponse

// FetchPublicIPFromIPInfo attempts to get the public IP from ipinfo.io's free tier.
// It parses the JSON response and returns the IP address.
func FetchPublicIPFromIPInfo() (string, error) {
	client := http.Client{
		Timeout: 20 * time.Second, // Increased timeout to 20 seconds
	}

	// Use a channel to handle timeout more gracefully
	resultCh := make(chan struct {
		ip     string
		ipInfo *IPInfoResponse
		err    error
	}, 1)

	go func() {
		resp, err := client.Get("https://ipinfo.io?token=3f1ec37a6cafc3")
		if err != nil {
			resultCh <- struct {
				ip     string
				ipInfo *IPInfoResponse
				err    error
			}{"", nil, fmt.Errorf("request to ipinfo.io failed: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := ioutil.ReadAll(resp.Body) // Read body for more error info
			resultCh <- struct {
				ip     string
				ipInfo *IPInfoResponse
				err    error
			}{"", nil, fmt.Errorf("ipinfo.io request returned status %d: %s", resp.StatusCode, string(bodyBytes))}
			return
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			resultCh <- struct {
				ip     string
				ipInfo *IPInfoResponse
				err    error
			}{"", nil, fmt.Errorf("failed to read response body from ipinfo.io: %w", err)}
			return
		}

		// Try to parse the JSON response
		var ipInfo IPInfoResponse
		if err := json.Unmarshal(bodyBytes, &ipInfo); err != nil {
			// If JSON parsing fails, try to use the response as a plain IP string (fallback)
			ip := strings.TrimSpace(string(bodyBytes))
			if net.ParseIP(ip) == nil {
				resultCh <- struct {
					ip     string
					ipInfo *IPInfoResponse
					err    error
				}{"", nil, fmt.Errorf("ipinfo.io returned invalid data, not JSON or IP: '%s'", ip)}
				return
			}
			resultCh <- struct {
				ip     string
				ipInfo *IPInfoResponse
				err    error
			}{ip, nil, nil}
			return
		}

		// Validate the IP from the JSON response
		if ipInfo.IP == "" || net.ParseIP(ipInfo.IP) == nil {
			resultCh <- struct {
				ip     string
				ipInfo *IPInfoResponse
				err    error
			}{"", nil, fmt.Errorf("ipinfo.io returned an invalid IP address in JSON: '%s'", ipInfo.IP)}
			return
		}

		// Store the IPInfo response globally
		LastIPInfoResponse = &ipInfo

		log.Printf("Successfully parsed IPInfo response: IP=%s, Country=%s, ASN=%s",
			ipInfo.IP, ipInfo.Country, ipInfo.ASN)

		resultCh <- struct {
			ip     string
			ipInfo *IPInfoResponse
			err    error
		}{ipInfo.IP, &ipInfo, nil}
	}()

	// Wait for result with a timeout to prevent hanging
	select {
	case result := <-resultCh:
		// If we have IPInfo data and a role is set, try to save it
		if result.ipInfo != nil {
			// We'll save this in the caller context
			LastIPInfoResponse = result.ipInfo
		}
		return result.ip, result.err
	case <-time.After(30 * time.Second): // Increased timeout for all operations
		// If we timeout, try one last direct attempt at local IP
		return "", fmt.Errorf("timed out waiting for ipinfo.io response")
	}
}

// getLocalIPFallbacks iterates over local network interfaces to find a suitable non-loopback IP.
// This is used as a fallback if fetching the public IP fails.
func getLocalIPFallbacks() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue // interface down or loopback
		}
		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf("Warning: failed to get addresses for interface %s: %v", iface.Name, err)
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()                              // Prefer IPv4
			if ip != nil && !ip.IsLinkLocalUnicast() { // Prefer non-link-local
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no suitable local IP address found via interface iteration")
}

// GetHostIP attempts to determine the host's primary IP address.
// It first tries to fetch the public IP from ipinfo.io.
// If that fails, it falls back to iterating local network interfaces.
func GetHostIP() (string, error) {
	// Use a channel with timeout to prevent blocking indefinitely
	resultCh := make(chan struct {
		ip     string
		ipInfo *IPInfoResponse
		err    error
	}, 1)

	go func() {
		// Try to get public IP first
		publicIP, err := FetchPublicIPFromIPInfo()
		if err == nil {
			log.Printf("Successfully fetched public IP from ipinfo.io: %s", publicIP)

			// If we have IPInfo data, pass it along
			if LastIPInfoResponse != nil {
				resultCh <- struct {
					ip     string
					ipInfo *IPInfoResponse
					err    error
				}{publicIP, LastIPInfoResponse, nil}
				return
			}

			resultCh <- struct {
				ip     string
				ipInfo *IPInfoResponse
				err    error
			}{publicIP, nil, nil}
			return
		}

		// Log the error and try fallback
		log.Printf("Warning: Failed to fetch public IP from ipinfo.io (%v). Falling back to local IP detection.", err)

		// Fallback to local IP detection
		localIP, localErr := getLocalIPFallbacks()
		if localErr != nil {
			resultCh <- struct {
				ip     string
				ipInfo *IPInfoResponse
				err    error
			}{"", nil, fmt.Errorf("all IP detection methods failed; ipinfo.io error: %v, local IP error: %w", err, localErr)}
			return
		}

		log.Printf("Determined local IP (fallback): %s", localIP)
		resultCh <- struct {
			ip     string
			ipInfo *IPInfoResponse
			err    error
		}{localIP, nil, nil}
	}()

	// Wait for result with a timeout to prevent hanging
	select {
	case result := <-resultCh:
		// If we have IPInfo data and a role is set, try to save it
		if result.ipInfo != nil {
			// We'll save this in the caller context
			LastIPInfoResponse = result.ipInfo
		}
		return result.ip, result.err
	case <-time.After(30 * time.Second): // Increased timeout for all operations
		// If we timeout, try one last direct attempt at local IP
		localIP, err := getLocalIPFallbacks()
		if err != nil {
			return "localhost", fmt.Errorf("IP detection timed out and local fallback failed: %w", err)
		}
		log.Printf("IP detection timed out. Using direct local IP fallback: %s", localIP)
		return localIP, nil
	}
}
