package main

/*
#cgo pkg-config: x11 xfixes
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

func runX11() error {
	errorBuffer := make([]byte, 512)
	result := C.clyp_x11_run((*C.char)(unsafe.Pointer(&errorBuffer[0])), C.size_t(len(errorBuffer)))
	if result == 0 {
		return nil
	}
	message := C.GoString((*C.char)(unsafe.Pointer(&errorBuffer[0])))
	if message == "" {
		message = "X11 clipboard backend failed"
	}
	return errors.New(message)
}

//export goX11ClipboardData
func goX11ClipboardData(mimeType *C.char, data unsafe.Pointer, length C.size_t) {
	if activeClipboardHandler == nil || data == nil || length == 0 {
		return
	}
	activeClipboardHandler(C.GoString(mimeType), C.GoBytes(data, C.int(length)))
}
