package oracle

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"go.uber.org/zap"
)

// authorityChainPrefix marks chain ids that represent authority/boot-node
// enrollments carrying a D14 bond hold.
const authorityChainPrefix = "authority/"

// CompleteBondReleases finds every registered authority chain that carries a bond
// hold and ensures the hold has been released to its boot node address by the
// root owner. This is the server-side trigger for D14 completion
// (design_root_tx_chain.md §2.4): registration reserves the hold (ConsumeHold
// binds it to the chain), and the root-owner-signed Release moves the escrow to
// the boot node address. A hold that is merely reserved (consumed) is released
// here; a hold already released is left untouched (idempotent). It is safe to
// call repeatedly — this is the restart-safety guarantee: a hold reserved by a
// registration immediately before a crash is released as soon as the Oracle is
// back up.
func (o *Oracle) CompleteBondReleases(ctx context.Context) error {
	if o.Mode() != ModeActive || o.replication == nil || o.replication.signer == nil {
		return nil
	}
	tok := o.GetNRNToken()
	now := time.Now()
	for _, reg := range o.checkpoint.registry.List() {
		if reg == nil {
			continue
		}
		if !strings.HasPrefix(reg.ChainID, authorityChainPrefix) {
			continue
		}
		if reg.HoldID == "" || reg.BondOwner == "" {
			continue
		}
		hold, err := tok.GetHold(reg.HoldID)
		if err != nil {
			o.logger.Warn("bond completion: cannot read hold",
				zap.String("hold", reg.HoldID), zap.Error(err))
			continue
		}
		switch hold.Status {
		case ledger.BondStatusReleased:
			// Already released by the root owner; nothing to do.
			continue
		case ledger.BondStatusPending, ledger.BondStatusConsumed:
			release := knirvsigning.BondRelease{
				HoldID:          reg.HoldID,
				BootNodeChainID: reg.ChainID,
				BootNodeAddress: reg.BondOwner,
			}
			payload, err := json.Marshal(release)
			if err != nil {
				o.logger.Error("bond completion: encode release", zap.Error(err))
				continue
			}
			signed, err := o.replication.signer.SignMessage(ctx, knirvsigning.MessageEnvelope{
				SchemaVersion: knirvsigning.MessageSchemaVersion,
				Domain:        knirvsigning.RegistryDomain,
				Purpose:       knirvsigning.PurposeBondRelease,
				ChainID:       tok.ChainID(),
				Nonce:         "bond-release:" + reg.HoldID,
				IssuedAtUnix:  now.Unix(),
				ExpiresAtUnix: now.Add(time.Hour).Unix(),
				Payload:       payload,
			})
			if err != nil {
				o.logger.Error("bond completion: sign release",
					zap.String("hold", reg.HoldID), zap.Error(err))
				continue
			}
			if err := tok.Release(reg.HoldID, release, signed, now); err != nil {
				o.logger.Error("bond completion: release",
					zap.String("hold", reg.HoldID), zap.Error(err))
				continue
			}
			o.logger.Info("bond completion: released hold to boot node",
				zap.String("hold", reg.HoldID),
				zap.String("chain", reg.ChainID),
				zap.String("boot", reg.BondOwner))
		default:
			o.logger.Warn("bond completion: unexpected hold status",
				zap.String("hold", reg.HoldID), zap.String("status", hold.Status))
		}
	}
	return nil
}

// SweepExpiredBondRefunds refunds every pending bond hold whose expiry has
// passed, returning the operator's escrow for abandoned authority enrollments
// (registration never completed). It is the safety net that prevents stranded
// escrow — "no partial registration".
func (o *Oracle) SweepExpiredBondRefunds(ctx context.Context) error {
	if o.Mode() != ModeActive {
		return nil
	}
	n, err := o.GetNRNToken().RefundExpiredHolds(time.Now())
	if err != nil {
		o.logger.Error("bond refund sweep failed", zap.Error(err))
		return err
	}
	if n > 0 {
		o.logger.Info("bond refund sweep released escrow for abandoned enrollments",
			zap.Int("holds", n))
	}
	return nil
}

// startBondSupervisor runs the bond completion + refund sweep. It performs an
// initial pass at startup (restart safety) and then repeats on a slow interval.
func (o *Oracle) startBondSupervisor(ctx context.Context) {
	o.runBondSupervisorPass(ctx)
	done := make(chan struct{})
	o.supervisorDone = done
	go func() {
		const interval = 5 * time.Minute
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.runBondSupervisorPass(ctx)
			}
		}
	}()
}

func (o *Oracle) runBondSupervisorPass(ctx context.Context) {
	if err := o.CompleteBondReleases(ctx); err != nil {
		o.logger.Error("bond completion pass failed", zap.Error(err))
	}
	if err := o.SweepExpiredBondRefunds(ctx); err != nil {
		o.logger.Error("bond refund sweep pass failed", zap.Error(err))
	}
}
