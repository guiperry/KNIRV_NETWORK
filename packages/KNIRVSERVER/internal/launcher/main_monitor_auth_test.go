package launcher

import (
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func TestIsAdminRequestTestnetAdminToken(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	req := httptest.NewRequest("GET", "/api/v1/knirvbase/metrics", nil)
	req.Header.Set("Authorization", "Bearer TESTNET_ADMIN_TOKEN")

	viper.Set("environment", "testnet")
	if !isAdminRequest(req) {
		t.Fatal("testnet admin token should access monitor endpoints in testnet")
	}

	viper.Set("environment", "production")
	viper.Set("testnet", false)
	if isAdminRequest(req) {
		t.Fatal("testnet admin token must not access monitor endpoints outside testnet")
	}
}
