package cognitiveengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ScalableNode defines the interface for nodes that can participate in horizontal scaling.
// Implement this interface to integrate custom node types with the DistributedScaler.
type ScalableNode interface {
	// NodeID returns the unique identifier for this node
	NodeID() string

	// NodeAddress returns the network address for this node
	NodeAddress() string

	// CurrentLoad returns the current load metric (0.0 - 1.0)
	CurrentLoad() float64

	// TasksProcessed returns the total number of tasks processed by this node
	TasksProcessed() int64

	// Start begins the node's processing loop
	Start(ctx context.Context) error

	// Stop gracefully stops the node
	Stop() error

	// ScaleUp increments the node's capacity
	ScaleUp(count int) error

	// ScaleDown decrements the node's capacity
	ScaleDown(count int) error

	// IsHealthy returns whether the node is operating normally
	IsHealthy() bool

	// GetMetrics returns current resource metrics for this node
	GetMetrics() NodeMetrics
}

// DefaultScalableNode provides a reference implementation of ScalableNode
// that can be embedded or extended by custom implementations.
type DefaultScalableNode struct {
	nodeID         string
	address        string
	mu             sync.RWMutex
	currentLoad    float64
	tasksProcessed int64
	healthy        bool
	startedAt      time.Time
	goroutineCount int
	memoryUsageMB  int64
	eventBus       *EventBus
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewDefaultScalableNode creates a new default scalable node
func NewDefaultScalableNode(nodeID, address string, eventBus *EventBus) *DefaultScalableNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &DefaultScalableNode{
		nodeID:         nodeID,
		address:        address,
		healthy:        true,
		startedAt:      time.Now(),
		goroutineCount: runtime.NumGoroutine(),
		eventBus:       eventBus,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (n *DefaultScalableNode) NodeID() string {
	return n.nodeID
}

func (n *DefaultScalableNode) NodeAddress() string {
	return n.address
}

func (n *DefaultScalableNode) CurrentLoad() float64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.currentLoad
}

func (n *DefaultScalableNode) TasksProcessed() int64 {
	return atomic.LoadInt64(&n.tasksProcessed)
}

func (n *DefaultScalableNode) Start(ctx context.Context) error {
	n.mu.Lock()
	n.healthy = true
	n.startedAt = time.Now()
	n.mu.Unlock()

	go n.metricsCollectorLoop()
	return nil
}

func (n *DefaultScalableNode) Stop() error {
	n.cancel()
	n.mu.Lock()
	n.healthy = false
	n.mu.Unlock()
	return nil
}

func (n *DefaultScalableNode) ScaleUp(count int) error {
	if count <= 0 {
		return fmt.Errorf("ScaleUp count must be positive, got %d", count)
	}
	log.Printf("DefaultScalableNode[%s]: scaling up by %d", n.nodeID, count)
	n.mu.Lock()
	n.goroutineCount += count
	n.mu.Unlock()
	return nil
}

func (n *DefaultScalableNode) ScaleDown(count int) error {
	if count <= 0 {
		return fmt.Errorf("ScaleDown count must be positive, got %d", count)
	}
	n.mu.Lock()
	newCount := n.goroutineCount - count
	if newCount < 1 {
		newCount = 1
	}
	n.goroutineCount = newCount
	n.mu.Unlock()
	log.Printf("DefaultScalableNode[%s]: scaling down by %d", n.nodeID, count)
	return nil
}

func (n *DefaultScalableNode) IsHealthy() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.healthy
}

func (n *DefaultScalableNode) GetMetrics() NodeMetrics {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return NodeMetrics{
		NodeID:            n.nodeID,
		TasksProcessed:    atomic.LoadInt64(&n.tasksProcessed),
		SuccessRate:       0.0,
		AvgProcessingTime: 0.0,
		ReliabilityScore:  0.0,
		Specializations:   []string{"default_scalable_node"},
		LastActive:        time.Now(),
	}
}

func (n *DefaultScalableNode) metricsCollectorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.updateMetrics()
		}
	}
}

func (n *DefaultScalableNode) updateMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	n.mu.Lock()
	n.goroutineCount = runtime.NumGoroutine()
	n.memoryUsageMB = int64(m.HeapAlloc / (1024 * 1024))
	n.currentLoad = float64(runtime.NumGoroutine()) / float64(runtime.GOMAXPROCS(0)*10)
	if n.currentLoad > 1.0 {
		n.currentLoad = 1.0
	}
	n.mu.Unlock()

	if n.eventBus != nil {
		n.eventBus.Publish(EngineEvent{
			Type:      EventNodeMetricsUpdated,
			Source:    n.nodeID,
			Payload:   n.GetMetrics(),
			Timestamp: time.Now(),
		})
	}
}

// IncrementTasksProcessed atomically increments the task counter
func (n *DefaultScalableNode) IncrementTasksProcessed() {
	atomic.AddInt64(&n.tasksProcessed, 1)
}

// GetGoroutineCount returns the current goroutine count
func (n *DefaultScalableNode) GetGoroutineCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.goroutineCount
}

// GetMemoryUsageMB returns the current memory usage in MB
func (n *DefaultScalableNode) GetMemoryUsageMB() int64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.memoryUsageMB
}

type MessageQueue interface {
	Publish(ctx context.Context, topic string, message interface{}) error
	Subscribe(ctx context.Context, topic string, handler func(interface{})) error
	Close() error
}

type ScalingEvent struct {
	Type         string
	SourceNodeID string
	TargetCount  int
	Timestamp    time.Time
	Metadata     map[string]interface{}
}

type NodeInfo struct {
	NodeID         string
	Address        string
	IsLeader       bool
	CurrentLoad    float64
	TasksProcessed int64
	StartedAt      time.Time
}

type DistributedScaler struct {
	nodeID            string
	isLeader          bool
	nodes             map[string]*NodeInfo
	messageQueue      MessageQueue
	scalingConfig     *ScalingConfig
	eventBus          *EventBus
	scalableNode      ScalableNode
	scalingStrategy   ScalingStrategy
	metricsCollector  *ScalingMetricsCollector
	lastScaleUpTime   time.Time
	lastScaleDownTime time.Time
	currentReplicas   int32
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// ScalingStrategy defines the interface for custom scaling algorithms
type ScalingStrategy interface {
	// CalculateDesiredReplicas computes the target replica count based on metrics
	CalculateDesiredReplicas(ctx context.Context, currentReplicas int, metrics *ScalingMetrics) int

	// ShouldScale determines if scaling should occur
	ShouldScale(currentMetrics, targetMetrics *ScalingMetrics) (bool, string)
}

// DefaultScalingStrategy implements a simple threshold-based scaling strategy
type DefaultScalingStrategy struct {
	scaleUpThreshold   float64
	scaleDownThreshold float64
	minReplicas        int
	maxReplicas        int
	cooldownUp         time.Duration
	cooldownDown       time.Duration
}

// NewDefaultScalingStrategy creates a new default scaling strategy
func NewDefaultScalingStrategy(minReplicas, maxReplicas int, scaleUpThresh, scaleDownThresh float64) *DefaultScalingStrategy {
	return &DefaultScalingStrategy{
		scaleUpThreshold:   scaleUpThresh,
		scaleDownThreshold: scaleDownThresh,
		minReplicas:        minReplicas,
		maxReplicas:        maxReplicas,
		cooldownUp:         5 * time.Minute,
		cooldownDown:       15 * time.Minute,
	}
}

func (s *DefaultScalingStrategy) CalculateDesiredReplicas(_ context.Context, currentReplicas int, metrics *ScalingMetrics) int {
	if metrics.AvgCPUUtilization > s.scaleUpThreshold {
		return min(currentReplicas+1, s.maxReplicas)
	}
	if metrics.AvgCPUUtilization < s.scaleDownThreshold && currentReplicas > s.minReplicas {
		return max(currentReplicas-1, s.minReplicas)
	}
	return currentReplicas
}

func (s *DefaultScalingStrategy) ShouldScale(currentMetrics, targetMetrics *ScalingMetrics) (bool, string) {
	if targetMetrics.AvgCPUUtilization > s.scaleUpThreshold {
		return true, "up"
	}
	if targetMetrics.AvgCPUUtilization < s.scaleDownThreshold {
		return true, "down"
	}
	return false, "none"
}

// ScalingMetricsCollector tracks scaling-related metrics over time
type ScalingMetricsCollector struct {
	metricsHistory []ScaledMetricSnapshot
	mu             sync.RWMutex
	maxHistorySize int
}

type ScaledMetricSnapshot struct {
	Timestamp         time.Time
	Replicas          int
	AvgCPUUtilization float64
	AvgMemoryUsage    float64
	TasksProcessed    int64
	QueueDepth        int
}

func NewScalingMetricsCollector(maxHistory int) *ScalingMetricsCollector {
	return &ScalingMetricsCollector{
		metricsHistory: make([]ScaledMetricSnapshot, 0, maxHistory),
		maxHistorySize: maxHistory,
	}
}

func (mc *ScalingMetricsCollector) Record(snapshot ScaledMetricSnapshot) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metricsHistory = append(mc.metricsHistory, snapshot)
	if len(mc.metricsHistory) > mc.maxHistorySize {
		mc.metricsHistory = mc.metricsHistory[1:]
	}
}

func (mc *ScalingMetricsCollector) GetRecent(count int) []ScaledMetricSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if count > len(mc.metricsHistory) {
		count = len(mc.metricsHistory)
	}
	result := make([]ScaledMetricSnapshot, count)
	copy(result, mc.metricsHistory[len(mc.metricsHistory)-count:])
	return result
}

func (mc *ScalingMetricsCollector) GetAverageCPU() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.metricsHistory) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, m := range mc.metricsHistory {
		sum += m.AvgCPUUtilization
	}
	return sum / float64(len(mc.metricsHistory))
}

// ScalingMetrics holds aggregated metrics for scaling decisions
type ScalingMetrics struct {
	AvgCPUUtilization float64
	AvgMemoryUsage    float64
	TotalTasks        int64
	QueueDepth        int
	ActiveConnections int
	Nodes             map[string]*NodeInfo
}

type ScalingConfig struct {
	MinReplicas        int
	MaxReplicas        int
	ScaleUpThreshold   float64
	ScaleDownThreshold float64
	ScaleUpCooldown    time.Duration
	ScaleDownCooldown  time.Duration
	LeaderElectionTTL  time.Duration
}

func NewDistributedScaler(nodeID string, mq MessageQueue, cfg *ScalingConfig) *DistributedScaler {
	if cfg == nil {
		cfg = &ScalingConfig{
			MinReplicas:        1,
			MaxReplicas:        10,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.2,
			ScaleUpCooldown:    5 * time.Minute,
			ScaleDownCooldown:  15 * time.Minute,
			LeaderElectionTTL:  30 * time.Second,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	ds := &DistributedScaler{
		nodeID:           nodeID,
		nodes:            make(map[string]*NodeInfo),
		messageQueue:     mq,
		scalingConfig:    cfg,
		scalingStrategy:  NewDefaultScalingStrategy(cfg.MinReplicas, cfg.MaxReplicas, cfg.ScaleUpThreshold, cfg.ScaleDownThreshold),
		metricsCollector: NewScalingMetricsCollector(100),
		currentReplicas:  int32(cfg.MinReplicas),
		ctx:              ctx,
		cancel:           cancel,
	}

	return ds
}

// SetEventBus wires an EventBus into the DistributedScaler for publishing scaling events
func (ds *DistributedScaler) SetEventBus(bus *EventBus) {
	ds.eventBus = bus
}

// SetScalableNode sets a ScalableNode implementation for direct node management
func (ds *DistributedScaler) SetScalableNode(node ScalableNode) {
	ds.scalableNode = node
}

// SetScalingStrategy sets a custom scaling strategy algorithm
func (ds *DistributedScaler) SetScalingStrategy(strategy ScalingStrategy) {
	ds.scalingStrategy = strategy
}

// RegisterNode registers a new node with the distributed scaler
func (ds *DistributedScaler) RegisterNode(nodeID, address string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if _, exists := ds.nodes[nodeID]; exists {
		return fmt.Errorf("node %s already registered", nodeID)
	}

	ds.nodes[nodeID] = &NodeInfo{
		NodeID:         nodeID,
		Address:        address,
		IsLeader:       false,
		CurrentLoad:    0.0,
		TasksProcessed: 0,
		StartedAt:      time.Now(),
	}

	log.Printf("DistributedScaler: registered node %s at %s", nodeID, address)
	return nil
}

// UnregisterNode removes a node from the distributed scaler
func (ds *DistributedScaler) UnregisterNode(nodeID string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if _, exists := ds.nodes[nodeID]; !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	delete(ds.nodes, nodeID)
	log.Printf("DistributedScaler: unregistered node %s", nodeID)
	return nil
}

func (ds *DistributedScaler) Start() error {
	ds.mu.Lock()
	ds.nodes[ds.nodeID] = &NodeInfo{
		NodeID:         ds.nodeID,
		Address:        ds.getNodeAddress(),
		IsLeader:       true,
		CurrentLoad:    0.0,
		TasksProcessed: 0,
		StartedAt:      time.Now(),
	}
	ds.isLeader = true
	ds.mu.Unlock()

	if ds.scalableNode != nil {
		if err := ds.scalableNode.Start(ds.ctx); err != nil {
			log.Printf("DistributedScaler: failed to start scalable node: %v", err)
		}
	}

	if ds.messageQueue != nil {
		if err := ds.messageQueue.Subscribe(ds.ctx, "cognitive:scaling", ds.handleScalingEvent); err != nil {
			log.Printf("DistributedScaler: failed to subscribe to scaling events: %v", err)
		}
		if err := ds.messageQueue.Subscribe(ds.ctx, "cognitive:node_heartbeat", ds.handleHeartbeat); err != nil {
			log.Printf("DistributedScaler: failed to subscribe to heartbeats: %v", err)
		}
	}

	go ds.leaderElectionLoop()
	go ds.healthCheckLoop()
	go ds.scalingEvaluationLoop()
	go ds.metricsAggregationLoop()

	log.Printf("DistributedScaler[%s]: started as leader=%v", ds.nodeID, ds.isLeader)
	return nil
}

func (ds *DistributedScaler) Stop() error {
	ds.cancel()

	if ds.messageQueue != nil {
		return ds.messageQueue.Close()
	}
	return nil
}

func (ds *DistributedScaler) handleScalingEvent(msg interface{}) {
	data, ok := msg.([]byte)
	if !ok {
		return
	}

	var event ScalingEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("DistributedScaler: failed to unmarshal scaling event: %v", err)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	switch event.Type {
	case "scale_request":
		ds.processScaleRequest(&event)
	case "scale_complete":
		ds.handleScaleComplete(&event)
	}
}

func (ds *DistributedScaler) handleHeartbeat(msg interface{}) {
	data, ok := msg.([]byte)
	if !ok {
		return
	}

	var node NodeInfo
	if err := json.Unmarshal(data, &node); err != nil {
		return
	}

	ds.mu.Lock()
	existing, exists := ds.nodes[node.NodeID]
	if exists {
		existing.CurrentLoad = node.CurrentLoad
		existing.TasksProcessed = node.TasksProcessed
	} else {
		ds.nodes[node.NodeID] = &node
	}
	ds.mu.Unlock()
}

func (ds *DistributedScaler) processScaleRequest(event *ScalingEvent) {
	if !ds.isLeader {
		return
	}

	targetCount := event.TargetCount
	if targetCount < ds.scalingConfig.MinReplicas {
		targetCount = ds.scalingConfig.MinReplicas
	}
	if targetCount > ds.scalingConfig.MaxReplicas {
		targetCount = ds.scalingConfig.MaxReplicas
	}

	log.Printf("DistributedScaler: processing scale request to %d replicas", targetCount)

	completeEvent := &ScalingEvent{
		Type:        "scale_complete",
		TargetCount: targetCount,
		Timestamp:   time.Now(),
	}
	ds.publishEvent(completeEvent)
}

func (ds *DistributedScaler) handleScaleComplete(event *ScalingEvent) {
	log.Printf("DistributedScaler: scale complete, target %d replicas", event.TargetCount)
}

func (ds *DistributedScaler) leaderElectionLoop() {
	ticker := time.NewTicker(ds.scalingConfig.LeaderElectionTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.mu.RLock()
			currentLeader := ds.findLeader()
			ds.mu.RUnlock()

			if currentLeader != ds.nodeID {
				ds.mu.Lock()
				ds.isLeader = false
				if node, ok := ds.nodes[ds.nodeID]; ok {
					node.IsLeader = false
				}
				ds.mu.Unlock()
			}
		}
	}
}

func (ds *DistributedScaler) findLeader() string {
	for _, node := range ds.nodes {
		if node.IsLeader {
			return node.NodeID
		}
	}
	return ""
}

func (ds *DistributedScaler) healthCheckLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.mu.Lock()
			now := time.Now()
			for nodeID, node := range ds.nodes {
				if nodeID != ds.nodeID && now.Sub(node.StartedAt) > 30*time.Second {
					delete(ds.nodes, nodeID)
					log.Printf("DistributedScaler: removed unresponsive node %s", nodeID)
				}
			}
			ds.mu.Unlock()

			ds.sendHeartbeat()
		}
	}
}

func (ds *DistributedScaler) sendHeartbeat() {
	node := &NodeInfo{
		NodeID:         ds.nodeID,
		Address:        ds.getNodeAddress(),
		IsLeader:       ds.isLeader,
		CurrentLoad:    ds.getCurrentLoad(),
		TasksProcessed: ds.getTasksProcessed(),
		StartedAt:      time.Now(),
	}

	data, _ := json.Marshal(node)
	ds.publishEventRaw("cognitive:node_heartbeat", data)
}

func (ds *DistributedScaler) publishEvent(event *ScalingEvent) {
	data, _ := json.Marshal(event)
	ds.publishEventRaw("cognitive:scaling", data)
}

func (ds *DistributedScaler) publishEventRaw(topic string, data []byte) {
	if ds.messageQueue != nil {
		_ = ds.messageQueue.Publish(ds.ctx, topic, data)
	}
}

func (ds *DistributedScaler) RequestScale(targetCount int) error {
	if !ds.isLeader {
		return ds.publishScaleRequest(ds.nodeID, targetCount)
	}

	ds.mu.Lock()
	event := &ScalingEvent{
		Type:         "scale_request",
		SourceNodeID: ds.nodeID,
		TargetCount:  targetCount,
		Timestamp:    time.Now(),
	}
	ds.mu.Unlock()

	ds.processScaleRequest(event)
	return nil
}

func (ds *DistributedScaler) publishScaleRequest(sourceID string, targetCount int) error {
	event := &ScalingEvent{
		Type:         "scale_request",
		SourceNodeID: sourceID,
		TargetCount:  targetCount,
		Timestamp:    time.Now(),
	}

	data, _ := json.Marshal(event)
	return ds.messageQueue.Publish(ds.ctx, "cognitive:scaling", data)
}

func (ds *DistributedScaler) EvaluateScalingNeeds(currentLoad float64) (shouldScale bool, direction string, targetCount int) {
	ds.mu.RLock()
	nodeCount := len(ds.nodes)
	ds.mu.RUnlock()

	if currentLoad > ds.scalingConfig.ScaleUpThreshold && nodeCount < ds.scalingConfig.MaxReplicas {
		return true, "up", nodeCount + 1
	}

	if currentLoad < ds.scalingConfig.ScaleDownThreshold && nodeCount > ds.scalingConfig.MinReplicas {
		return true, "down", nodeCount - 1
	}

	return false, "none", nodeCount
}

func (ds *DistributedScaler) GetNodeCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return len(ds.nodes)
}

func (ds *DistributedScaler) IsLeader() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.isLeader
}

func (ds *DistributedScaler) GetNodes() map[string]*NodeInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	result := make(map[string]*NodeInfo, len(ds.nodes))
	for k, v := range ds.nodes {
		result[k] = v
	}
	return result
}

func (ds *DistributedScaler) getNodeAddress() string {
	return "localhost"
}

func (ds *DistributedScaler) getCurrentLoad() float64 {
	return 0.0
}

func (ds *DistributedScaler) getTasksProcessed() int64 {
	return 0
}

// scalingEvaluationLoop periodically evaluates scaling needs and triggers scaling actions
func (ds *DistributedScaler) scalingEvaluationLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.evaluateAndTriggerScaling()
		}
	}
}

// evaluateAndTriggerScaling performs scaling evaluation and triggers actions if needed
func (ds *DistributedScaler) evaluateAndTriggerScaling() {
	if !ds.isLeader {
		return
	}

	metrics := ds.aggregateMetrics()
	desiredReplicas := ds.scalingStrategy.CalculateDesiredReplicas(ds.ctx, int(ds.currentReplicas), metrics)

	if desiredReplicas != int(ds.currentReplicas) {
		direction := "up"
		if desiredReplicas < int(ds.currentReplicas) {
			direction = "down"
		}

		ds.publishScalingEvent(ScalingEvent{
			Type:         "auto_scale_decision",
			SourceNodeID: ds.nodeID,
			TargetCount:  desiredReplicas,
			Timestamp:    time.Now(),
			Metadata: map[string]interface{}{
				"direction":        direction,
				"current_replicas": ds.currentReplicas,
				"avg_cpu":          metrics.AvgCPUUtilization,
			},
		})

		if err := ds.executeScaling(desiredReplicas); err != nil {
			log.Printf("DistributedScaler: scaling execution failed: %v", err)
		}
	}
}

// aggregateMetrics collects and aggregates metrics from all nodes
func (ds *DistributedScaler) aggregateMetrics() *ScalingMetrics {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var totalLoad float64
	var totalTasks int64
	nodeCount := len(ds.nodes)

	for _, node := range ds.nodes {
		totalLoad += node.CurrentLoad
		totalTasks += node.TasksProcessed
	}

	avgLoad := 0.0
	if nodeCount > 0 {
		avgLoad = totalLoad / float64(nodeCount)
	}

	return &ScalingMetrics{
		AvgCPUUtilization: avgLoad,
		TotalTasks:        totalTasks,
		Nodes:             ds.nodes,
	}
}

// executeScaling performs the actual scaling action
func (ds *DistributedScaler) executeScaling(targetReplicas int) error {
	if targetReplicas < ds.scalingConfig.MinReplicas {
		targetReplicas = ds.scalingConfig.MinReplicas
	}
	if targetReplicas > ds.scalingConfig.MaxReplicas {
		targetReplicas = ds.scalingConfig.MaxReplicas
	}

	delta := targetReplicas - int(ds.currentReplicas)
	if delta == 0 {
		return nil
	}

	if delta > 0 {
		if ds.scalableNode != nil {
			if err := ds.scalableNode.ScaleUp(delta); err != nil {
				return fmt.Errorf("scale up failed: %w", err)
			}
		}
		ds.lastScaleUpTime = time.Now()
	} else {
		if ds.scalableNode != nil {
			if err := ds.scalableNode.ScaleDown(-delta); err != nil {
				return fmt.Errorf("scale down failed: %w", err)
			}
		}
		ds.lastScaleDownTime = time.Now()
	}

	atomic.StoreInt32(&ds.currentReplicas, int32(targetReplicas))

	ds.publishScalingEvent(ScalingEvent{
		Type:         "scale_complete",
		SourceNodeID: ds.nodeID,
		TargetCount:  targetReplicas,
		Timestamp:    time.Now(),
		Metadata: map[string]interface{}{
			"delta": delta,
		},
	})

	log.Printf("DistributedScaler: scaled to %d replicas (delta=%d)", targetReplicas, delta)
	return nil
}

// publishScalingEvent publishes a scaling event to the event bus and message queue
func (ds *DistributedScaler) publishScalingEvent(event ScalingEvent) {
	if ds.eventBus != nil {
		ds.eventBus.Publish(EngineEvent{
			Type:      EventScalingAction,
			Source:    ds.nodeID,
			Payload:   event,
			Timestamp: time.Now(),
		})
	}

	data, _ := json.Marshal(event)
	ds.publishEventRaw("cognitive:scaling", data)
}

// metricsAggregationLoop periodically collects and stores scaling metrics
func (ds *DistributedScaler) metricsAggregationLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.recordMetricsSnapshot()
		}
	}
}

// recordMetricsSnapshot captures current metrics for historical analysis
func (ds *DistributedScaler) recordMetricsSnapshot() {
	metrics := ds.aggregateMetrics()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := ScaledMetricSnapshot{
		Timestamp:         time.Now(),
		Replicas:          int(ds.currentReplicas),
		AvgCPUUtilization: metrics.AvgCPUUtilization,
		AvgMemoryUsage:    float64(m.HeapAlloc) / float64(1024*1024),
		TasksProcessed:    metrics.TotalTasks,
		QueueDepth:        0,
	}

	ds.metricsCollector.Record(snapshot)
}

// GetScalingMetrics returns aggregated scaling metrics
func (ds *DistributedScaler) GetScalingMetrics() *ScalingMetrics {
	return ds.aggregateMetrics()
}

// GetMetricsHistory returns recent scaling metric snapshots
func (ds *DistributedScaler) GetMetricsHistory(count int) []ScaledMetricSnapshot {
	return ds.metricsCollector.GetRecent(count)
}

// GetAverageCPU returns the average CPU utilization over the metric history
func (ds *DistributedScaler) GetAverageCPU() float64 {
	return ds.metricsCollector.GetAverageCPU()
}

// GetCurrentReplicas returns the current replica count
func (ds *DistributedScaler) GetCurrentReplicas() int {
	return int(atomic.LoadInt32(&ds.currentReplicas))
}

// IsInCooldown returns whether scaling is in cooldown period
func (ds *DistributedScaler) IsInCooldown(direction string) bool {
	if direction == "up" {
		return time.Since(ds.lastScaleUpTime) < ds.scalingConfig.ScaleUpCooldown
	}
	return time.Since(ds.lastScaleDownTime) < ds.scalingConfig.ScaleDownCooldown
}
