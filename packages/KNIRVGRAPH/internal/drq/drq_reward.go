package drq



// CalculateReward computes immediate reward for DRQ update
func CalculateReward(action DRQAction, outcome ActionOutcome) float64 {
	baseReward := 0.0

	// Solution quality component
	if outcome.SolutionValidated {
		baseReward += outcome.ValidationScore * 100.0
	}

	// Cluster efficiency component
	clusterEfficiency := 0.0
	if outcome.AgentHours > 0 { // Avoid division by zero
		clusterEfficiency = float64(outcome.SolutionsGenerated) / float64(outcome.AgentHours)
	}
	baseReward += clusterEfficiency * 50.0

	// Skill reusability component
	if outcome.SkillMinted {
		baseReward += outcome.DependencyCount * 25.0
	}

	// Network effect component
	networkBonus := outcome.DownstreamResolutions * 10.0
	baseReward += networkBonus

	// Penalty for resource waste
	resourcePenalty := outcome.WastedDVEHours * -5.0
	baseReward += resourcePenalty

	return baseReward
}
