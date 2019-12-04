# go-handle
Iterate over Windows Handles

## Usage

Before querying the handles, create a buffer that can hold all process handles. On most machines, ~5MB should be fine.

Then call the function `handle.QueryHandles(...)`. This function takes the following arguments:

 - `buf`: A buffer that can hold all process handles
 - `processFilter` _(optional)_: Only show process handles of process with this id
 - `handleTypes` _(optional)_: Only return handles of the specified types, e.g. `File`, `Event` or `Mutant`
 - `queryTimeout`: Some handles can not be queried and cause a freeze. This timeout will be used to prevent freezes

The function returns a list of handles.

## Example
The following example iterates over all handles of this process.

```golang
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
handles, err := handle.QueryHandles(buf, &pid, []string{"File", "Event", "Mutant"}, time.Second*20)
if err != nil {
	panic(err)
}
for _, fh := range handles {
	fmt.Printf("PID %05d | %8.8s | HANDLE %04X | '%s'\n", fh.Process, fh.Type, fh.Handle, fh.Name)
}
```

`Z:\go\src\github.com\Codehardt\go-handle>go run examples/example.go`
```
PID 08108 |     File | HANDLE 0004 | '\Device\ConDrv'
PID 08108 |    Event | HANDLE 0008 | ''
PID 08108 |    Event | HANDLE 000C | ''
PID 08108 |    Event | HANDLE 003C | ''
PID 08108 |    Event | HANDLE 0040 | ''
PID 08108 |     File | HANDLE 0044 | '\Device\Mup\VBoxSvr\win10\'
PID 08108 |     File | HANDLE 0048 | '\Device\ConDrv'
PID 08108 |     File | HANDLE 00A0 | '\Device\DeviceApi'
PID 08108 |     File | HANDLE 00AC | '\Device\KsecDD'
PID 08108 |     File | HANDLE 00B0 | '\Device\CNG'
PID 08108 |    Event | HANDLE 00B4 | ''
PID 08108 |    Event | HANDLE 00C4 | ''
PID 08108 |    Event | HANDLE 00DC | ''
PID 08108 |    Event | HANDLE 00E8 | ''
PID 08108 |    Event | HANDLE 00F0 | ''
PID 08108 |    Event | HANDLE 00FC | ''
PID 08108 |    Event | HANDLE 0104 | ''
PID 08108 |    Event | HANDLE 0110 | '\BaseNamedObjects\TestHandleEvent'
PID 08108 |    Event | HANDLE 0114 | '\Sessions\1\BaseNamedObjects\TestHandleEvent2'
PID 08108 |   Mutant | HANDLE 0118 | '\BaseNamedObjects\TestHandleMutex'
PID 08108 |   Mutant | HANDLE 011C | '\Sessions\1\BaseNamedObjects\TestHandleMutex2'
PID 08108 |     File | HANDLE 0120 | '\Device\Mup\VBoxSvr\win10\TestFile'
PID 08108 |    Event | HANDLE 0124 | ''
PID 08108 |     File | HANDLE 018C | '\Device\ConDrv'
PID 08108 |     File | HANDLE 01BC | '\Device\ConDrv'
PID 08108 |     File | HANDLE 01DC | '\Device\ConDrv'
```