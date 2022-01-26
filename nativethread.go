package handle

import (
	"math"
	"time"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazyDLL("kernel32.dll")

	createThread    = kernel32.NewProc("CreateThread")
	terminateThread = kernel32.NewProc("TerminateThread")
)

type nativeThread struct {
	handle windows.Handle
	done   chan struct{}
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
	// If a native thread is running, the runtime might detect a deadlock if we wait for results from the native thread.
	// To circumvent this, we run a background routine that does nothing, but is technically not dead yet.
	thread.done = make(chan struct{})
	go func() {
		for {
			select {
			case <-thread.done:
				return
			case <-time.After(time.Duration(math.MaxInt64)):
			}
		}
	}()
	return thread, nil
}

func (t nativeThread) Terminate() error {
	close(t.done)
	r1, _, err := terminateThread.Call(
		uintptr(t.handle),
		0,
	)
	if r1 == 0 {
		return err
	}
	windows.CloseHandle(t.handle)
	t.handle = 0
	return nil
}

func (t nativeThread) IsZero() bool {
	return t.handle == 0
}
