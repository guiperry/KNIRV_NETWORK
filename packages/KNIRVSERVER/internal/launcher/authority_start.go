package launcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guiperry/knirv-sdk-go/secrets"
	"github.com/spf13/viper"

	knirvauthority "knirv-server/internal/authority"
	"knirv-server/pkg/embedded/validationchain/checkpoint"
)

const authorityAnchorInterval = 30 * time.Second

// authoritySocketPath is where backend_server reaches the authority service.
func authoritySocketPath() string {
	return embeddedChainSocketPath("KNIRV_AUTHORITY_SOCKET_PATH", "authority.sock")
}

// startAuthority starts the authority-record service (Task 2b.1).
//
// OBSERVE MODE ONLY: records are built, signed, stored and anchored, and the
// self-verification verdict is logged and counted, but nothing is ever rejected.
// authority.mode=enforce is accepted as a request and downgraded with a warning.
//
// It is never fatal. A node with neither root.key nor boot.key, a locked key, a bad
// policy.json or a damaged store gets a DEGRADED service: the socket still answers
// /authority/status with the reason, and authorize refuses. Config keys:
//
//	authority.mode        observe (default) | enforce (downgraded to observe)
//	authority.bond        chain bond, default 0 (WARNs: the operator must set it)
//	authority.bond_owner  address that owns the bond
func (app *ServerApp) startAuthority(ctx context.Context) error {
	appDataDir, err := getAppDataDir()
	if err != nil {
		return fmt.Errorf("resolve app data dir: %w", err)
	}
	mode := strings.TrimSpace(os.Getenv("KNIRV_AUTHORITY_MODE"))
	if mode == "" {
		mode = viper.GetString("authority.mode")
	}

	gatewayURL, gwErr := resolvePublicURL(app.config)
	if gwErr != nil {
		gatewayURL = checkpoint.DefaultOracleGatewayURL
	}
	gatewayURL = strings.TrimRight(gatewayURL, "/")
	oracleURL := strings.TrimRight(strings.TrimSpace(os.Getenv("KNIRV_ORACLE_URL")), "/")
	if oracleURL == "" {
		oracleURL = gatewayURL
	}

	svc, anchor := knirvauthority.Assemble(ctx, knirvauthority.AssembleConfig{
		Dir:       filepath.Join(appDataDir, "authority"),
		Mode:      mode,
		Provider:  secrets.NewFileProvider(),
		Poster:    checkpoint.NewPoster(gatewayURL, oracleURL),
		Bond:      viper.GetUint64("authority.bond"),
		BondOwner: strings.TrimSpace(viper.GetString("authority.bond_owner")),
		Logf:      log.Printf,
	})

	socketPath := authoritySocketPath()
	server := &knirvauthority.Server{Svc: svc, Anchor: anchor, Token: app.internalAuthToken}
	go func() {
		if err := server.Serve(ctx, socketPath); err != nil {
			log.Printf("Warning: authority service stopped serving on %s: %v", socketPath, err)
		}
	}()
	go svc.Run(ctx) // keeps the ephemeral session key fresh
	if anchor != nil {
		go anchor.Run(ctx, authorityAnchorInterval)
	}
	st := svc.Status()
	log.Printf("Authority service on %s: state=%s mode=%s", socketPath, st.State, st.Mode)
	return nil
}
