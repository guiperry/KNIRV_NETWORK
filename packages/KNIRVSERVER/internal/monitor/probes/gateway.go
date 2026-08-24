package probes

import "time"

type GatewayProbe struct {
	url string
}

func NewGatewayProbe(url string) *GatewayProbe {
	return &GatewayProbe{url: url}
}

func (p *GatewayProbe) Name() string {
	return "gateway"
}

func (p *GatewayProbe) Scrape() (*ProbeResult, error) {
	return scrapeURL(p.url)
}

type GatewayRoute struct {
	Name       string `json:"name"`
	PathPrefix string `json:"pathPrefix"`
	Target     string `json:"target"`
	Protocol   string `json:"protocol"`
	Status     string `json:"status"`
	LatencyMs  int64  `json:"latencyMs"`
}

type GatewayHealth struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}
