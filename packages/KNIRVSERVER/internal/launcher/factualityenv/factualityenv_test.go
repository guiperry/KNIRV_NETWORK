package factualityenv

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestFactualityEmbeddingConfigPassesToMapper(t *testing.T) {
	for _, tc := range []struct{ name, yaml, want string }{
		{"explicit opt-in", "factuality:\n  embeddings:\n    data_class: Private\n    allow_cloudflare_for_private: true\n", "KNIRV_EMBEDDINGS_ALLOW_CLOUDFLARE_FOR_PRIVATE=true"},
		{"explicit denial overrides inherited opt-in", "factuality:\n  embeddings:\n    allow_cloudflare_for_private: false\n", "KNIRV_EMBEDDINGS_ALLOW_CLOUDFLARE_FOR_PRIVATE=false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := viper.New()
			cfg.SetConfigType("yaml")
			if err := cfg.ReadConfig(strings.NewReader(tc.yaml)); err != nil {
				t.Fatal(err)
			}
			env := EmbeddingEnv([]string{"KNIRV_EMBEDDINGS_ALLOW_CLOUDFLARE_FOR_PRIVATE=true", "UNRELATED=1"}, cfg)
			matches := 0
			for _, item := range env {
				if strings.HasPrefix(item, "KNIRV_EMBEDDINGS_ALLOW_CLOUDFLARE_FOR_PRIVATE=") {
					matches++
					if item != tc.want {
						t.Fatalf("got %q want %q", item, tc.want)
					}
				}
			}
			if matches != 1 {
				t.Fatalf("opt-in occurrences=%d env=%v", matches, env)
			}
			if tc.name == "explicit opt-in" && !strings.Contains(strings.Join(env, "\n"), "KNIRV_EMBEDDING_DATA_CLASS=private") {
				t.Fatal("class not propagated")
			}
		})
	}
}

func TestFactualityFlashConfigPassesToModels(t *testing.T) {
	for _, tc := range []struct {
		name        string
		yaml        string
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name:        "disabled turns flash off",
			yaml:        "factuality:\n  flash:\n    enabled: false\n",
			wantPresent: []string{"KNIRV_FLASH=false"},
		},
		{
			name: "metasearch configured maps all keys",
			yaml: "factuality:\n  flash:\n    enabled: true\n    metasearch_url: http://127.0.0.1:8080\n    metasearch_engines: [duckduckgo, wikipedia]\n    allow_domains: [example.com]\n    deny_domains: [spam.example]\n",
			wantPresent: []string{
				"KNIRV_FLASH=true",
				"KNIRV_METASEARCH_URL=http://127.0.0.1:8080",
				"KNIRV_METASEARCH_ENGINES=duckduckgo,wikipedia",
				"KNIRV_METASEARCH_ALLOW=example.com",
				"KNIRV_METASEARCH_DENY=spam.example",
			},
		},
		{
			name:        "unset leaves flash at daemon default",
			yaml:        "factuality:\n  embeddings:\n    data_class: Public\n",
			wantAbsent:  []string{"KNIRV_FLASH", "KNIRV_METASEARCH_URL", "KNIRV_METASEARCH_ENGINES", "KNIRV_METASEARCH_ALLOW", "KNIRV_METASEARCH_DENY"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := viper.New()
			cfg.SetConfigType("yaml")
			if err := cfg.ReadConfig(strings.NewReader(tc.yaml)); err != nil {
				t.Fatal(err)
			}
			env := FlashEnv([]string{}, cfg)
			for _, want := range tc.wantPresent {
				found := false
				for _, item := range env {
					if item == want {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("missing %q in env=%v", want, env)
				}
			}
			for _, absent := range tc.wantAbsent {
				for _, item := range env {
					if strings.HasPrefix(item, absent+"=") {
						t.Fatalf("unexpected %q present in env=%v", item, env)
					}
				}
			}
		})
	}
}
