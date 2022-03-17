package handle

import (
	"fmt"
	"reflect"
	"unsafe"

	"golang.org/x/sys/windows"
)

type SystemHandle struct {
	UniqueProcessID       uint16
	CreatorBackTraceIndex uint16
	ObjectTypeIndex       uint8
	HandleAttributes      uint8
	HandleValue           uint16
	Object                uint3264
	GrantedAccess         uint3264
}

type systemHandleInformation struct {
	Count uint3264
	// ... followed by the specified number of handles
}

type InsufficientBufferError struct {
	RequiredBufferSize uint32
}

func (i InsufficientBufferError) Error() string {
	return fmt.Sprintf("a buffer of at least %d bytes is required", i.RequiredBufferSize)
}

func NtQuerySystemHandles(buf []byte) ([]SystemHandle, error) {
	// reset buffer, querying system information seem to require a 0-valued buffer.
	// Without this reset, the below sysinfo.Count might be wrong.
	for i := 0; i < len(buf); i++ {
		buf[i] = 0
	}
	// load all handle information to buffer and convert it to systemHandleInformation
	var requiredBuffer uint32
	if err := windows.NtQuerySystemInformation(
		16,
		unsafe.Pointer(&buf[0]),
		uint32(len(buf)),
		&requiredBuffer,
	); err != nil {
		if err == windows.STATUS_INFO_LENGTH_MISMATCH {
			return nil, InsufficientBufferError{requiredBuffer}
		}
		return nil, err
	}
	sysinfo := (*systemHandleInformation)(unsafe.Pointer(&buf[0]))
	var handles []SystemHandle
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&handles))
	sh.Data = uintptr(unsafe.Pointer(&buf[int(unsafe.Sizeof(sysinfo.Count))]))
	sh.Len = int(sysinfo.Count)
	sh.Cap = int(sysinfo.Count)
	return handles, nil
}
