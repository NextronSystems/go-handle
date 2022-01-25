package handle

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

// Inspector describes a structure that queries details (name and type name) to a specific handle.
// Common elements such as type ID to name mappings and process handles are cached and reused.
type Inspector struct {
	queryId        int // unique identifier for this object, used to identify object to native thread
	typeMapping    map[uint8]string
	processHandles map[uint16]windows.Handle
	timeout        time.Duration
	ntQueryThread  nativeThread
}

func NewInspector(timeout time.Duration) *Inspector {
	queryMapMutex.Lock()
	defer queryMapMutex.Unlock()
	query := &Inspector{
		queryId:        nextQueryId,
		typeMapping:    map[uint8]string{},
		processHandles: map[uint16]windows.Handle{},
		timeout:        timeout,
	}
	nextQueryId++
	queryMap[query.queryId] = ioChannel{
		input:  make(chan ntObjectQuery),
		output: make(chan interface{}),
	}
	return query
}

// ioChannel describes a pair of channels that is used to communicate with the native thread that calls NtQueryObject.
// The input channel receives each parameter pair for NtQueryObject and the native thread responds with the name or
// type name, if successful, or and error value if not successful.
type ioChannel struct {
	input  chan ntObjectQuery
	output chan interface{}
}

// maps the unique ID of each Inspector object to its native thread communication channels.
var queryMap = map[int]ioChannel{}
var queryMapMutex sync.Mutex
var nextQueryId int

// Close the Inspector object, removing any cached data and stopping the native thread
func (i *Inspector) Close() {
	queryMapMutex.Lock()
	defer queryMapMutex.Unlock()
	close(queryMap[i.queryId].input) // Implicitly causes the native thread to exit if it is running
	delete(queryMap, i.queryId)
	for _, handle := range i.processHandles {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
	}
}

var ownpid = uint16(os.Getpid())

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
	p, hasCachedHandle := i.processHandles[handle.UniqueProcessID]
	if !hasCachedHandle {
		var err error
		p, err = windows.OpenProcess(
			windows.PROCESS_DUP_HANDLE,
			true,
			uint32(handle.UniqueProcessID),
		)
		i.processHandles[handle.UniqueProcessID] = p
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

// ntObjectQuery describes the parameters for a single call to NtQueryObject.
type ntObjectQuery struct {
	informationClass int
	handle           windows.Handle
}

var ErrTimeout = errors.New("NtQueryObject deadlocked")

// ntQueryObject wraps NtQueryObject and supports a timeout logic.
// Because NtQueryObject can deadlock on specific handles, we do
// not want to call it directly. We also can't call it in a separate
// go routine because then that go routine might be permanently blocked.
//
// Instead, we use a native ntQueryThread that starts queryObjects and
// communicate with it via a pair of pipes. If the response pipe times
// out, we assume a deadlock and kill the native ntQueryThread.
func (i *Inspector) ntQueryObject(h windows.Handle, informationClass int) (handleType string, err error) {
	if i.ntQueryThread.IsZero() {
		i.ntQueryThread, err = createNativeThread(queryObjectsCallback, uintptr(i.queryId))
		if err != nil {
			return "", err
		}
	}
	queryMap[i.queryId].input <- ntObjectQuery{
		informationClass: informationClass,
		handle:           h,
	}
	select {
	case result := <-queryMap[i.queryId].output:
		if err, isErr := result.(error); isErr {
			return "", err
		} else {
			return result.(string), nil
		}
	case <-time.After(i.timeout):
		i.ntQueryThread.Terminate()
		return "", ErrTimeout
	}
}

var queryObjectsCallback = windows.NewCallback(queryObjects)

// queryObjects runs a loop where it receives handles on an input channel
// and sends the results on an output channel. The channel pair is identified
// by the passed ID. This is meant to be run in a native ntQueryThread to be able to
// catch NtQueryObject deadlocks.
func queryObjects(id uintptr) uintptr {
	queryMapMutex.Lock()
	channels := queryMap[int(id)]
	queryMapMutex.Unlock()
	for query := range channels.input {
		var (
			result string
			err    error
		)
		switch query.informationClass {
		case nameInformationClass:
			result, err = ntQueryObjectName(query.handle)
		case typeInformationClass:
			result, err = ntQueryObjectType(query.handle)
		}
		if err != nil {
			channels.output <- err
		} else {
			channels.output <- result
		}
	}
	return 0
}
