package main

import (
	"fmt"
	"os"
	"time"

	"github.com/NextronSystems/go-handle"
	"golang.org/x/sys/windows"
)

func createEvent(name string) (windows.Handle, error) {
	u16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	h, err := windows.CreateEvent(nil, 0, 0, u16)
	if err != nil {
		return 0, err
	}
	return h, nil
}

func createMutex(name string) (windows.Handle, error) {
	u16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	h, err := windows.CreateMutex(nil, false, u16)
	if err != nil {
		return 0, err
	}
	return h, nil
}

func main() {
	// create an example global and local event
	eventHandle, err := createEvent(`Global\TestHandleEvent`)
	if err != nil {
		panic(err)
	}
	defer windows.CloseHandle(eventHandle)
	eventHandle2, err := createEvent(`Local\TestHandleEvent2`)
	if err != nil {
		panic(err)
	}
	defer windows.CloseHandle(eventHandle2)
	// create an example global and local mutex
	mutexHandle, err := createMutex(`Global\TestHandleMutex`)
	if err != nil {
		panic(err)
	}
	defer windows.CloseHandle(mutexHandle)
	mutexHandle2, err := createMutex(`Local\TestHandleMutex2`)
	if err != nil {
		panic(err)
	}
	defer windows.CloseHandle(mutexHandle2)
	// create an example file
	f, err := os.OpenFile("TestFile", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}
	defer os.Remove("TestFile")
	defer f.Close()
	// create 6MB buffer
	buf := make([]byte, 6000000)
	pid := uint16(os.Getpid())
	handles, err := handle.QueryHandles(buf, &pid, nil, time.Millisecond*500)
	if err != nil {
		panic(err)
	}
	for _, fh := range handles {
		fmt.Printf("PID %05d | %8.8s | HANDLE %04X | '%s'\n", fh.Process, fh.Type, fh.Handle, fh.Name)
	}
}
