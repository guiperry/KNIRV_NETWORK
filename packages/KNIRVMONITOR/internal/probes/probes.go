package probes

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Metric struct {
	Name   string
	Labels map[string]string
	Value  float64
}

type ProbeResult struct {
	Metrics map[string]Metric
	Scraped time.Time
}

type Probe interface {
	Name() string
	Scrape() (*ProbeResult, error)
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func scrapeURL(url string) (*ProbeResult, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	metrics := make(map[string]Metric)
	scanner := bufio.NewScanner(resp.Body)

	var currentName string
	var currentHelp string
	var currentType string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 2 {
				continue
			}
			switch parts[1] {
			case "HELP":
				if len(parts) >= 3 {
					currentHelp = parts[2]
				}
			case "TYPE":
				if len(parts) >= 3 {
					currentType = parts[2]
				}
			}
			continue
		}

		name, value, labels, err := parseMetricLine(line)
		if err != nil {
			continue
		}

		if currentType == "SUMMARY" || currentType == "HISTOGRAM" {
			continue
		}

		key := name
		if len(labels) > 0 {
			var pairs []string
			for k, v := range labels {
				pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
			}
			sortPairs(pairs)
			key = fmt.Sprintf("%s{%s}", name, strings.Join(pairs, ","))
		}

		metrics[key] = Metric{
			Name:   name,
			Labels: labels,
			Value:  value,
		}
		_ = currentName
		_ = currentHelp
		_ = currentType
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", url, err)
	}

	return &ProbeResult{
		Metrics: metrics,
		Scraped: time.Now(),
	}, nil
}

func parseMetricLine(line string) (string, float64, map[string]string, error) {
	idx := strings.Index(line, "{")
	if idx >= 0 {
		closeIdx := strings.LastIndex(line, "}")
		if closeIdx < 0 {
			return "", 0, nil, fmt.Errorf("malformed labels")
		}
		name := line[:idx]
		labelStr := line[idx+1 : closeIdx]
		labels := parseLabels(labelStr)
		rest := strings.TrimSpace(line[closeIdx+1:])
		parts := strings.SplitN(rest, " ", 2)
		if len(parts) != 2 {
			return "", 0, nil, fmt.Errorf("malformed value")
		}
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return "", 0, nil, err
		}
		return name, val, labels, nil
	}

	parts := strings.Fields(line)
	if len(parts) != 2 {
		return "", 0, nil, fmt.Errorf("malformed metric line")
	}
	val, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return "", 0, nil, err
	}
	return parts[0], val, nil, nil
}

func parseLabels(s string) map[string]string {
	labels := make(map[string]string)
	var currentKey string
	var currentValue strings.Builder
	inValue := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if !inValue {
			if ch == '=' {
				currentKey = strings.TrimSpace(s[max(0, i-len(strings.TrimLeft(s[:i], " "))):i])
				inValue = true
				currentValue.Reset()
			}
		} else {
			if ch == '"' {
				if i+1 < len(s) && s[i+1] == '"' {
					currentValue.WriteByte('"')
					i++
				}
			} else if ch == ',' {
				val := strings.TrimSpace(currentValue.String())
				labels[currentKey] = val
				inValue = false
				currentKey = ""
			}
		}
	}

	if inValue && currentKey != "" {
		labels[currentKey] = strings.TrimSpace(currentValue.String())
	}

	return labels
}

func sortPairs(pairs []string) {
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j] < pairs[i] {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type ProbeManager struct {
	mu      sync.RWMutex
	probes  map[string]Probe
	results map[string]*ProbeResult
}

func NewProbeManager() *ProbeManager {
	return &ProbeManager{
		probes:  make(map[string]Probe),
		results: make(map[string]*ProbeResult),
	}
}

func (pm *ProbeManager) Register(p Probe) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.probes[p.Name()] = p
}

func (pm *ProbeManager) ScrapeAll() map[string]*ProbeResult {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	results := make(map[string]*ProbeResult)
	for name, probe := range pm.probes {
		result, err := probe.Scrape()
		if err != nil {
			result = &ProbeResult{
				Metrics: make(map[string]Metric),
				Scraped: time.Now(),
			}
		}
		results[name] = result
		pm.results[name] = result
	}
	return results
}

func (pm *ProbeManager) GetResult(name string) (*ProbeResult, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	r, ok := pm.results[name]
	return r, ok
}
