#include <windows.h>

typedef struct {
    // Ini is waited upon by the native thread and is triggered when a valid handle and information class have been placed in the input buffers
    HANDLE ini;
    // Ini is triggered by the native thread when NtQueryObject is finished and the output buffer has been filled
    HANDLE done;

    // Input data for NtQueryObject
    HANDLE handle;
    int informationClass;
    // Output buffer for NtQueryObject
    byte *buffer;
    int bufferLength;
    // NtQueryObject return value
    int result;
} exchange_t;

int queryObjects(exchange_t* exchange);