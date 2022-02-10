package handle

import (
	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazyDLL("kernel32.dll")

	createThread    = kernel32.NewProc("CreateThread")
	terminateThread = kernel32.NewProc("TerminateThread")
)

type nativeThread struct {
	handle windows.Handle
}

func createNativeThread(callback uintptr, param uintptr) (nativeThread, error) {
	var thread nativeThread
	h, _, err := createThread.Call(
		0,
		0,
		callback,
		param,
		0,
		0,
	)
	if h == 0 {
		return nativeThread{}, err
	}
	thread.handle = windows.Handle(h)
	return thread, nil
}

func (t nativeThread) Terminate() error {
	r1, _, err := terminateThread.Call(
		uintptr(t.handle),
		0,
	)
	windows.CloseHandle(t.handle)
	if r1 == 0 {
		return err
	}
	return nil
}

func (t nativeThread) IsZero() bool {
	return t.handle == 0
}
