package controller

import "testing"

func TestPipelineRunsTrainerBeforeSeeder(t *testing.T) {
	stages := buildPipelineStages("goat")
	trainerIndex, seederIndex := -1, -1
	for i, stage := range stages {
		switch stage.BinName {
		case "data-trainer":
			trainerIndex = i
		case "data-seeder":
			seederIndex = i
		}
	}
	if trainerIndex < 0 || seederIndex < 0 {
		t.Fatalf("pipeline stages = %#v; expected both data-trainer and data-seeder", stages)
	}
	if trainerIndex >= seederIndex {
		t.Fatalf("data-trainer index = %d, data-seeder index = %d; trainer must run first", trainerIndex, seederIndex)
	}
	if got := stages[trainerIndex].Args; len(got) == 0 || got[len(got)-1] != "1" {
		t.Fatalf("data-trainer args = %v; expected a single pass over the mapper batch", got)
	}
	seederArgs := stages[seederIndex].Args
	for i := 0; i+1 < len(seederArgs); i++ {
		if seederArgs[i] == "-epochs" && seederArgs[i+1] == "1" {
			return
		}
	}
	t.Fatalf("data-seeder args = %v; expected a single pass over the mapper batch", seederArgs)
}
