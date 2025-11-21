package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/graph"
	"KNIRVCHAIN/internal/types"
)

// NoveltyChecker handles novelty assessment of ideas
type NoveltyChecker struct {
	graphQueries *graph.GraphQueries
	chromaDB     database.ChromemDBManager
}

// NewNoveltyChecker creates a new novelty checker
func NewNoveltyChecker(graphQueries *graph.GraphQueries, chromaDB database.ChromemDBManager) *NoveltyChecker {
	return &NoveltyChecker{
		graphQueries: graphQueries,
		chromaDB:     chromaDB,
	}
}

// NoveltyResult represents the result of a novelty assessment
type NoveltyResult struct {
	IsNovel       bool                   `json:"is_novel"`
	Score         float64                `json:"score"`          // 0.0 to 1.0, higher means more novel
	SimilarIdeas  []SimilarIdea          `json:"similar_ideas"`
	AssessmentBy  string                 `json:"assessment_by"`
	AssessedAt    int64                  `json:"assessed_at"`
	Reasoning     string                 `json:"reasoning"`
}

// SimilarIdea represents an idea similar to the assessed idea
type SimilarIdea struct {
	IdeaNodeID string  `json:"idea_node_id"`
	Similarity float64 `json:"similarity"` // 0.0 to 1.0
	Content    string  `json:"content"`
}

// AssessNovelty assesses the novelty of an idea
func (nc *NoveltyChecker) AssessNovelty(ideaNode *types.IdeaNode) (*NoveltyResult, error) {
	result := &NoveltyResult{
		AssessmentBy: "novelty_checker",
		AssessedAt:   time.Now().Unix(),
	}

	// Get all existing idea nodes for comparison
	existingIdeas, err := nc.getAllExistingIdeas()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing ideas: %w", err)
	}

	if len(existingIdeas) == 0 {
		// First idea is automatically novel
		result.IsNovel = true
		result.Score = 1.0
		result.Reasoning = "First idea in the system"
		return result, nil
	}

	// Perform semantic similarity search
	similarIdeas, err := nc.findSimilarIdeas(ideaNode, existingIdeas)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar ideas: %w", err)
	}

	result.SimilarIdeas = similarIdeas

	// Calculate novelty score
	noveltyScore := nc.calculateNoveltyScore(ideaNode, similarIdeas)
	result.Score = noveltyScore

	// Determine if novel based on threshold
	noveltyThreshold := 0.7 // Ideas with score > 0.7 are considered novel
	result.IsNovel = noveltyScore > noveltyThreshold

	// Generate reasoning
	result.Reasoning = nc.generateReasoning(result, similarIdeas)

	log.Printf("Novelty assessment completed for idea %s: score=%.2f, novel=%t", ideaNode.ID, noveltyScore, result.IsNovel)

	return result, nil
}

// getAllExistingIdeas gets all existing idea nodes
func (nc *NoveltyChecker) getAllExistingIdeas() ([]*types.IdeaNode, error) {
	nodeIDs, err := nc.graphQueries.GetNodeStore().ListNodesByType("idea_node")
	if err != nil {
		return nil, err
	}

	var ideas []*types.IdeaNode
	for _, id := range nodeIDs {
		idea, err := nc.graphQueries.GetNodeStore().GetIdeaNode(id)
		if err != nil {
			continue // Skip on error
		}
		ideas = append(ideas, idea)
	}

	return ideas, nil
}

// findSimilarIdeas finds ideas similar to the given idea
func (nc *NoveltyChecker) findSimilarIdeas(targetIdea *types.IdeaNode, existingIdeas []*types.IdeaNode) ([]SimilarIdea, error) {
	var similarIdeas []SimilarIdea

	// Use ChromaDB for semantic search if available
	if nc.chromaDB != nil {
		return nc.findSimilarIdeasSemantic(targetIdea, existingIdeas)
	}

	// Fallback to simple text similarity
	return nc.findSimilarIdeasText(targetIdea, existingIdeas)
}

// findSimilarIdeasSemantic uses semantic search to find similar ideas
func (nc *NoveltyChecker) findSimilarIdeasSemantic(targetIdea *types.IdeaNode, existingIdeas []*types.IdeaNode) ([]SimilarIdea, error) {
	// Create search query from the target idea
	query := nc.generateIdeaQuery(targetIdea)

	// Perform semantic search
	results, err := nc.chromaDB.Search("knirvchain_nodes", query, 10)
	if err != nil {
		log.Printf("Semantic search failed, falling back to text similarity: %v", err)
		return nc.findSimilarIdeasText(targetIdea, existingIdeas)
	}

	var similarIdeas []SimilarIdea
	for _, result := range results {
		// Only consider idea nodes
		if !strings.Contains(result.Content, "idea_type") {
			continue
		}

		similarIdea := SimilarIdea{
			IdeaNodeID: result.ID,
			Similarity: float64(result.Score) / 100.0, // Convert to 0-1 scale
			Content:    result.Content,
		}
		similarIdeas = append(similarIdeas, similarIdea)
	}

	return similarIdeas, nil
}

// findSimilarIdeasText uses simple text similarity to find similar ideas
func (nc *NoveltyChecker) findSimilarIdeasText(targetIdea *types.IdeaNode, existingIdeas []*types.IdeaNode) ([]SimilarIdea, error) {
	var similarIdeas []SimilarIdea

	targetContent := nc.generateIdeaContent(targetIdea)

	for _, existingIdea := range existingIdeas {
		// Skip self-comparison
		if existingIdea.ID == targetIdea.ID {
			continue
		}

		existingContent := nc.generateIdeaContent(existingIdea)
		similarity := nc.calculateTextSimilarity(targetContent, existingContent)

		if similarity > 0.3 { // Only include reasonably similar ideas
			similarIdeas = append(similarIdeas, SimilarIdea{
				IdeaNodeID: existingIdea.ID,
				Similarity: similarity,
				Content:    existingContent,
			})
		}
	}

	return similarIdeas, nil
}

// generateIdeaQuery generates a search query from an idea
func (nc *NoveltyChecker) generateIdeaQuery(idea *types.IdeaNode) string {
	return fmt.Sprintf("%s idea about %s", idea.IdeaType, idea.ContentHash)
}

// generateIdeaContent generates searchable content from an idea
func (nc *NoveltyChecker) generateIdeaContent(idea *types.IdeaNode) string {
	content := fmt.Sprintf("Type: %s, Origin: %s", idea.IdeaType, idea.OriginNIM)

	// Add dependency information
	if len(idea.Dependencies) > 0 {
		content += fmt.Sprintf(", Dependencies: %v", idea.Dependencies)
	}

	return content
}

// calculateTextSimilarity calculates simple text similarity
func (nc *NoveltyChecker) calculateTextSimilarity(text1, text2 string) float64 {
	// Simple Jaccard similarity based on words
	words1 := nc.tokenizeText(text1)
	words2 := nc.tokenizeText(text2)

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Create word sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		set1[word] = true
	}
	for _, word := range words2 {
		set2[word] = true
	}

	// Calculate intersection
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	// Calculate union
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// tokenizeText splits text into words
func (nc *NoveltyChecker) tokenizeText(text string) []string {
	// Simple tokenization: split on spaces and remove punctuation
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, ".", "")
	text = strings.ReplaceAll(text, ":", "")
	text = strings.ReplaceAll(text, ";", "")

	return strings.Fields(text)
}

// calculateNoveltyScore calculates the novelty score based on similar ideas
func (nc *NoveltyChecker) calculateNoveltyScore(idea *types.IdeaNode, similarIdeas []SimilarIdea) float64 {
	if len(similarIdeas) == 0 {
		return 1.0 // Completely novel
	}

	// Calculate weighted average of similarities
	totalWeight := 0.0
	weightedSum := 0.0

	for _, similar := range similarIdeas {
		weight := 1.0 // Could be weighted by idea importance/authority
		weightedSum += similar.Similarity * weight
		totalWeight += weight
	}

	avgSimilarity := weightedSum / totalWeight

	// Novelty score is inverse of average similarity
	noveltyScore := 1.0 - avgSimilarity

	// Ensure score is between 0 and 1
	if noveltyScore < 0 {
		noveltyScore = 0
	}
	if noveltyScore > 1 {
		noveltyScore = 1
	}

	return noveltyScore
}

// generateReasoning generates human-readable reasoning for the assessment
func (nc *NoveltyChecker) generateReasoning(result *NoveltyResult, similarIdeas []SimilarIdea) string {
	if result.IsNovel {
		if len(similarIdeas) == 0 {
			return "This appears to be a completely novel idea with no similar concepts found in the existing knowledge graph."
		}
		return fmt.Sprintf("This idea shows sufficient novelty (score: %.2f) despite %d similar concepts found. The highest similarity was %.2f.",
			result.Score, len(similarIdeas), nc.getMaxSimilarity(similarIdeas))
	}

	return fmt.Sprintf("This idea lacks sufficient novelty (score: %.2f) with %d similar concepts found. The highest similarity was %.2f.",
		result.Score, len(similarIdeas), nc.getMaxSimilarity(similarIdeas))
}

// getMaxSimilarity returns the maximum similarity score from similar ideas
func (nc *NoveltyChecker) getMaxSimilarity(similarIdeas []SimilarIdea) float64 {
	if len(similarIdeas) == 0 {
		return 0.0
	}

	maxSim := 0.0
	for _, idea := range similarIdeas {
		if idea.Similarity > maxSim {
			maxSim = idea.Similarity
		}
	}

	return maxSim
}

// BatchAssessNovelty assesses novelty for multiple ideas
func (nc *NoveltyChecker) BatchAssessNovelty(ideaNodes []*types.IdeaNode) ([]*NoveltyResult, error) {
	results := make([]*NoveltyResult, len(ideaNodes))

	for i, idea := range ideaNodes {
		result, err := nc.AssessNovelty(idea)
		if err != nil {
			return nil, fmt.Errorf("failed to assess novelty for idea %s: %w", idea.ID, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetNoveltyStatistics returns statistics about novelty assessments
func (nc *NoveltyChecker) GetNoveltyStatistics() (*NoveltyStatistics, error) {
	// Get all idea nodes
	ideaNodes, err := nc.getAllExistingIdeas()
	if err != nil {
		return nil, err
	}

	if len(ideaNodes) == 0 {
		return &NoveltyStatistics{}, nil
	}

	stats := &NoveltyStatistics{
		TotalIdeas:     len(ideaNodes),
		NovelIdeas:     0,
		AverageNovelty: 0.0,
	}

	totalNovelty := 0.0

	for _, idea := range ideaNodes {
		// This would need to be enhanced to get stored novelty assessments
		// For now, simulate assessment
		result, err := nc.AssessNovelty(idea)
		if err != nil {
			continue
		}

		totalNovelty += result.Score
		if result.IsNovel {
			stats.NovelIdeas++
		}
	}

	stats.AverageNovelty = totalNovelty / float64(len(ideaNodes))

	return stats, nil
}

// NoveltyStatistics represents statistics about novelty assessments
type NoveltyStatistics struct {
	TotalIdeas     int     `json:"total_ideas"`
	NovelIdeas     int     `json:"novel_ideas"`
	AverageNovelty float64 `json:"average_novelty"`
}

// MonitorNoveltyTrends monitors trends in novelty over time
func (nc *NoveltyChecker) MonitorNoveltyTrends(ctx context.Context, interval time.Duration) chan *NoveltyTrend {
	trends := make(chan *NoveltyTrend, 1)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				close(trends)
				return
			case <-ticker.C:
				trend, err := nc.calculateNoveltyTrend()
				if err != nil {
					log.Printf("Failed to calculate novelty trend: %v", err)
					continue
				}
				trends <- trend
			}
		}
	}()

	return trends
}

// NoveltyTrend represents a novelty trend
type NoveltyTrend struct {
	Timestamp      int64   `json:"timestamp"`
	AverageNovelty float64 `json:"average_novelty"`
	Trend          string  `json:"trend"` // increasing, decreasing, stable
	Period         string  `json:"period"`
}

// calculateNoveltyTrend calculates the current novelty trend
func (nc *NoveltyChecker) calculateNoveltyTrend() (*NoveltyTrend, error) {
	stats, err := nc.GetNoveltyStatistics()
	if err != nil {
		return nil, err
	}

	// This is a simplified trend calculation
	// In a real system, this would compare with historical data
	trend := &NoveltyTrend{
		Timestamp:      time.Now().Unix(),
		AverageNovelty: stats.AverageNovelty,
		Trend:          "stable", // Placeholder
		Period:         "current",
	}

	return trend, nil
}