package cognitiveengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type PolicyLoader struct {
	basePath     string
	opaURL       string
	opaEnabled   bool
	httpClient   *http.Client
	watchEnabled bool
	stopCh       chan struct{}
	eventBus     *EventBus
}

type PolicyLoaderConfig struct {
	BasePath     string
	OPAURL       string
	OPAEnabled   bool
	WatchEnabled bool
}

type ExternalPolicy struct {
	ID          string   `json:"id" yaml:"id"`
	Description string   `json:"description" yaml:"description"`
	Version     string   `json:"version" yaml:"version"`
	DVEIDs      []string `json:"dve_ids" yaml:"dve_ids"`
	Rules       []struct {
		ID         string   `json:"id" yaml:"id"`
		Metric     string   `json:"metric" yaml:"metric"`
		Operator   string   `json:"operator" yaml:"operator"`
		Threshold  float64  `json:"threshold" yaml:"threshold"`
		Severity   string   `json:"severity" yaml:"severity"`
		Action     string   `json:"action" yaml:"action"`
		Conditions []string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
		Cooldown   string   `json:"cooldown,omitempty" yaml:"cooldown,omitempty"`
	} `json:"rules" yaml:"rules"`
}

type PolicyBundle struct {
	Policies []ExternalPolicy `json:"policies" yaml:"policies"`
	Metadata struct {
		Version   string    `json:"version" yaml:"version"`
		UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	} `json:"metadata" yaml:"metadata"`
}

type OPARule struct {
	Modules []struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	} `json:"modules,omitempty"`
	Rules []struct {
		Head struct {
			Variable string      `json:"variable"`
			Value    interface{} `json:"value"`
		} `json:"head"`
		Body []struct {
			Unify interface{} `json:"Unify"`
			Vars  []struct {
				Text  string      `json:"Text"`
				Value interface{} `json:"value"`
			} `json:"vars,omitempty"`
		} `json:"body,omitempty"`
	} `json:"rules"`
	Packages []struct {
		Path string `json:"path"`
	} `json:"packages,omitempty"`
}

func DefaultPolicyLoaderConfig() *PolicyLoaderConfig {
	return &PolicyLoaderConfig{
		BasePath:     "/etc/knirv/policies",
		WatchEnabled: true,
	}
}

func NewPolicyLoader(cfg *PolicyLoaderConfig, eventBus *EventBus) (*PolicyLoader, error) {
	if cfg == nil {
		cfg = DefaultPolicyLoaderConfig()
	}

	pl := &PolicyLoader{
		basePath:     cfg.BasePath,
		opaURL:       cfg.OPAURL,
		opaEnabled:   cfg.OPAEnabled,
		watchEnabled: cfg.WatchEnabled,
		stopCh:       make(chan struct{}),
		eventBus:     eventBus,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if err := os.MkdirAll(pl.basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create policy directory: %w", err)
	}

	return pl, nil
}

func (pl *PolicyLoader) LoadPolicies(ctx context.Context) ([]*PolicyRule, error) {
	var allPolicies []*PolicyRule

	if pl.opaEnabled && pl.opaURL != "" {
		opaPolicies, err := pl.loadFromOPA(ctx)
		if err != nil {
			log.Printf("PolicyLoader: OPA loading failed, falling back to file-based: %v", err)
		} else {
			allPolicies = append(allPolicies, opaPolicies...)
		}
	}

	filePolicies, err := pl.loadFromFiles()
	if err != nil {
		log.Printf("PolicyLoader: file-based loading failed: %v", err)
	} else {
		allPolicies = append(allPolicies, filePolicies...)
	}

	if len(allPolicies) == 0 {
		log.Printf("PolicyLoader: no policies loaded, using defaults")
		return pl.getDefaultPolicies(), nil
	}

	log.Printf("PolicyLoader: loaded %d policies", len(allPolicies))
	return allPolicies, nil
}

func (pl *PolicyLoader) loadFromOPA(ctx context.Context) ([]*PolicyRule, error) {
	url := fmt.Sprintf("%s/v1/policies", strings.TrimSuffix(pl.opaURL, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := pl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var bundle PolicyBundle
	if err := json.Unmarshal(body, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse OPA response: %w", err)
	}

	return pl.convertBundleToPolicies(&bundle), nil
}

func (pl *PolicyLoader) loadFromFiles() ([]*PolicyRule, error) {
	entries, err := os.ReadDir(pl.basePath)
	if err != nil {
		return nil, err
	}

	var policies []*PolicyRule

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		filePath := filepath.Join(pl.basePath, entry.Name())
		filePolicies, err := pl.loadPolicyFile(filePath)
		if err != nil {
			log.Printf("PolicyLoader: failed to load %s: %v", filePath, err)
			continue
		}

		policies = append(policies, filePolicies...)
	}

	return policies, nil
}

func (pl *PolicyLoader) loadPolicyFile(path string) ([]*PolicyRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var bundle PolicyBundle
	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &bundle); err != nil {
			return nil, fmt.Errorf("JSON parse error: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &bundle); err != nil {
			return nil, fmt.Errorf("YAML parse error: %w", err)
		}
	}

	return pl.convertBundleToPolicies(&bundle), nil
}

func (pl *PolicyLoader) convertBundleToPolicies(bundle *PolicyBundle) []*PolicyRule {
	var policies []*PolicyRule

	for _, extPolicy := range bundle.Policies {
		for _, rule := range extPolicy.Rules {
			dveID := ""
			if len(extPolicy.DVEIDs) > 0 {
				dveID = extPolicy.DVEIDs[0]
			}

			policy := &PolicyRule{
				ID:                rule.ID,
				Description:       extPolicy.Description,
				DVEID:             dveID,
				Metric:            rule.Metric,
				Operator:          rule.Operator,
				Threshold:         rule.Threshold,
				Severity:          rule.Severity,
				RemediationAction: rule.Action,
				Enabled:           true,
				CreatedAt:         time.Now(),
			}

			if rule.Cooldown != "" {
				if _, err := time.ParseDuration(rule.Cooldown); err == nil {
					log.Printf("PolicyLoader: parsed cooldown %s for rule %s", rule.Cooldown, rule.ID)
				}
			}

			policies = append(policies, policy)
		}
	}

	return policies
}

func (pl *PolicyLoader) getDefaultPolicies() []*PolicyRule {
	return []*PolicyRule{
		{
			ID:                "dveguard_low_success",
			Description:       "DVE node has critically low task success rate",
			Metric:            "success_rate",
			Operator:          "lt",
			Threshold:         0.4,
			Severity:          "critical",
			RemediationAction: "quarantine_node",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                "dveguard_slow_response",
			Description:       "DVE node average response time exceeds safety threshold",
			Metric:            "avg_processing_time",
			Operator:          "gt",
			Threshold:         300.0,
			Severity:          "warning",
			RemediationAction: "redistribute_tasks",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                "dveguard_high_resource",
			Description:       "DVE node resource utilization is critically high",
			Metric:            "resource_utilization",
			Operator:          "gt",
			Threshold:         0.95,
			Severity:          "critical",
			RemediationAction: "scale_resources",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                "dveguard_panic_trigger",
			Description:       "DVE node has breached multiple critical policies",
			Metric:            "violation_count",
			Operator:          "gt",
			Threshold:         5.0,
			Severity:          "panic",
			RemediationAction: "kernel_isolation",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
	}
}

func (pl *PolicyLoader) SavePolicy(policy *PolicyRule, format string) error {
	ext := ".yaml"
	if format == "json" {
		ext = ".json"
	}

	filename := fmt.Sprintf("%s%s", policy.ID, ext)
	filePath := filepath.Join(pl.basePath, filename)

	var data []byte
	var err error

	if format == "json" {
		data, err = json.MarshalIndent(policy, "", "  ")
	} else {
		data, err = yaml.Marshal(policy)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write policy file: %w", err)
	}

	log.Printf("PolicyLoader: saved policy %s to %s", policy.ID, filePath)
	return nil
}

func (pl *PolicyLoader) DeletePolicy(policyID string) error {
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		filePath := filepath.Join(pl.basePath, policyID+ext)
		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete policy file: %w", err)
			}
			log.Printf("PolicyLoader: deleted policy %s", policyID)
			return nil
		}
	}

	return fmt.Errorf("policy %s not found", policyID)
}

func (pl *PolicyLoader) EvaluateWithOPA(ctx context.Context, input map[string]interface{}) (bool, error) {
	if !pl.opaEnabled || pl.opaURL == "" {
		return false, fmt.Errorf("OPA not configured")
	}

	url := fmt.Sprintf("%s/v1/data/knirv/allow", strings.TrimSuffix(pl.opaURL, "/"))

	body, err := json.Marshal(map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pl.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("OPA evaluation failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Result bool `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Result, nil
}

func (pl *PolicyLoader) ValidatePolicy(policy *PolicyRule) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}

	validMetrics := map[string]bool{
		"success_rate":         true,
		"avg_processing_time":  true,
		"resource_utilization": true,
		"violation_count":      true,
		"cpu_usage":            true,
		"memory_usage":         true,
		"latency_p99":          true,
		"error_rate":           true,
	}

	if !validMetrics[policy.Metric] {
		return fmt.Errorf("invalid metric: %s", policy.Metric)
	}

	validOperators := map[string]bool{
		"lt":  true,
		"gt":  true,
		"eq":  true,
		"lte": true,
		"gte": true,
	}

	if !validOperators[policy.Operator] {
		return fmt.Errorf("invalid operator: %s", policy.Operator)
	}

	validSeverities := map[string]bool{
		"warning":  true,
		"critical": true,
		"panic":    true,
	}

	if !validSeverities[policy.Severity] {
		return fmt.Errorf("invalid severity: %s", policy.Severity)
	}

	return nil
}

func (pl *PolicyLoader) StartWatching() error {
	if !pl.watchEnabled {
		return nil
	}

	go pl.watchLoop()
	log.Printf("PolicyLoader: started watching %s", pl.basePath)
	return nil
}

func (pl *PolicyLoader) StopWatching() {
	close(pl.stopCh)
	pl.stopCh = make(chan struct{})
}

func (pl *PolicyLoader) watchLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pl.stopCh:
			return
		case <-ticker.C:
			pl.checkForUpdates()
		}
	}
}

func (pl *PolicyLoader) checkForUpdates() {
	policies, err := pl.loadFromFiles()
	if err != nil {
		log.Printf("PolicyLoader: watch check failed: %v", err)
		return
	}

	if pl.eventBus != nil && len(policies) > 0 {
		pl.eventBus.Publish(EngineEvent{
			Type:      EventPatternDetected,
			Source:    "policy_loader",
			Payload:   map[string]interface{}{"action": "policies_updated", "count": len(policies)},
			Timestamp: time.Now(),
		})
	}
}

func (pl *PolicyLoader) GetPolicyPaths() []string {
	var paths []string
	entries, err := os.ReadDir(pl.basePath)
	if err != nil {
		return paths
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			paths = append(paths, filepath.Join(pl.basePath, entry.Name()))
		}
	}

	return paths
}

type PolicyManager struct {
	loader     *PolicyLoader
	engine     *GuardrailEngine
	ctx        context.Context
	cancel     context.CancelFunc
	autoReload bool
}

func NewPolicyManager(loader *PolicyLoader, engine *GuardrailEngine) *PolicyManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &PolicyManager{
		loader:     loader,
		engine:     engine,
		ctx:        ctx,
		cancel:     cancel,
		autoReload: true,
	}
}

func (pm *PolicyManager) Start() error {
	if err := pm.loader.StartWatching(); err != nil {
		return err
	}

	go pm.reloadLoop()
	log.Println("PolicyManager: started")
	return nil
}

func (pm *PolicyManager) Stop() {
	pm.cancel()
	pm.loader.StopWatching()
	log.Println("PolicyManager: stopped")
}

func (pm *PolicyManager) reloadLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			if pm.autoReload {
				pm.ReloadPolicies()
			}
		}
	}
}

func (pm *PolicyManager) ReloadPolicies() error {
	policies, err := pm.loader.LoadPolicies(pm.ctx)
	if err != nil {
		return fmt.Errorf("failed to reload policies: %w", err)
	}

	for _, policy := range policies {
		pm.engine.AddPolicy(policy)
	}

	log.Printf("PolicyManager: reloaded %d policies", len(policies))
	return nil
}
