package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/ipc"
)

type KnirvInputConfig struct {
	InputDir string
	Format   string // "arrow", "json", "auto"
}

func (c *Config) LoadKnirvInput() ([]*DocumentRecord, error) {
	format := c.Format
	if format == "" {
		format = "auto"
	}

	var records []*DocumentRecord
	var err error

	switch format {
	case "arrow":
		records, err = loadArrowFiles(c.InputDir)
	case "json":
		records, err = loadJSONFiles(c.InputDir)
	case "auto":
		records, err = loadAutoDetect(c.InputDir)
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}

	return records, err
}

func loadArrowFiles(dir string) ([]*DocumentRecord, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.arrow"))
	if err != nil {
		return nil, fmt.Errorf("glob arrow files: %w", err)
	}

	var records []*DocumentRecord
	for _, f := range files {
		recs, err := loadSingleArrow(f)
		if err != nil {
			continue
		}
		records = append(records, recs...)
	}

	return records, nil
}

func loadSingleArrow(path string) ([]*DocumentRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer file.Close()

	reader, err := ipc.NewFileReader(file)
	if err != nil {
		return nil, fmt.Errorf("new reader: %w", err)
	}
	defer reader.Close()

	schema := reader.Schema()
	colIndex := make(map[string]int)
	for i, field := range schema.Fields() {
		colIndex[field.Name] = i
	}

	var records []*DocumentRecord
	batchIndex := 0

	for reader.Next() {
		rec := reader.Record()
		if rec == nil {
			break
		}

		chunkID := int32(batchIndex)

		for row := 0; row < int(rec.NumRows()); row++ {
			record := &DocumentRecord{
				ChunkID: chunkID,
			}

			if idx, ok := colIndex["file_name"]; ok {
				if arr := getStringArray(rec.Column(idx), row); arr != "" {
					record.FileName = arr
				}
			}

			if idx, ok := colIndex["content"]; ok {
				if arr := getStringArray(rec.Column(idx), row); arr != "" {
					record.Content = arr
				}
			}

			if idx, ok := colIndex["tokens"]; ok {
				record.Tokens = getStringListArray(rec.Column(idx), row)
			}

			if idx, ok := colIndex["pos_tags"]; ok {
				record.POSTags = getIntListArray(rec.Column(idx), row)
			}

			if idx, ok := colIndex["dep_hashes"]; ok {
				record.DepHashes = getUint32ListArray(rec.Column(idx), row)
			}

			records = append(records, record)
			chunkID++
		}

		batchIndex++
	}

	return records, nil
}

func loadJSONFiles(dir string) ([]*DocumentRecord, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob json files: %w", err)
	}

	var records []*DocumentRecord
	for _, f := range files {
		recs, err := loadSingleJSON(f)
		if err != nil {
			continue
		}
		records = append(records, recs...)
	}

	return records, nil
}

func loadSingleJSON(path string) ([]*DocumentRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var jsonRecords []struct {
		FileName  string   `json:"file_name"`
		ChunkID   int32    `json:"chunk_id"`
		Content   string   `json:"content"`
		Tokens    []string `json:"tokens"`
		POSTags   []int    `json:"pos_tags"`
		DepHashes []uint32 `json:"dep_hashes"`
	}

	if err := json.Unmarshal(data, &jsonRecords); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	records := make([]*DocumentRecord, len(jsonRecords))
	for i, jr := range jsonRecords {
		records[i] = &DocumentRecord{
			FileName:  jr.FileName,
			ChunkID:   jr.ChunkID,
			Content:   jr.Content,
			Tokens:    jr.Tokens,
			POSTags:   jr.POSTags,
			DepHashes: jr.DepHashes,
		}
	}

	return records, nil
}

func loadAutoDetect(dir string) ([]*DocumentRecord, error) {
	arrowFiles, _ := filepath.Glob(filepath.Join(dir, "*.arrow"))
	jsonFiles, _ := filepath.Glob(filepath.Join(dir, "*.json"))

	if len(arrowFiles) > 0 && len(jsonFiles) == 0 {
		return loadArrowFiles(dir)
	}
	if len(jsonFiles) > 0 && len(arrowFiles) == 0 {
		return loadJSONFiles(dir)
	}

	var all []*DocumentRecord
	if arrow, err := loadArrowFiles(dir); err == nil {
		all = append(all, arrow...)
	}
	if json, err := loadJSONFiles(dir); err == nil {
		all = append(all, json...)
	}

	return all, nil
}

func (c *Config) SetKnirvFormat(format string) {
	c.Format = format
}

func getStringArray(col arrow.Column, row int) string {
	return ""
}

func getStringListArray(col arrow.Column, row int) []string {
	return nil
}

func getIntListArray(col arrow.Column, row int) []int {
	return nil
}

func getUint32ListArray(col arrow.Column, row int) []uint32 {
	return nil
}

func init() {
	RegisterKnirvFormats()
}

func RegisterKnirvFormats() {
	supportedFormats["arrow"] = "Apache Arrow IPC format"
	supportedFormats["json"] = "JSON Lines format"
	supportedFormats["auto"] = "Auto-detect format"
}

var supportedFormats = make(map[string]string)

func GetSupportedFormats() []string {
	formats := make([]string, 0, len(supportedFormats))
	for f := range supportedFormats {
		formats = append(formats, f)
	}
	return formats
}

func FormatFromExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".arrow":
		return "arrow"
	case ".json":
		return "json"
	case ".parquet":
		return "parquet"
	default:
		return "unknown"
	}
}
