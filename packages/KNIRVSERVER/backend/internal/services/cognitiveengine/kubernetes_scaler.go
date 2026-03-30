package cognitiveengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type KubernetesScaler struct {
	nodeID          string
	deploymentName  string
	namespace       string
	enabled         bool
	mu              sync.RWMutex
	config          *KubernetesScalerConfig
	metrics         *K8sScalerMetrics
	client          *http.Client
	baseURL         string
	token           string
	currentReplicas int32
	targetReplicas  int32
	ctx             context.Context
	cancel          context.CancelFunc
}

type KubernetesScalerConfig struct {
	Enabled            bool
	APIURL             string
	Token              string
	Namespace          string
	DeploymentName     string
	MinReplicas        int32
	MaxReplicas        int32
	ScaleUpThreshold   float64
	ScaleDownThreshold float64
	PollingInterval    time.Duration
	CooldownPeriod     time.Duration
	HPAEnabled         bool
	HPAName            string
}

type K8sScalerMetrics struct {
	CurrentReplicas   int32
	ReadyReplicas     int32
	AvailableReplicas int32
	CPUUtilization    float64
	MemoryUtilization float64
	LastSyncTime      time.Time
	Errors            []string
}

type K8sDeployment struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		ResourceVersion string `json:"resourceVersion"`
	} `json:"metadata"`
	Spec struct {
		Replicas int32 `json:"replicas"`
	} `json:"spec"`
	Status struct {
		Replicas          int32 `json:"replicas"`
		ReadyReplicas     int32 `json:"readyReplicas"`
		AvailableReplicas int32 `json:"availableReplicas"`
		Conditions        []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
	} `json:"status"`
}

type K8sHPA struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		ResourceVersion string `json:"resourceVersion"`
	} `json:"metadata"`
	Spec struct {
		ScaleTargetRef struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"scaleTargetRef"`
		MinReplicas *int32 `json:"minReplicas,omitempty"`
		MaxReplicas int32  `json:"maxReplicas"`
		Metrics     []struct {
			Type     string `json:"type"`
			Resource *struct {
				Name   string `json:"name"`
				Target struct {
					Type               string   `json:"type"`
					AverageUtilization *float64 `json:"averageUtilization,omitempty"`
				} `json:"target"`
			} `json:"resource,omitempty"`
		} `json:"metrics,omitempty"`
	} `json:"spec"`
	Status struct {
		CurrentReplicas int32 `json:"currentReplicas"`
		DesiredReplicas int32 `json:"desiredReplicas"`
		LastScaleTime   *struct {
			Time time.Time `json:"time"`
		} `json:"lastScaleTime,omitempty"`
		Conditions []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
	} `json:"status"`
}

type K8sScaleRequest struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Spec       struct {
		Replicas int32 `json:"replicas"`
	} `json:"spec"`
}

type K8sPodMetrics struct {
	Items []struct {
		Metadata struct {
			Name              string    `json:"name"`
			Namespace         string    `json:"namespace"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
		} `json:"metadata"`
		Containers []struct {
			Name  string `json:"name"`
			Usage struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			} `json:"usage"`
		} `json:"containers"`
	} `json:"items"`
}

func DefaultKubernetesScalerConfig() *KubernetesScalerConfig {
	return &KubernetesScalerConfig{
		Enabled:            false,
		APIURL:             "https://kubernetes.default.svc",
		Namespace:          "default",
		DeploymentName:     "knirv-server",
		MinReplicas:        1,
		MaxReplicas:        10,
		ScaleUpThreshold:   0.7,
		ScaleDownThreshold: 0.3,
		PollingInterval:    30 * time.Second,
		CooldownPeriod:     5 * time.Minute,
		HPAEnabled:         false,
		HPAName:            "knirv-server-hpa",
	}
}

func NewKubernetesScaler(nodeID string, cfg *KubernetesScalerConfig) (*KubernetesScaler, error) {
	if cfg == nil {
		cfg = DefaultKubernetesScalerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	ks := &KubernetesScaler{
		nodeID:         nodeID,
		deploymentName: cfg.DeploymentName,
		namespace:      cfg.Namespace,
		enabled:        false,
		config:         cfg,
		metrics:        &K8sScalerMetrics{},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:         cfg.APIURL,
		token:           cfg.Token,
		currentReplicas: cfg.MinReplicas,
		targetReplicas:  cfg.MinReplicas,
		ctx:             ctx,
		cancel:          cancel,
	}

	return ks, nil
}

func (ks *KubernetesScaler) Start() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if ks.enabled {
		return fmt.Errorf("kubernetes scaler already running")
	}

	ks.enabled = true

	if !ks.config.HPAEnabled {
		go ks.pollingLoop()
	}

	log.Printf("KubernetesScaler[%s]: started for deployment %s/%s", ks.nodeID, ks.namespace, ks.deploymentName)
	return nil
}

func (ks *KubernetesScaler) Stop() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.enabled {
		return nil
	}

	ks.cancel()
	ks.enabled = false

	log.Printf("KubernetesScaler[%s]: stopped", ks.nodeID)
	return nil
}

func (ks *KubernetesScaler) pollingLoop() {
	ticker := time.NewTicker(ks.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ks.ctx.Done():
			return
		case <-ticker.C:
			ks.syncMetrics()
		}
	}
}

func (ks *KubernetesScaler) syncMetrics() error {
	deployment, err := ks.getDeployment()
	if err != nil {
		log.Printf("KubernetesScaler: failed to get deployment: %v", err)
		ks.recordError(err)
		return err
	}

	metrics, err := ks.getPodMetrics()
	if err != nil {
		log.Printf("KubernetesScaler: failed to get pod metrics: %v", err)
		ks.recordError(err)
	}

	ks.mu.Lock()
	ks.metrics.CurrentReplicas = deployment.Status.Replicas
	ks.metrics.ReadyReplicas = deployment.Status.ReadyReplicas
	ks.metrics.AvailableReplicas = deployment.Status.AvailableReplicas
	ks.metrics.LastSyncTime = time.Now()
	ks.mu.Unlock()

	if metrics != nil {
		avgCPU, avgMem := ks.calculateAverageUtilization(metrics)
		ks.mu.Lock()
		ks.metrics.CPUUtilization = avgCPU
		ks.metrics.MemoryUtilization = avgMem
		ks.mu.Unlock()
	}

	return nil
}

func (ks *KubernetesScaler) getDeployment() (*K8sDeployment, error) {
	url := fmt.Sprintf("%s/apis/apps/v1/namespaces/%s/deployments/%s",
		ks.baseURL, ks.namespace, ks.deploymentName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ks.setAuthHeader(req)

	resp, err := ks.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deployment K8sDeployment
	if err := json.Unmarshal(body, &deployment); err != nil {
		return nil, err
	}

	return &deployment, nil
}

func (ks *KubernetesScaler) getPodMetrics() (*K8sPodMetrics, error) {
	url := fmt.Sprintf("%s/apis/metrics.k8s.io/v1beta1/namespaces/%s/pods",
		ks.baseURL, ks.namespace)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ks.setAuthHeader(req)

	resp, err := ks.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var metrics K8sPodMetrics
	if err := json.Unmarshal(body, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func (ks *KubernetesScaler) calculateAverageUtilization(metrics *K8sPodMetrics) (cpuAvg, memAvg float64) {
	if len(metrics.Items) == 0 {
		return 0, 0
	}

	var totalCPU, totalMem float64
	for _, item := range metrics.Items {
		for _, container := range item.Containers {
			cpu, _ := parseResourceValue(container.Usage.CPU)
			mem, _ := parseResourceValue(container.Usage.Memory)
			totalCPU += cpu
			totalMem += mem
		}
	}

	return totalCPU / float64(len(metrics.Items)), totalMem / float64(len(metrics.Items))
}

func parseResourceValue(s string) (float64, error) {
	var value float64
	var unit string
	fmt.Sscanf(s, "%f%s", &value, &unit)

	switch unit {
	case "n":
		return value / 1e9, nil
	case "u":
		return value / 1e6, nil
	case "m":
		return value / 1e3, nil
	case "Ki":
		return value * 1024, nil
	case "Mi":
		return value * 1024 * 1024, nil
	case "Gi":
		return value * 1024 * 1024 * 1024, nil
	case "Ti":
		return value * 1024 * 1024 * 1024 * 1024, nil
	default:
		return value, nil
	}
}

func (ks *KubernetesScaler) setAuthHeader(req *http.Request) {
	if ks.token != "" {
		req.Header.Set("Authorization", "Bearer "+ks.token)
	}
	req.Header.Set("Accept", "application/json")
}

func (ks *KubernetesScaler) ScaleReplicas(replicas int32) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if replicas < ks.config.MinReplicas {
		replicas = ks.config.MinReplicas
	}
	if replicas > ks.config.MaxReplicas {
		replicas = ks.config.MaxReplicas
	}

	url := fmt.Sprintf("%s/apis/apps/v1/namespaces/%s/deployments/%s/scale",
		ks.baseURL, ks.namespace, ks.deploymentName)

	scaleReq := K8sScaleRequest{
		APIVersion: "autoscaling/v1",
		Kind:       "Scale",
		Spec: struct {
			Replicas int32 `json:"replicas"`
		}{Replicas: replicas},
	}

	body, err := json.Marshal(scaleReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	ks.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ks.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scale request failed with status: %d", resp.StatusCode)
	}

	ks.currentReplicas = replicas
	log.Printf("KubernetesScaler[%s]: scaled to %d replicas", ks.nodeID, replicas)

	return nil
}

func (ks *KubernetesScaler) GetCurrentReplicas() int32 {
	return atomic.LoadInt32(&ks.currentReplicas)
}

func (ks *KubernetesScaler) GetTargetReplicas() int32 {
	return atomic.LoadInt32(&ks.targetReplicas)
}

func (ks *KubernetesScaler) GetMetrics() *K8sScalerMetrics {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	metricsCopy := *ks.metrics
	metricsCopy.Errors = make([]string, len(ks.metrics.Errors))
	copy(metricsCopy.Errors, ks.metrics.Errors)

	return &metricsCopy
}

func (ks *KubernetesScaler) recordError(err error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.metrics.Errors = append(ks.metrics.Errors, err.Error())
	if len(ks.metrics.Errors) > 10 {
		ks.metrics.Errors = ks.metrics.Errors[1:]
	}
}

func (ks *KubernetesScaler) CreateHPA(minReplicas, maxReplicas int32, targetCPUUtilization float64) error {
	url := fmt.Sprintf("%s/apis/autoscaling/v2/namespaces/%s/horizontalpodautoscalers",
		ks.baseURL, ks.namespace)

	cpuTarget := targetCPUUtilization
	hpa := K8sHPA{
		APIVersion: "autoscaling/v2",
		Kind:       "HorizontalPodAutoscaler",
	}
	hpa.Metadata.Name = ks.config.HPAName
	hpa.Metadata.Namespace = ks.namespace
	hpa.Spec.ScaleTargetRef.Kind = "Deployment"
	hpa.Spec.ScaleTargetRef.Name = ks.deploymentName
	hpa.Spec.MinReplicas = &minReplicas
	hpa.Spec.MaxReplicas = maxReplicas
	hpa.Spec.Metrics = []struct {
		Type     string `json:"type"`
		Resource *struct {
			Name   string `json:"name"`
			Target struct {
				Type               string   `json:"type"`
				AverageUtilization *float64 `json:"averageUtilization,omitempty"`
			} `json:"target"`
		} `json:"resource,omitempty"`
	}{
		{
			Type: "Resource",
			Resource: &struct {
				Name   string `json:"name"`
				Target struct {
					Type               string   `json:"type"`
					AverageUtilization *float64 `json:"averageUtilization,omitempty"`
				} `json:"target"`
			}{
				Name: "cpu",
				Target: struct {
					Type               string   `json:"type"`
					AverageUtilization *float64 `json:"averageUtilization,omitempty"`
				}{
					Type:               "Utilization",
					AverageUtilization: &cpuTarget,
				},
			},
		},
	}

	body, err := json.Marshal(hpa)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	ks.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ks.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HPA creation failed with status: %d", resp.StatusCode)
	}

	log.Printf("KubernetesScaler[%s]: HPA created with min=%d, max=%d, targetCPU=%.1f%%",
		ks.nodeID, minReplicas, maxReplicas, targetCPUUtilization)

	return nil
}

func (ks *KubernetesScaler) DeleteHPA() error {
	url := fmt.Sprintf("%s/apis/autoscaling/v2/namespaces/%s/horizontalpodautoscalers/%s",
		ks.baseURL, ks.namespace, ks.config.HPAName)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	ks.setAuthHeader(req)

	resp, err := ks.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("HPA deletion failed with status: %d", resp.StatusCode)
	}

	log.Printf("KubernetesScaler[%s]: HPA deleted", ks.nodeID)
	return nil
}

func (ks *KubernetesScaler) GetHPA() (*K8sHPA, error) {
	url := fmt.Sprintf("%s/apis/autoscaling/v2/namespaces/%s/horizontalpodautoscalers/%s",
		ks.baseURL, ks.namespace, ks.config.HPAName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ks.setAuthHeader(req)

	resp, err := ks.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var hpa K8sHPA
	if err := json.Unmarshal(body, &hpa); err != nil {
		return nil, err
	}

	return &hpa, nil
}

func (ks *KubernetesScaler) IsEnabled() bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.enabled
}

func (ks *KubernetesScaler) GetConfig() *KubernetesScalerConfig {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.config
}

func (ks *KubernetesScaler) SetConfig(cfg *KubernetesScalerConfig) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.config = cfg
}

func (ks *KubernetesScaler) GetStats() map[string]interface{} {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return map[string]interface{}{
		"node_id":          ks.nodeID,
		"enabled":          ks.enabled,
		"deployment":       ks.deploymentName,
		"namespace":        ks.namespace,
		"current_replicas": ks.currentReplicas,
		"target_replicas":  ks.targetReplicas,
		"min_replicas":     ks.config.MinReplicas,
		"max_replicas":     ks.config.MaxReplicas,
		"hpa_enabled":      ks.config.HPAEnabled,
		"current_cpu_util": ks.metrics.CPUUtilization,
		"current_mem_util": ks.metrics.MemoryUtilization,
		"last_sync_time":   ks.metrics.LastSyncTime,
	}
}
