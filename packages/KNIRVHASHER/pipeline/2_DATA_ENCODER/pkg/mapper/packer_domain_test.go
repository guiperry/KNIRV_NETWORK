package mapper

import "testing"

func TestHeadingDrivenDomainDetection(t *testing.T) {
	tests := []struct {
		heading string
		want    uint32
	}{
		{"arXiv heading: A theorem for algebraic topology", 0x2000},
		{"arXiv heading: Compiler optimization for source code", 0x3000},
		{"arXiv heading: Neural language models for scientific research", 0x4000},
	}
	for _, tt := range tests {
		if got := detectDomain(tt.heading, ""); got != tt.want {
			t.Errorf("detectDomain(%q)=%#x, want %#x", tt.heading, got, tt.want)
		}
	}
}
