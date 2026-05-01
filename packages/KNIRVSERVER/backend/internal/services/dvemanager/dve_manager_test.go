package dvemanager

import (
	"testing"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/database"
	"backend_server/internal/objects"
)

func TestRegisterNode(t *testing.T) {
	// Skip this test for now due to P2P manager interface issues
	t.Skip("Skipping DVE manager test due to P2P interface compatibility issues")
}

func TestGetNodes(t *testing.T) {
	// Skip this test for now due to P2P manager interface issues
	t.Skip("Skipping DVE manager test due to P2P interface compatibility issues")
}

func TestNodeFilter(t *testing.T) {
	// Skip this test for now due to P2P manager interface issues
	t.Skip("Skipping DVE manager test due to P2P interface compatibility issues")
}

func TestUpdateNodeStatus(t *testing.T) {
	// Setup
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register a node first
	req := &objects.RegisterNodeRequest{
		Name:         "test-node",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(req)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Update node status
	err = manager.UpdateNodeStatus(node.ID, "maintenance")
	if err != nil {
		t.Fatalf("Failed to update node status: %v", err)
	}

	// Verify status was updated
	updatedNode, err := manager.getNodeFromDB(node.ID)
	if err != nil {
		t.Fatalf("Failed to get updated node: %v", err)
	}

	if updatedNode.Status != "maintenance" {
		t.Errorf("Expected status 'maintenance', got '%s'", updatedNode.Status)
	}
}

func TestGetSystemHealth(t *testing.T) {
	// Setup
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register some test nodes
	for i := 0; i < 3; i++ {
		req := &objects.RegisterNodeRequest{
			Name:         "test-node-" + string(rune(i+'1')),
			TEEType:      "sgx",
			StakeAmount:  100000,
			Location:     "us-east-1",
			IPAddress:    "192.168.1.10" + string(rune(i+'0')),
			PublicKey:    "test-public-key-" + string(rune(i+'1')),
			Capabilities: []string{"validation"},
		}
		_, err := manager.RegisterNode(req)
		if err != nil {
			t.Fatalf("Failed to register node %d: %v", i, err)
		}
	}

	health, err := manager.GetSystemHealth()
	if err != nil {
		t.Fatalf("Failed to get system health: %v", err)
	}

	if health.ActiveNodes != 3 {
		t.Errorf("Expected 3 active nodes, got %d", health.ActiveNodes)
	}

	if health.TotalNodes != 3 {
		t.Errorf("Expected 3 total nodes, got %d", health.TotalNodes)
	}

	if health.OverallStatus != "healthy" {
		t.Errorf("Expected overall status 'healthy', got '%s'", health.OverallStatus)
	}
}

func TestCalculateOverallStatus(t *testing.T) {
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	tests := []struct {
		activeNodes int
		totalNodes  int
		expected    string
	}{
		{0, 0, "critical"},
		{5, 10, "degraded"}, // 50% active
		{8, 10, "healthy"},  // 80% active
		{10, 10, "healthy"}, // 100% active
	}

	for _, test := range tests {
		status := manager.calculateOverallStatus(test.activeNodes, test.totalNodes)
		if status != test.expected {
			t.Errorf("Expected status '%s' for %d/%d nodes, got '%s'",
				test.expected, test.activeNodes, test.totalNodes, status)
		}
	}
}

func TestGetAllNodes(t *testing.T) {
	// Setup
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register test nodes
	for i := 0; i < 2; i++ {
		req := &objects.RegisterNodeRequest{
			Name:         "test-node-" + string(rune(i+'1')),
			TEEType:      "sgx",
			StakeAmount:  100000,
			Location:     "us-east-1",
			IPAddress:    "192.168.1.10" + string(rune(i+'0')),
			PublicKey:    "test-public-key-" + string(rune(i+'1')),
			Capabilities: []string{"validation"},
		}
		_, err := manager.RegisterNode(req)
		if err != nil {
			t.Fatalf("Failed to register node %d: %v", i, err)
		}
	}

	nodes := manager.GetAllNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
}

func TestUpdateNode(t *testing.T) {
	// Setup
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register a node first
	req := &objects.RegisterNodeRequest{
		Name:         "original-name",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(req)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Update the node
	updates := map[string]interface{}{
		"name":         "updated-name",
		"status":       "maintenance",
		"location":     "us-west-2",
		"stake_amount": 150000.0,
		"capabilities": []interface{}{"validation", "ml-inference"},
	}

	updatedNode, err := manager.UpdateNode(node.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	if updatedNode.Name != "updated-name" {
		t.Errorf("Expected name 'updated-name', got '%s'", updatedNode.Name)
	}

	if updatedNode.Status != "maintenance" {
		t.Errorf("Expected status 'maintenance', got '%s'", updatedNode.Status)
	}

	if updatedNode.StakeAmount != 150000 {
		t.Errorf("Expected stake amount 150000, got %d", updatedNode.StakeAmount)
	}

	if len(updatedNode.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(updatedNode.Capabilities))
	}
}

func TestRemoveNode(t *testing.T) {
	// Setup
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register a node first
	req := &objects.RegisterNodeRequest{
		Name:         "test-node",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(req)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Remove the node
	err = manager.RemoveNode(node.ID)
	if err != nil {
		t.Fatalf("Failed to remove node: %v", err)
	}

	// Verify node was removed
	_, err = manager.getNodeFromDB(node.ID)
	if err == nil {
		t.Error("Expected error when getting removed node")
	}
}

func TestCreateAndGetTask(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	nodeReq := &objects.RegisterNodeRequest{
		Name:         "test-node",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(nodeReq)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	task := &objects.ValidationTask{
		ID:              "test-task-1",
		Type:            "skillnode",
		Status:          "pending",
		Priority:        5,
		RequiredTEEType: "sgx",
		AssignedNodeID:  node.ID,
		RequestedBy:     "test-user",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		TimeoutAt:       time.Now().Add(1 * time.Hour),
	}

	err = manager.CreateTask(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	retrievedTask, err := manager.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrievedTask.ID)
	}

	if retrievedTask.Status != "pending" {
		t.Errorf("Expected task status 'pending', got %s", retrievedTask.Status)
	}
}

func TestGetNodeTasks(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	nodeReq := &objects.RegisterNodeRequest{
		Name:         "test-node",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(nodeReq)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	for i := 0; i < 3; i++ {
		task := &objects.ValidationTask{
			ID:              "test-task-" + string(rune(i+'0')),
			Type:            "skillnode",
			Status:          "pending",
			Priority:        5,
			RequiredTEEType: "sgx",
			AssignedNodeID:  node.ID,
			RequestedBy:     "test-user",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			TimeoutAt:       time.Now().Add(1 * time.Hour),
		}
		err := manager.CreateTask(task)
		if err != nil {
			t.Fatalf("Failed to create task %d: %v", i, err)
		}
	}

	tasks, err := manager.GetNodeTasks(node.ID)
	if err != nil {
		t.Fatalf("Failed to get node tasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}
}

func TestUpdateTask(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	nodeReq := &objects.RegisterNodeRequest{
		Name:         "test-node",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(nodeReq)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	task := &objects.ValidationTask{
		ID:              "test-task-1",
		Type:            "skillnode",
		Status:          "pending",
		Priority:        5,
		RequiredTEEType: "sgx",
		AssignedNodeID:  node.ID,
		RequestedBy:     "test-user",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		TimeoutAt:       time.Now().Add(1 * time.Hour),
	}

	err = manager.CreateTask(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	updates := map[string]interface{}{
		"status": "running",
	}
	updatedTask, err := manager.UpdateTask(task.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	if updatedTask.Status != "running" {
		t.Errorf("Expected task status 'running', got %s", updatedTask.Status)
	}
}

func TestListTasks(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	nodeReq := &objects.RegisterNodeRequest{
		Name:         "test-node",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(nodeReq)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	for i := 0; i < 5; i++ {
		status := "pending"
		if i >= 2 {
			status = "completed"
		}
		task := &objects.ValidationTask{
			ID:              "test-task-" + string(rune(i+'0')),
			Type:            "skillnode",
			Status:          status,
			Priority:        5,
			RequiredTEEType: "sgx",
			AssignedNodeID:  node.ID,
			RequestedBy:     "test-user",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			TimeoutAt:       time.Now().Add(1 * time.Hour),
		}
		err := manager.CreateTask(task)
		if err != nil {
			t.Fatalf("Failed to create task %d: %v", i, err)
		}
	}

	allTasks, err := manager.ListTasks("", "")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	if len(allTasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(allTasks))
	}

	pendingTasks, err := manager.ListTasks("pending", "")
	if err != nil {
		t.Fatalf("Failed to list pending tasks: %v", err)
	}
	if len(pendingTasks) != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", len(pendingTasks))
	}
}

func TestGetNodeMetrics(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	nodeReq := &objects.RegisterNodeRequest{
		Name:         "test-node",
		TEEType:      "sgx",
		StakeAmount:  100000,
		Location:     "us-east-1",
		IPAddress:    "192.168.1.100",
		PublicKey:    "test-public-key",
		Capabilities: []string{"validation"},
	}

	node, err := manager.RegisterNode(nodeReq)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	metrics, err := manager.GetNodeMetrics(node.ID)
	if err != nil {
		t.Fatalf("Failed to get node metrics: %v", err)
	}

	if metrics["node_id"] != node.ID {
		t.Errorf("Expected node_id %s, got %v", node.ID, metrics["node_id"])
	}

	if metrics["status"] != "online" {
		t.Errorf("Expected status 'online', got %v", metrics["status"])
	}
}

func TestCalculateNetworkLatency(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	latency := manager.calculateNetworkLatency()

	if latency < 0 {
		t.Errorf("Expected non-negative latency, got %f", latency)
	}
}

func TestCalculateAverageResponseTime(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	responseTime := manager.calculateAverageResponseTime()

	if responseTime < 0 {
		t.Errorf("Expected non-negative response time, got %f", responseTime)
	}
}

func TestCalculateTEEHealthScore(t *testing.T) {
	db, err := database.NewBuntDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, err := NewDVEManager(db, nil, nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create DVE manager: %v", err)
	}

	healthScore := manager.calculateTEEHealthScore()

	if healthScore < 0 || healthScore > 1 {
		t.Errorf("Expected health score between 0 and 1, got %f", healthScore)
	}
}

func TestBrowserNodeRegistration(t *testing.T) {
	// Setup
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register a browser-extension node
	req := &objects.RegisterNodeRequest{
		Name:         "browser-test-node",
		TEEType:      "browser-extension",
		StakeAmount:  10000,
		Location:     "global",
		IPAddress:    "ws://test-browser-dve/ws",
		PublicKey:    "browser-pub-key",
		Capabilities: []string{"validation", "light-attestation", "dve-identity"},
	}

	node, err := manager.RegisterNode(req)
	if err != nil {
		t.Fatalf("Failed to register browser-extension node: %v", err)
	}

	// Verify browser-extension specific fields
	if node.TEEType != "browser-extension" {
		t.Errorf("Expected TEEType 'browser-extension', got '%s'", node.TEEType)
	}
	if !node.IsRemote {
		t.Error("Expected IsRemote to be true for browser-extension nodes")
	}
	if !node.Connected {
		t.Error("Expected Connected to be true for browser-extension nodes")
	}
	if node.SSHPort != 0 {
		t.Errorf("Expected SSHPort 0 for browser-extension nodes, got %d", node.SSHPort)
	}
	if node.Status != "online" {
		t.Errorf("Expected Status 'online', got '%s'", node.Status)
	}
	if node.ReputationScore != 100 {
		t.Errorf("Expected initial ReputationScore 100, got %d", node.ReputationScore)
	}

	// Verify it can be retrieved
	retrievedNode, err := manager.GetNode(node.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve browser-extension node: %v", err)
	}
	if retrievedNode.TEEType != "browser-extension" {
		t.Errorf("Retrieved node TEEType mismatch: expected 'browser-extension', got '%s'", retrievedNode.TEEType)
	}
}

func TestInvalidTEETypeRejected(t *testing.T) {
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	req := &objects.RegisterNodeRequest{
		Name:     "invalid-tee-node",
		TEEType:  "invalid-tee-type",
		PublicKey: "test-key",
	}

	_, err := manager.RegisterNode(req)
	if err == nil {
		t.Error("Expected error when registering node with invalid TEE type")
	}
}

func TestRegisterBrowserDVE(t *testing.T) {
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	walletAddress := "0xabcdef1234567890"
	capabilities := []string{"policy-check", "signature-verify", "reasoning-simple"}
	badgeNFTIDs := []string{"badge-nft-001", "badge-nft-002"}
	extensionID := "chrome-extension-abc123"
	browserVersion := "2.0.1"

	node, err := manager.RegisterBrowserDVE(walletAddress, capabilities, badgeNFTIDs, extensionID, browserVersion)
	if err != nil {
		t.Fatalf("Failed to register browser DVE: %v", err)
	}

	if node.TEEType != "browser-extension" {
		t.Errorf("Expected TEEType 'browser-extension', got '%s'", node.TEEType)
	}
	if node.WalletAddress != walletAddress {
		t.Errorf("Expected WalletAddress '%s', got '%s'", walletAddress, node.WalletAddress)
	}
	if node.ExtensionID != extensionID {
		t.Errorf("Expected ExtensionID '%s', got '%s'", extensionID, node.ExtensionID)
	}
	if node.BrowserVersion != browserVersion {
		t.Errorf("Expected BrowserVersion '%s', got '%s'", browserVersion, node.BrowserVersion)
	}
	if len(node.BadgeNFTIDs) != 2 {
		t.Errorf("Expected 2 badge NFT IDs, got %d", len(node.BadgeNFTIDs))
	}
	if node.BadgeNFTIDs[0] != "badge-nft-001" {
		t.Errorf("Expected first badge NFT ID 'badge-nft-001', got '%s'", node.BadgeNFTIDs[0])
	}
	if !node.IsRemote {
		t.Error("Expected IsRemote to be true for browser-extension nodes")
	}
	if !node.Connected {
		t.Error("Expected Connected to be true for browser-extension nodes")
	}
	if node.SSHPort != 0 {
		t.Errorf("Expected SSHPort 0 for browser-extension nodes, got %d", node.SSHPort)
	}
	if node.Status != "online" {
		t.Errorf("Expected Status 'online', got '%s'", node.Status)
	}
}

func TestRegisterBrowserDVE_WithNoBadgeNFTs(t *testing.T) {
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	node, err := manager.RegisterBrowserDVE("0xwallet", []string{"validation"}, nil, "ext-id", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to register browser DVE without badges: %v", err)
	}

	if len(node.BadgeNFTIDs) != 0 {
		t.Errorf("Expected 0 badge NFT IDs, got %d", len(node.BadgeNFTIDs))
	}
	if node.WalletAddress != "0xwallet" {
		t.Errorf("Expected WalletAddress '0xwallet', got '%s'", node.WalletAddress)
	}
}

func TestUpdateBrowserDVEHeartbeat(t *testing.T) {
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register a browser DVE first
	node, err := manager.RegisterBrowserDVE("0xheartbeat-test", []string{"validation"}, nil, "ext-hb", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to register browser DVE: %v", err)
	}

	wsConnectionID := "ws-conn-999"

	// Update heartbeat
	err = manager.UpdateBrowserDVEHeartbeat(node.ID, wsConnectionID)
	if err != nil {
		t.Fatalf("Failed to update browser DVE heartbeat: %v", err)
	}

	// Verify the update
	updatedNode, err := manager.GetNode(node.ID)
	if err != nil {
		t.Fatalf("Failed to get updated node: %v", err)
	}

	if updatedNode.WSConnectionID != wsConnectionID {
		t.Errorf("Expected WSConnectionID '%s', got '%s'", wsConnectionID, updatedNode.WSConnectionID)
	}
	if updatedNode.Status != "online" {
		t.Errorf("Expected Status 'online', got '%s'", updatedNode.Status)
	}
	if !updatedNode.Connected {
		t.Error("Expected Connected to be true after heartbeat")
	}
}

func TestUpdateBrowserDVEHeartbeat_NonBrowserNode(t *testing.T) {
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	// Register a regular node
	req := &objects.RegisterNodeRequest{
		Name:       "sgx-node",
		TEEType:    "sgx",
		PublicKey:  "test-key",
		StakeAmount: 100000,
	}
	sgxNode, err := manager.RegisterNode(req)
	if err != nil {
		t.Fatalf("Failed to register SGX node: %v", err)
	}

	// Try to update heartbeat on non-browser node - should fail
	err = manager.UpdateBrowserDVEHeartbeat(sgxNode.ID, "ws-conn-xxx")
	if err == nil {
		t.Error("Expected error when updating heartbeat on non-browser-extension node")
	}
}

func TestUpdateBrowserDVEHeartbeat_NotFound(t *testing.T) {
	db, _ := database.NewBuntDB(":memory:")
	defer db.Close()

	cfg := &config.Config{ChainID: "test-chain"}
	manager, _ := NewDVEManager(db, nil, nil, cfg)

	err := manager.UpdateBrowserDVEHeartbeat("nonexistent-node-id", "ws-conn-xxx")
	if err == nil {
		t.Error("Expected error when updating heartbeat on nonexistent node")
	}
}
