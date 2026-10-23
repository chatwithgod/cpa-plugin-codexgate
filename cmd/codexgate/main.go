// Native ABI shell adapted from CLIProxyAPI's MIT-licensed native Go example.
// See THIRD_PARTY_NOTICES.md and licenses/CLIProxyAPI-MIT.txt.
package main

/*
#include "bridge.h"
*/
import "C"

import (
	"encoding/json"
	"unsafe"

	"codexgate/internal/plugin"
)

var manager plugin.Manager

const nativeABIVersion = uint32(C.CODEXGATE_ABI_VERSION)

func main() {}

func respond(out *C.cliproxy_buffer, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte(`{"ok":false,"error":{"code":"encode_error","message":"response encoding failed"}}`)
	}
	out.ptr, out.len = C.CBytes(raw), C.size_t(len(raw))
}

func fail(out *C.cliproxy_buffer, message string) {
	respond(out, map[string]any{"ok": false, "error": map[string]string{"code": "codexgate_error", "message": message}})
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, length C.size_t, out *C.cliproxy_buffer) (code C.int) {
	if out == nil {
		return 1
	}
	out.ptr, out.len = nil, 0
	defer func() {
		if recover() != nil {
			manager.Close()
			fail(out, "CodexGate internal error")
			code = 1
		}
	}()
	if method == nil || length > 128*1024*1024 || (request == nil && length > 0) {
		fail(out, "invalid ABI input")
		return 1
	}
	var raw []byte
	if length > 0 {
		raw = C.GoBytes(unsafe.Pointer(request), C.int(length))
	}
	result, err := manager.Handle(C.GoString(method), raw)
	if err != nil {
		fail(out, err.Error())
		return 1
	}
	respond(out, map[string]any{"ok": true, "result": result})
	return 0
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() { manager.Close() }
