#include <windows.h>
#include <stdint.h>

typedef struct {
    // Ini is waited upon by the native thread and is triggered when a valid handle and information class have been placed in the input buffers
    uintptr_t ini;
    // Ini is triggered by the native thread when NtQueryObject is finished and the output buffer has been filled
    uintptr_t done;

    // Input data for NtQueryObject
    uintptr_t handle;
    int informationClass;
    // Output buffer for NtQueryObject
    byte *buffer;
    int bufferLength;
    // NtQueryObject return value
    int result;
} exchange_t;

int queryObjects(exchange_t* exchange);