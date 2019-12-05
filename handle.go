//+build windows

package handle

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modntdll                     = syscall.NewLazyDLL("ntdll.dll")
	procNtQuerySystemInformation = modntdll.NewProc("NtQuerySystemInformation")
	procNtQueryObject            = modntdll.NewProc("NtQueryObject")
)

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

type systemHandle struct {
	UniqueProcessID       uint16
	CreatorBackTraceIndex uint16
	ObjectTypeIndex       uint8
	HandleAttributes      uint8
	HandleValue           uint16
	Object                uint3264
	GrantedAccess         uint3264
}

type systemHandleInformation struct {
	Count        uint3264
	SystemHandle []systemHandle
}

type objectTypeInformation struct {
	TypeName unicodeString
	_        [22]uint64 // unused
}

type objectNameInformation struct {
	Name unicodeString
}

func NtSuccess(rt uint32) bool {
	return rt < 0x8000000
}

type Handle struct {
	Process uint16
	Handle  uint16
	Name    string
	Type    string
}

func QueryHandles(buf []byte, processFilter *uint16, handleTypes []string, queryTimeout time.Duration) (handles []Handle, err error) {
	// reset buffer, querying system information seem to require a 0-valued buffer.
	// Without this reset, the below sysinfo.Count might be wrong.
	for i := 0; i < len(buf); i++ {
		buf[i] = 0
	}
	ownpid := uint16(os.Getpid())
	ownprocess, err := windows.GetCurrentProcess()
	if err != nil {
		return nil, fmt.Errorf("could not get current process: %s", err)
	}
	defer windows.CloseHandle(ownprocess)
	// load all handle information to buffer and convert it to systemHandleInformation
	if err := querySystemInformation(buf); err != nil {
		return nil, fmt.Errorf("could not query system information: %s", err)
	}
	sysinfo := (*systemHandleInformation)(unsafe.Pointer(&buf[0]))
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&sysinfo.SystemHandle))
	sh.Data = uintptr(unsafe.Pointer(&buf[int(unsafe.Sizeof(sysinfo.Count))]))
	sh.Len = int(sysinfo.Count)
	sh.Cap = int(sysinfo.Count)
	var (
		typeMapping    = make(map[uint8]string) // what objecttypeindex equals which handletype
		typeMappingErr = make(map[uint8]int)
		typeFilter     map[string]struct{}
		processErrs    = make(map[uint16]struct{})
	)
	if len(handleTypes) > 0 {
		typeFilter = make(map[string]struct{})
		for _, handleType := range handleTypes {
			typeFilter[handleType] = struct{}{}
		}
	}
	log("type filter: %#v", typeFilter)
	log("sysinfo count: %d", sysinfo.Count)
	for i := uint3264(0); i < sysinfo.Count; i++ {
		handle := sysinfo.SystemHandle[i]
		// some handles cause freeze, skip them
		if (handle.GrantedAccess == 0x0012019f) ||
			(handle.GrantedAccess == 0x001a019f) ||
			(handle.GrantedAccess == 0x00120189) ||
			(handle.GrantedAccess == 0x00100000) {
			continue
		}
		if processFilter != nil && *processFilter != handle.UniqueProcessID {
			log("skipping handle of process %d due to process filter %d", handle.UniqueProcessID, processFilter)
			continue
		}
		if _, ok := processErrs[handle.UniqueProcessID]; ok {
			continue
		}
		// unknown type, query the type information
		if _, ok := typeMapping[handle.ObjectTypeIndex]; !ok {
			log("handle type %d of handle 0x%X is unknown, querying for type ...", handle.UniqueProcessID, handle.HandleValue)
			done := make(chan struct{}, 1)
			var (
				handleTypeRoutine string
				errRoutine        error
			)
			go func() {
				handleTypeRoutine, errRoutine = queryTypeInformation(handle, ownprocess, ownpid == handle.UniqueProcessID)
				done <- struct{}{}
			}()
			select {
			case <-done:
				if errRoutine == errOpenProcess {
					log("skipping process %d due to open error", handle.UniqueProcessID)
					processErrs[handle.UniqueProcessID] = struct{}{}
					continue
				}
				if errRoutine != nil {
					log("handle type %d could not be queried: %s", handle.ObjectTypeIndex, errRoutine)
					// to prevent querying tons of types that can't be queries, count errors per
					// handle type and ignore this type if more than X tries failed.
					typeMappingErr[handle.ObjectTypeIndex]++
					if typeMappingErr[handle.ObjectTypeIndex] >= 10 {
						typeMapping[handle.ObjectTypeIndex] = "unknown"
					}
				} else {
					log("handle type %d is of type %s", handle.ObjectTypeIndex, handleTypeRoutine)
					typeMapping[handle.ObjectTypeIndex] = handleTypeRoutine
				}
			case <-time.After(queryTimeout):
				log("timeout when querying process %d handle 0x%X with granted access 0x%X", handle.ObjectTypeIndex, handle.HandleValue, handle.GrantedAccess)
				continue
			}
		}
		handleType := typeMapping[handle.ObjectTypeIndex]
		if typeFilter != nil {
			if _, ok := typeFilter[handleType]; !ok {
				// handle type not in filter list, skip
				log("skipping handle type %q due to handle filters", handleType)
				continue
			}
		}
		switch handleType {
		default:
			done := make(chan struct{}, 1)
			var (
				nameRoutine string
				errRoutine  error
			)
			go func() {
				nameRoutine, errRoutine = queryNameInformation(handle, ownprocess, ownpid == handle.UniqueProcessID)
				done <- struct{}{}
			}()
			var name string
			select {
			case <-done:
				if errRoutine != nil {
					log("could not get handle name for handle 0x%X of type %s: %s", handle.HandleValue, handleType, errRoutine)
				} else {
					name = nameRoutine
				}
			case <-time.After(queryTimeout):
				log("timeout when querying for handle name of process %d's handle 0x%X (type %s) and granted access 0x%X", handle.UniqueProcessID, handle.HandleValue, handleType, handle.GrantedAccess)
			}
			handle := Handle{Process: handle.UniqueProcessID, Handle: handle.HandleValue, Name: name, Type: handleType}
			log("handle found: process: %d handle: 0x%X name: %10.10s type: %s", handle.Process, handle.Handle, handle.Name, handle.Type)
			// add handle to result set
			handles = append(handles, handle)
		}
	}
	runtime.KeepAlive(buf)
	return handles, nil
}

func querySystemInformation(buf []byte) error {
	ret, _, _ := procNtQuerySystemInformation.Call(
		16,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if !NtSuccess(uint32(ret)) {
		return fmt.Errorf("NTStatus(0x%X)", ret)
	}
	return nil
}

var nameAndTypeBuffer = make([]byte, 4096)
var errOpenProcess = errors.New("could not open process")

func queryTypeInformation(handle systemHandle, ownprocess windows.Handle, ownpid bool) (string, error) {
	// duplicate handle if it's not from our own process
	var h windows.Handle
	if !ownpid {
		p, err := windows.OpenProcess(
			windows.PROCESS_DUP_HANDLE,
			true,
			uint32(handle.UniqueProcessID),
		)
		if err != nil {
			log("could not open process %d: %s", handle.UniqueProcessID, err)
			return "", errOpenProcess
		}
		defer windows.CloseHandle(p)
		if err := windows.DuplicateHandle(
			p,
			windows.Handle(handle.HandleValue),
			ownprocess,
			&h,
			0,
			false,
			windows.DUPLICATE_SAME_ACCESS,
		); err != nil {
			log("could not duplicate process handle 0x%X of process %d: %s", handle.HandleValue, handle.UniqueProcessID, err)
			return "", err
		}
		defer windows.CloseHandle(h)
	} else {
		h = windows.Handle(handle.HandleValue)
	}
	ret, _, _ := procNtQueryObject.Call(
		uintptr(h), 2,
		uintptr(unsafe.Pointer(&nameAndTypeBuffer[0])),
		uintptr(len(nameAndTypeBuffer)),
		0,
	)
	if !NtSuccess(uint32(ret)) {
		return "", fmt.Errorf("NTStatus(0x%X)", ret)
	}
	name := (*objectTypeInformation)(unsafe.Pointer(&nameAndTypeBuffer[0])).TypeName.String()
	runtime.KeepAlive(nameAndTypeBuffer)
	return name, nil
}

func queryNameInformation(handle systemHandle, ownprocess windows.Handle, ownpid bool) (string, error) {
	// duplicate handle if it's not from our own process
	var h windows.Handle
	if !ownpid {
		p, err := windows.OpenProcess(
			windows.PROCESS_DUP_HANDLE,
			true,
			uint32(handle.UniqueProcessID),
		)
		if err != nil {
			return "", err
		}
		defer windows.CloseHandle(p)
		if err := windows.DuplicateHandle(
			p,
			windows.Handle(handle.HandleValue),
			ownprocess,
			&h,
			0,
			false,
			windows.DUPLICATE_SAME_ACCESS,
		); err != nil {
			return "", err
		}
		defer windows.CloseHandle(h)
	} else {
		h = windows.Handle(handle.HandleValue)
	}
	log("query (access 0x%X)", handle.GrantedAccess)
	ret, _, _ := procNtQueryObject.Call(
		uintptr(h),
		1,
		uintptr(unsafe.Pointer(&nameAndTypeBuffer[0])),
		uintptr(len(nameAndTypeBuffer)),
		0,
	)
	if !NtSuccess(uint32(ret)) {
		return "", fmt.Errorf("NTStatus(0x%X)", ret)
	}
	name := (*objectNameInformation)(unsafe.Pointer(&nameAndTypeBuffer[0])).Name.String()
	runtime.KeepAlive(nameAndTypeBuffer)
	return name, nil
}

func (u unicodeString) String() string {
	var s []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	hdr.Data = uintptr(unsafe.Pointer(u.Buffer))
	hdr.Len = int(u.Length / 2)
	hdr.Cap = int(u.MaximumLength / 2)
	log("converting unicode string with length %d and capacity %d", u.Length, u.MaximumLength)
	return string(utf16.Decode(s))
}

var writer io.Writer

// DebugWriter sets a debug writer for debug logging, e.g. os.Stdout
func DebugWriter(w io.Writer) {
	writer = w
}

func log(format string, a ...interface{}) {
	if writer == nil {
		return
	}
	fmt.Fprintf(writer, format+"\n", a...)
}
