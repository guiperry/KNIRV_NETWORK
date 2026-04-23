package installer

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type ServiceConfig struct {
	Enabled        bool
	BaseURL        string
	OracleURL     string
	CheckInterval time.Duration
	Timeout       time.Duration
}

type InstallerService struct {
	config  *ServiceConfig
	logger  *zap.Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewService(cfg *ServiceConfig, logger *zap.Logger) *InstallerService {
	return &InstallerService{
		config: cfg,
		logger: logger,
	}
}

func (s *InstallerService) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if !s.config.Enabled {
		s.logger.Info("Installer service disabled")
		return nil
	}

	s.logger.Info("Starting installer service",
		zap.String("base_url", s.config.BaseURL),
		zap.String("oracle_url", s.config.OracleURL))

	go s.runMonitor()

	return nil
}

func (s *InstallerService) Stop(ctx context.Context) error {
	s.logger.Info("Stopping installer service")

	if s.cancel != nil {
		s.cancel()
	}

	return nil
}

func (s *InstallerService) runMonitor() {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkInstallations()
		}
	}
}

func (s *InstallerService) checkInstallations() {
	s.logger.Debug("Checking install completions")
}

type InstallProgress struct {
	NodeID        string    `json:"node_id"`
	Phase        string    `json:"phase"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error        string    `json:"error,omitempty"`
}

func (s *InstallerService) GetProgress(nodeID string) (*InstallProgress, error) {
	return nil, fmt.Errorf("not implemented")
}

type InstallRequest struct {
	NodeID         string `json:"node_id"`
	OwnerID       string `json:"owner_id"`
	DesiredURI   string `json:"desired_uri,omitempty"`
	BootnodeIP   string `json:"bootnode_ip,omitempty"`
	BootnodePort int    `json:"bootnode_port,omitempty"`
}

type InstallResponse struct {
	NodeID       string `json:"node_id"`
	DVEID        string `json:"dve_id"`
	FullURI      string `json:"full_uri"`
	WalletAddr  string `json:"wallet_address"`
	TxHash      string `json:"tx_hash"`
	Status      string `json:"status"`
}