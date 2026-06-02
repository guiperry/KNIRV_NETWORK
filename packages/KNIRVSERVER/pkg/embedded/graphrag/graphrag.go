package graphrag

/*
#cgo LDFLAGS: ${SRCDIR}/../../../build/embedded/graphrag/libgraphrag_core.a -ldl -pthread -lm
#include "graphrag.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

func Init(config []byte) error {
	if len(config) == 0 {
		return fmt.Errorf("config cannot be empty")
	}

	cConfig := C.CString(string(config))
	defer C.free(unsafe.Pointer(cConfig))

	result := C.graphrag_init(cConfig, C.size_t(len(config)))
	if result != 0 {
		return fmt.Errorf("graphrag init failed with code: %d", result)
	}

	return nil
}

func Shutdown() error {
	result := C.graphrag_shutdown()
	if result != 0 {
		return fmt.Errorf("graphrag shutdown failed with code: %d", result)
	}
	return nil
}

func HealthCheck() error {
	result := C.graphrag_health_check()
	if result != 0 {
		return fmt.Errorf("graphrag health check failed with code: %d", result)
	}
	return nil
}

func IndexDocument(docID string, content []byte) error {
	if docID == "" {
		return fmt.Errorf("docID cannot be empty")
	}

	cDocID := C.CString(docID)
	defer C.free(unsafe.Pointer(cDocID))

	cContent := C.CString(string(content))
	defer C.free(unsafe.Pointer(cContent))

	result := C.graphrag_index_document(cDocID, cContent, C.size_t(len(content)))
	if result != 0 {
		return fmt.Errorf("graphrag index document failed with code: %d", result)
	}

	return nil
}

func Query(query string, limit int) ([]byte, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	var bufLen C.size_t
	resultPtr := C.graphrag_query(cQuery, C.int(limit), &bufLen)
	if resultPtr == nil {
		return nil, fmt.Errorf("graphrag query returned null pointer")
	}
	defer C.free(unsafe.Pointer(resultPtr))

	return C.GoBytes(unsafe.Pointer(resultPtr), C.int(bufLen)), nil
}

func EmbedTexts(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	textsJSON, err := json.Marshal(texts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal texts: %w", err)
	}

	cTextsJSON := C.CString(string(textsJSON))
	defer C.free(unsafe.Pointer(cTextsJSON))

	var bufLen C.size_t
	resultPtr := C.graphrag_embed_texts(cTextsJSON, C.size_t(len(textsJSON)), &bufLen)
	if resultPtr == nil {
		return nil, fmt.Errorf("graphrag embed texts returned null pointer")
	}
	defer C.free(unsafe.Pointer(resultPtr))

	raw := C.GoBytes(unsafe.Pointer(resultPtr), C.int(bufLen))
	var embeddings [][]float32
	if err := json.Unmarshal(raw, &embeddings); err != nil {
		return nil, fmt.Errorf("failed to parse embeddings: %w", err)
	}

	return embeddings, nil
}

func IndexDocumentWithResult(docID string, content []byte) ([]byte, error) {
	if docID == "" {
		return nil, fmt.Errorf("docID cannot be empty")
	}

	if err := IndexDocument(docID, content); err != nil {
		return nil, err
	}

	var bufLen C.size_t
	resultPtr := C.graphrag_get_last_extraction(&bufLen)
	if resultPtr == nil {
		return nil, fmt.Errorf("graphrag get last extraction returned null pointer")
	}
	defer C.free(unsafe.Pointer(resultPtr))

	return C.GoBytes(unsafe.Pointer(resultPtr), C.int(bufLen)), nil
}
