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
package main

import (
	"log"

	"github.com/Codehardt/go-handle"
)

func main() {
	buf := make([]byte, 6000000) // create 6MB buffer
	handles, err := handle.QueryHandles(buf, nil, []handle.HandleType{handle.HandleTypeFile})
	if err != nil {
		log.Fatal(err)
	}
	for i, h := range handles {
		if fh, ok := h.(*handle.FileHandle); ok {
			log.Printf("file handle 0x%04X for process %05d with name '%s'", fh.Handle(), fh.Process(), fh.Name())
		} else {
			log.Fatal("no a file handle")
		}
		if i > 50 {
			break
		}
	}
}
```

```
2019/11/29 15:49:05 file handle 0x0040 for process 03596 with name '\Device\HarddiskVolume2\Windows\System32'
2019/11/29 15:49:05 file handle 0x0080 for process 03596 with name '\Device\CNG'
2019/11/29 15:49:05 file handle 0x049C for process 03596 with name '\Device\DeviceApi'
2019/11/29 15:49:05 file handle 0x05D4 for process 03596 with name '\Device\KsecDD'
2019/11/29 15:49:05 file handle 0x0608 for process 03596 with name '\Device\HarddiskVolume2\Windows\System32\de-DE\KernelBase.dll.mui'
2019/11/29 15:49:05 file handle 0x0048 for process 03632 with name '\Device\HarddiskVolume2\Windows\System32'
2019/11/29 15:49:05 file handle 0x008C for process 03632 with name '\Device\CNG'
2019/11/29 15:49:05 file handle 0x0140 for process 03632 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\svchost.exe.mui'
2019/11/29 15:49:05 file handle 0x0174 for process 03632 with name '\Device\DeviceApi'
2019/11/29 15:49:05 file handle 0x022C for process 03632 with name '\Device\KsecDD'
2019/11/29 15:49:05 file handle 0x0380 for process 03632 with name '\Device\DeviceApi'
2019/11/29 15:49:05 file handle 0x03CC for process 03632 with name '\Device\Nsi'
2019/11/29 15:49:05 file handle 0x042C for process 03632 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\crypt32.dll.mui'
2019/11/29 15:49:05 file handle 0x0444 for process 03632 with name '\Device\DeviceApi'
2019/11/29 15:49:05 file handle 0x0458 for process 03632 with name '\Device\KsecDD'
2019/11/29 15:49:05 file handle 0x0460 for process 03632 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\ConnectedDevicesPlatform\L.Marcel\ActivitiesCache.db'
2019/11/29 15:49:05 file handle 0x0468 for process 03632 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\ConnectedDevicesPlatform\L.Marcel\ActivitiesCache.db-wal'
2019/11/29 15:49:05 file handle 0x046C for process 03632 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\ConnectedDevicesPlatform\L.Marcel\ActivitiesCache.db-shm'
2019/11/29 15:49:05 file handle 0x0048 for process 03664 with name '\Device\HarddiskVolume2\Windows\System32'
2019/11/29 15:49:05 file handle 0x008C for process 03664 with name '\Device\CNG'
2019/11/29 15:49:05 file handle 0x0130 for process 03664 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\svchost.exe.mui'
2019/11/29 15:49:05 file handle 0x0204 for process 03664 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\Microsoft\Windows\Notifications\wpndatabase.db'
2019/11/29 15:49:05 file handle 0x0268 for process 03664 with name '\Device\KsecDD'
2019/11/29 15:49:05 file handle 0x027C for process 03664 with name '\Device\DeviceApi'
2019/11/29 15:49:05 file handle 0x02E8 for process 03664 with name '\Device\HarddiskVolume2\Windows\System32\de-DE\KernelBase.dll.mui'
2019/11/29 15:49:05 file handle 0x0330 for process 03664 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\Microsoft\Windows\Notifications\wpndatabase.db-wal'
2019/11/29 15:49:05 file handle 0x0334 for process 03664 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\Microsoft\Windows\Notifications\wpndatabase.db-shm'
2019/11/29 15:49:05 file handle 0x03CC for process 03664 with name '\Device\KsecDD'
2019/11/29 15:49:05 file handle 0x04E0 for process 03664 with name '\Device\Nsi'
2019/11/29 15:49:05 file handle 0x04F8 for process 03664 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\QuietHours.dll.mui'
2019/11/29 15:49:05 file handle 0x062C for process 03664 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\netmsg.dll.mui'
2019/11/29 15:49:05 file handle 0x06FC for process 03664 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\winnlsres.dll.mui'
2019/11/29 15:49:05 file handle 0x0784 for process 03664 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\NotificationController.dll.mui'
2019/11/29 15:49:05 file handle 0x0790 for process 03664 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\mswsock.dll.mui'
2019/11/29 15:49:05 file handle 0x0040 for process 03744 with name '\Device\HarddiskVolume2\Windows\System32'
2019/11/29 15:49:05 file handle 0x007C for process 03744 with name '\Device\CNG'
2019/11/29 15:49:05 file handle 0x0114 for process 03744 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\taskhostw.exe.mui'
2019/11/29 15:49:05 file handle 0x0150 for process 03744 with name '\Device\KsecDD'
2019/11/29 15:49:05 file handle 0x01E8 for process 03744 with name '\Device\HarddiskVolume2\Windows\System32\de-DE\ESENT.dll.mui'
2019/11/29 15:49:05 file handle 0x01F8 for process 03744 with name '\Device\HarddiskVolume2\Windows\System32\en-US\MsCtfMonitor.dll.mui'
2019/11/29 15:49:05 file handle 0x0298 for process 03744 with name '\Device\Beep'
2019/11/29 15:49:05 file handle 0x02B8 for process 03744 with name '\Device\DeviceApi'
2019/11/29 15:49:05 file handle 0x02CC for process 03744 with name '\Device\KsecDD'
2019/11/29 15:49:05 file handle 0x02D0 for process 03744 with name '\Device\Harddisk0\DR0'
2019/11/29 15:49:05 file handle 0x0324 for process 03744 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\Microsoft\Windows\WebCacheLock.dat'
2019/11/29 15:49:05 file handle 0x033C for process 03744 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\Microsoft\Windows\WebCache\V01.log'
2019/11/29 15:49:05 file handle 0x0354 for process 03744 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\Microsoft\Windows\WebCache\WebCacheV01.jfm'
2019/11/29 15:49:05 file handle 0x03B0 for process 03744 with name '\Device\HarddiskVolume2\Program Files\WindowsApps\Microsoft.LanguageExperiencePackde-DE_18362.15.56.0_neutral__8wekyb3d8bbwe\Windows\System32\de-DE\winmm.dll.mui'
2019/11/29 15:49:05 file handle 0x03D0 for process 03744 with name '\Device\HarddiskVolume2\Users\Marcel\AppData\Local\Microsoft\Windows\WebCache\WebCacheV01.dat'
```