package icme

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"backend_server/internal/database"
)

const (
	kbKeyGlobalObj  = "icme:global:objectives:"
	kbKeyDVEObj     = "icme:dve:%s:objectives:"
	kbKeyGlobalBind = "icme:global:agent_bindings:"
	kbKeyDVEBind    = "icme:dve:%s:agent_bindings:"
	kbKeyAlignment  = "icme:alignment:%s:%s:"
)

type IntentRegistry struct {
	db             *database.BuntDBManager
	mu             sync.RWMutex
	cache          map[string]*IntentObjective
	dveCache       map[string]map[string]*IntentObjective
	globalBinds    map[string]string
	dveBindings    map[string]map[string]string
	alignmentCache []*AlignmentRecord
	logger         *zap.Logger
}

func NewIntentRegistry(db *database.BuntDBManager, logger *zap.Logger) (*IntentRegistry, error) {
	r := &IntentRegistry{
		db:             db,
		cache:          make(map[string]*IntentObjective),
		dveCache:       make(map[string]map[string]*IntentObjective),
		globalBinds:    make(map[string]string),
		dveBindings:    make(map[string]map[string]string),
		alignmentCache: make([]*AlignmentRecord, 0, 100),
		logger:         logger,
	}
	if err := r.loadFromDB(); err != nil {
		return nil, fmt.Errorf("intent registry load: %w", err)
	}
	return r, nil
}

func (r *IntentRegistry) RegisterObjective(obj *IntentObjective) error {
	if obj.Name == "" {
		return fmt.Errorf("objective name is required")
	}
	obj.UpdatedAt = time.Now()
	if obj.CreatedAt.IsZero() {
		obj.CreatedAt = obj.UpdatedAt
		obj.Version = 1
	} else {
		obj.Version++
	}

	var key string
	if obj.Scope == ScopeGlobal {
		key = kbKeyGlobalObj + obj.Name
	} else {
		key = fmt.Sprintf(kbKeyDVEObj+obj.Name, obj.DVEID)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal objective: %w", err)
	}

	if err := r.db.StoreJSON(key, data); err != nil {
		return fmt.Errorf("knirvbase store objective: %w", err)
	}

	r.mu.Lock()
	if obj.Scope == ScopeGlobal {
		r.cache[obj.Name] = obj
	} else {
		if r.dveCache[obj.DVEID] == nil {
			r.dveCache[obj.DVEID] = make(map[string]*IntentObjective)
		}
		r.dveCache[obj.DVEID][obj.Name] = obj
	}
	r.mu.Unlock()

	r.logger.Info("icme intent objective registered",
		zap.String("name", obj.Name),
		zap.String("scope", string(obj.Scope)),
		zap.Int("version", obj.Version),
	)
	return nil
}

func (r *IntentRegistry) BindAgentToObjective(agentID, objectiveName string, dveID string) error {
	r.mu.RLock()
	var exists bool
	if dveID == "" {
		_, exists = r.cache[objectiveName]
	} else {
		if dveCache, ok := r.dveCache[dveID]; ok {
			_, exists = dveCache[objectiveName]
		}
	}
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("objective %q not found", objectiveName)
	}

	var bindKey string
	if dveID == "" {
		bindKey = kbKeyGlobalBind + agentID
	} else {
		bindKey = fmt.Sprintf(kbKeyDVEBind+agentID, dveID)
	}

	if err := r.db.StoreJSON(bindKey, []byte(`"`+objectiveName+`"`)); err != nil {
		return fmt.Errorf("knirvbase bind agent: %w", err)
	}

	r.mu.Lock()
	if dveID == "" {
		r.globalBinds[agentID] = objectiveName
	} else {
		if r.dveBindings[dveID] == nil {
			r.dveBindings[dveID] = make(map[string]string)
		}
		r.dveBindings[dveID][agentID] = objectiveName
	}
	r.mu.Unlock()
	return nil
}

func (r *IntentRegistry) GetObjectiveForAgent(agentID, dveID string) *IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if dveID != "" {
		if binds, ok := r.dveBindings[dveID]; ok {
			if objName, ok := binds[agentID]; ok {
				if objs, ok := r.dveCache[dveID]; ok {
					return objs[objName]
				}
			}
		}
	}

	if objName, ok := r.globalBinds[agentID]; ok {
		return r.cache[objName]
	}
	return nil
}

func (r *IntentRegistry) GetGlobalObjectiveForAgent(agentID string) *IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if objName, ok := r.globalBinds[agentID]; ok {
		return r.cache[objName]
	}
	return nil
}

func (r *IntentRegistry) GetObjectiveForDVE(agentID, dveID string) *IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if binds, ok := r.dveBindings[dveID]; ok {
		if objName, ok := binds[agentID]; ok {
			if objs, ok := r.dveCache[dveID]; ok {
				return objs[objName]
			}
		}
	}
	return nil
}

func (r *IntentRegistry) EvaluateTradeOffs(agentID, dveID string, context map[string]float64) float64 {
	obj := r.GetObjectiveForAgent(agentID, dveID)
	if obj == nil {
		return 0.5
	}
	var weighted, total float64
	for key, weight := range obj.TradeOffs {
		if val, ok := context[key]; ok {
			weighted += val * weight
			total += weight
		}
	}
	if total == 0 {
		return 0.5
	}
	return weighted / total
}

func (r *IntentRegistry) IsActionAuthorized(agentID, dveID, action string) bool {
	obj := r.GetObjectiveForAgent(agentID, dveID)
	if obj == nil {
		return true
	}
	for _, a := range obj.AuthorizedActions {
		if a == action {
			return true
		}
	}
	return false
}

func (r *IntentRegistry) ViolatesHardBoundary(agentID, dveID, action string) bool {
	obj := r.GetObjectiveForAgent(agentID, dveID)
	if obj == nil {
		return false
	}
	for _, boundary := range obj.HardBoundaries {
		if boundary == action {
			return true
		}
	}
	return false
}

func (r *IntentRegistry) RecordAlignment(rec *AlignmentRecord) error {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(kbKeyAlignment, rec.AgentID, rec.ID)
	if err := r.db.StoreJSON(key, data); err != nil {
		return err
	}

	r.mu.Lock()
	r.alignmentCache = append(r.alignmentCache, rec)
	if len(r.alignmentCache) > 100 {
		r.alignmentCache = r.alignmentCache[len(r.alignmentCache)-100:]
	}
	r.mu.Unlock()

	return nil
}

func (r *IntentRegistry) ListAlignmentRecords(agentID string) ([]*AlignmentRecord, error) {
	return nil, nil
}

func (r *IntentRegistry) GetRecentAlignmentRecords(objectiveName string, limit int) []*AlignmentRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var records []*AlignmentRecord
	for _, rec := range r.alignmentCache {
		if rec.ObjectiveName == objectiveName {
			records = append(records, rec)
		}
	}

	if len(records) > limit {
		return records[len(records)-limit:]
	}
	return records
}

func (r *IntentRegistry) ListObjectives(dveID string) []*IntentObjective {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*IntentObjective
	for _, obj := range r.cache {
		result = append(result, obj)
	}
	if dveID != "" {
		if dveObjs, ok := r.dveCache[dveID]; ok {
			for _, obj := range dveObjs {
				result = append(result, obj)
			}
		}
	}
	return result
}

func (r *IntentRegistry) loadFromDB() error {
	return nil
}
