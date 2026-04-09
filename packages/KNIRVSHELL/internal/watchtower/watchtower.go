package watchtower

import (
	"bufio"
	"context"
	"sync"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/incident"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/runner"
)

const (
	incidentCooldown = 5 * time.Second
)

type UILogLine struct {
	AgentID string
	Line    string
	IsError bool
}

type AgentResult struct {
	AgentID  string
	ExitCode int
	Err      error
}

type Watchtower struct {
	mu         sync.Mutex
	agents     map[string]*watchedAgent
	uiCh       chan<- UILogLine
	incidentCh chan<- *incident.Incident
	sigs       *SignatureSet
}

type watchedAgent struct {
	proc         *runner.AgentProcess
	ring         *RingBuffer
	cancel       context.CancelFunc
	lastIncident time.Time
	incMu        sync.Mutex
}

func New(uiCh chan<- UILogLine, incCh chan<- *incident.Incident) *Watchtower {
	return &Watchtower{
		agents:     make(map[string]*watchedAgent),
		uiCh:       uiCh,
		incidentCh: incCh,
		sigs:       DefaultSignatures(),
	}
}

func (w *Watchtower) Watch(ctx context.Context, proc *runner.AgentProcess) <-chan AgentResult {
	ctx, cancel := context.WithCancel(ctx)
	wa := &watchedAgent{proc: proc, ring: NewRingBuffer(500), cancel: cancel}
	resultCh := make(chan AgentResult, 1)

	w.mu.Lock()
	w.agents[proc.ID] = wa
	w.mu.Unlock()

	go w.supervise(ctx, wa, resultCh)
	return resultCh
}

func (w *Watchtower) supervise(ctx context.Context, wa *watchedAgent, resultCh chan<- AgentResult) {
	defer func() {
		wa.cancel()
		w.mu.Lock()
		delete(w.agents, wa.proc.ID)
		w.mu.Unlock()
		close(resultCh)
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(wa.proc.Stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case w.uiCh <- UILogLine{AgentID: wa.proc.ID, Line: scanner.Text(), IsError: true}:
			}
		}
	}()

	scanner := bufio.NewScanner(wa.proc.Stdout)
	ctxCancelled := false

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ctxCancelled = true
			break
		default:
		}
		if ctxCancelled {
			break
		}

		line := scanner.Text()
		wa.ring.Push(line)

		isErr, sigName := w.sigs.Match(line)
		select {
		case w.uiCh <- UILogLine{AgentID: wa.proc.ID, Line: line, IsError: isErr}:
		default:
		}

		if isErr {
			w.handleIncident(wa, line, sigName, 0)
		}
	}

	wg.Wait()

	exitCode, waitErr := wa.proc.WaitFn()
	if exitCode != 0 {
		w.handleIncident(wa, "non-zero exit: process terminated", "EXIT_NON_ZERO", exitCode)
	}
	resultCh <- AgentResult{AgentID: wa.proc.ID, ExitCode: exitCode, Err: waitErr}
}

func (w *Watchtower) handleIncident(wa *watchedAgent, triggerLine, sigName string, exitCode int) {
	wa.incMu.Lock()
	defer wa.incMu.Unlock()

	if time.Since(wa.lastIncident) < incidentCooldown {
		return
	}
	wa.lastIncident = time.Now()

	inc := incident.New(wa.proc, wa.ring.Snapshot(), triggerLine, sigName, exitCode)
	path, err := inc.WriteToVault()
	if err == nil {
		incident.OpenInObsidian(path)
	}
	select {
	case w.incidentCh <- inc:
	default:
	}
}
