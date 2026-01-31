package mlc

/*
#cgo LDFLAGS: -L/usr/local/lib -lmlc_llm -Wl,-rpath,/usr/local/lib
#cgo CFLAGS: -I${SRCDIR}/../../third_party/include
#include <stdlib.h>

// Note: You must place the MLC-LLM header files (like mlc_chat.h) 
// in your repo at /third_party/include/ so the compiler can see them.
#include "mlc_chat.h" 
*/
import "C"
import (
	"unsafe"
	"fmt"
)

type MLCEngine struct {
	handle C.MLCChatBackendHandle
}

func NewEngine(modelPath string) (*MLCEngine, error) {
	cPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cPath))

	// Calling the actual C function from libmlc_llm.so
	handle := C.MLCChatBackendCreate(cPath)
	if handle == nil {
		return nil, fmt.Errorf("failed to initialize MLC backend")
	}
	return &MLCEngine{handle: handle}, nil
}