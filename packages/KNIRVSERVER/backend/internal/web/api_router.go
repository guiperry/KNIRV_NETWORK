package web

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// APIRouter provides unified API routing with versioning
type APIRouter struct {
	dveHandlers             *DVEHandlers
	pluginHandlers          *PluginManagementHandlers
	agentHandlers           *AgentHandlers
	paymentHandlers         *PaymentHandlers
	knirvshellHandlers      *KNIRVSHELLHandlers
	onboardingHandlers      *OnboardingHandlers
	cognitiveEngineHandlers *CognitiveEngineHandlers
	knowledgeBaseHandlers   *KnowledgeBaseHandlers
	governanceHandlers      *GovernanceHandlers
	authMiddleware          *middleware.AuthMiddleware
	browserDVEHub           *BrowserDVEHub
	dvePodHandler           *DVEPodHandler
}

// NewAPIRouter creates a new unified API router
func NewAPIRouter(
	dveHandlers *DVEHandlers,
	pluginHandlers *PluginManagementHandlers,
	agentHandlers *AgentHandlers,
	paymentHandlers *PaymentHandlers,
	knirvshellHandlers *KNIRVSHELLHandlers,
	onboardingHandlers *OnboardingHandlers,
	cognitiveEngineHandlers *CognitiveEngineHandlers,
	knowledgeBaseHandlers *KnowledgeBaseHandlers,
	governanceHandlers *GovernanceHandlers,
	authMiddleware *middleware.AuthMiddleware,
	browserDVEHub *BrowserDVEHub,
	dvePodHandler *DVEPodHandler,
) *APIRouter {
	return &APIRouter{
		dveHandlers:             dveHandlers,
		pluginHandlers:          pluginHandlers,
		agentHandlers:           agentHandlers,
		paymentHandlers:         paymentHandlers,
		knirvshellHandlers:      knirvshellHandlers,
		onboardingHandlers:      onboardingHandlers,
		cognitiveEngineHandlers: cognitiveEngineHandlers,
		knowledgeBaseHandlers:   knowledgeBaseHandlers,
		governanceHandlers:      governanceHandlers,
		authMiddleware:          authMiddleware,
		browserDVEHub:           browserDVEHub,
		dvePodHandler:           dvePodHandler,
	}
}

// RegisterRoutes registers all unified API routes
func (ar *APIRouter) RegisterRoutes(r *mux.Router) {
	// Create versioned API subrouter
	apiV1 := r.PathPrefix("/api/v1").Subrouter()

	// Health endpoint — returns backend status.  Proxied from the gateway
	// as /api/v1/health.
	apiV1.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}).Methods("GET")

	// Info endpoint — returns server role/mode info for the WebGUI.
	// Proxied from the gateway as /api/v1/info.
	apiV1.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"role":    "General",
			"network": "local",
			"mode":    "standalone",
		})
	}).Methods("GET")

	// Register DVE routes under /api/v1/dve/
	ar.registerDVERoutes(apiV1)

	// Register Plugin routes under /api/v1/plugin/
	ar.registerPluginRoutes(apiV1)

	// Register Agent routes under /api/v1/dve/{id}/agent/
	ar.registerAgentRoutes(apiV1)

	// Register Payment routes under /api/v1/payments/
	ar.registerPaymentRoutes(apiV1)

	// Register KNIRVSHELL routes under /api/v1/shell/
	ar.registerKNIRVSHELLRoutes(apiV1)

	// Register Onboarding routes under /api/v1/onboarding/
	ar.registerOnboardingRoutes(apiV1)

	// Register Cognitive Engine routes under /api/v1/cognitive/
	ar.registerCognitiveRoutes(apiV1)

	// Register Knowledge Base routes under /api/v1/knowledge-base/
	ar.registerKnowledgeBaseRoutes(apiV1)

	// Register Governance routes under /api/v1/governance/
	if ar.governanceHandlers != nil {
		ar.governanceHandlers.RegisterRoutes(apiV1)
	}

	// Register backward compatibility redirects
	ar.registerBackwardCompatibilityRoutes(r)

	// Browser DVE WebSocket endpoint (no version prefix for WS upgrade compatibility)
	if ar.browserDVEHub != nil {
		r.HandleFunc("/api/dve/browser/ws", ar.browserDVEHub.HandleWebSocket)
	}
}

// registerDVERoutes registers DVE-related routes
func (ar *APIRouter) registerDVERoutes(apiV1 *mux.Router) {
	dveRouter := apiV1.PathPrefix("/dve").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		dveRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// DVE node operations
	dveRouter.HandleFunc("/nodes", ar.dveHandlers.GetDVENodes).Methods("GET", "OPTIONS")
	dveRouter.HandleFunc("/nodes", ar.dveHandlers.PostDVENodes).Methods("POST", "OPTIONS")
	dveRouter.HandleFunc("/nodes/{id}", ar.dveHandlers.GetDVENode).Methods("GET", "OPTIONS")
	dveRouter.HandleFunc("/nodes/{id}", ar.dveHandlers.UpdateDVENode).Methods("PUT", "OPTIONS")
	dveRouter.HandleFunc("/nodes/{id}", ar.dveHandlers.DeleteDVENode).Methods("DELETE", "OPTIONS")

	// DVE node endpoints
	dveRouter.HandleFunc("/nodes/{id}/endpoints", ar.dveHandlers.GetDVENodeEndpoints).Methods("GET", "OPTIONS")
	dveRouter.HandleFunc("/nodes/{id}/ssh-endpoint", ar.dveHandlers.GetDVENodeSSHEndpoint).Methods("GET", "OPTIONS")
	dveRouter.HandleFunc("/nodes/{id}/validation-endpoint", ar.dveHandlers.GetDVENodeValidationEndpoint).Methods("GET", "OPTIONS")
	dveRouter.HandleFunc("/nodes/{id}/error-resolution-endpoint", ar.dveHandlers.GetDVENodeErrorResolutionEndpoint).Methods("GET", "OPTIONS")

	// DVE monitoring
	dveRouter.HandleFunc("/workers", ar.dveHandlers.GetDVEWorkers).Methods("GET", "OPTIONS")
	dveRouter.HandleFunc("/{nodeId}/tasks", ar.dveHandlers.GetDVENodeTasksAlias).Methods("GET", "OPTIONS")
	dveRouter.HandleFunc("/{nodeId}/metrics", ar.dveHandlers.GetDVENodeMetricsAlias).Methods("GET", "OPTIONS")

	// SSH session management
	dveRouter.HandleFunc("/{nodeId}/ssh-session", ar.dveHandlers.CreateNodeSSHSession).Methods("POST", "OPTIONS")
	dveRouter.HandleFunc("/{nodeId}/agent/response", ar.dveHandlers.PostAgentResponse).Methods("POST", "OPTIONS")

	// P2P network
	dveRouter.HandleFunc("/peers", ar.dveHandlers.GetP2PPeers).Methods("GET", "OPTIONS")

	// DVE Pod registration (portable DVE)
	if ar.dvePodHandler != nil {
		ar.dvePodHandler.RegisterRoutes(apiV1)
	}
}

// registerPluginRoutes registers Plugin management routes
func (ar *APIRouter) registerPluginRoutes(apiV1 *mux.Router) {
	pluginRouter := apiV1.PathPrefix("/plugin").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		pluginRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// Plugin CRUD operations
	pluginRouter.HandleFunc("/objects", ar.pluginHandlers.GetPlugins).Methods("GET", "OPTIONS")
	pluginRouter.HandleFunc("/objects", ar.pluginHandlers.PostPlugin).Methods("POST", "OPTIONS")
	pluginRouter.HandleFunc("/objects/{id}", ar.pluginHandlers.GetPlugin).Methods("GET", "OPTIONS")
	pluginRouter.HandleFunc("/objects/{id}", ar.pluginHandlers.PutPlugin).Methods("PUT", "OPTIONS")
	pluginRouter.HandleFunc("/objects/{id}", ar.pluginHandlers.DeletePlugin).Methods("DELETE", "OPTIONS")

	// Plugin actions
	pluginRouter.HandleFunc("/objects/{id}/actions", ar.pluginHandlers.PostPluginAction).Methods("POST", "OPTIONS")

	// Plugin monitoring
	pluginRouter.HandleFunc("/objects/{id}/metrics", ar.pluginHandlers.GetPluginMetrics).Methods("GET", "OPTIONS")
	pluginRouter.HandleFunc("/objects/{id}/logs", ar.pluginHandlers.GetPluginLogs).Methods("GET", "OPTIONS")
	pluginRouter.HandleFunc("/objects/{id}/events", ar.pluginHandlers.GetPluginEvents).Methods("GET", "OPTIONS")

	// Templates
	pluginRouter.HandleFunc("/templates", ar.pluginHandlers.GetPluginTemplates).Methods("GET", "OPTIONS")
	pluginRouter.HandleFunc("/templates", ar.pluginHandlers.PostPluginTemplate).Methods("POST", "OPTIONS")

	// Summary
	pluginRouter.HandleFunc("/summary", ar.pluginHandlers.GetPluginSummary).Methods("GET", "OPTIONS")
}

// registerAgentRoutes registers Agent-related routes
func (ar *APIRouter) registerAgentRoutes(apiV1 *mux.Router) {
	agentRouter := apiV1.PathPrefix("/dve/{id}/agent").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		agentRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// Agent operations
	agentRouter.HandleFunc("/status", ar.agentHandlers.GetAgentStatus).Methods("GET", "OPTIONS")
	agentRouter.HandleFunc("/launch", ar.agentHandlers.LaunchAgent).Methods("POST", "OPTIONS")
	agentRouter.HandleFunc("", ar.agentHandlers.StopAgent).Methods("DELETE", "OPTIONS")
	agentRouter.HandleFunc("/tasks", ar.agentHandlers.GetAgentTasks).Methods("GET", "OPTIONS")
	agentRouter.HandleFunc("/tasks", ar.agentHandlers.SubmitAgentTask).Methods("POST", "OPTIONS")
	agentRouter.HandleFunc("/tasks/{taskID}", ar.agentHandlers.GetAgentTask).Methods("GET", "OPTIONS")
}

// registerPaymentRoutes registers Payment-related routes
func (ar *APIRouter) registerPaymentRoutes(apiV1 *mux.Router) {
	paymentRouter := apiV1.PathPrefix("/payments").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		paymentRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// Stripe payment operations
	paymentRouter.HandleFunc("/stripe/create-session", ar.paymentHandlers.CreateStripeCheckoutSession).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/charge-status", ar.paymentHandlers.GetStripeChargeStatus).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/refund", ar.paymentHandlers.RefundStripeCharge).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/webhook", ar.paymentHandlers.StripeWebhook).Methods("POST", "OPTIONS")

	// PayPal payment operations
	paymentRouter.HandleFunc("/paypal/create-order", ar.paymentHandlers.CreatePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/order-status", ar.paymentHandlers.GetPayPalOrderStatus).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/capture", ar.paymentHandlers.CapturePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/refund", ar.paymentHandlers.RefundPayPalCapture).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/webhook", ar.paymentHandlers.PayPalWebhook).Methods("POST", "OPTIONS")
}

// registerKNIRVSHELLRoutes registers KNIRVSHELL-related routes
func (ar *APIRouter) registerKNIRVSHELLRoutes(apiV1 *mux.Router) {
	cliRouter := apiV1.PathPrefix("/shell").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		cliRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// KNIRVSHELL operations
	cliRouter.HandleFunc("/execute", ar.knirvshellHandlers.ExecuteCommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/wallet/info", ar.knirvshellHandlers.GetWalletInfo).Methods("GET", "OPTIONS")
	cliRouter.HandleFunc("/wallet/send", ar.knirvshellHandlers.SendToken).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/validation/execute", ar.knirvshellHandlers.ExecuteValidation).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/tee/execute", ar.knirvshellHandlers.ExecuteTEECommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/p2p/execute", ar.knirvshellHandlers.ExecuteP2PCommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/chain/execute", ar.knirvshellHandlers.ExecuteChainCommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/sessions", ar.knirvshellHandlers.ListSessions).Methods("GET", "OPTIONS")
	cliRouter.HandleFunc("/sessions", ar.knirvshellHandlers.CreateSession).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/sessions/{id}", ar.knirvshellHandlers.GetSession).Methods("GET", "OPTIONS")
	cliRouter.HandleFunc("/sessions/{id}/stop", ar.knirvshellHandlers.StopSession).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/sessions/{id}/input", ar.knirvshellHandlers.SendInput).Methods("POST", "OPTIONS")
}

// registerOnboardingRoutes registers Onboarding-related routes
func (ar *APIRouter) registerOnboardingRoutes(apiV1 *mux.Router) {
	onboardingRouter := apiV1.PathPrefix("/onboarding").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		onboardingRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// Onboarding operations
	onboardingRouter.HandleFunc("/organizations", ar.onboardingHandlers.OnboardOrganization).Methods("POST", "OPTIONS")
	onboardingRouter.HandleFunc("/organizations", ar.onboardingHandlers.ListConfigurations).Methods("GET", "OPTIONS")
	onboardingRouter.HandleFunc("/organizations/{id}", ar.onboardingHandlers.GetConfiguration).Methods("GET", "OPTIONS")
	onboardingRouter.HandleFunc("/organizations/{id}", ar.onboardingHandlers.UpdateConfiguration).Methods("PUT", "OPTIONS")
	onboardingRouter.HandleFunc("/organizations/{id}", ar.onboardingHandlers.DeleteConfiguration).Methods("DELETE", "OPTIONS")
	onboardingRouter.HandleFunc("/organizations/{id}/validate", ar.onboardingHandlers.ValidateAction).Methods("POST", "OPTIONS")
}

// registerCognitiveRoutes registers Cognitive Engine-related routes
func (ar *APIRouter) registerCognitiveRoutes(apiV1 *mux.Router) {
	cognitiveRouter := apiV1.PathPrefix("/cognitive").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		cognitiveRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// Cognitive Engine operations
	cognitiveRouter.HandleFunc("/metrics/{nodeId}", ar.cognitiveEngineHandlers.GetCognitiveMetrics).Methods("GET", "OPTIONS")
	cognitiveRouter.HandleFunc("/learning-state", ar.cognitiveEngineHandlers.GetLearningState).Methods("GET", "OPTIONS")
	cognitiveRouter.HandleFunc("/adaptations", ar.cognitiveEngineHandlers.GetAdaptationHistory).Methods("GET", "OPTIONS")
	cognitiveRouter.HandleFunc("/patterns", ar.cognitiveEngineHandlers.GetFailurePatterns).Methods("GET", "OPTIONS")
	cognitiveRouter.HandleFunc("/performance/tasks", ar.cognitiveEngineHandlers.GetTaskPerformance).Methods("GET", "OPTIONS")
	cognitiveRouter.HandleFunc("/performance/nodes", ar.cognitiveEngineHandlers.GetNodePerformance).Methods("GET", "OPTIONS")
	cognitiveRouter.HandleFunc("/learning/trigger", ar.cognitiveEngineHandlers.TriggerLearningCycle).Methods("POST", "OPTIONS")
	cognitiveRouter.HandleFunc("/status", ar.cognitiveEngineHandlers.GetStatus).Methods("GET", "OPTIONS")
}

// registerKnowledgeBaseRoutes registers Knowledge Base routes (GraphRAG-RS integration)
func (ar *APIRouter) registerKnowledgeBaseRoutes(apiV1 *mux.Router) {
	if ar.knowledgeBaseHandlers == nil {
		return
	}

	kbRouter := apiV1.PathPrefix("/knowledge-base").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		kbRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// CRUD operations
	kbRouter.HandleFunc("/objects", ar.knowledgeBaseHandlers.GetKnowledgeBases).Methods("GET", "OPTIONS")
	kbRouter.HandleFunc("/objects", ar.knowledgeBaseHandlers.PostKnowledgeBase).Methods("POST", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}", ar.knowledgeBaseHandlers.GetKnowledgeBase).Methods("GET", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}", ar.knowledgeBaseHandlers.PutKnowledgeBase).Methods("PUT", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}", ar.knowledgeBaseHandlers.DeleteKnowledgeBase).Methods("DELETE", "OPTIONS")

	// GraphRAG operations
	kbRouter.HandleFunc("/objects/{id}/query", ar.knowledgeBaseHandlers.QueryKnowledgeBase).Methods("POST", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}/index", ar.knowledgeBaseHandlers.IndexKnowledgeBase).Methods("POST", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}/index-status", ar.knowledgeBaseHandlers.GetIndexStatus).Methods("GET", "OPTIONS")
	kbRouter.HandleFunc("/objects/{id}/deploy", ar.knowledgeBaseHandlers.DeployKnowledgeBase).Methods("POST", "OPTIONS")

	// Summary
	kbRouter.HandleFunc("/summary", ar.knowledgeBaseHandlers.GetKnowledgeBaseSummary).Methods("GET", "OPTIONS")
}

// registerBackwardCompatibilityRoutes registers redirects for old API paths
func (ar *APIRouter) registerBackwardCompatibilityRoutes(r *mux.Router) {
	// Redirect old /api/dve-nodes/ to /api/v1/dve/nodes/
	r.PathPrefix("/api/dve-nodes/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/dve/nodes/" + path[len("/api/dve-nodes/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})

	// Redirect old /api/dve/ to /api/v1/dve/
	r.PathPrefix("/api/dve/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/dve/" + path[len("/api/dve/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})

	// Redirect old /api/plugin-management/ to /api/v1/plugin/
	r.PathPrefix("/api/plugin-management/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/plugin/" + path[len("/api/plugin-management/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})

	// Redirect old /api/payments/ to /api/v1/payments/
	r.PathPrefix("/api/payments/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/payments/" + path[len("/api/payments/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})

	// Redirect old /api/knirvshell/ to /api/v1/shell/
	r.PathPrefix("/api/knirvshell/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/shell/" + path[len("/api/knirvshell/"):]
		http.Redirect(w, r, newPath, http.StatusTemporaryRedirect)
	})

	// Redirect old /api/v1/cli/ to /api/v1/shell/ (the frontend console-panel uses this path)
	r.PathPrefix("/api/v1/cli/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/shell/" + path[len("/api/v1/cli/"):]
		http.Redirect(w, r, newPath, http.StatusTemporaryRedirect)
	})

	// Redirect old /api/cognitive/ to /api/v1/cognitive/
	r.PathPrefix("/api/cognitive/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/cognitive/" + path[len("/api/cognitive/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})

	// Redirect old /api/fabric-management/ to /api/v1/knowledge-base/
	r.PathPrefix("/api/fabric-management/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/knowledge-base/" + path[len("/api/fabric-management/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})
}
