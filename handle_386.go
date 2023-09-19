//+build !amd64

package handle

import "unsafe"

type uint3264 uint32

const maxHandleCount = (1 << 31) / unsafe.Sizeof(SystemHandle{})
