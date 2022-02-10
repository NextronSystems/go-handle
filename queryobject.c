#include "queryobject.h"

#include <winternl.h>

typedef NTSTATUS NTAPI (*NtQueryObjectType)(HANDLE Handle,OBJECT_INFORMATION_CLASS ObjectInformationClass,PVOID ObjectInformation,ULONG ObjectInformationLength,PULONG ReturnLength);

int queryObjects(exchange_t* exchange) {
    HMODULE ntdll = LoadLibraryA("ntdll.dll");
    NtQueryObjectType ntQueryObject = (NtQueryObjectType) GetProcAddress(ntdll, "NtQueryObject");

    while(1) {
        if (WaitForSingleObject(exchange->ini, INFINITE) != WAIT_OBJECT_0) {
            return 1;
        }
        exchange->result = ntQueryObject(exchange->handle, exchange->informationClass, exchange->buffer, exchange->bufferLength, 0);
        SetEvent(exchange->done);
    }
    return 0;
}