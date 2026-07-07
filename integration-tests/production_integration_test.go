package integration_tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

// ProductionTestConfig represents the production deployment test configuration
type ProductionTestConfig struct {
	ProductionDeployment struct {
		Enabled         bool `yaml:"enabled"`
		DeploymentModes struct {
			Local struct {
				Enabled       bool              `yaml:"enabled"`
				GatewayURL    string            `yaml:"gateway_url"`
				MonitoringURL string            `yaml:"monitoring_url"`
				Services      map[string]string `yaml:"services"`
			} `yaml:"local"`
			Kubernetes struct {
				Enabled       bool   `yaml:"enabled"`
				Namespace     string `yaml:"namespace"`
				GatewayURL    string `yaml:"gateway_url"`
				MonitoringURL string `yaml:"monitoring_url"`
				IngressHost   string `yaml:"ingress_host"`
			} `yaml:"kubernetes"`
			DockerCompose struct {
				Enabled     bool   `yaml:"enabled"`
				ComposeFile string `yaml:"compose_file"`
				Network     string `yaml:"network"`
				GatewayURL  string `yaml:"gateway_url"`
			} `yaml:"docker_compose"`
		} `yaml:"deployment_modes"`
		DeploymentScripts struct {
			DeployScript        string `yaml:"deploy_script"`
			ManageScript        string `yaml:"manage_script"`
			ProductionTestSuite string `yaml:"production_test_suite"`
		} `yaml:"deployment_scripts"`
		ProductionTests struct {
			Enabled           bool   `yaml:"enabled"`
			Timeout           string `yaml:"timeout"`
			LoadTestDuration  string `yaml:"load_test_duration"`
			ConcurrentUsers   int    `yaml:"concurrent_users"`
			ConnectivityTests bool   `yaml:"connectivity_tests"`
			BridgeTests       bool   `yaml:"bridge_tests"`
			MonitoringTests   bool   `yaml:"monitoring_tests"`
			RealNetworkTests  bool   `yaml:"real_network_tests"`
		} `yaml:"production_tests"`
		Monitoring struct {
			PrometheusURL   string `yaml:"prometheus_url"`
			GrafanaURL      string `yaml:"grafana_url"`
			AlertmanagerURL string `yaml:"alertmanager_url"`
			CollectMetrics  bool   `yaml:"collect_metrics"`
			CreateSnapshots bool   `yaml:"create_snapshots"`
		} `yaml:"monitoring"`
		Orchestration struct {
			PreTestDeployment   bool   `yaml:"pre_test_deployment"`
			PostTestCleanup     bool   `yaml:"post_test_cleanup"`
			WaitForServices     bool   `yaml:"wait_for_services"`
			ServiceReadyTimeout string `yaml:"service_ready_timeout"`
			HealthCheckInterval string `yaml:"health_check_interval"`
		} `yaml:"orchestration"`
	} `yaml:"production_deployment"`
}

// ProductionTestSuite manages production deployment integration tests
type ProductionTestSuite struct {
	config     *ProductionTestConfig
	deployMode string
	baseURL    string
	client     *http.Client
}

// NewProductionTestSuite creates a new production test suite
func NewProductionTestSuite(configPath string) (*ProductionTestSuite, error) {
	config, err := loadProductionConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Determine deployment mode from environment or config
	deployMode := os.Getenv("KNIRV_DEPLOYMENT_MODE")
	if deployMode == "" {
		deployMode = "local" // default
	}

	var baseURL string
	switch deployMode {
	case "kubernetes":
		baseURL = config.ProductionDeployment.DeploymentModes.Kubernetes.GatewayURL
	case "docker_compose":
		baseURL = config.ProductionDeployment.DeploymentModes.DockerCompose.GatewayURL
	default:
		baseURL = config.ProductionDeployment.DeploymentModes.Local.GatewayURL
	}

	return &ProductionTestSuite{
		config:     config,
		deployMode: deployMode,
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// loadProductionConfig loads the production test configuration
func loadProductionConfig(configPath string) (*ProductionTestConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ProductionTestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// TestProductionDeploymentIntegration tests the production deployment integration
func TestProductionDeploymentIntegration(t *testing.T) {
	suite, err := NewProductionTestSuite("config/test-config.yaml")
	if err != nil {
		t.Fatalf("Failed to create production test suite: %v", err)
	}

	if !suite.config.ProductionDeployment.Enabled {
		t.Skip("Production deployment integration tests are disabled")
	}

	t.Run("DeploymentModeValidation", suite.testDeploymentModeValidation)
	t.Run("ServiceHealthChecks", suite.testServiceHealthChecks)
	t.Run("ProductionTestSuiteExecution", suite.testProductionTestSuiteExecution)
	t.Run("MonitoringIntegration", suite.testMonitoringIntegration)
	t.Run("ConnectivityProofEngine", suite.testConnectivityProofEngine)
	t.Run("BridgeIntegration", suite.testBridgeIntegration)
}

// testDeploymentModeValidation validates the current deployment mode
func (pts *ProductionTestSuite) testDeploymentModeValidation(t *testing.T) {
	t.Logf("Testing deployment mode: %s", pts.deployMode)

	switch pts.deployMode {
	case "local":
		if !pts.config.ProductionDeployment.DeploymentModes.Local.Enabled {
			t.Skip("Local deployment mode is disabled")
		}
	case "kubernetes":
		if !pts.config.ProductionDeployment.DeploymentModes.Kubernetes.Enabled {
			t.Skip("Kubernetes deployment mode is disabled")
		}
		pts.testKubernetesDeployment(t)
	case "docker_compose":
		if !pts.config.ProductionDeployment.DeploymentModes.DockerCompose.Enabled {
			t.Skip("Docker Compose deployment mode is disabled")
		}
		pts.testDockerComposeDeployment(t)
	default:
		t.Fatalf("Unknown deployment mode: %s", pts.deployMode)
	}
}

// testKubernetesDeployment validates Kubernetes deployment
func (pts *ProductionTestSuite) testKubernetesDeployment(t *testing.T) {
	namespace := pts.config.ProductionDeployment.DeploymentModes.Kubernetes.Namespace

	// Check if namespace exists
	cmd := exec.Command("kubectl", "get", "namespace", namespace)
	if err := cmd.Run(); err != nil {
		t.Errorf("Kubernetes namespace %s not found: %v", namespace, err)
		return
	}

	// Check if pods are running
	cmd = exec.Command("kubectl", "get", "pods", "-n", namespace, "--field-selector=status.phase=Running")
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("Failed to get running pods: %v", err)
		return
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		t.Error("No running pods found in namespace")
	}

	t.Logf("Kubernetes deployment validation passed")
}

// testDockerComposeDeployment validates Docker Compose deployment
func (pts *ProductionTestSuite) testDockerComposeDeployment(t *testing.T) {
	composeFile := pts.config.ProductionDeployment.DeploymentModes.DockerCompose.ComposeFile

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Errorf("Docker Compose file not found: %s", composeFile)
		return
	}

	// Check if services are running
	cmd := exec.Command("docker-compose", "-f", composeFile, "ps", "--services", "--filter", "status=running")
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("Failed to check Docker Compose services: %v", err)
		return
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		t.Error("No running Docker Compose services found")
	}

	t.Logf("Docker Compose deployment validation passed")
}

// testServiceHealthChecks validates all service health endpoints
func (pts *ProductionTestSuite) testServiceHealthChecks(t *testing.T) {
	services := pts.config.ProductionDeployment.DeploymentModes.Local.Services

	for serviceName, serviceURL := range services {
		t.Run(serviceName, func(t *testing.T) {
			healthURL := serviceURL + "/health"

			resp, err := pts.client.Get(healthURL)
			if err != nil {
				t.Errorf("Health check failed for %s: %v", serviceName, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Health check returned non-200 status for %s: %d", serviceName, resp.StatusCode)
			}

			t.Logf("Health check passed for %s", serviceName)
		})
	}
}

// testProductionTestSuiteExecution runs the production test suite
func (pts *ProductionTestSuite) testProductionTestSuiteExecution(t *testing.T) {
	if !pts.config.ProductionDeployment.ProductionTests.Enabled {
		t.Skip("Production tests are disabled")
	}

	testSuitePath := pts.config.ProductionDeployment.DeploymentScripts.ProductionTestSuite

	// Check if test suite exists
	if _, err := os.Stat(testSuitePath); os.IsNotExist(err) {
		t.Errorf("Production test suite not found: %s", testSuitePath)
		return
	}

	// Set environment variables for the test suite
	env := os.Environ()
	env = append(env, fmt.Sprintf("GATEWAY_URL=%s", pts.baseURL))

	// Execute the production test suite
	cmd := exec.Command("bash", testSuitePath)
	cmd.Env = env
	cmd.Dir = filepath.Dir(testSuitePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Production test suite failed: %v\nOutput: %s", err, string(output))
		return
	}

	t.Logf("Production test suite executed successfully")
}

// testMonitoringIntegration validates monitoring stack integration
func (pts *ProductionTestSuite) testMonitoringIntegration(t *testing.T) {
	if !pts.config.ProductionDeployment.ProductionTests.MonitoringTests {
		t.Skip("Monitoring tests are disabled")
	}

	monitoring := pts.config.ProductionDeployment.Monitoring

	// Test Prometheus
	if monitoring.PrometheusURL != "" {
		t.Run("Prometheus", func(t *testing.T) {
			resp, err := pts.client.Get(monitoring.PrometheusURL + "/api/v1/query?query=up")
			if err != nil {
				t.Errorf("Prometheus health check failed: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Prometheus returned non-200 status: %d", resp.StatusCode)
			}
		})
	}

	// Test Grafana
	if monitoring.GrafanaURL != "" {
		t.Run("Grafana", func(t *testing.T) {
			resp, err := pts.client.Get(monitoring.GrafanaURL + "/api/health")
			if err != nil {
				t.Errorf("Grafana health check failed: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Grafana returned non-200 status: %d", resp.StatusCode)
			}
		})
	}

	// Test Alertmanager
	if monitoring.AlertmanagerURL != "" {
		t.Run("Alertmanager", func(t *testing.T) {
			resp, err := pts.client.Get(monitoring.AlertmanagerURL + "/-/healthy")
			if err != nil {
				t.Errorf("Alertmanager health check failed: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Alertmanager returned non-200 status: %d", resp.StatusCode)
			}
		})
	}
}

// testConnectivityProofEngine validates KNIRV-ROUTER connectivity proof engine
func (pts *ProductionTestSuite) testConnectivityProofEngine(t *testing.T) {
	if !pts.config.ProductionDeployment.ProductionTests.ConnectivityTests {
		t.Skip("Connectivity tests are disabled")
	}

	routerAPIURL := pts.config.ProductionDeployment.DeploymentModes.Local.Services["knirvrouter_api"]
	if routerAPIURL == "" {
		t.Skip("KNIRV-ROUTER API URL not configured")
	}

	// Test connectivity status endpoint
	t.Run("ConnectivityStatus", func(t *testing.T) {
		resp, err := pts.client.Get(routerAPIURL + "/api/connectivity/status")
		if err != nil {
			t.Errorf("Connectivity status check failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Connectivity status returned non-200 status: %d", resp.StatusCode)
			return
		}

		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Errorf("Failed to decode connectivity status: %v", err)
			return
		}

		if proofEngineActive, ok := status["proof_engine_active"].(bool); !ok || !proofEngineActive {
			t.Error("Proof engine is not active")
		}
	})

	// Test proof generation
	t.Run("ProofGeneration", func(t *testing.T) {
		resp, err := pts.client.Post(routerAPIURL+"/api/connectivity/proofs", "application/json", nil)
		if err != nil {
			t.Errorf("Proof generation request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Proof generation returned non-200 status: %d", resp.StatusCode)
		}
	})
}

// testBridgeIntegration validates KNIRV-ORACLE bridge integration
func (pts *ProductionTestSuite) testBridgeIntegration(t *testing.T) {
	if !pts.config.ProductionDeployment.ProductionTests.BridgeTests {
		t.Skip("Bridge tests are disabled")
	}

	rootURL := pts.config.ProductionDeployment.DeploymentModes.Local.Services["knirvoracle"]
	if rootURL == "" {
		t.Skip("KNIRV-ORACLE URL not configured")
	}

	// Test bridge health
	t.Run("BridgeHealth", func(t *testing.T) {
		resp, err := pts.client.Get(rootURL + "/bridge/health")
		if err != nil {
			t.Errorf("Bridge health check failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Bridge health returned non-200 status: %d", resp.StatusCode)
		}
	})

	// Test bridge metrics
	t.Run("BridgeMetrics", func(t *testing.T) {
		resp, err := pts.client.Get(rootURL + "/bridge/metrics")
		if err != nil {
			t.Errorf("Bridge metrics check failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Bridge metrics returned non-200 status: %d", resp.StatusCode)
		}
	})
}
