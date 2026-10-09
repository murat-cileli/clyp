package main

/*
#cgo pkg-config: wayland-client
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

var errBackendUnavailable = errors.New("clipboard backend unavailable")

func runWayland() error {
	errorBuffer := make([]byte, 512)
	result := C.clyp_wayland_run((*C.char)(unsafe.Pointer(&errorBuffer[0])), C.size_t(len(errorBuffer)))
	if result == 0 {
		return nil
	}
	if result == 2 {
		return errBackendUnavailable
	}
	message := C.GoString((*C.char)(unsafe.Pointer(&errorBuffer[0])))
	if message == "" {
		message = "Wayland clipboard backend failed"
	}
	return errors.New(message)
}

//export goWaylandClipboardData
func goWaylandClipboardData(mimeType *C.char, data unsafe.Pointer, length C.size_t) {
	if activeClipboardHandler == nil || data == nil || length == 0 {
		return
	}
	activeClipboardHandler(C.GoString(mimeType), C.GoBytes(data, C.int(length)))
}
