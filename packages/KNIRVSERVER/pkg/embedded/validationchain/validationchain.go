package validationchain

/*
#cgo LDFLAGS: ${SRCDIR}/../../../build/embedded/validation_chain/libvalidationchain.a -ldl -pthread
#include "validationchain.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Init initializes validation chain engine
func Init(config []byte) error {
	if len(config) == 0 {
		return fmt.Errorf("config cannot be empty")
	}

	cConfig := C.CString(string(config))
	defer C.free(unsafe.Pointer(cConfig))

	result := C.validation_chain_init(cConfig, C.size_t(len(config)))
	if result != 0 {
		return fmt.Errorf("validation chain init failed with code: %d", result)
	}

	return nil
}

// Shutdown gracefully stops validation chain
func Shutdown() error {
	result := C.validation_chain_shutdown()
	if result != 0 {
		return fmt.Errorf("validation chain shutdown failed with code: %d", result)
	}
	return nil
}

// HealthCheck returns current health status
func HealthCheck() error {
	result := C.validation_chain_health_check()
	if result != 0 {
		return fmt.Errorf("validation chain health check failed with code: %d", result)
	}
	return nil
}

// ValidateTransaction validates a single transaction
func ValidateTransaction(txData []byte) (bool, []byte, error) {
	if len(txData) == 0 {
		return false, nil, fmt.Errorf("transaction data cannot be empty")
	}

	cTxData := C.CString(string(txData))
	defer C.free(unsafe.Pointer(cTxData))

	var resultCode C.int
	var resultLen C.size_t
	resultPtr := C.validation_chain_validate(cTxData, C.size_t(len(txData)), &resultCode, &resultLen)

	if resultPtr == nil {
		return false, nil, fmt.Errorf("validation returned null pointer")
	}
	defer C.free(unsafe.Pointer(resultPtr))

	resultBytes := C.GoBytes(unsafe.Pointer(resultPtr), C.int(resultLen))
	return resultCode == 0, resultBytes, nil
}
