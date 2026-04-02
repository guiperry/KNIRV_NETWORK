package writer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apache/arrow/go/v18/arrow/ipc"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/encoder"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/normalizer"
)

type FileWriter struct {
	arrowDir string
	jsonDir  string
}

func NewFileWriter(arrowDir, jsonDir string) *FileWriter {
	return &FileWriter{
		arrowDir: arrowDir,
		jsonDir:  jsonDir,
	}
}

func (w *FileWriter) WriteArrow(records []*normalizer.SecurityRecord, filename string) (string, error) {
	if len(records) == 0 {
		return "", nil
	}

	enc := encoder.NewArrowEncoder()
	data, err := enc.EncodeToBytes(records)
	if err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}

	path := filepath.Join(w.arrowDir, filename+".arrow")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	return path, nil
}

func (w *FileWriter) WriteJSON(records []*normalizer.SecurityRecord, filename string) (string, error) {
	if len(records) == 0 {
		return "", nil
	}

	enc := encoder.NewJSONEncoder()
	path := filepath.Join(w.jsonDir, filename+".json")
	if err := enc.EncodeToFile(records, path); err != nil {
		return "", fmt.Errorf("encode to file: %w", err)
	}

	return path, nil
}

func (w *FileWriter) WriteBoth(records []*normalizer.SecurityRecord, filename string) (arrowPath, jsonPath string, err error) {
	arrowPath, err = w.WriteArrow(records, filename)
	if err != nil {
		return "", "", err
	}

	jsonPath, err = w.WriteJSON(records, filename)
	if err != nil {
		return arrowPath, "", err
	}

	return arrowPath, jsonPath, nil
}

func (w *FileWriter) WriteArrowIPC(records []*normalizer.SecurityRecord, filename string) (string, error) {
	if len(records) == 0 {
		return "", nil
	}

	enc := encoder.NewArrowEncoder()
	table, err := enc.Encode(records)
	if err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}
	defer table.Release()

	path := filepath.Join(w.arrowDir, filename+".arrow")
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	defer file.Close()

	writer := ipc.NewFileWriter(file)
	defer writer.Close()

	for i := 0; i < int(table.NumCols()); i++ {
		col := table.Column(i)
		arr := col.Data().Slice(0, col.Len())
		if err := writer.Write(*arr); err != nil {
			return "", fmt.Errorf("write: %w", err)
		}
	}

	return path, nil
}
