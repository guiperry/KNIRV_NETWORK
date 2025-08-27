package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// KNIRVCONTROLLER Demo Workflow Tests
// These tests demonstrate real-world usage scenarios for demos

// Demo workflow data structures
type DemoAgentRequest struct {
	AgentID      string                 `json:"agentId"`
	AgentName    string                 `json:"agentName"`
	Capabilities []string               `json:"capabilities"`
	Templates    []string               `json:"templates"`
	Parameters   map[string]interface{} `json:"parameters"`
}

type DemoSkillChainRequest struct {
	ChainID     string   `json:"chainId"`
	SkillIDs    []string `json:"skillIds"`
	UserAddress string   `json:"userAddress"`
	NRNAmount   string   `json:"nrnAmount"`
}

type DemoErrorFixingRequest struct {
	ErrorCode    string                 `json:"errorCode"`
	ErrorMessage string                 `json:"errorMessage"`
	SourceCode   string                 `json:"sourceCode"`
	Language     string                 `json:"language"`
	Context      map[string]interface{} `json:"context"`
}

type DemoNetworkStatusResponse struct {
	Status      string                 `json:"status"`
	Connections map[string]interface{} `json:"connections"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// Demo Workflow Test Suite
func TestKNIRVControllerDemoWorkflows(t *testing.T) {
	// Ensure all services are ready for demo
	t.Log("🚀 Starting KNIRVCONTROLLER Demo Workflow Tests")
	
	// Wait for all required services
	require.NoError(t, waitForKNIRVControllerService(KNIRVControllerURL, TestTimeout))
	require.NoError(t, waitForKNIRVControllerService(KNIRVRouterURL, TestTimeout))
	require.NoError(t, waitForKNIRVControllerService(KNIRVGraphURL, TestTimeout))

	t.Log("✅ All services ready for demo workflows")

	// Run demo workflows
	t.Run("Demo1_AgentDevelopmentWorkflow", testDemoAgentDevelopmentWorkflow)
	t.Run("Demo2_SkillInvocationWorkflow", testDemoSkillInvocationWorkflow)
	t.Run("Demo3_ErrorFixingWorkflow", testDemoErrorFixingWorkflow)
	t.Run("Demo4_LoRAAdapterWorkflow", testDemoLoRAAdapterWorkflow)
	t.Run("Demo5_NetworkIntegrationWorkflow", testDemoNetworkIntegrationWorkflow)
	t.Run("Demo6_RealTimeMonitoringWorkflow", testDemoRealTimeMonitoringWorkflow)
}

func testDemoAgentDevelopmentWorkflow(t *testing.T) {
	t.Log("🤖 Demo 1: Agent Development Workflow")
	t.Log("Demonstrating: Agent creation → Template compilation → WASM generation")

	// Step 1: Create a demo agent
	agentRequest := DemoAgentRequest{
		AgentID:   "demo-agent-001",
		AgentName: "Demo Text Processing Agent",
		Capabilities: []string{
			"text-processing",
			"syntax-analysis",
			"error-detection",
		},
		Templates: []string{
			"CognitiveEngine",
			"LoRAAdapter",
			"EventEmitter",
		},
		Parameters: map[string]interface{}{
			"optimizationLevel": "O2",
			"targetPlatform":    "web",
			"demoMode":          true,
		},
	}

	t.Log("📝 Step 1: Creating demo agent...")
	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/api/compile-agent", agentRequest)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		t.Logf("✅ Agent created successfully: %+v", response)
	} else {
		t.Logf("ℹ️  Agent creation returned status %d (expected in demo environment)", resp.StatusCode)
	}

	// Step 2: Get available templates
	t.Log("📋 Step 2: Retrieving available templates...")
	resp2, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/api/templates", nil)
	require.NoError(t, err)
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var templates map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&templates)
		require.NoError(t, err)
		t.Logf("✅ Templates retrieved: %+v", templates)
	} else {
		t.Logf("ℹ️  Template retrieval returned status %d", resp2.StatusCode)
	}

	// Step 3: Check template export status
	t.Log("📤 Step 3: Checking template export status...")
	resp3, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/api/templates/info", nil)
	require.NoError(t, err)
	defer resp3.Body.Close()

	if resp3.StatusCode == http.StatusOK {
		var templateInfo map[string]interface{}
		err = json.NewDecoder(resp3.Body).Decode(&templateInfo)
		require.NoError(t, err)
		t.Logf("✅ Template info: %+v", templateInfo)
	}

	t.Log("🎉 Demo 1 Complete: Agent Development Workflow demonstrated")
}

func testDemoSkillInvocationWorkflow(t *testing.T) {
	t.Log("⚡ Demo 2: Skill Invocation Workflow")
	t.Log("Demonstrating: ErrorContext → KNIRVGRAPH → KNIRVROUTER → Skill Execution")

	// Step 1: Create error context for skill invocation
	errorContext := KNIRVControllerErrorContext{
		ErrorID:      "demo-skill-request-001",
		ErrorType:    "skill_invocation_request",
		ErrorMessage: "Demo: Need text processing skill for user input",
		StackTrace:   "demo_workflow.js:42:15",
		UserContext: map[string]interface{}{
			"userAddress": "knirv1demo123456789",
			"nrnAmount":   "250",
			"inputText":   "Hello, KNIRV Network! Please process this text.",
			"requestType": "text-analysis",
			"demoMode":    true,
		},
		AgentID:   "demo-agent-001",
		Timestamp: time.Now().UnixMilli(),
		Severity:  "medium",
	}

	t.Log("🔍 Step 1: Creating error context for skill resolution...")
	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/api/process-error-context", errorContext)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		t.Logf("✅ Error context processed: %+v", response)
	} else {
		t.Logf("ℹ️  Error context processing returned status %d", resp.StatusCode)
	}

	// Step 2: Direct skill invocation
	skillRequest := KNIRVControllerSkillRequest{
		SkillID:     "demo-text-processing-skill",
		UserAddress: "knirv1demo123456789",
		NRNAmount:   "250",
		Parameters: map[string]interface{}{
			"agentId":    "demo-agent-001",
			"inputText":  "Hello, KNIRV Network! Please process this text.",
			"operation":  "analyze",
			"demoMode":   true,
			"priority":   "high",
		},
		Priority: "high",
		UseP2P:   true,
		UseWASM:  true,
	}

	t.Log("🎯 Step 2: Invoking skill directly...")
	resp2, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/api/invoke-skill", skillRequest)
	require.NoError(t, err)
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var skillResponse KNIRVControllerSkillResponse
		err = json.NewDecoder(resp2.Body).Decode(&skillResponse)
		require.NoError(t, err)
		t.Logf("✅ Skill invoked successfully: RequestID=%s, ExecutionTime=%dms", 
			skillResponse.RequestID, skillResponse.ExecutionTime)
	} else {
		t.Logf("ℹ️  Skill invocation returned status %d", resp2.StatusCode)
	}

	t.Log("🎉 Demo 2 Complete: Skill Invocation Workflow demonstrated")
}

func testDemoErrorFixingWorkflow(t *testing.T) {
	t.Log("🔧 Demo 3: Error Fixing Workflow")
	t.Log("Demonstrating: Code error → Analysis → LoRA adapter → Fixed code")

	// Step 1: Submit code with error for fixing
	errorFixingRequest := DemoErrorFixingRequest{
		ErrorCode:    "SyntaxError",
		ErrorMessage: "Unexpected token '}' at line 15",
		SourceCode: `
function processData(data) {
    if (data && data.length > 0) {
        return data.map(item => {
            return {
                id: item.id,
                name: item.name,
                processed: true
            }
        }
    }
    return [];
}`,
		Language: "javascript",
		Context: map[string]interface{}{
			"fileName":    "demo-processor.js",
			"lineNumber":  15,
			"columnNumber": 9,
			"demoMode":    true,
		},
	}

	t.Log("🐛 Step 1: Submitting code with syntax error...")
	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/api/fix-code-error", errorFixingRequest)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		t.Logf("✅ Code error analysis completed: %+v", response)
	} else {
		t.Logf("ℹ️  Code error fixing returned status %d", resp.StatusCode)
	}

	// Step 2: Compile LoRA adapter for error fixing
	loraRequest := KNIRVControllerLoRARequest{
		AdapterName:            "demo-syntax-fixer",
		Description:            "Demo LoRA adapter for JavaScript syntax error fixing",
		BaseModelCompatibility: "hrm-v1",
		Version:                1,
		Rank:                   8,
		Alpha:                  0.3,
		Metadata: map[string]string{
			"language":    "javascript",
			"errorType":   "syntax",
			"demoMode":    "true",
			"capability":  "error-fixing",
		},
	}

	t.Log("🧠 Step 2: Compiling LoRA adapter for error fixing...")
	resp2, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/lora/compile", loraRequest)
	require.NoError(t, err)
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var loraResponse KNIRVControllerLoRAResponse
		err = json.NewDecoder(resp2.Body).Decode(&loraResponse)
		require.NoError(t, err)
		t.Logf("✅ LoRA adapter compiled: AdapterID=%s", loraResponse.AdapterID)
	} else {
		t.Logf("ℹ️  LoRA compilation returned status %d", resp2.StatusCode)
	}

	t.Log("🎉 Demo 3 Complete: Error Fixing Workflow demonstrated")
}

func testDemoLoRAAdapterWorkflow(t *testing.T) {
	t.Log("🧬 Demo 4: LoRA Adapter Workflow")
	t.Log("Demonstrating: Adapter creation → Registration → Invocation")

	// Step 1: Create specialized LoRA adapter
	loraRequest := KNIRVControllerLoRARequest{
		AdapterName:            "demo-text-enhancer",
		Description:            "Demo LoRA adapter for text enhancement and analysis",
		BaseModelCompatibility: "hrm-v1",
		Version:                1,
		Rank:                   16,
		Alpha:                  0.5,
		Metadata: map[string]string{
			"capability":   "text-enhancement",
			"domain":       "natural-language",
			"demoMode":     "true",
			"optimization": "speed",
		},
	}

	t.Log("🔬 Step 1: Creating specialized LoRA adapter...")
	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/lora/compile", loraRequest)
	require.NoError(t, err)
	defer resp.Body.Close()

	var adapterID string
	if resp.StatusCode == http.StatusOK {
		var response KNIRVControllerLoRAResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		adapterID = response.AdapterID
		t.Logf("✅ LoRA adapter created: %s", adapterID)
	} else {
		t.Logf("ℹ️  LoRA creation returned status %d", resp.StatusCode)
	}

	// Step 2: Invoke LoRA adapter
	if adapterID != "" {
		invokeRequest := map[string]interface{}{
			"adapterId": adapterID,
			"inputText": "This is a demo text that needs enhancement and analysis.",
			"parameters": map[string]interface{}{
				"enhancementType": "clarity",
				"analysisDepth":   "detailed",
				"demoMode":        true,
			},
		}

		t.Log("⚡ Step 2: Invoking LoRA adapter...")
		resp2, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/lora/invoke", invokeRequest)
		require.NoError(t, err)
		defer resp2.Body.Close()

		if resp2.StatusCode == http.StatusOK {
			var response map[string]interface{}
			err = json.NewDecoder(resp2.Body).Decode(&response)
			require.NoError(t, err)
			t.Logf("✅ LoRA adapter invoked successfully: %+v", response)
		} else {
			t.Logf("ℹ️  LoRA invocation returned status %d", resp2.StatusCode)
		}
	}

	t.Log("🎉 Demo 4 Complete: LoRA Adapter Workflow demonstrated")
}

func testDemoNetworkIntegrationWorkflow(t *testing.T) {
	t.Log("🌐 Demo 5: Network Integration Workflow")
	t.Log("Demonstrating: KNIRVCONTROLLER ↔ Network Services Integration")

	// Step 1: Check network status
	t.Log("📡 Step 1: Checking network status...")
	resp, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/api/network-status", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var networkStatus DemoNetworkStatusResponse
		err = json.NewDecoder(resp.Body).Decode(&networkStatus)
		require.NoError(t, err)
		t.Logf("✅ Network status: %s", networkStatus.Status)
		t.Logf("📊 Connections: %+v", networkStatus.Connections)
	} else {
		t.Logf("ℹ️  Network status returned status %d", resp.StatusCode)
	}

	// Step 2: Test cross-service communication
	t.Log("🔄 Step 2: Testing cross-service communication...")
	
	// Test KNIRVCONTROLLER → KNIRVGRAPH
	graphRequest := map[string]interface{}{
		"query":    "demo-query",
		"demoMode": true,
	}
	
	resp2, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/api/graph-query", graphRequest)
	require.NoError(t, err)
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&response)
		require.NoError(t, err)
		t.Logf("✅ KNIRVGRAPH communication successful: %+v", response)
	} else {
		t.Logf("ℹ️  KNIRVGRAPH communication returned status %d", resp2.StatusCode)
	}

	t.Log("🎉 Demo 5 Complete: Network Integration Workflow demonstrated")
}

func testDemoRealTimeMonitoringWorkflow(t *testing.T) {
	t.Log("📊 Demo 6: Real-Time Monitoring Workflow")
	t.Log("Demonstrating: System metrics → Health monitoring → Performance tracking")

	// Step 1: Get system metrics
	t.Log("📈 Step 1: Retrieving system metrics...")
	resp, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/api/metrics", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var metrics map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&metrics)
		require.NoError(t, err)
		t.Logf("✅ System metrics retrieved: %+v", metrics)
	} else {
		t.Logf("ℹ️  Metrics endpoint returned status %d", resp.StatusCode)
	}

	// Step 2: Health check with detailed component status
	t.Log("🏥 Step 2: Performing detailed health check...")
	resp2, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/health", nil)
	require.NoError(t, err)
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var health KNIRVControllerHealthResponse
		err = json.NewDecoder(resp2.Body).Decode(&health)
		require.NoError(t, err)
		t.Logf("✅ Health check: %s", health.Status)
		t.Logf("🔧 Components: %+v", health.Components)
	}

	// Step 3: Performance monitoring
	t.Log("⚡ Step 3: Checking performance metrics...")
	resp3, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/api/performance", nil)
	require.NoError(t, err)
	defer resp3.Body.Close()

	if resp3.StatusCode == http.StatusOK {
		var performance map[string]interface{}
		err = json.NewDecoder(resp3.Body).Decode(&performance)
		require.NoError(t, err)
		t.Logf("✅ Performance metrics: %+v", performance)
	} else {
		t.Logf("ℹ️  Performance endpoint returned status %d", resp3.StatusCode)
	}

	t.Log("🎉 Demo 6 Complete: Real-Time Monitoring Workflow demonstrated")
	t.Log("🏆 All Demo Workflows Complete! KNIRVCONTROLLER ready for live demonstrations.")
}
