package web

import (
	"net/http"

	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// APIRouter provides unified API routing with versioning
type APIRouter struct {
	dveHandlers             *DVEHandlers
	fabricHandlers          *FabricManagementHandlers
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
	fabricHandlers *FabricManagementHandlers,
	agentHandlers *AgentHandlers,
	paymentHandlers *PaymentHandlers,
	knirvcliHandlers *KNIRVCLIHandlers,
	onboardingHandlers *OnboardingHandlers,
	cognitiveEngineHandlers *CognitiveEngineHandlers,
	authMiddleware *middleware.AuthMiddleware,
) *APIRouter {
	return &APIRouter{
		dveHandlers:             dveHandlers,
		fabricHandlers:          fabricHandlers,
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

	// Register Fabric routes under /api/v1/fabric/
	ar.registerFabricRoutes(apiV1)

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

// registerFabricRoutes registers Fabric management routes
func (ar *APIRouter) registerFabricRoutes(apiV1 *mux.Router) {
	fabricRouter := apiV1.PathPrefix("/fabric").Subrouter()

	// Apply optional auth middleware
	if ar.authMiddleware != nil {
		fabricRouter.Use(ar.authMiddleware.OptionalAuth)
	}

	// Fabric CRUD operations
	fabricRouter.HandleFunc("/objects", ar.fabricHandlers.GetFabrics).Methods("GET", "OPTIONS")
	fabricRouter.HandleFunc("/objects", ar.fabricHandlers.PostFabric).Methods("POST", "OPTIONS")
	fabricRouter.HandleFunc("/objects/{id}", ar.fabricHandlers.GetFabric).Methods("GET", "OPTIONS")
	fabricRouter.HandleFunc("/objects/{id}", ar.fabricHandlers.PutFabric).Methods("PUT", "OPTIONS")
	fabricRouter.HandleFunc("/objects/{id}", ar.fabricHandlers.DeleteFabric).Methods("DELETE", "OPTIONS")

	// Fabric actions
	fabricRouter.HandleFunc("/objects/{id}/actions", ar.fabricHandlers.PostFabricAction).Methods("POST", "OPTIONS")

	// Fabric monitoring
	fabricRouter.HandleFunc("/objects/{id}/metrics", ar.fabricHandlers.GetFabricMetrics).Methods("GET", "OPTIONS")
	fabricRouter.HandleFunc("/objects/{id}/logs", ar.fabricHandlers.GetFabricLogs).Methods("GET", "OPTIONS")
	fabricRouter.HandleFunc("/objects/{id}/events", ar.fabricHandlers.GetFabricEvents).Methods("GET", "OPTIONS")

	// Templates
	fabricRouter.HandleFunc("/templates", ar.fabricHandlers.GetFabricTemplates).Methods("GET", "OPTIONS")
	fabricRouter.HandleFunc("/templates", ar.fabricHandlers.PostFabricTemplate).Methods("POST", "OPTIONS")

	// Summary
	fabricRouter.HandleFunc("/summary", ar.fabricHandlers.GetFabricSummary).Methods("GET", "OPTIONS")
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
	paymentRouter.HandleFunc("/stripe/create-intent", ar.paymentHandlers.CreateStripePaymentIntent).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/intent", ar.paymentHandlers.GetStripePaymentIntent).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/stripe/refund", ar.paymentHandlers.RefundStripePayment).Methods("POST", "OPTIONS")

	// PayPal payment operations
	paymentRouter.HandleFunc("/paypal/create-order", ar.paymentHandlers.CreatePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/order", ar.paymentHandlers.GetPayPalOrder).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/capture", ar.paymentHandlers.CapturePayPalOrder).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/paypal/refund", ar.paymentHandlers.RefundPayPalCapture).Methods("POST", "OPTIONS")

	// Blockchain payment operations
	paymentRouter.HandleFunc("/blockchain/balance", ar.paymentHandlers.GetBlockchainWalletBalance).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/blockchain/payment", ar.paymentHandlers.CreateBlockchainPayment).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/blockchain/transaction", ar.paymentHandlers.GetBlockchainTransaction).Methods("GET", "OPTIONS")
	paymentRouter.HandleFunc("/blockchain/verify", ar.paymentHandlers.VerifyBlockchainTransaction).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/blockchain/estimate-gas", ar.paymentHandlers.EstimateBlockchainGas).Methods("POST", "OPTIONS")
	paymentRouter.HandleFunc("/blockchain/history", ar.paymentHandlers.GetBlockchainTransactionHistory).Methods("GET", "OPTIONS")
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

	// Redirect old /api/fabric-management/ to /api/v1/fabric/
	r.PathPrefix("/api/fabric-management/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		newPath := "/api/v1/fabric/" + path[len("/api/fabric-management/"):]
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
