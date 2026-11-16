package embedded

import "testing"

// The status registry must list the trainer the Makefile embeds
// (build-heart-decision-trainer), so `CheckEmbeddedBinaries` reports it.
func TestCheckEmbeddedBinariesListsHeartDecisionTrainer(t *testing.T) {
	for _, b := range CheckEmbeddedBinaries() {
		if b.Name == "heart-decision-trainer" {
			return
		}
	}
	t.Fatal("heart-decision-trainer missing from the embedded-binary registry")
}
