package handle

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// #include "queryobject.h"
import "C"

// Inspector describes a structure that queries details (name and type name) to a specific handle.
// Common elements such as type ID to name mappings and process handles are cached and reused.
type Inspector struct {
	nativeExchange *C.exchange_t
	typeMapping    map[uint16]string
	processHandles map[uint32]windows.Handle
	timeout        time.Duration
	ntQueryThread  nativeThread
}

func NewInspector(timeout time.Duration) *Inspector {
	query := &Inspector{
		typeMapping:    map[uint16]string{},
		processHandles: map[uint32]windows.Handle{},
		timeout:        timeout,
		nativeExchange: &C.exchange_t{},
	}
	ini, _ := windows.CreateEvent(nil, 0, 0, nil)
	query.nativeExchange.ini = C.uintptr_t(ini)
	done, _ := windows.CreateEvent(nil, 0, 0, nil)
	query.nativeExchange.done = C.uintptr_t(done)

	return query
}

// Close the Inspector object, removing any cached data and stopping the native thread
func (i *Inspector) Close() {
	if !i.ntQueryThread.IsZero() {
		i.ntQueryThread.Terminate()
		i.ntQueryThread = nativeThread{}
	}
	windows.CloseHandle(windows.Handle(i.nativeExchange.ini))
	i.nativeExchange.ini = 0
	windows.CloseHandle(windows.Handle(i.nativeExchange.done))
	i.nativeExchange.done = 0
	for _, handle := range i.processHandles {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
	}
	i.processHandles = map[uint32]windows.Handle{}
}

var ownpid = uint3264(os.Getpid())

// LookupHandleType returns the type name for the handle. If possible, a cached type
// is used; otherwise, the handle is duplicated and its type is looked up.
func (i *Inspector) LookupHandleType(handle SystemHandle) (handleType string, err error) {
	handleType, knownType := i.typeMapping[handle.ObjectTypeIndex]
	if knownType {
		return handleType, nil
	}
	var h windows.Handle
	// duplicate handle if it's not from our own process
	if handle.UniqueProcessID != ownpid {
		h, err = i.duplicateHandle(handle)
		if err != nil {
			return "", fmt.Errorf("could not duplicate handle: %w", err)
		}
		defer windows.CloseHandle(h)
	} else {
		h = windows.Handle(handle.HandleValue)
	}
	handleType, err = i.ntQueryObject(h, typeInformationClass)
	i.typeMapping[handle.ObjectTypeIndex] = handleType
	if err != nil {
		return "", fmt.Errorf("could not query handle type: %w", err)
	}
	return
}

func (i *Inspector) LookupHandleName(handle SystemHandle) (name string, err error) {
	var h windows.Handle
	// duplicate handle if it's not from our own process
	if handle.UniqueProcessID != ownpid {
		h, err = i.duplicateHandle(handle)
		if err != nil {
			return "", fmt.Errorf("could not duplicate handle: %w", err)
		}
		defer windows.CloseHandle(h)
	} else {
		h = windows.Handle(handle.HandleValue)
	}
	name, err = i.ntQueryObject(h, nameInformationClass)
	return
}

// duplicateHandle duplicates a handle into our own process. It uses a cache for
// process handles that are used repeatedly.
func (i *Inspector) duplicateHandle(handle SystemHandle) (windows.Handle, error) {
	pid := uint32(handle.UniqueProcessID)
	p, hasCachedHandle := i.processHandles[pid]
	if !hasCachedHandle {
		var err error
		p, err = windows.OpenProcess(
			windows.PROCESS_DUP_HANDLE,
			true,
			pid,
		)
		i.processHandles[pid] = p
		if err != nil {
			return 0, err
		}
	} else if p == 0 { // Error was cached
		return 0, errors.New("failed to open process")
	}
	var h windows.Handle
	if err := windows.DuplicateHandle(
		p,
		windows.Handle(handle.HandleValue),
		windows.CurrentProcess(),
		&h,
		0,
		false,
		windows.DUPLICATE_SAME_ACCESS,
	); err != nil {
		return 0, err
	}
	return h, nil
}

var ErrTimeout = errors.New("NtQueryObject deadlocked")
