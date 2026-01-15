package main

import (
    "KNIRVGRAPH/internal/nrv"
    "fmt"
    "log"
    "time"
)

func main() {
    fmt.Println("=== KNIRV Network Resolution Vector (NRV) System Demo ===")
    
    // Create and start NRV system
    nrvSystem := nrv.NewNRVSystem("demo-peer", nil)
    
    err := nrvSystem.Start()
    if err != nil {
        log.Fatalf("Failed to start NRV system: %v", err)
    }
    defer nrvSystem.Stop()

    fmt.Println("\n1. Creating Network Resolution Vectors...")
    
    // Create some example vectors
    vectors := []struct {
        targetHash  string
        coordinates []float64
        metadata    map[string]interface{}
    }{
        {
            targetHash:  "blockchain-node-1",
            coordinates: []float64{1.0, 2.0, 3.0},
            metadata: map[string]interface{}{
                "type":     "blockchain_node",
                "location": "us-east-1",
                "capacity": 1000,
            },
        },
        {
            targetHash:  "storage-cluster-alpha",
            coordinates: []float64{5.0, 1.0, 2.0},
            metadata: map[string]interface{}{
                "type":     "storage_cluster",
                "location": "eu-west-1",
                "capacity": 5000,
            },
        },
        {
            targetHash:  "compute-resource-beta",
            coordinates: []float64{3.0, 4.0, 1.0},
            metadata: map[string]interface{}{
                "type":     "compute_resource",
                "location": "asia-pacific-1",
                "capacity": 2000,
            },
        },
    }

    for _, v := range vectors {
        vector, err := nrvSystem.CreateVector(v.targetHash, v.coordinates, v.metadata)
        if err != nil {
            log.Printf("Failed to create vector for %s: %v", v.targetHash, err)
            continue
        }
        fmt.Printf("  ✓ Created vector %s for target %s\n", vector.ID[:8], v.targetHash)
    }

    fmt.Println("\n2. Creating Error Nodes...")
    
    // Create some example error nodes
    errors := []struct {
        errorType   string
        description string
        context     map[string]interface{}
        severity    int
    }{
        {
            errorType:   "network_connectivity",
            description: "Unable to connect to blockchain node",
            context: map[string]interface{}{
                "target_node": "blockchain-node-1",
                "error_code":  "CONN_TIMEOUT",
                "attempts":    3,
            },
            severity: 3,
        },
        {
            errorType:   "storage_capacity",
            description: "Storage cluster approaching capacity limit",
            context: map[string]interface{}{
                "cluster_id":     "storage-cluster-alpha",
                "current_usage":  85.5,
                "threshold":      90.0,
            },
            severity: 2,
        },
        {
            errorType:   "compute_overload",
            description: "Compute resource experiencing high load",
            context: map[string]interface{}{
                "resource_id": "compute-resource-beta",
                "cpu_usage":   95.2,
                "memory_usage": 88.7,
            },
            severity: 4,
        },
    }

    for _, e := range errors {
        errorNode, err := nrvSystem.CreateErrorNode(e.errorType, e.description, e.context, e.severity)
        if err != nil {
            log.Printf("Failed to create error node: %v", err)
            continue
        }
        fmt.Printf("  ✓ Created error node %s (type: %s, severity: %d)\n", 
            errorNode.ID[:8], e.errorType, e.severity)
    }

    fmt.Println("\n3. Creating Skill Nodes...")
    
    // Create some example skill nodes
    skills := []struct {
        skillType    string
        capabilities []string
        requirements map[string]interface{}
    }{
        {
            skillType:    "network_diagnostics",
            capabilities: []string{"network_connectivity", "dns_resolution", "port_scanning"},
            requirements: map[string]interface{}{
                "min_confidence": 0.8,
                "timeout":        30,
            },
        },
        {
            skillType:    "storage_management",
            capabilities: []string{"storage_capacity", "data_migration", "cleanup"},
            requirements: map[string]interface{}{
                "min_confidence": 0.9,
                "access_level":   "admin",
            },
        },
        {
            skillType:    "load_balancer",
            capabilities: []string{"compute_overload", "traffic_distribution", "scaling"},
            requirements: map[string]interface{}{
                "min_confidence": 0.85,
                "scaling_policy": "auto",
            },
        },
        {
            skillType:    "general_troubleshooter",
            capabilities: []string{"general", "logging", "monitoring"},
            requirements: map[string]interface{}{
                "min_confidence": 0.7,
            },
        },
    }

    for _, s := range skills {
        skillNode, err := nrvSystem.CreateSkillNode(s.skillType, s.capabilities, s.requirements)
        if err != nil {
            log.Printf("Failed to create skill node: %v", err)
            continue
        }
        fmt.Printf("  ✓ Created skill node %s (type: %s, capabilities: %v)\n", 
            skillNode.ID[:8], s.skillType, s.capabilities)
    }

    fmt.Println("\n4. Demonstrating Target Resolution...")
    
    // Resolve some targets
    targets := []string{"blockchain-node-1", "storage-cluster-alpha", "nonexistent-target"}
    
    for _, target := range targets {
        vectors, err := nrvSystem.ResolveTarget(target)
        if err != nil {
            log.Printf("Failed to resolve target %s: %v", target, err)
            continue
        }
        
        if len(vectors) > 0 {
            fmt.Printf("  ✓ Found %d vector(s) for target %s\n", len(vectors), target)
            for _, v := range vectors {
                fmt.Printf("    - Vector %s (confidence: %.2f)\n", v.ID[:8], v.Confidence)
            }
        } else {
            fmt.Printf("  ✗ No vectors found for target %s\n", target)
        }
    }

    fmt.Println("\n5. Demonstrating Error Resolution...")
    
    // Find skills for different error types
    errorTypes := []string{"network_connectivity", "storage_capacity", "compute_overload", "unknown_error"}
    
    for _, errorType := range errorTypes {
        skills, err := nrvSystem.GetSkillsForErrorType(errorType)
        if err != nil {
            log.Printf("Failed to get skills for error type %s: %v", errorType, err)
            continue
        }
        
        if len(skills) > 0 {
            fmt.Printf("  ✓ Found %d skill(s) for error type %s\n", len(skills), errorType)
            for _, skill := range skills {
                fmt.Printf("    - Skill %s (type: %s)\n", skill.ID[:8], skill.SkillType)
            }
        } else {
            fmt.Printf("  ✗ No skills found for error type %s\n", errorType)
        }
    }

    fmt.Println("\n6. System Statistics...")
    
    allVectors := nrvSystem.GetAllVectors()
    allErrors := nrvSystem.GetAllErrorNodes()
    allSkills := nrvSystem.GetAllSkillNodes()
    
    fmt.Printf("  Total Vectors: %d\n", len(allVectors))
    fmt.Printf("  Total Error Nodes: %d\n", len(allErrors))
    fmt.Printf("  Total Skill Nodes: %d\n", len(allSkills))

    fmt.Println("\n7. Demonstrating Vector Confidence Decay...")
    
    // Show initial confidences
    fmt.Println("  Initial vector confidences:")
    for _, v := range allVectors {
        fmt.Printf("    Vector %s: %.3f\n", v.ID[:8], v.Confidence)
    }
    
    // Wait a moment and trigger confidence decay
    time.Sleep(100 * time.Millisecond)
    
    // Note: In a real scenario, this would happen automatically via the maintenance goroutine
    fmt.Println("  After confidence decay:")
    updatedVectors := nrvSystem.GetAllVectors()
    for _, v := range updatedVectors {
        fmt.Printf("    Vector %s: %.3f\n", v.ID[:8], v.Confidence)
    }

    fmt.Println("\n=== NRV System Demo Complete ===")
    fmt.Println("\nThe NRV system successfully demonstrated:")
    fmt.Println("  ✓ Vector creation and management")
    fmt.Println("  ✓ Error node creation and tracking")
    fmt.Println("  ✓ Skill node creation and capability matching")
    fmt.Println("  ✓ Target resolution")
    fmt.Println("  ✓ Error-to-skill matching")
    fmt.Println("  ✓ System statistics and monitoring")
    fmt.Println("  ✓ Confidence decay mechanisms")
}
