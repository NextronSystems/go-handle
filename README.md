# go-handle
Iterate over Windows Handles

Currently supported handle types are:

```golang
const (
	HandleTypeFile   HandleType = "File"
	HandleTypeEvent  HandleType = "Event"
	HandleTypeMutant HandleType = "Mutant"
)
```

## Usage

Before querying the handles, create a buffer that can hold all process handles. On most machines, ~5MB should be fine.

Then call the function `handle.QueryHandles(...)`. This function takes the following arguments:

 - `buf`: A buffer that can hold all process handles
 - `processFilter` _(optional)_: Only show process handles of process with this id
 - `handleTypes` _(optional)_: Only return handles of the specified types _(see above for all available handle types)_

The function returns a list of handles. You can convert the generic handle to `FileHandle`, `EventHandle`, `...` or just use the generic functions `.Process()`, `.Handle()` and `.Name()`. 

## Example
The following example iterates over all file handles.

```golang
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
	handles, err := handle.QueryHandles(buf, &pid, []handle.HandleType{
		handle.HandleTypeMutant,
		handle.HandleTypeEvent,
		handle.HandleTypeFile,
	}, time.Second*20)
	if err != nil {
		panic(err)
	}
	for _, fh := range handles {
		fmt.Printf("[pid %d] +0x%03X handle '%s'\n", fh.Process(), fh.Handle(), fh.Name())
	}
}
```

`Z:\go\src\github.com\Codehardt\go-handle>go run examples/example.go`
```
[pid 7360] +0x004 handle '\Device\ConDrv'
[pid 7360] +0x008 handle ''
[pid 7360] +0x00C handle ''
[pid 7360] +0x03C handle ''
[pid 7360] +0x040 handle ''
[pid 7360] +0x044 handle '\Device\Mup\VBoxSvr\win10\go\src\github.com\Codehardt\go-handle'
[pid 7360] +0x048 handle '\Device\ConDrv'
[pid 7360] +0x0A8 handle '\Device\KsecDD'
[pid 7360] +0x0AC handle '\Device\CNG'
[pid 7360] +0x0B8 handle '\Device\DeviceApi'
[pid 7360] +0x0C4 handle ''
[pid 7360] +0x0D0 handle ''
[pid 7360] +0x0E4 handle ''
[pid 7360] +0x0E8 handle ''
[pid 7360] +0x0F0 handle ''
[pid 7360] +0x0FC handle ''
[pid 7360] +0x104 handle ''
[pid 7360] +0x110 handle '\BaseNamedObjects\TestHandleEvent'
[pid 7360] +0x114 handle '\Sessions\1\BaseNamedObjects\TestHandleEvent2'
[pid 7360] +0x118 handle '\BaseNamedObjects\TestHandleMutex'
[pid 7360] +0x11C handle '\Sessions\1\BaseNamedObjects\TestHandleMutex2'
[pid 7360] +0x120 handle '\Device\Mup\VBoxSvr\win10\go\src\github.com\Codehardt\go-handle\TestFile'
[pid 7360] +0x124 handle ''
[pid 7360] +0x178 handle '\Device\ConDrv'
[pid 7360] +0x18C handle '\Device\ConDrv'
[pid 7360] +0x190 handle '\Device\ConDrv'
```