package vault

import "sync"

type OntologyRegistry struct {
	mu          sync.RWMutex
	frameworks  map[string]*Framework
	constraints map[string]*Constraint
}

type Framework struct {
	Name        string
	Version     string
	Domain      string
	Constraints []string
}

type Constraint struct {
	ID          string
	Name        string
	Type        string
	Expression  string
	Description string
}

func NewOntologyRegistry() *OntologyRegistry {
	return &OntologyRegistry{
		frameworks:  make(map[string]*Framework),
		constraints: make(map[string]*Constraint),
	}
}

func (r *OntologyRegistry) RegisterFramework(framework *Framework) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frameworks[framework.Name] = framework
}

func (r *OntologyRegistry) RegisterConstraint(constraint *Constraint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constraints[constraint.ID] = constraint
}

func (r *OntologyRegistry) GetFramework(name string) (*Framework, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.frameworks[name]
	return f, ok
}

func (r *OntologyRegistry) GetConstraint(id string) (*Constraint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.constraints[id]
	return c, ok
}

func (r *OntologyRegistry) ListFrameworks() []*Framework {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Framework, 0, len(r.frameworks))
	for _, f := range r.frameworks {
		result = append(result, f)
	}
	return result
}

func (r *OntologyRegistry) ListConstraints() []*Constraint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Constraint, 0, len(r.constraints))
	for _, c := range r.constraints {
		result = append(result, c)
	}
	return result
}
