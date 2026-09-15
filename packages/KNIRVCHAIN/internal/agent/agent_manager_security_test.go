package agent

import "testing"

func TestLegacyBlockVerificationFailsClosed(t *testing.T) {
	if (&Block{}).VerifyBlock() {
		t.Fatal("legacy agent block must not be accepted without canonical blockchain verification")
	}
}
