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

	"knirvmodels/internal/cli/embedded"
	"knirvmodels/internal/ops"
	"knirvmodels/internal/policy"
	handspkg "knirvmodels/pkg/hands"
	"knirvmodels/pkg/hands/gpt"
	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/heartqwen"
	"knirvmodels/pkg/livesearch"
	"knirvmodels/pkg/models"
	"knirvmodels/pkg/vlm"
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
	s.enableVerifier(hasher)
	s.hasher = hasher
	reg.Register(hasher)

	// Task 3.9 follow-up: drain the witness queue off the request path,
	// witnessing each batch root via the 3_DATA_SEEDER socket. The loop stops
	// with the daemon.
	wctx, wcancel := context.WithCancel(context.Background())
	go func() {
		<-s.shutdownCh
		wcancel()
	}()
	go hasher.StartWitnessing(wctx)
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

	if models.HandsEnabled() {
		fin := transformer.NewWASMFinisher(heartConfigFromEnv())
		if h := s.buildHands(reg, hasher, fin); h != nil {
			reg.Register(h)
		}
	} else {
		log.Printf("HANDS disabled (set KNIRV_HANDS_ENABLED=1 to enable)")
	}

	// HEAD is the VLM + mirrored text generator after the migration. It is always
	// registered so its routes exist; with KNIRV_VLM_BACKEND=none (default) every
	// operation returns a structured 503 and HEAD never fabricates output.
	headBackend := vlm.FromEnv()
	reg.Register(models.NewHead(headBackend))
	log.Printf("HEAD served (/models/head/*): VLM backend %q", headBackend.Name())

	reg.SetManifest(frame.Default(), liveFacts)
	if heart != nil {
		s.enableQwen(reg, heart, hasher)
	}
	reg.Mount(s.mux)
	s.registry = reg
}

// handsEngineConfig is the GPT config the production HANDS engine is built with.
// Tests override it with a tiny config so the default (engineBuild == nil)
// constructor path is exercised cheaply.
var handsEngineConfig = gpt.DefaultConfig

// buildHands wires the HANDS server-tasks model when KNIRV_HANDS_ENABLED=1: a
// persistent audit log, the daemon's T0/T1 tool set, a policy gate, the default
// frame registry, and a checkpoint loader (model-latest in HandsDir). With no
// checkpoint the engine stays untrained and the routes return 503 (no training
// runs are triggered here). It returns nil (and logs) if the audit log cannot be
// opened, leaving HANDS disabled rather than serving unaudited.
func (s *Server) buildHands(reg *models.Registry, hasher *models.Hasher, fin *transformer.WASMFinisher) *models.Hands {
	return s.buildHandsWith(reg, hasher, fin, nil)
}

// buildHandsWith is buildHands with an optional engine constructor (engineBuild).
// When nil, the production constructor (loadHandsEngine) builds the HANDS GPT and
// loads HandsDir()/model-latest if present. Tests pass a lightweight engine (or a
// scripted generator) so the wiring (audit, gate, tools, registration, checkpoint
// load) is exercised without the full GPT.
func (s *Server) buildHandsWith(reg *models.Registry, hasher *models.Hasher, fin *transformer.WASMFinisher, engineBuild func() (models.Generator, error)) *models.Hands {
	frames := frame.Default()
	audit, aerr := policy.OpenAudit(filepath.Join(auditDir(), "hands_tool_audit.jsonl"))
	if aerr != nil {
		log.Printf("HANDS disabled: cannot open audit log: %v", aerr)
		return nil
	}
	tools := ops.Standard(s.daemonToolDeps(frames, reg, hasher, fin))
	gate := policy.NewGate(frames, audit)
	build := engineBuild
	if build == nil {
		build = func() (models.Generator, error) {
			return loadHandsEngine(models.HandsDir(), handsEngineConfig())
		}
	}
	log.Printf("HANDS served (/models/hands/*)")
	return models.NewHandsWith(build, gate, audit, tools, frames)
}

// loadHandsEngine builds the HANDS GPT with cfg and loads HandsDir()/model-latest
// when present. With no checkpoint the engine stays untrained and the routes
// return 503 (no training runs are triggered here). A corrupt checkpoint is a
// load error, not a silent fallback.
func loadHandsEngine(dir string, cfg *gpt.Config) (models.Generator, error) {
	eng := handspkg.NewEngine(cfg, "")
	if dir == "" {
		return eng, nil
	}
	cp := filepath.Join(dir, "model-latest")
	if _, serr := os.Stat(cp); serr != nil {
		return eng, nil
	}
	if lerr := eng.Load(cp); lerr != nil {
		return nil, fmt.Errorf("load HANDS checkpoint %s: %w", cp, lerr)
	}
	return eng, nil
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
	// Live lookup is on unless the deployment turned it off (KNIRV_FLASH=0 or KNIRV_NO_FLASH=1,
	// Task 3.8, owner decision O5); the frame tells the model the truth about it.
	f.Capabilities["flash_lookup"] = livesearch.Enabled()
	f.Capabilities["evidence_index"] = false // no index is built yet (Phase 2)
	return f
}

// enableQwen makes the local Qwen HEART's backend (spec: HEART is Qwen behind a role
// frame). The socket is KNIRV_LLAMA_SOCKET, else <app data>/sockets/llama.sock, the
// path KNIRVSERVER gives the model harness; the daemon does not need llama to be up
// yet, health reads "qwen-unavailable" until it is and the harness falls back to the
// external generator meanwhile. KNIRV_HEART_BACKEND=gpt keeps the old backend.
func (s *Server) enableQwen(reg *models.Registry, heart *models.Heart, hasher *models.Hasher) {
	// KNIRV_HEART_BACKEND=gpt and =hands are aliases for the same HANDS GPT
	// backend (spec §3.3): both keep the local Gorgonite/HANDS model and skip the
	// Qwen wrapper. Only the default (qwen) path enables Qwen.
	if models.HeartBackendIsHands() {
		log.Printf("HEART stays on the Gorgonite/HANDS GPT backend (KNIRV_HEART_BACKEND=%s)", models.HeartBackend())
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
	tools := ops.Standard(s.daemonToolDeps(frames, reg, hasher, fin))
	heart.UseQwen(&heartqwen.Engine{Reg: frames, Facts: liveFacts, LLM: llm,
		Gate: policy.NewGate(frames, audit), Exec: tools},
		func(ctx context.Context) bool {
			cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			return llm.Ready(cctx)
		}, fin)
	log.Printf("HEART generate served by Qwen via %s; tool audit %s", sock, filepath.Join(auditDir(), "tool_audit.jsonl"))
}

// daemonToolDeps builds the T0/T1 tool set the daemon serves (controller state,
// hasher, index, head status, wasm render/verify, and the registry/manifest/
// health surfaces). Both HEART and HANDS are wired with the same set so a HANDS
// request can run the same core-authorized tools a HEART request can.
func (s *Server) daemonToolDeps(frames *frame.Registry, reg *models.Registry, hasher *models.Hasher, fin *transformer.WASMFinisher) ops.Deps {
	return ops.Deps{
		PipelineStatus: func(context.Context, json.RawMessage) (any, error) {
			return s.controller.PipelineStatus(), nil
		},
		PipelineLogs: func(_ context.Context, raw json.RawMessage) (any, error) {
			var a struct {
				Stage string `json:"stage"`
				Lines int    `json:"lines"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			return s.controller.PipelineLogs(a.Stage, a.Lines), nil
		},
		ErrorsHistory: func(_ context.Context, raw json.RawMessage) (any, error) {
			var a struct {
				Limit int `json:"limit"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			return s.controller.ErrorHistory(a.Limit), nil
		},
		HeartDecide: func(_ context.Context, raw json.RawMessage) (any, error) {
			var a struct {
				Prompt string `json:"prompt"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			// HEART's decision core is byte-based. Sending its unsigned byte ids
			// through the route retains the model's own vocab and length checks;
			// unsupported model vocabularies fail closed.
			tokens := make([]uint32, len(a.Prompt))
			for i := range []byte(a.Prompt) {
				tokens[i] = uint32(a.Prompt[i])
			}
			body, _ := json.Marshal(map[string]any{"tokens": tokens})
			return s.routeJSON(http.MethodPost, sdk.HeartDecisionPrefix+"/reason", body, nil)
		},
		HasherVerify: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var a struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			return hasher.VerifyText(ctx, a.Text, heartqwen.TenantFrom(ctx))
		},
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
		HeartDecisionStatus: func(context.Context, json.RawMessage) (any, error) {
			return s.routeJSON(http.MethodGet, sdk.HeartDecisionPrefix+"/models", nil, nil)
		},
		HasherValidate: func(_ context.Context, raw json.RawMessage) (any, error) {
			return s.routeJSON(http.MethodPost, sdk.HasherPrefix+"/validate", raw, nil)
		},
		WASMRender: func(_ context.Context, raw json.RawMessage) (any, error) {
			typ, fields, err := s.moduleArgs(raw)
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
			typ, fields, err := s.moduleArgs(raw)
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
	}
}

// moduleArgs unpacks a wasm render/verify request's {type, fields}.
func (s *Server) moduleArgs(raw json.RawMessage) (transformer.WASMType, json.RawMessage, error) {
	var a struct {
		Type   transformer.WASMType `json:"type"`
		Fields json.RawMessage      `json:"fields"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", nil, err
	}
	return a.Type, a.Fields, nil
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
