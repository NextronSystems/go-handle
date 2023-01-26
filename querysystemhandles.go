package handle

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SystemHandle is the OS based definition of SYSTEM_HANDLE_TABLE_ENTRY_INFO_EX
type SystemHandle struct {
	Object                uint3264
	UniqueProcessID       uint3264
	HandleValue           uint3264
	GrantedAccess         uint32
	CreatorBackTraceIndex uint16
	ObjectTypeIndex       uint16
	HandleAttributes      uint32
	_                     uint32
}

type systemHandleInformationEx struct {
	Count uint3264
	_     uint3264
	// ... followed by the specified number of handles
	Handles [1 << 20]SystemHandle
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
		0x40, // SystemExtendedHandleInformation
		unsafe.Pointer(&buf[0]),
		uint32(len(buf)),
		&requiredBuffer,
	); err != nil {
		if err == windows.STATUS_INFO_LENGTH_MISMATCH {
			return nil, InsufficientBufferError{requiredBuffer}
		}
		return nil, err
	}
	sysinfo := (*systemHandleInformationEx)(unsafe.Pointer(&buf[0]))
	var handles = sysinfo.Handles[:sysinfo.Count:sysinfo.Count]
	return handles, nil
}
