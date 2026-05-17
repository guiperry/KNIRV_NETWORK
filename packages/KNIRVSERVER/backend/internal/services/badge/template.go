// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package badge provides Badge lifecycle services including template management.
package badge

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BadgeTemplate is a reusable blueprint for minting badges.
// Admins create templates via the Badge Lab, which then become available
// in the DVE Workspace Policy panel and WebGUI Vault for minting.
type BadgeTemplate struct {
	ID                string            `json:"id"`                 // UUID
	Name              string            `json:"name"`               // e.g. "Certified Validator"
	BadgeType         string            `json:"badge_type"`         // "skill", "capability", "property"
	Description       string            `json:"description"`        // Human-readable purpose
	ValueSignals      []string          `json:"value_signals"`      // Selected Values
	OntologySignals   []string          `json:"ontology_signals"`   // Selected Ontology elements
	AITextTag         string            `json:"ai_text_tag"`        // e.g. "Confidential • v2.1"
	PrimaryColor      string            `json:"primary_color"`      // hex e.g. "#FBBF24"
	SecondaryColor    string            `json:"secondary_color"`    // hex e.g. "#D97706"
	BackgroundColor   string            `json:"background_color"`   // hex e.g. "#02040a"
	AlignmentThreshold float64          `json:"alignment_threshold"`
	AuthCredentials   AuthCredentialSet `json:"auth_credentials"`   // scoped API keys, JWT, role
	SVGDesign         string            `json:"svg_design"`         // pre-rendered SVG
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	IsActive          bool              `json:"is_active"`          // soft delete / archive
}

// AuthCredentialSet captures optional auth credential scoping for a badge template.
type AuthCredentialSet struct {
	APIKeyProvider string `json:"api_key_provider,omitempty"`
	APIKeyScope    string `json:"api_key_scope,omitempty"`
	JWTClaim       string `json:"jwt_claim,omitempty"`
	JWTValue       string `json:"jwt_value,omitempty"`
	SelectedRole   string `json:"selected_role,omitempty"`
}

// BadgeTemplateRegistry manages badge template CRUD with JSON file persistence.
// Thread-safe via sync.RWMutex. Persists to a JSON file on every mutation.
type BadgeTemplateRegistry struct {
	mu       sync.RWMutex
	templates map[string]*BadgeTemplate // id → template
	filePath string                     // path to JSON persistence file
}

// NewBadgeTemplateRegistry creates a registry that persists to the given file path.
// The directory must already exist (or be creatable). Loads existing data on creation.
func NewBadgeTemplateRegistry(filePath string) (*BadgeTemplateRegistry, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("badge template registry: create directory %s: %w", dir, err)
	}

	r := &BadgeTemplateRegistry{
		templates: make(map[string]*BadgeTemplate),
		filePath:  filePath,
	}

	if err := r.load(); err != nil {
		log.Printf("[badge-template] loaded 0 templates from %s (new file)", filePath)
	} else {
		log.Printf("[badge-template] loaded %d templates from %s", len(r.templates), filePath)
	}

	return r, nil
}

// DefaultTemplatePath returns the default path for badge template persistence,
// using the KNIRV application data directory (/var/lib/knirvserver/ or KNIRV_DATA_DIR env).
func DefaultTemplatePath() string {
	dir := os.Getenv("KNIRV_DATA_DIR")
	if dir == "" {
		dir = os.Getenv("KNIRV_APP_DATA_DIR")
	}
	if dir == "" {
		dir = "/var/lib/knirvserver"
	}
	return filepath.Join(dir, "badge-templates.json")
}

// Create adds a new template, assigns a UUID, sets timestamps, and persists.
func (r *BadgeTemplateRegistry) Create(tmpl *BadgeTemplate) error {
	if tmpl == nil {
		return fmt.Errorf("template is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tmpl.ID = uuid.New().String()
	tmpl.CreatedAt = time.Now().UTC()
	tmpl.UpdatedAt = time.Now().UTC()
	tmpl.IsActive = true

	r.templates[tmpl.ID] = tmpl
	return r.saveLocked()
}

// Get retrieves a template by ID. Returns nil and an error if not found.
func (r *BadgeTemplateRegistry) Get(id string) (*BadgeTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, ok := r.templates[id]
	if !ok {
		return nil, fmt.Errorf("badge template %s not found", id)
	}
	return tmpl, nil
}

// List returns all templates. If activeOnly is true, only IsActive templates are returned.
func (r *BadgeTemplateRegistry) List(activeOnly bool) ([]*BadgeTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*BadgeTemplate
	for _, tmpl := range r.templates {
		if activeOnly && !tmpl.IsActive {
			continue
		}
		result = append(result, tmpl)
	}
	if result == nil {
		result = []*BadgeTemplate{}
	}
	return result, nil
}

// Update replaces a template's mutable fields, regenerates timestamps, and persists.
func (r *BadgeTemplateRegistry) Update(id string, tmpl *BadgeTemplate) error {
	if tmpl == nil {
		return fmt.Errorf("template is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.templates[id]
	if !ok {
		return fmt.Errorf("badge template %s not found", id)
	}

	// Preserve ID, created_at, and carry over the ID from the URL.
	existing.Name = tmpl.Name
	existing.BadgeType = tmpl.BadgeType
	existing.Description = tmpl.Description
	existing.ValueSignals = tmpl.ValueSignals
	existing.OntologySignals = tmpl.OntologySignals
	existing.AITextTag = tmpl.AITextTag
	existing.PrimaryColor = tmpl.PrimaryColor
	existing.SecondaryColor = tmpl.SecondaryColor
	existing.BackgroundColor = tmpl.BackgroundColor
	existing.AlignmentThreshold = tmpl.AlignmentThreshold
	existing.AuthCredentials = tmpl.AuthCredentials
	existing.SVGDesign = tmpl.SVGDesign
	existing.UpdatedAt = time.Now().UTC()

	return r.saveLocked()
}

// Delete soft-deletes (archives) a template by setting IsActive=false.
// Returns error if the template does not exist.
func (r *BadgeTemplateRegistry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tmpl, ok := r.templates[id]
	if !ok {
		return fmt.Errorf("badge template %s not found", id)
	}

	tmpl.IsActive = false
	tmpl.UpdatedAt = time.Now().UTC()
	return r.saveLocked()
}

// ── persistence ──

// saveLocked writes all templates to the JSON file. Must be called with mu held.
func (r *BadgeTemplateRegistry) saveLocked() error {
	data, err := json.MarshalIndent(r.templates, "", "  ")
	if err != nil {
		return fmt.Errorf("badge template registry: marshal error: %w", err)
	}
	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("badge template registry: write error: %w", err)
	}
	return nil
}

// load reads all templates from the JSON file on startup.
func (r *BadgeTemplateRegistry) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return err // new file — not an error
		}
		return fmt.Errorf("badge template registry: read error: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &r.templates); err != nil {
		return fmt.Errorf("badge template registry: unmarshal error: %w", err)
	}
	return nil
}

// Stats returns registry statistics.
func (r *BadgeTemplateRegistry) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.templates)
	active := 0
	for _, t := range r.templates {
		if t.IsActive {
			active++
		}
	}
	return map[string]interface{}{
		"total_templates":  total,
		"active_templates": active,
		"persistence_path": r.filePath,
	}
}
