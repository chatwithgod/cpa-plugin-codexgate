// Native ABI shell adapted from CLIProxyAPI's MIT-licensed native Go example.
// See THIRD_PARTY_NOTICES.md and licenses/CLIProxyAPI-MIT.txt.
package main

/*
#include <stdint.h>
#include <stdlib.h>
typedef struct { void* ptr; size_t len; } cliproxy_buffer;
typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);
typedef struct {
    uint32_t abi_version; void* host_ctx;
    cliproxy_host_call_fn call; cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;
typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);
typedef struct {
    uint32_t abi_version; cliproxy_plugin_call_fn call;
    cliproxy_plugin_free_fn free_buffer; cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;
extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);
*/
import "C"

import (
	"encoding/json"
	"unsafe"

	"codexgate/internal/plugin"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
)

var manager plugin.Manager

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, api *C.cliproxy_plugin_api) C.int {
	if host == nil || api == nil || uint32(host.abi_version) != pluginabi.ABIVersion {
		return 1
	}
	api.abi_version = C.uint32_t(pluginabi.ABIVersion)
	api.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	api.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	api.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

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

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, length C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() { manager.Close() }
