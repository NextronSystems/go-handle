package handle

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

type objectTypeInformation struct {
	TypeName windows.NTUnicodeString
	_        [22]uint64 // unused
}

type objectNameInformation struct {
	Name windows.NTUnicodeString
}

const (
	nameInformationClass = iota + 1
	typeInformationClass
)

func ntQueryObjectName(handle windows.Handle) (string, error) {
	buf, err := ntQueryObject(handle, nameInformationClass)
	if err != nil {
		return "", err
	}
	name := (*objectNameInformation)(unsafe.Pointer(&buf[0])).Name.String()
	runtime.KeepAlive(buf)
	return name, nil
}

func ntQueryObjectType(handle windows.Handle) (string, error) {
	buf, err := ntQueryObject(handle, typeInformationClass)
	if err != nil {
		return "", err
	}
	name := (*objectTypeInformation)(unsafe.Pointer(&buf[0])).TypeName.String()
	runtime.KeepAlive(buf)
	return name, nil
}

var (
	modntdll          = windows.NewLazyDLL("ntdll.dll")
	procNtQueryObject = modntdll.NewProc("NtQueryObject")
)

func NtSuccess(rt uint32) bool {
	return rt < 0x8000000
}

func ntQueryObject(handle windows.Handle, informationClass int) ([]byte, error) {
	buf := make([]byte, 0x1000)
	ret, _, _ := procNtQueryObject.Call(
		uintptr(handle),
		uintptr(informationClass),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if !NtSuccess(uint32(ret)) {
		return nil, windows.NTStatus(ret)
	}
	return buf, nil
}
