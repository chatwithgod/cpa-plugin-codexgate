#include "bridge.h"
#include "_cgo_export.h"

#if defined(__APPLE__) && defined(__x86_64__)
#include <pthread.h>

/* Darwin/amd64 Go runtimes use the same fixed TLS slot for g. Entering a
 * c-shared Go callback on CPA's Go thread can borrow the host's g and crash
 * during stack growth (reproduced against official CPA 8.0.38).
 * Run each callback on a fresh C-owned thread and join before returning.
 * No Go pointer or host callback crosses between the two runtimes here.
 * There is no daemon, persistent worker, or change to authentication policy.
 */
typedef struct {
    char *method;
    uint8_t *request;
    size_t length;
    cliproxy_buffer *out;
    int shutdown;
    int result;
} codexgate_task;

static void *run_task(void *value) {
    codexgate_task *task = (codexgate_task *)value;
    if (task->shutdown) {
        cliproxyPluginShutdown();
        task->result = 0;
    } else {
        task->result = cliproxyPluginCall(task->method, task->request, task->length, task->out);
    }
    return NULL;
}

static int dispatch(codexgate_task *task) {
    pthread_t thread;
    if (pthread_create(&thread, NULL, run_task, task) != 0) {
        return 1;
    }
    /* A valid joinable thread, joined exactly once by its creating caller. */
    if (pthread_join(thread, NULL) != 0) {
        abort(); /* Never return while a worker could still use caller memory. */
    }
    return task->result;
}
#endif

static int bridge_call(char *method, uint8_t *request, size_t length, cliproxy_buffer *out) {
    if (out == NULL) {
        return 1;
    }
    out->ptr = NULL;
    out->len = 0;
#if defined(__APPLE__) && defined(__x86_64__)
    codexgate_task task = {method, request, length, out, 0, 1};
    return dispatch(&task);
#else
    return cliproxyPluginCall(method, request, length, out);
#endif
}

static void bridge_free(void *ptr, size_t length) {
    (void)length;
    /* C.CBytes allocates with this library's C allocator; no Go entry needed. */
    free(ptr);
}

static void bridge_shutdown(void) {
#if defined(__APPLE__) && defined(__x86_64__)
    codexgate_task task = {NULL, NULL, 0, NULL, 1, 1};
    (void)dispatch(&task);
#else
    cliproxyPluginShutdown();
#endif
}

CODEXGATE_EXPORT int cliproxy_plugin_init(cliproxy_host_api *host, cliproxy_plugin_api *api) {
    if (host == NULL || api == NULL || host->abi_version != CODEXGATE_ABI_VERSION) {
        return 1;
    }
    api->abi_version = CODEXGATE_ABI_VERSION;
    api->call = bridge_call;
    api->free_buffer = bridge_free;
    api->shutdown = bridge_shutdown;
    return 0;
}
