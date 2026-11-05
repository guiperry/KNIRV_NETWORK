package knirvbao

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestConfigUsesLoopbackAndNeverContainsSeal(t *testing.T) {
	key := bytes.Repeat([]byte{0x6d}, 32)
	sha := strings.Repeat("a1", 32)
	config, err := renderConfig("/tmp/knirv-test", "/tmp/knirv-test/bin", sha, "127.0.0.1:12345", "127.0.0.1:12346", key)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`address = "127.0.0.1:12345"`, `cluster_address = "127.0.0.1:12346"`, `socket_mode = "0600"`, `socket_user = "` + strconv.Itoa(os.Getuid()) + `"`, `socket_group = "` + strconv.Itoa(os.Getgid()) + `"`, `current_key = "env://KNIRV_BAO_SEAL_KEY"`, `sha256sum = "` + sha + `"`, `audit "file" "knirv"`} {
		if !strings.Contains(config, required) {
			t.Errorf("missing configuration field %q", required)
		}
	}
	if strings.Contains(config, hex.EncodeToString(key)) || strings.Contains(config, base64.StdEncoding.EncodeToString(key)) {
		t.Fatal("configuration contains seal key")
	}
	for _, tc := range []struct {
		root, dir, digest, api, cluster string
		key                             []byte
	}{
		{"relative", "/tmp/bin", sha, "127.0.0.1:1", "127.0.0.1:2", key},
		{"/tmp/${unsafe}", "/tmp/bin", sha, "127.0.0.1:1", "127.0.0.1:2", key},
		{"/tmp/" + strings.Repeat("x", 100), "/tmp/bin", sha, "127.0.0.1:1", "127.0.0.1:2", key},
		{"/tmp/root", "/tmp/bin", sha, "0.0.0.0:1", "127.0.0.1:2", key},
		{"/tmp/root", "/tmp/bin", sha, "127.0.0.1:1", "127.0.0.1:1", key},
		{"/tmp/root", "/tmp/bin", "bad", "127.0.0.1:1", "127.0.0.1:2", key},
		{"/tmp/root", "/tmp/bin", sha, "127.0.0.1:1", "127.0.0.1:2", key[:1]},
	} {
		if _, err := renderConfig(tc.root, tc.dir, tc.digest, tc.api, tc.cluster, tc.key); err == nil {
			t.Fatal("invalid config was accepted")
		}
	}
}
