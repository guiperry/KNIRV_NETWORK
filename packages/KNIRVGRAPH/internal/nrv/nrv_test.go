package nrv

import (
    "testing"
    "time"
)

func TestNRVSystem(t *testing.T) {
    // Create NRV system
    nrvSystem := NewNRVSystem("test-peer", nil)
    
    // Start the system
    err := nrvSystem.Start()
    if err != nil {
        t.Fatalf("Failed to start NRV system: %v", err)
    }
    defer nrvSystem.Stop()

    // Test vector creation
    t.Run("CreateVector", func(t *testing.T) {
        targetHash := "test-hash-123"
        coordinates := []float64{1.0, 2.0, 3.0}
        metadata := map[string]interface{}{
            "test": "data",
        }

        vector, err := nrvSystem.CreateVector(targetHash, coordinates, metadata)
        if err != nil {
            t.Fatalf("Failed to create vector: %v", err)
        }

        if vector.TargetHash != targetHash {
            t.Errorf("Expected target hash %s, got %s", targetHash, vector.TargetHash)
        }

        if len(vector.Coordinates) != len(coordinates) {
            t.Errorf("Expected %d coordinates, got %d", len(coordinates), len(vector.Coordinates))
        }

        if vector.Confidence != 1.0 {
            t.Errorf("Expected initial confidence 1.0, got %f", vector.Confidence)
        }
    })

    // Test target resolution
    t.Run("ResolveTarget", func(t *testing.T) {
        targetHash := "resolve-test-hash"
        coordinates := []float64{4.0, 5.0}
        metadata := map[string]interface{}{"type": "test"}

        // Create a vector first
        _, err := nrvSystem.CreateVector(targetHash, coordinates, metadata)
        if err != nil {
            t.Fatalf("Failed to create vector: %v", err)
        }

        // Resolve the target
        vectors, err := nrvSystem.ResolveTarget(targetHash)
        if err != nil {
            t.Fatalf("Failed to resolve target: %v", err)
        }

        if len(vectors) != 1 {
            t.Errorf("Expected 1 vector, got %d", len(vectors))
        }

        if vectors[0].TargetHash != targetHash {
            t.Errorf("Expected target hash %s, got %s", targetHash, vectors[0].TargetHash)
        }
    })

    // Test error node creation
    t.Run("CreateErrorNode", func(t *testing.T) {
        errorType := "network_error"
        description := "Connection timeout"
        context := map[string]interface{}{
            "host": "example.com",
            "port": 8080,
        }
        severity := 3

        errorNode, err := nrvSystem.CreateErrorNode(errorType, description, context, severity)
        if err != nil {
            t.Fatalf("Failed to create error node: %v", err)
        }

        if errorNode.ErrorType != errorType {
            t.Errorf("Expected error type %s, got %s", errorType, errorNode.ErrorType)
        }

        if errorNode.Severity != severity {
            t.Errorf("Expected severity %d, got %d", severity, errorNode.Severity)
        }
    })

    // Test skill node creation
    t.Run("CreateSkillNode", func(t *testing.T) {
        skillType := "network_repair"
        capabilities := []string{"network_error", "connection_timeout"}
        requirements := map[string]interface{}{
            "min_confidence": 0.8,
        }

        skillNode, err := nrvSystem.CreateSkillNode(skillType, capabilities, requirements)
        if err != nil {
            t.Fatalf("Failed to create skill node: %v", err)
        }

        if skillNode.SkillType != skillType {
            t.Errorf("Expected skill type %s, got %s", skillType, skillNode.SkillType)
        }

        if len(skillNode.Capabilities) != len(capabilities) {
            t.Errorf("Expected %d capabilities, got %d", len(capabilities), len(skillNode.Capabilities))
        }
    })

    // Test skill matching for error types
    t.Run("GetSkillsForErrorType", func(t *testing.T) {
        // Create a skill that can handle network errors
        skillType := "network_troubleshooter"
        capabilities := []string{"network_error", "dns_error"}
        requirements := map[string]interface{}{}

        _, err := nrvSystem.CreateSkillNode(skillType, capabilities, requirements)
        if err != nil {
            t.Fatalf("Failed to create skill node: %v", err)
        }

        // Find skills for network_error
        skills, err := nrvSystem.GetSkillsForErrorType("network_error")
        if err != nil {
            t.Fatalf("Failed to get skills for error type: %v", err)
        }

        if len(skills) == 0 {
            t.Error("Expected at least one skill for network_error")
        }

        found := false
        for _, skill := range skills {
            if skill.SkillType == skillType {
                found = true
                break
            }
        }

        if !found {
            t.Error("Expected to find the created skill")
        }
    })

    // Test vector confidence decay
    t.Run("VectorConfidenceDecay", func(t *testing.T) {
        targetHash := "decay-test-hash"
        coordinates := []float64{1.0, 1.0}
        metadata := map[string]interface{}{}

        vector, err := nrvSystem.CreateVector(targetHash, coordinates, metadata)
        if err != nil {
            t.Fatalf("Failed to create vector: %v", err)
        }

        initialConfidence := vector.Confidence

        // Manually trigger confidence update
        nrvSystem.updateVectorConfidences()

        // Get the vector again
        vectors, err := nrvSystem.ResolveTarget(targetHash)
        if err != nil {
            t.Fatalf("Failed to resolve target: %v", err)
        }

        if len(vectors) == 0 {
            t.Fatal("No vectors found after confidence update")
        }

        updatedConfidence := vectors[0].Confidence
        if updatedConfidence >= initialConfidence {
            t.Errorf("Expected confidence to decay from %f, but got %f", initialConfidence, updatedConfidence)
        }
    })

    // Test getting all vectors, errors, and skills
    t.Run("GetAllMethods", func(t *testing.T) {
        vectors := nrvSystem.GetAllVectors()
        errors := nrvSystem.GetAllErrorNodes()
        skills := nrvSystem.GetAllSkillNodes()

        if len(vectors) == 0 {
            t.Error("Expected some vectors to exist")
        }

        if len(errors) == 0 {
            t.Error("Expected some error nodes to exist")
        }

        if len(skills) == 0 {
            t.Error("Expected some skill nodes to exist")
        }
    })
}

func TestNRVConfig(t *testing.T) {
    config := DefaultNRVConfig()

    if config.MaxVectors <= 0 {
        t.Error("Expected positive max vectors")
    }

    if config.VectorTTL <= 0 {
        t.Error("Expected positive vector TTL")
    }

    if config.ConfidenceDecay <= 0 || config.ConfidenceDecay >= 1 {
        t.Error("Expected confidence decay between 0 and 1")
    }

    if config.ValidationTimeout <= 0 {
        t.Error("Expected positive validation timeout")
    }
}

func TestVectorExpiration(t *testing.T) {
    // Create config with very short TTL for testing
    config := &NRVConfig{
        MaxVectors:        100,
        VectorTTL:         100 * time.Millisecond,
        ConfidenceDecay:   0.95,
        ValidationTimeout: 1 * time.Second,
        DHTBootstrapPeers: []string{},
    }

    nrvSystem := NewNRVSystem("test-peer", config)
    err := nrvSystem.Start()
    if err != nil {
        t.Fatalf("Failed to start NRV system: %v", err)
    }
    defer nrvSystem.Stop()

    // Create a vector
    targetHash := "expiration-test"
    coordinates := []float64{1.0}
    metadata := map[string]interface{}{}

    _, err = nrvSystem.CreateVector(targetHash, coordinates, metadata)
    if err != nil {
        t.Fatalf("Failed to create vector: %v", err)
    }

    // Verify vector exists
    vectors := nrvSystem.GetAllVectors()
    if len(vectors) == 0 {
        t.Fatal("Expected vector to exist")
    }

    // Wait for expiration
    time.Sleep(200 * time.Millisecond)

    // Manually trigger cleanup
    nrvSystem.cleanupExpiredVectors()

    // Verify vector is gone
    vectors = nrvSystem.GetAllVectors()
    if len(vectors) != 0 {
        t.Errorf("Expected vector to be expired, but found %d vectors", len(vectors))
    }
}
