package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArxivPaperRemainsOneMapperRecord(t *testing.T) {
	raw := RawRecord{
		DatasetID: "arxiv:cs.LG",
		Heading:   "A heading used for domain classification",
		Text:      strings.Repeat("long abstract text ", 200),
	}
	chunks := stagedRecordChunks(raw, &Config{ChunkSize: 128, ChunkOverlap: 16})
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want one arXiv paper unit", len(chunks))
	}
}

func TestStagedInputIgnoresConnectorStateObjects(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "records.jsonl"), []byte(`{"dataset_id":"arxiv:cs.LG","text":"paper abstract"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source_state.json"), []byte(`{"tier":"arxiv"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "arxiv_cursor.json"), []byte(`{"next_offset":12}`), 0644); err != nil {
		t.Fatal(err)
	}
	files, err := stagedInputFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "records.jsonl" {
		t.Fatalf("staged files=%v", files)
	}
}
