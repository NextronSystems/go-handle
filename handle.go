//+build windows

package handle

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
)

type HandleType string

const (
	HandleTypeFile   HandleType = "File"
	HandleTypeEvent  HandleType = "Event"
	HandleTypeMutant HandleType = "Mutant"
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

type Handle interface {
	Process() uint16
	Handle() uint16
	Name() string
}

type basicHandle struct {
	p uint16
	h uint16
	n string
}

func (b basicHandle) Process() uint16 { return b.p }
func (b basicHandle) Handle() uint16  { return b.h }
func (b basicHandle) Name() string    { return b.n }

type FileHandle struct {
	basicHandle
}

type EventHandle struct {
	basicHandle
}

type MutantHandle struct {
	basicHandle
}

func QueryHandles(buf []byte, processFilter *uint16, handleTypes []HandleType) (handles []Handle, err error) {
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
		typeMapping    = make(map[uint8]HandleType) // what objecttypeindex equals which handletype
		typeMappingErr = make(map[uint8]int)
		typeFilter     = make(map[HandleType]struct{})
		processErrs    = make(map[uint16]struct{})
	)
	if len(handleTypes) == 0 {
		// use all handle types if no handle type filter is set
		typeFilter[HandleTypeFile] = struct{}{}
		typeFilter[HandleTypeEvent] = struct{}{}
		typeFilter[HandleTypeMutant] = struct{}{}
	} else {
		for _, handleType := range handleTypes {
			typeFilter[handleType] = struct{}{}
		}
	}
	log("type filter: %#v", typeFilter)
	log("sysinfo count: %d", sysinfo.Count)
	for i := uint3264(0); i < sysinfo.Count; i++ {
		handle := sysinfo.SystemHandle[i]
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
			handleType, err := queryTypeInformation(handle, ownprocess, ownpid == handle.UniqueProcessID)
			if err == errOpenProcess {
				log("skipping process %d due to open error", handle.UniqueProcessID)
				processErrs[handle.UniqueProcessID] = struct{}{}
				continue
			}
			if err != nil {
				log("handle type %d could not be queried: %s", handle.ObjectTypeIndex, err)
				// to prevent querying tons of types that can't be queries, count errors per
				// handle type and ignore this type if more than X tries failed.
				typeMappingErr[handle.ObjectTypeIndex]++
				if typeMappingErr[handle.ObjectTypeIndex] >= 10 {
					typeMapping[handle.ObjectTypeIndex] = "unknown"
				}
			} else {
				log("handle type %d is of type %s", handle.ObjectTypeIndex, handleType)
				typeMapping[handle.ObjectTypeIndex] = handleType
			}
		}
		handleType := typeMapping[handle.ObjectTypeIndex]
		if _, ok := typeFilter[handleType]; !ok {
			// handle type not in filter list, skip
			log("skipping handle type %q due to handle filters", handleType)
			continue
		}
		switch handleType {
		case HandleTypeFile, HandleTypeEvent, HandleTypeMutant:
			// get name of handle (same for file, event and mutant)
			name, err := queryNameInformation(handle, ownprocess, ownpid == handle.UniqueProcessID)
			if err != nil {
				log("could not get handle name for handle 0x%X of type %s: %s", handle.HandleValue, handleType, err)
				name = ""
			}
			basic := basicHandle{p: handle.UniqueProcessID, h: handle.HandleValue, n: name}
			log("handle found: process: %d handle: 0x%X name: %s", basic.p, basic.h, basic.n)
			// add handle to result set
			switch handleType {
			case HandleTypeFile:
				handles = append(handles, &FileHandle{basic})
			case HandleTypeEvent:
				handles = append(handles, &EventHandle{basic})
			case HandleTypeMutant:
				handles = append(handles, &MutantHandle{basic})
			}
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

func queryTypeInformation(handle systemHandle, ownprocess windows.Handle, ownpid bool) (HandleType, error) {
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
	name := HandleType((*objectTypeInformation)(unsafe.Pointer(&nameAndTypeBuffer[0])).TypeName.String())
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
	var b []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	hdr.Data = uintptr(unsafe.Pointer(u.Buffer))
	hdr.Len = int(u.Length)
	hdr.Cap = int(u.MaximumLength)
	utf := make([]uint16, (len(b)+(2-1))/2)
	for i := 0; i+(2-1) < len(b); i += 2 {
		utf[i/2] = binary.LittleEndian.Uint16(b[i:])
	}
	if len(b)/2 < len(utf) {
		utf[len(utf)-1] = utf8.RuneError
	}
	runtime.KeepAlive(u.Buffer)
	return strings.Trim(string(utf16.Decode(utf)), "\x00")
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
