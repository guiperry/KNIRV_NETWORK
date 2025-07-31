// Example usage of KNIRV Gateway SDK in Go

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway"
	"github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway/option"
)

func main() {
	// Example 1: Basic client setup
	basicExample()

	// Example 2: Economics service operations
	economicsExample()

	// Example 3: Health monitoring
	healthExample()

	// Example 4: Integration status
	integrationExample()
}

func basicExample() {
	fmt.Println("=== Basic Client Setup ===")

	// Create a client with default options
	client := gateway.NewClient()

	// Or create with custom options
	customClient := gateway.NewClient(
		option.WithEnvironmentDevelopment(),
		option.WithDebug(true),
		option.WithTimeout(60000),
	)

	// Create economics-specific client
	economicsClient := gateway.NewEconomicsClient(
		option.WithEconomicsURL("http://localhost:8090"),
		option.WithAPIKey("your-api-key"),
	)

	fmt.Printf("Default client created: %+v\n", client != nil)
	fmt.Printf("Custom client created: %+v\n", customClient != nil)
	fmt.Printf("Economics client created: %+v\n", economicsClient != nil)
}

func economicsExample() {
	fmt.Println("\n=== Economics Service Operations ===")

	// Create economics client
	client := gateway.NewEconomicsClient(
		option.WithEnvironmentDevelopment(),
		option.WithDebug(true),
	)

	ctx := context.Background()

	// Example 1: Process skill invocation
	skillRequest := gateway.SkillInvocationRequest{
		UserID:  "user123",
		SkillID: "skill456",
		Amount:  "100000", // 0.1 NRN
	}

	skillResp, err := client.Economics.Skills.Invoke(ctx, skillRequest)
	if err != nil {
		log.Printf("Skill invocation failed: %v", err)
	} else {
		fmt.Printf("Skill invocation successful: %+v\n", skillResp)
	}

	// Example 2: Register LLM
	llmRequest := gateway.LLMRegistrationRequest{
		UserID:          "user123",
		LLMID:           "llm789",
		RegistrationFee: "1000000", // 1 NRN
	}

	llmResp, err := client.Economics.LLM.Register(ctx, llmRequest)
	if err != nil {
		log.Printf("LLM registration failed: %v", err)
	} else {
		fmt.Printf("LLM registration successful: %+v\n", llmResp)
	}

	// Example 3: Process validation reward
	validationRequest := gateway.ValidationRewardRequest{
		ValidatorID:      "validator123",
		TargetID:         "target456",
		ValidationResult: true,
	}

	validationResp, err := client.Economics.Validation.Reward(ctx, validationRequest)
	if err != nil {
		log.Printf("Validation reward failed: %v", err)
	} else {
		fmt.Printf("Validation reward successful: %+v\n", validationResp)
	}

	// Example 4: Calculate network fees
	feesRequest := gateway.NetworkFeesRequest{
		GasUsed:  21000,
		Priority: "medium",
	}

	feesResp, err := client.Economics.Fees.Calculate(ctx, feesRequest)
	if err != nil {
		log.Printf("Fee calculation failed: %v", err)
	} else {
		fmt.Printf("Fee calculation successful: %+v\n", feesResp)
	}

	// Example 5: Get economic metrics
	metrics, err := client.Economics.Metrics.Get(ctx)
	if err != nil {
		log.Printf("Metrics retrieval failed: %v", err)
	} else {
		fmt.Printf("Economic metrics: Total Supply: %s, Total Burned: %s\n", 
			metrics.TotalSupply, metrics.TotalBurned)
	}

	// Example 6: Get transaction history
	transactions, err := client.Economics.Transactions.List(ctx, 10, "")
	if err != nil {
		log.Printf("Transaction list failed: %v", err)
	} else {
		fmt.Printf("Retrieved %d transactions\n", len(transactions))
	}

	// Example 7: Get burn history
	burnEvents, err := client.Economics.Burn.GetHistory(ctx, 5)
	if err != nil {
		log.Printf("Burn history failed: %v", err)
	} else {
		fmt.Printf("Retrieved %d burn events\n", len(burnEvents))
	}

	// Example 8: Get economic rules
	rules, err := client.Economics.Rules.Get(ctx)
	if err != nil {
		log.Printf("Rules retrieval failed: %v", err)
	} else {
		fmt.Printf("Skill invocation cost: %s, LLM registration fee: %s\n", 
			rules.SkillInvocationCost, rules.LLMRegistrationFee)
	}
}

func healthExample() {
	fmt.Println("\n=== Health Monitoring ===")

	client := gateway.NewClient(
		option.WithEnvironmentDevelopment(),
	)

	ctx := context.Background()

	// Check economics service health
	isHealthy, err := client.Health.Check(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("Economics service healthy: %v\n", isHealthy)
	}
}

func integrationExample() {
	fmt.Println("\n=== Integration Status ===")

	client := gateway.NewClient(
		option.WithEnvironmentDevelopment(),
		option.WithDefaultKNIRVServices(),
	)

	ctx := context.Background()

	// Get integration status
	status, err := client.Integration.GetStatus(ctx)
	if err != nil {
		log.Printf("Integration status failed: %v", err)
	} else {
		fmt.Printf("Integration status: %+v\n", status)
	}
}

// Advanced example: Custom transaction processing
func advancedExample() {
	fmt.Println("\n=== Advanced Usage ===")

	client := gateway.NewClient(
		option.WithEnvironmentDevelopment(),
		option.WithVerbose(true),
	)

	ctx := context.Background()

	// Custom workflow: Check balance, process skill, verify transaction
	userID := "advanced_user"
	skillID := "advanced_skill"
	amount := "500000" // 0.5 NRN

	// 1. Get current metrics to check system state
	metrics, err := client.Economics.Metrics.Get(ctx)
	if err != nil {
		log.Printf("Failed to get metrics: %v", err)
		return
	}

	fmt.Printf("Current system state - Total Supply: %s, Network Utilization: %f\n", 
		metrics.TotalSupply, metrics.NetworkUtilization)

	// 2. Calculate fees for the operation
	feesResp, err := client.Economics.Fees.Calculate(ctx, gateway.NetworkFeesRequest{
		GasUsed:  50000,
		Priority: "high",
	})
	if err != nil {
		log.Printf("Fee calculation failed: %v", err)
		return
	}

	fmt.Printf("Estimated fees: %s\n", feesResp.TotalFee)

	// 3. Process the skill invocation
	skillResp, err := client.Economics.Skills.Invoke(ctx, gateway.SkillInvocationRequest{
		UserID:  userID,
		SkillID: skillID,
		Amount:  amount,
	})
	if err != nil {
		log.Printf("Skill invocation failed: %v", err)
		return
	}

	fmt.Printf("Skill invocation completed: Transaction ID: %s\n", skillResp.TransactionID)

	// 4. Verify the transaction
	transaction, err := client.Economics.Transactions.Get(ctx, skillResp.TransactionID)
	if err != nil {
		log.Printf("Transaction verification failed: %v", err)
		return
	}

	fmt.Printf("Transaction verified: Status: %s, Amount: %s\n", 
		transaction.Status, transaction.Amount)

	// 5. Get updated metrics
	updatedMetrics, err := client.Economics.Metrics.Get(ctx)
	if err != nil {
		log.Printf("Failed to get updated metrics: %v", err)
		return
	}

	fmt.Printf("Updated system state - Total Burned: %s\n", updatedMetrics.TotalBurned)
}

// Environment setup helper
func setupEnvironment() {
	// Set environment variables for development
	os.Setenv("ECONOMICS_SERVICE_URL", "http://localhost:8090")
	os.Setenv("GATEWAY_SERVICE_URL", "http://localhost:8000")
	os.Setenv("KNIRVCHAIN_URL", "http://localhost:8080")
	os.Setenv("KNIRVNEXUS_URL", "http://localhost:8081")
	os.Setenv("KNIRVROOT_URL", "http://localhost:8082")
	os.Setenv("KNIRVGRAPH_URL", "http://localhost:8083")
	os.Setenv("KNIRV_DEBUG", "true")
}
