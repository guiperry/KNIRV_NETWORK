package web

import (
	"encoding/json"
	"net/http"
	"time"

	"nexus-backend/internal/models"
	"nexus-backend/internal/services/dverental"
	"nexus-backend/internal/web/middleware"

	"github.com/gorilla/mux"
)

// DVERentalHandlers handles DVE rental API requests
type DVERentalHandlers struct {
	dveRentalService *dverental.DVERentalService
}

// NewDVERentalHandlers creates new DVE rental handlers
func NewDVERentalHandlers(dveRentalService *dverental.DVERentalService) *DVERentalHandlers {
	return &DVERentalHandlers{dveRentalService: dveRentalService}
}

// DVERentalResponse represents a standard API response for DVE rental operations
type DVERentalResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetRentalPlans handles GET /api/dve-rental/plans
func (h *DVERentalHandlers) GetRentalPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.dveRentalService.GetRentalPlans()
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to fetch rental plans: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      plans,
		Message:   "Rental plans retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateRental handles POST /api/dve-rental/rentals
func (h *DVERentalHandlers) CreateRental(w http.ResponseWriter, r *http.Request) {
	var req models.RentalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	// For now, use the user ID from the request
	if req.UserID == "" {
		req.UserID = "test-user-" + time.Now().Format("20060102150405")
	}

	rentalResponse, err := h.dveRentalService.CreateRental(&req)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to create rental: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !rentalResponse.Success {
		response := DVERentalResponse{
			Success:   false,
			Error:     rentalResponse.Error,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      rentalResponse,
		Message:   "DVE rental created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetUserRentals handles GET /api/dve-rental/rentals
func (h *DVERentalHandlers) GetUserRentals(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	rentals, err := h.dveRentalService.GetActiveRentals(userID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to fetch user rentals: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      rentals,
		Message:   "User rentals retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRentalStats handles GET /api/dve-rental/stats
func (h *DVERentalHandlers) GetRentalStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.dveRentalService.GetRentalStats()
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to fetch rental stats: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      stats,
		Message:   "Rental statistics retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExtendRental handles POST /api/dve-rental/rentals/{id}/extend
func (h *DVERentalHandlers) ExtendRental(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	var req struct {
		AdditionalDuration int64  `json:"additional_duration"`
		PaymentTxHash      string `json:"payment_tx_hash"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.dveRentalService.ExtendRental(rentalID, req.AdditionalDuration, req.PaymentTxHash)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to extend rental: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Message:   "Rental extended successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelRental handles DELETE /api/dve-rental/rentals/{id}
func (h *DVERentalHandlers) CancelRental(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	err := h.dveRentalService.CancelRental(rentalID, userID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to cancel rental: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Message:   "Rental cancelled successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the DVE rental routes with the router
func (h *DVERentalHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DVE rental endpoints
	rentalRouter := r.PathPrefix("/api/dve-rental").Subrouter()

	// Public routes for viewing plans and stats
	rentalRouter.HandleFunc("/plans", h.GetRentalPlans).Methods("GET")
	rentalRouter.HandleFunc("/stats", h.GetRentalStats).Methods("GET")

	// Protected routes for rental management
	if authMiddleware != nil {
		protectedRentalRouter := rentalRouter.PathPrefix("").Subrouter()
		protectedRentalRouter.Use(authMiddleware.RequireAuth)
		protectedRentalRouter.HandleFunc("/rentals", h.CreateRental).Methods("POST")
		protectedRentalRouter.HandleFunc("/rentals", h.GetUserRentals).Methods("GET")
		protectedRentalRouter.HandleFunc("/rentals/{id}/extend", h.ExtendRental).Methods("POST")
		protectedRentalRouter.HandleFunc("/rentals/{id}", h.CancelRental).Methods("DELETE")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		rentalRouter.HandleFunc("/rentals", h.CreateRental).Methods("POST")
		rentalRouter.HandleFunc("/rentals", h.GetUserRentals).Methods("GET")
		rentalRouter.HandleFunc("/rentals/{id}/extend", h.ExtendRental).Methods("POST")
		rentalRouter.HandleFunc("/rentals/{id}", h.CancelRental).Methods("DELETE")
	}
}
