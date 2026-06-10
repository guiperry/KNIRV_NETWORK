package did

import "sync"

type MemoryStore struct {
	mu          sync.RWMutex
	docs        map[string]*DIDDocument
	deactivated map[string]bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		docs:        make(map[string]*DIDDocument),
		deactivated: make(map[string]bool),
	}
}

func (s *MemoryStore) Get(did string) (*DIDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.deactivated[did] {
		return nil, ErrDIDDeactivated
	}

	doc, ok := s.docs[did]
	if !ok {
		return nil, ErrDIDNotFound
	}
	return doc, nil
}

func (s *MemoryStore) Put(doc *DIDDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.docs[doc.ID] = doc
	return nil
}

func (s *MemoryStore) Deactivate(did string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deactivated[did] = true
	delete(s.docs, did)
	return nil
}
