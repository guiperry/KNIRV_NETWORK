package main

import "testing"

func TestUpdateEnabled(t *testing.T) {
	env := func(vals map[string]string) func(string) string {
		return func(k string) string { return vals[k] }
	}
	tests := []struct {
		name       string
		serverPath string
		env        map[string]string
		want       bool
	}{
		{name: "default checks for updates", want: true},
		{name: "KNIRV_LLAMA_UPDATE=0 pins the binary", env: map[string]string{"KNIRV_LLAMA_UPDATE": "0"}, want: false},
		{name: "any other value keeps updates on", env: map[string]string{"KNIRV_LLAMA_UPDATE": "1"}, want: true},
		{name: "-server-path pins the binary", serverPath: "/opt/llama-server", want: false},
		{name: "-server-path and the gate both pin", serverPath: "/opt/llama-server", env: map[string]string{"KNIRV_LLAMA_UPDATE": "0"}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := updateEnabled(tc.serverPath, env(tc.env)); got != tc.want {
				t.Fatalf("updateEnabled = %v, want %v", got, tc.want)
			}
		})
	}
}
