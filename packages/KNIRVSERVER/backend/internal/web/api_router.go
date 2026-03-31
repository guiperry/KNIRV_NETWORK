package web

import (
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
	knirvcliHandlers        *KNIRVCLIHandlers
	onboardingHandlers      *OnboardingHandlers
	cognitiveEngineHandlers *CognitiveEngineHandlers
	authMiddleware          *middleware.AuthMiddleware
}

// NewAPIRouter creates a new unified API router
func NewAPIRouter(
	dveHandlers *DVEHandlers,
	pluginHandlers *PluginManagementHandlers,
	agentHandlers *AgentHandlers,
	paymentHandlers *PaymentHandlers,
	knirvcliHandlers *KNIRVCLIHandlers,
	onboardingHandlers *OnboardingHandlers,
	cognitiveEngineHandlers *CognitiveEngineHandlers,
	authMiddleware *middleware.AuthMiddleware,
) *APIRouter {
	return &APIRouter{
		dveHandlers:             dveHandlers,
		pluginHandlers:          pluginHandlers,
		agentHandlers:           agentHandlers,
		paymentHandlers:         paymentHandlers,
		knirvcliHandlers:        knirvcliHandlers,
		onboardingHandlers:      onboardingHandlers,
		cognitiveEngineHandlers: cognitiveEngineHandlers,
		authMiddleware:          authMiddleware,
	}
}

// RegisterRoutes registers all unified API routes
func (ar *APIRouter) RegisterRoutes(r *mux.Router) {
	// Create versioned API subrouter
	apiV1 := r.PathPrefix("/api/v1").Subrouter()

	// Register DVE routes under /api/v1/dve/
	ar.registerDVERoutes(apiV1)

	// Register Plugin routes under /api/v1/plugin/
	ar.registerPluginRoutes(apiV1)

	// Register Agent routes under /api/v1/dve/{id}/agent/
	ar.registerAgentRoutes(apiV1)

	// Register Payment routes under /api/v1/payments/
	ar.registerPaymentRoutes(apiV1)

	// Register KNIRVCLI routes under /api/v1/cli/
	ar.registerKNIRVCLIRoutes(apiV1)

	// Register Onboarding routes under /api/v1/onboarding/
	ar.registerOnboardingRoutes(apiV1)

	// Register Cognitive Engine routes under /api/v1/cognitive/
	ar.registerCognitiveRoutes(apiV1)

	// Register backward compatibility redirects
	ar.registerBackwardCompatibilityRoutes(r)
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

	// P2P network
	dveRouter.HandleFunc("/peers", ar.dveHandlers.GetP2PPeers).Methods("GET", "OPTIONS")
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

	// PayPal payment operations
	paymentRouter.HandleFunc("/paypal/create-order", ar.paymentHandlers.CreatePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/order-status", ar.paymentHandlers.GetPayPalOrderStatus).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/capture", ar.paymentHandlers.CapturePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/refund", ar.paymentHandlers.RefundPayPalCapture).Methods("POST", "OPTIONS")
}

// registerKNIRVCLIRoutes registers KNIRVCLI-related routes
func (ar *APIRouter) registerKNIRVCLIRoutes(apiV1 *mux.Router) {
	cliRouter := apiV1.PathPrefix("/cli").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		cliRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// KNIRVCLI operations
	cliRouter.HandleFunc("/execute", ar.knirvcliHandlers.ExecuteCommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/wallet/info", ar.knirvcliHandlers.GetWalletInfo).Methods("GET", "OPTIONS")
	cliRouter.HandleFunc("/wallet/send", ar.knirvcliHandlers.SendToken).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/validation/execute", ar.knirvcliHandlers.ExecuteValidation).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/tee/execute", ar.knirvcliHandlers.ExecuteTEECommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/p2p/execute", ar.knirvcliHandlers.ExecuteP2PCommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/chain/execute", ar.knirvcliHandlers.ExecuteChainCommand).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/sessions", ar.knirvcliHandlers.ListSessions).Methods("GET", "OPTIONS")
	cliRouter.HandleFunc("/sessions/{id}", ar.knirvcliHandlers.GetSession).Methods("GET", "OPTIONS")
	cliRouter.HandleFunc("/sessions/{id}/stop", ar.knirvcliHandlers.StopSession).Methods("POST", "OPTIONS")
	cliRouter.HandleFunc("/sessions/{id}/input", ar.knirvcliHandlers.SendInput).Methods("POST", "OPTIONS")
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

	// Redirect old /api/knirvcli/ to /api/v1/cli/
	r.PathPrefix("/api/knirvcli/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/cli/" + path[len("/api/knirvcli/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})

	// Redirect old /api/onboarding/ to /api/v1/onboarding/
	r.PathPrefix("/api/onboarding/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/onboarding/" + path[len("/api/onboarding/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})

	// Redirect old /api/cognitive/ to /api/v1/cognitive/
	r.PathPrefix("/api/cognitive/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/cognitive/" + path[len("/api/cognitive/"):]
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})
}
