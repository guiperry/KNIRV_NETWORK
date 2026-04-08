package storage

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

const (
	iFrameInterval = 50
	driftThreshold = 0.25
)

type PendingBracket struct {
	Bracket    *nrv.Bracket
	DeltaType  nrv.DeltaType
	AnchorID   *string
	DriftScore float64
}

type FrameTicker struct {
	writer   *NRVWriter
	interval time.Duration
	mu       sync.Mutex
	pending  []PendingBracket
	lastIBkt *nrv.Bracket
	lastIID  string
	bktCount int

	thermoSamples []nrv.ThermoAtmosphere
	linguistic    nrv.LinguisticMapping

	ticker *time.Ticker
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewFrameTicker(w *NRVWriter, interval time.Duration) *FrameTicker {
	ft := &FrameTicker{
		writer:   w,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
	ft.ticker = time.NewTicker(interval)
	ft.wg.Add(1)
	go ft.run()
	return ft
}

func (ft *FrameTicker) AppendBracket(ctx context.Context, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	b.ID = uuid.New().String()

	var deltaType nrv.DeltaType
	var anchorID *string
	var driftScore float64

	if ft.lastIBkt == nil || ft.bktCount%iFrameInterval == 0 {
		deltaType = nrv.DeltaTypeI
		ft.lastIBkt = b
		ft.lastIID = b.ID
	} else {
		driftScore = euclideanDrift(b.Projections, ft.lastIBkt.Projections)
		if driftScore > driftThreshold {
			deltaType = nrv.DeltaTypeI
			ft.lastIBkt = b
			ft.lastIID = b.ID
		} else {
			deltaType = nrv.DeltaTypeP
			id := ft.lastIID
			anchorID = &id
			b.Projections = nrv.XORProjections(b.Projections, ft.lastIBkt.Projections)
		}
	}

	ft.pending = append(ft.pending, PendingBracket{
		Bracket:    b,
		DeltaType:  deltaType,
		AnchorID:   anchorID,
		DriftScore: driftScore,
	})
	ft.thermoSamples = append(ft.thermoSamples, thermo)
	ft.bktCount++

	return nil
}

func (ft *FrameTicker) SetLinguistic(token, unit string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.linguistic = nrv.LinguisticMapping{Token: token, Unit: unit}
}

func (ft *FrameTicker) Stop() {
	select {
	case <-ft.stopCh:
		return
	default:
		close(ft.stopCh)
	}
	ft.wg.Wait()
}

func (ft *FrameTicker) run() {
	defer ft.wg.Done()
	for {
		select {
		case <-ft.ticker.C:
			ft.flush()
		case <-ft.stopCh:
			ft.ticker.Stop()
			ft.flush()
			return
		}
	}
}

func (ft *FrameTicker) flush() {
	ft.mu.Lock()
	pending := ft.pending
	thermo := ft.thermoSamples
	ling := ft.linguistic
	ft.pending = nil
	ft.thermoSamples = nil
	ft.mu.Unlock()

	if len(pending) == 0 {
		return
	}

	frameID := uuid.New().String()
	atmosphere := aggregateThermo(thermo)
	bracketMetas := make([]nrv.BracketMeta, len(pending))
	for i, pb := range pending {
		bracketMetas[i] = nrv.BracketMeta{
			ID:         pb.Bracket.ID,
			Type:       pb.DeltaType,
			AnchorID:   pb.AnchorID,
			Offset:     i * nrv.BracketSize,
			DriftScore: pb.DriftScore,
		}
	}

	buf := make([]byte, len(pending)*nrv.BracketSize)
	for i, pb := range pending {
		encoded := nrv.EncodeBracket(pb.Bracket)
		copy(buf[i*nrv.BracketSize:], encoded[:])
	}

	_ = ft.writer.AppendFrame(frameID, buf, bracketMetas, atmosphere, ling)
}

func euclideanDrift(a, b [64]byte) float64 {
	var sum float64
	for i := 0; i < 16; i++ {
		av := math.Float32frombits(uint32(a[i*4]) | uint32(a[i*4+1])<<8 | uint32(a[i*4+2])<<16 | uint32(a[i*4+3])<<24)
		bv := math.Float32frombits(uint32(b[i*4]) | uint32(b[i*4+1])<<8 | uint32(b[i*4+2])<<16 | uint32(b[i*4+3])<<24)
		diff := float64(av) - float64(bv)
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

func aggregateThermo(samples []nrv.ThermoAtmosphere) nrv.ThermoAtmosphere {
	if len(samples) == 0 {
		return nrv.ThermoAtmosphere{}
	}

	var sumTemp, sumVolt, sumClock float32
	for _, s := range samples {
		sumTemp += s.AvgTempC
		sumVolt += s.PeakVoltV
		sumClock += s.ClockMHz
	}
	count := float32(len(samples))

	return nrv.ThermoAtmosphere{
		AvgTempC:  sumTemp / count,
		PeakVoltV: sumVolt / count,
		ClockMHz:  sumClock / count,
	}
}
