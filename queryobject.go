package handle

// #include "queryobject.h"
import "C"
import (
	"time"
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

// ntQueryObject wraps NtQueryObject and supports a timeout logic.
// Because NtQueryObject can deadlock on specific handles, we do
// not want to call it directly. We also can't call it in a separate
// go routine because then that go routine might be permanently blocked.
//
// Instead, we use a native ntQueryThread that starts queryObjects and
// communicate with it via a CGO struct. If the response pipe times
// out, we assume a deadlock and kill the native ntQueryThread.
func (i *Inspector) ntQueryObject(h windows.Handle, informationClass int) (string, error) {
	if i.ntQueryThread.IsZero() {
		if err := windows.ResetEvent(windows.Handle(i.nativeExchange.ini)); err != nil {
			return "", err
		}
		if err := windows.ResetEvent(windows.Handle(i.nativeExchange.done)); err != nil {
			return "", err
		}
		var err error
		i.ntQueryThread, err = createNativeThread(uintptr(C.queryObjects), uintptr(unsafe.Pointer(i.nativeExchange)))
		if err != nil {
			return "", err
		}
	}
	i.nativeExchange.handle = C.HANDLE(h)
	i.nativeExchange.informationClass = C.int(informationClass)

	if err := windows.SetEvent(windows.Handle(i.nativeExchange.ini)); err != nil {
		return "", err
	}

	if s, err := windows.WaitForSingleObject(windows.Handle(i.nativeExchange.done), uint32(i.timeout/time.Millisecond)); s == uint32(windows.WAIT_TIMEOUT) || err != nil {
		i.ntQueryThread.Terminate()
		i.ntQueryThread = nativeThread{}
		return "", ErrTimeout
	}
	if i.nativeExchange.result != 0 {
		return "", windows.NTStatus(i.nativeExchange.result)
	}
	if informationClass == nameInformationClass {
		return (*objectNameInformation)(unsafe.Pointer(i.nativeExchange.buffer)).Name.String(), nil
	} else if informationClass == typeInformationClass {
		return (*objectTypeInformation)(unsafe.Pointer(i.nativeExchange.buffer)).TypeName.String(), nil
	} else {
		panic(informationClass)
	}
}
