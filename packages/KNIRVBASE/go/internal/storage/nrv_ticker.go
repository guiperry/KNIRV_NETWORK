package storage

import (
	"context"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

const (
	iFrameInterval = 50
	driftThreshold = 0.25
)

type MemoryZone struct {
	HistoryXOR [3]uint32
}

type PendingBracket struct {
	Bracket    *nrv.Bracket
	DeltaType  nrv.DeltaType
	AnchorID   *string
	DriftScore float64
	HistoryXOR [3]uint32
}

type FrameTicker struct {
	writer   *NRVWriter
	interval time.Duration
	mu       sync.Mutex
	pending  []PendingBracket
	lastIBkt *nrv.Bracket
	lastIID  string
	bktCount int

	memoryZone MemoryZone

	thermoSamples []nrv.ThermoAtmosphere
	linguistic    nrv.LinguisticMapping

	flushMu  sync.Mutex
	lastErr  error
	ticker   *time.Ticker
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func cloneBracketShallow(b *nrv.Bracket) *nrv.Bracket {
	if b == nil {
		return nil
	}
	c := *b
	c.Meta = nil
	return &c
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

// LastFlushError returns the most recent error from flushing to the writer, if any.
func (ft *FrameTicker) LastFlushError() error {
	ft.flushMu.Lock()
	defer ft.flushMu.Unlock()
	return ft.lastErr
}

func (ft *FrameTicker) AppendBracket(ctx context.Context, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if b.ID == "" {
		b.ID = uuid.New().String()
	}

	var deltaType nrv.DeltaType
	var anchorID *string
	var driftScore float64

	historyXOR := ft.computeMemoryXOR(b)

	if ft.lastIBkt == nil || ft.bktCount%iFrameInterval == 0 {
		deltaType = nrv.DeltaTypeI
		ft.lastIBkt = cloneBracketShallow(b)
		ft.lastIID = b.ID
	} else {
		driftScore = euclideanDrift(b.Projections, ft.lastIBkt.Projections)
		if driftScore > driftThreshold {
			deltaType = nrv.DeltaTypeI
			ft.lastIBkt = cloneBracketShallow(b)
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
		HistoryXOR: historyXOR,
	})
	ft.thermoSamples = append(ft.thermoSamples, thermo)
	ft.bktCount++

	return nil
}

func (ft *FrameTicker) computeMemoryXOR(b *nrv.Bracket) [3]uint32 {
	slot6 := ft.memoryZone.HistoryXOR[0] ^ hashBracket(b)
	slot7 := ft.memoryZone.HistoryXOR[1] ^ slot6
	slot8 := ft.memoryZone.HistoryXOR[2] ^ slot7

	ft.memoryZone.HistoryXOR[0] = slot6
	ft.memoryZone.HistoryXOR[1] = slot7
	ft.memoryZone.HistoryXOR[2] = slot8

	return [3]uint32{slot6, slot7, slot8}
}

func hashBracket(b *nrv.Bracket) uint32 {
	h := b.GoldenSeed
	for i := 0; i < 32; i += 4 {
		h ^= binary.LittleEndian.Uint32(b.Projections[i : i+4])
	}
	h ^= b.SubSecondUS
	return h
}

func (ft *FrameTicker) SetLinguistic(token, unit string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.linguistic = nrv.LinguisticMapping{Token: token, Unit: unit}
}

func (ft *FrameTicker) Stop() {
	ft.stopOnce.Do(func() { close(ft.stopCh) })
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

func writeHistoryToMemory(dst *[14]byte, hx [3]uint32) {
	binary.LittleEndian.PutUint32(dst[0:4], hx[0])
	binary.LittleEndian.PutUint32(dst[4:8], hx[1])
	binary.LittleEndian.PutUint32(dst[8:12], hx[2])
	for i := 12; i < 14; i++ {
		dst[i] = 0
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
		writeHistoryToMemory(&pb.Bracket.Memory, pb.HistoryXOR)
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

	err := ft.writer.AppendFrame(frameID, buf, bracketMetas, atmosphere, ling)
	ft.flushMu.Lock()
	ft.lastErr = err
	ft.flushMu.Unlock()
}

func euclideanDrift(a, b [32]byte) float64 {
	var sum float64
	for i := 0; i < 16; i++ {
		av := float64(uint16(a[i*2])|uint16(a[i*2+1])<<8) / 65535.0
		bv := float64(uint16(b[i*2])|uint16(b[i*2+1])<<8) / 65535.0
		diff := av - bv
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
