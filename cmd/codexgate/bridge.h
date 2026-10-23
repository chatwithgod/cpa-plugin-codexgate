#ifndef CODEXGATE_BRIDGE_H
#define CODEXGATE_BRIDGE_H

#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

#define CODEXGATE_ABI_VERSION 1
typedef struct { void *ptr; size_t len; } cliproxy_buffer;
typedef int (*cliproxy_host_call_fn)(void *, const char *, const uint8_t *, size_t, cliproxy_buffer *);
typedef void (*cliproxy_host_free_fn)(void *, size_t);
typedef struct {
    uint32_t abi_version;
    void *host_ctx;
    cliproxy_host_call_fn call;
    cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;
typedef int (*cliproxy_plugin_call_fn)(char *, uint8_t *, size_t, cliproxy_buffer *);
typedef void (*cliproxy_plugin_free_fn)(void *, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);
typedef struct {
    uint32_t abi_version;
    cliproxy_plugin_call_fn call;
    cliproxy_plugin_free_fn free_buffer;
    cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

#ifdef _WIN32
#define CODEXGATE_EXPORT __declspec(dllexport)
#else
#define CODEXGATE_EXPORT __attribute__((visibility("default")))
#endif
CODEXGATE_EXPORT int cliproxy_plugin_init(cliproxy_host_api *, cliproxy_plugin_api *);

#endif
