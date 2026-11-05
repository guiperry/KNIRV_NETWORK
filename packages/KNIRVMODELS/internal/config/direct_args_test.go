package config

import (
	"reflect"
	"testing"
)

func TestDirectArgs(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  *DeviceConfig
		want []string
	}{
		{"nil", nil, nil},
		{"default", &DeviceConfig{}, nil},
		{"jitter without direct", &DeviceConfig{DirectJitter: true}, nil},
		{"direct", &DeviceConfig{DirectMode: true}, []string{"--direct"}},
		{"direct and jitter", &DeviceConfig{DirectMode: true, DirectJitter: true}, []string{"--direct", "--direct-jitter"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.DirectArgs(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("DirectArgs() = %v, want %v", got, tc.want)
			}
		})
	}
}
