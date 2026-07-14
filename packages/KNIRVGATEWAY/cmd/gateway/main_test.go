package main

import "testing"

func TestCanStartCloudflareTunnel(t *testing.T) {
	tests := []struct {
		name                  string
		apiToken, zone, token string
		want                  bool
	}{
		{name: "pre-provisioned tunnel token", token: "tunnel-token", want: true},
		{name: "API provisioning credentials", apiToken: "api-token", zone: "zone-id", want: true},
		{name: "API token without zone", apiToken: "api-token", want: false},
		{name: "zone without API token", zone: "zone-id", want: false},
		{name: "no credentials", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canStartCloudflareTunnel(tt.apiToken, tt.zone, tt.token); got != tt.want {
				t.Fatalf("canStartCloudflareTunnel() = %v, want %v", got, tt.want)
			}
		})
	}
}
