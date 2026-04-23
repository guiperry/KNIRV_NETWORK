package services

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type DVEURI struct {
	ID         string `json:"id"`
	DVEID      string `json:"dve_id"`
	FullURI    string `json:"full_uri"`
	WalletAddr string `json:"wallet_address"`
	Endpoint  string `json:"endpoint"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type DVEURIRegistry struct {
	mu     sync.RWMutex
	uris   map[string]*DVEURI
	logger *zap.Logger
}

func NewDVEURIRegistry(_dbPath string, _chainClient interface{}, logger *zap.Logger) (*DVEURIRegistry, error) {
	return &DVEURIRegistry{
		uris:   make(map[string]*DVEURI),
		logger: logger,
	}, nil
}

func (r *DVEURIRegistry) Register(dve *DVEURI) error {
	if dve.ID == "" {
		dve.ID = fmt.Sprintf("dve_uri:%s", dve.DVEID)
	}
	if dve.CreatedAt == 0 {
		dve.CreatedAt = time.Now().Unix()
	}
	if dve.Status == "" {
		dve.Status = "active"
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.uris[dve.DVEID] = dve

	r.logger.Info("DVE URI registered",
		zap.String("dve_id", dve.DVEID),
		zap.String("full_uri", dve.FullURI))

	return nil
}

func (r *DVEURIRegistry) GetByDVEID(dveID string) (*DVEURI, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dve, ok := r.uris[dveID]
	if !ok {
		return nil, fmt.Errorf("DVE URI not found: %s", dveID)
	}

	return dve, nil
}

func (r *DVEURIRegistry) Resolve(uri string) (*DVEURI, error) {
	var dveID string
	fmt.Sscanf(uri, "knirv://%s", &dveID)
	if dveID == "" {
		return nil, fmt.Errorf("invalid DVE URI format")
	}

	return r.GetByDVEID(dveID)
}

func (r *DVEURIRegistry) Close() error {
	return nil
}