package controller

import (
	"strings"
	"testing"
)

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

func TestPipelineHistoryIsBoundedAndNeverReadsPaths(t *testing.T) {
	c := NewController(false)
	for i := 0; i < maxPipelineLogHistory+5; i++ {
		c.recordPipelineLog(PipelineLogMsg{Log: "line", Stage: "mapper", Error: i%2 == 0})
	}
	logs := c.PipelineLogs("mapper", 999)
	if len(logs) != maxPipelineLogHistory {
		t.Fatalf("logs = %d, want %d", len(logs), maxPipelineLogHistory)
	}
	if got := c.PipelineLogs("../../etc", 10); len(got) != 0 {
		t.Fatalf("stage filter must be exact, got %#v", got)
	}
	errs := c.ErrorHistory(999)
	if len(errs) != 50 {
		t.Fatalf("errors = %d, want 50", len(errs))
	}
	for _, entry := range errs {
		if !entry.Error || strings.Contains(entry.Log, "/etc") {
			t.Fatalf("bad error record %#v", entry)
		}
	}
}

func TestErrorHistoryIncludesBoundedAPIErrors(t *testing.T) {
	c := NewController(false)
	c.RecordError("api", "request rejected")
	errs := c.ErrorHistory(1)
	if len(errs) != 1 || errs[0].Stage != "api" || errs[0].Log != "request rejected" || !errs[0].Complete {
		t.Fatalf("errors = %#v", errs)
	}
}
