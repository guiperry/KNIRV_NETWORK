//go:build js && wasm

// KnirvAgent browser WebAssembly entrypoint.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"syscall/js"

	"github.com/knirvcorp/knirvagent/pkg/agent"
	"github.com/knirvcorp/knirvagent/pkg/bus"
	"github.com/knirvcorp/knirvagent/pkg/config"
	"github.com/knirvcorp/knirvagent/pkg/providers"
	"github.com/knirvcorp/knirvagent/pkg/relay"
)

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string

	browserLoop  *agent.AgentLoop
	browserRelay *relay.Config
)

func formatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

func buildInfo() map[string]any {
	goVer := goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}

	return map[string]any{
		"name":      "knirvagent",
		"version":   formatVersion(),
		"buildTime": buildTime,
		"goVersion": goVer,
		"goos":      runtime.GOOS,
		"goarch":    runtime.GOARCH,
	}
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("buildInfo", js.FuncOf(func(js.Value, []js.Value) any {
		return js.ValueOf(buildInfo())
	}))
	api.Set("init", js.FuncOf(func(this js.Value, args []js.Value) any {
		configJSON := ""
		if len(args) > 0 && args[0].Type() == js.TypeString {
			configJSON = args[0].String()
		}
		return promise(func() (any, error) {
			if err := initBrowserAgent(configJSON); err != nil {
				return nil, err
			}
			return buildInfo(), nil
		})
	}))
	api.Set("ask", js.FuncOf(func(this js.Value, args []js.Value) any {
		message := ""
		session := "browser:default"
		if len(args) > 0 {
			message = args[0].String()
		}
		if len(args) > 1 && args[1].Type() == js.TypeString {
			session = args[1].String()
		}
		return promise(func() (any, error) {
			return browserAsk(message, session)
		})
	}))

	js.Global().Set("KNIRVAgent", api)
	js.Global().Set("knirvagentBuildInfo", api.Get("buildInfo"))
	fmt.Printf("knirvagent %s browser wasm loaded\n", formatVersion())
	select {}
}

type browserInitEnvelope struct {
	Config     *config.Config `json:"config,omitempty"`
	GatewayURL string         `json:"gateway_url,omitempty"`
	DVEID      string         `json:"dve_id,omitempty"`
	AuthToken  string         `json:"auth_token,omitempty"`
	Relay      bool           `json:"relay,omitempty"`
}

func initBrowserAgent(configJSON string) error {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = "/knirvagent/browser-workspace"
	relayCfg := defaultBrowserRelay()

	if configJSON != "" {
		var envelope browserInitEnvelope
		if err := json.Unmarshal([]byte(configJSON), &envelope); err != nil {
			return fmt.Errorf("invalid browser agent config JSON: %w", err)
		}
		if envelope.Config != nil {
			cfg = envelope.Config
		} else if err := json.Unmarshal([]byte(configJSON), cfg); err != nil {
			return fmt.Errorf("invalid browser agent config JSON: %w", err)
		}
		if envelope.GatewayURL != "" {
			relayCfg.GatewayURL = envelope.GatewayURL
		}
		if envelope.DVEID != "" {
			relayCfg.DVEID = envelope.DVEID
		}
		if envelope.AuthToken != "" {
			relayCfg.AuthToken = envelope.AuthToken
		}
		if envelope.Relay {
			relayCfg.Enabled = true
		}
	}

	if relayCfg.Ready() {
		relayCfg.Enabled = true
		browserRelay = &relayCfg
		browserLoop = nil
		return nil
	}
	if relayCfg.Enabled {
		return fmt.Errorf("relay mode requires gateway_url and dve_id")
	}

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}

	msgBus := bus.NewMessageBus()
	browserLoop = agent.NewAgentLoop(cfg, msgBus, provider)
	return nil
}

func browserAsk(message, session string) (string, error) {
	if message == "" {
		return "", fmt.Errorf("message is required")
	}
	if browserRelay != nil && browserRelay.Ready() {
		return browserRelay.Execute(context.Background(), message)
	}
	if browserLoop == nil {
		if err := initBrowserAgent(""); err != nil {
			return "", err
		}
	}
	return browserLoop.ProcessDirect(context.Background(), message, session)
}

func defaultBrowserRelay() relay.Config {
	cfg := relay.Config{}
	global := js.Global()
	if location := global.Get("location"); location.Truthy() {
		origin := location.Get("origin")
		if origin.Type() == js.TypeString {
			cfg.GatewayURL = origin.String()
		}
	}
	if gatewayConfig := global.Get("KNIRV_GATEWAY_CONFIG"); gatewayConfig.Truthy() {
		if services := gatewayConfig.Get("oracle_services"); services.Truthy() {
			if webgui := services.Get("webgui"); webgui.Truthy() {
				if baseURL := webgui.Get("base_url"); baseURL.Type() == js.TypeString && baseURL.String() != "" {
					cfg.GatewayURL = baseURL.String()
				}
			}
		}
	}
	return cfg
}

func promise(fn func() (any, error)) js.Value {
	promiseCtor := js.Global().Get("Promise")
	return promiseCtor.New(js.FuncOf(func(this js.Value, args []js.Value) any {
		resolve := args[0]
		reject := args[1]
		go func() {
			result, err := fn()
			if err != nil {
				reject.Invoke(err.Error())
				return
			}
			resolve.Invoke(js.ValueOf(result))
		}()
		return nil
	}))
}
