//+build amd64

package handle

import (
	"unsafe"
)

type uint3264 uint64

const maxHandleCount = (1 << 50) / unsafe.Sizeof(SystemHandle{})
