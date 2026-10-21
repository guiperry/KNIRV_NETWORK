package headless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
	"github.com/guiperry/knirv-sdk-go/models/frame"

	"knirvmodels/internal/cli/controller"
	"knirvmodels/internal/cli/embedded"
	"knirvmodels/internal/ops"
	"knirvmodels/internal/policy"
	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/head"
	"knirvmodels/pkg/heartqwen"
	"knirvmodels/pkg/livesearch"
	"knirvmodels/pkg/models"
)

// registerModels mounts the daemon's model surface under /models/*: the registry
// plus HASHER, HEART and HEAD. backend_server reaches it over this socket and
// authenticates callers; nothing here does.
//
// HEART is built on first use (it loads a tokenizer, semantic memory and an
// engine), so a daemon that never serves a generate request never pays for it.
func (s *Server) registerModels() {
	reg := models.NewRegistry()

	hasher := models.NewHasher(nil)
	hasher.UseIndex(evidenceDir(), "")
	hasher.UseLive(livesearch.NewProduction)
	reg.Register(hasher)
	heart := models.NewHeart(func() (*transformer.HEARTService, error) {
		cfg := heartConfigFromEnv()
		// Share the engine the HASHER model serves; if it cannot be built HEART
		// builds its own, as it always has.
		if e, err := hasher.Engine(); err == nil {
			cfg.Engine = e
		} else {
			log.Printf("HEART: shared HASHER engine unavailable (%v); building its own", err)
		}
		return transformer.NewHEARTServiceWithConfig(cfg)
	})
	reg.Register(heart)

	if dir, err := controller.HEADDir(); err != nil {
		log.Printf("HEAD serving disabled: %v", err)
	} else {
		reg.Register(models.NewHead(head.NewService(dir)))
		log.Printf("HEAD served from %s (/models/head/*)", dir)
	}

	reg.SetManifest(frame.Default(), liveFacts)
	if heart != nil {
		s.enableQwen(reg, heart)
	}
	reg.Mount(s.mux)
	s.registry = reg
}

// heartConfigFromEnv is the daemon's equivalent of cmd/heart's flags.
//
//	KNIRV_HEART_EXTERNAL_GENERATOR_URL  bootstrap: an external LLM writes WASM until the internal model is trained
//	KNIRV_HEART_CHECKPOINT              opt in to the legacy Gorgonite GPT checkpoint
func heartConfigFromEnv() *transformer.HEARTConfig {
	cfg := transformer.DefaultHEARTConfig(false, false)
	if u := strings.TrimSpace(os.Getenv("KNIRV_HEART_EXTERNAL_GENERATOR_URL")); u != "" {
		cfg.ExternalGeneratorURL = u
		log.Printf("HEART: external generator %s (bootstrap mode)", u)
	}
	if p := strings.TrimSpace(os.Getenv("KNIRV_HEART_CHECKPOINT")); p != "" {
		cfg.ModelCheckpointPath = p
	}
	return cfg
}

// liveFacts is what the node tells every frame about itself, read from the same
// environment the llama wrapper was started with. Unset values are omitted.
func liveFacts() frame.LiveFacts {
	f := frame.LiveFacts{Model: "local-llama", NodeMode: "local", Capabilities: map[string]bool{}}
	if v := strings.TrimSpace(os.Getenv("KNIRV_LLAMA_MODEL_NAME")); v != "" {
		f.Model = v
	}
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("KNIRV_LLAMA_CTX_SIZE"))); err == nil && n > 0 {
		f.CtxSize = n
	}
	f.Capabilities["flash_lookup"] = os.Getenv("KNIRV_FLASH_LOOKUP") == "1"
	f.Capabilities["evidence_index"] = false // no index is built yet (Phase 2)
	return f
}

// enableQwen makes the local Qwen HEART's backend (spec: HEART is Qwen behind a role
// frame). The socket is KNIRV_LLAMA_SOCKET, else <app data>/sockets/llama.sock, the
// path KNIRVSERVER gives the model harness; the daemon does not need llama to be up
// yet, health reads "qwen-unavailable" until it is and the harness falls back to the
// external generator meanwhile. KNIRV_HEART_BACKEND=gpt keeps the old backend.
func (s *Server) enableQwen(reg *models.Registry, heart *models.Heart) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("KNIRV_HEART_BACKEND")), "gpt") {
		log.Printf("HEART stays on the Gorgonite GPT (KNIRV_HEART_BACKEND=gpt)")
		return
	}
	sock := llamaSocketPath()
	llm := &heartqwen.HTTPLLM{Client: sdk.NewUnixHTTPClient(sock, 10*time.Minute), Base: "http://llama", Model: liveFacts().Model}
	audit, err := policy.OpenAudit(filepath.Join(auditDir(), "tool_audit.jsonl"))
	if err != nil {
		log.Printf("HEART on Qwen disabled: %v", err)
		return
	}
	frames := frame.Default()
	fin := transformer.NewWASMFinisher(heartConfigFromEnv())
	moduleArgs := func(raw json.RawMessage) (transformer.WASMType, json.RawMessage, error) {
		var a struct {
			Type   transformer.WASMType `json:"type"`
			Fields json.RawMessage      `json:"fields"`
		}
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", nil, err
		}
		return a.Type, a.Fields, nil
	}
	tools := ops.Standard(ops.Deps{
		RulesList: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a struct {
				Domain string `json:"domain"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			out, err := s.controller.ManageRules(ctx, "list", a.Domain)
			return map[string]string{"rules": out}, err
		},
		IndexStatus: func(context.Context, json.RawMessage) (any, error) {
			return s.routeJSON(http.MethodGet, sdk.HasherPrefix+"/index/status", nil, nil)
		},
		IndexSearch: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a struct {
				Query string `json:"query"`
				K     int    `json:"k"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			body, _ := json.Marshal(map[string]any{"text": a.Query, "k": a.K})
			// Runs as the caller's tenant and never above user privilege: a model's
			// tool call cannot reach quarantine or another tenant's documents.
			return s.routeJSON(http.MethodPost, sdk.HasherPrefix+"/index/search", body,
				map[string]string{models.TenantHeader: heartqwen.TenantFrom(ctx)})
		},
		HeadStatus: func(context.Context, json.RawMessage) (any, error) {
			return s.routeJSON(http.MethodGet, sdk.HeadPrefix+"/models", nil, nil)
		},
		HasherValidate: func(_ context.Context, raw json.RawMessage) (any, error) {
			return s.routeJSON(http.MethodPost, sdk.HasherPrefix+"/validate", raw, nil)
		},
		WASMRender: func(_ context.Context, raw json.RawMessage) (any, error) {
			typ, fields, err := moduleArgs(raw)
			if err != nil {
				return nil, err
			}
			src, err := transformer.RenderFields(typ, fields)
			if err != nil {
				return nil, err
			}
			return map[string]any{"source": src}, nil
		},
		WASMVerify: func(_ context.Context, raw json.RawMessage) (any, error) {
			typ, fields, err := moduleArgs(raw)
			if err != nil {
				return nil, err
			}
			_, ok, msg, err := fin.Check(typ, fields)
			if err != nil {
				return nil, err
			}
			return map[string]any{"source_contract_ok": ok, "message": msg, "compiled": false}, nil
		},
		Manifest: func() any { return frames.Manifest(liveFacts()) },
		Registry: func() any { return reg.Statuses() },
		Health:   func() any { return map[string]any{"status": "ok", "models": len(reg.Statuses())} },
	})
	heart.UseQwen(&heartqwen.Engine{Reg: frames, Facts: liveFacts, LLM: llm,
		Gate: policy.NewGate(frames, audit), Exec: tools},
		func(ctx context.Context) bool {
			cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			return llm.Ready(cctx)
		}, fin)
	log.Printf("HEART generate served by Qwen via %s; tool audit %s", sock, filepath.Join(auditDir(), "tool_audit.jsonl"))
}

func auditDir() string {
	if d := strings.TrimSpace(os.Getenv("KNIRV_AUDIT_DIR")); d != "" {
		return d
	}
	if d, err := embedded.GetAppDataDir(); err == nil {
		return d
	}
	return os.TempDir()
}

// routeJSON calls one of the daemon's own routes in process and returns its JSON
// body, so a tool reports exactly what the HTTP route reports. A non-200 is an error.
func (s *Server) routeJSON(method, path string, body []byte, headers map[string]string) (json.RawMessage, error) {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("%s %s: HTTP %d: %s", method, path, rec.Code, strings.TrimSpace(rec.Body.String()))
	}
	return json.RawMessage(rec.Body.Bytes()), nil
}

func llamaSocketPath() string {
	if v := strings.TrimSpace(os.Getenv("KNIRV_LLAMA_SOCKET")); v != "" {
		return v
	}
	if d, err := embedded.GetAppDataDir(); err == nil {
		return filepath.Join(d, "sockets", "llama.sock")
	}
	return "/var/lib/knirvserver/sockets/llama.sock"
}

// evidenceDir is where the evidence index lives: KNIRV_EVIDENCE_DIR, else next to the
// HEAD data under <app data>/knirvmodels/data/evidence. The same path the
// `evidence` CLI writes to, so documents a user submits are what the daemon searches.
func evidenceDir() string {
	if v := strings.TrimSpace(os.Getenv("KNIRV_EVIDENCE_DIR")); v != "" {
		return v
	}
	if d, err := embedded.GetAppDataDir(); err == nil {
		return filepath.Join(d, "knirvmodels", "data", "evidence")
	}
	return filepath.Join(os.TempDir(), "knirvmodels", "data", "evidence")
}
