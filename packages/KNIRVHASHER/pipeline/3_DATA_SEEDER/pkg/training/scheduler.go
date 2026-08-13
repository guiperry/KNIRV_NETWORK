package training

import (
	"time"
)

type SchedulerConfig struct {
	OnDemand      bool
	Scheduled     bool
	ScheduledCron string
	OnViolation   bool
}

type TrainingScheduler struct {
	onDemand  chan *UserSecurityGates
	violation chan *GuardrailViolation
	scheduler *CronScheduler
	trainer   *UserTrainer
}

type GuardrailViolation struct {
	NodeID        string
	GuardrailType string
	Message       string
	Severity      string
}

func NewTrainingScheduler(cfg *SchedulerConfig) *TrainingScheduler {
	ts := &TrainingScheduler{
		onDemand:  make(chan *UserSecurityGates, 10),
		violation: make(chan *GuardrailViolation, 100),
	}

	if cfg.Scheduled {
		ts.scheduler = NewCronScheduler(cfg.ScheduledCron)
	}

	return ts
}

func (ts *TrainingScheduler) Start() {
	go ts.handleOnDemand()

	if ts.scheduler != nil {
		go ts.scheduler.Run(ts.runScheduledTraining)
	}

	go ts.handleViolations()
}

func (ts *TrainingScheduler) handleOnDemand() {
	for {
		select {
		case gates := <-ts.onDemand:
			ts.runTraining(gates)
		}
	}
}

func (ts *TrainingScheduler) handleViolations() {
	for {
		select {
		case v := <-ts.violation:
			gates := &UserSecurityGates{
				OrgID:  v.NodeID,
				UserID: v.NodeID,
				Constraints: []SecurityConstraint{
					{
						RuleID:   v.GuardrailType,
						Text:     v.Message,
						Type:     "deny",
						Severity: v.Severity,
					},
				},
				MaxGenerations: 50,
			}
			ts.runTraining(gates)
		}
	}
}

func (ts *TrainingScheduler) runScheduledTraining() {
	gates := &UserSecurityGates{
		OrgID:          "scheduled",
		UserID:         "all",
		MaxGenerations: 100,
	}
	ts.runTraining(gates)
}

func (ts *TrainingScheduler) runTraining(gates *UserSecurityGates) {
	trained, err := ts.trainer.TrainUserGates(nil, gates)
	if err != nil {
		return
	}

	ts.saveTrainedGates(trained)
}

func (ts *TrainingScheduler) saveTrainedGates(gates *TrainedGates) error {
	return nil
}

func (ts *TrainingScheduler) TriggerOnDemand(gates *UserSecurityGates) {
	ts.onDemand <- gates
}

func (ts *TrainingScheduler) TriggerOnViolation(v *GuardrailViolation) {
	ts.violation <- v
}

type CronScheduler struct {
	expression string
	stop       chan struct{}
}

func NewCronScheduler(expr string) *CronScheduler {
	return &CronScheduler{
		expression: expr,
		stop:       make(chan struct{}),
	}
}

func (cs *CronScheduler) Run(fn func()) {
	ticker := newTicker(24 * 60 * 60 * 1000)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stop:
			return
		case <-ticker.C:
			fn()
		}
	}
}

func (cs *CronScheduler) Stop() {
	close(cs.stop)
}

type ticker struct {
	C    chan struct{}
	done chan struct{}
}

func newTicker(ms int) *ticker {
	t := &ticker{
		C:    make(chan struct{}, 1),
		done: make(chan struct{}),
	}
	go func() {
		ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case t.C <- struct{}{}:
				default:
				}
			case <-t.done:
				return
			}
		}
	}()
	return t
}

func (t *ticker) Stop() {
	close(t.done)
}
