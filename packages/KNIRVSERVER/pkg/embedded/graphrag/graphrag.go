package graphrag

/*
#cgo LDFLAGS: ${SRCDIR}/../../../build/embedded/graphrag/libgraphrag_core.a -ldl -pthread
#include "graphrag.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Init initializes the graphrag engine with provided configuration
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

// Shutdown gracefully stops the graphrag engine
func Shutdown() error {
	result := C.graphrag_shutdown()
	if result != 0 {
		return fmt.Errorf("graphrag shutdown failed with code: %d", result)
	}
	return nil
}

// HealthCheck returns current health status of the graphrag engine
func HealthCheck() error {
	result := C.graphrag_health_check()
	if result != 0 {
		return fmt.Errorf("graphrag health check failed with code: %d", result)
	}
	return nil
}

// IndexDocument indexes a single document into the graph
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

// Query executes a graph query and returns results
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
