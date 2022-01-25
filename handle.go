//+build windows

package handle

import (
	"fmt"
	"io"
	"runtime"
	"time"
)

type Handle struct {
	Process uint16
	Handle  uint16
	Name    string
	Type    string
}

func QueryHandles(buf []byte, processFilter *uint16, handleTypes []string, queryTimeout time.Duration) (handles []Handle, err error) {
	systemHandles, err := NtQuerySystemHandles(buf)
	if err != nil {
		return nil, err
	}
	var typeFilter map[string]struct{}
	if len(handleTypes) > 0 {
		typeFilter = make(map[string]struct{})
		for _, handleType := range handleTypes {
			typeFilter[handleType] = struct{}{}
		}
	}
	log("type filter: %#v", typeFilter)
	log("handle count: %d", len(systemHandles))
	inspector := NewInspector(queryTimeout)
	defer inspector.Close()
	for _, handle := range systemHandles {
		log("handle: %#v", handle)
		if processFilter != nil && *processFilter != handle.UniqueProcessID {
			log("skipping handle of process %d due to process filter %d", handle.UniqueProcessID, processFilter)
			continue
		}
		handleType, err := inspector.LookupHandleType(handle)
		if err != nil {
			log("could not query handle type for handle %d in process %d with access mask %d, error: %v", handle.HandleValue, handle.UniqueProcessID, handle.GrantedAccess, err)
			continue
		}
		if typeFilter != nil {
			if _, isTargetType := typeFilter[handleType]; !isTargetType {
				continue
			}
		}
		name, err := inspector.LookupHandleName(handle)
		if err != nil {
			log("could not query handle name for handle %d in process %d with access mask %d, error: %v", handle.HandleValue, handle.UniqueProcessID, handle.GrantedAccess, err)
			continue
		}
		handles = append(handles, Handle{
			Process: handle.UniqueProcessID,
			Handle:  handle.HandleValue,
			Name:    name,
			Type:    handleType,
		})
	}
	runtime.KeepAlive(buf)
	return handles, nil
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
