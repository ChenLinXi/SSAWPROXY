package unsafe2

/*
	cgo_malloc.go : go c动态开辟内存空间
 */

// #include <stdlib.h>
import "C"

import (
	"unsafe"
)

func cgo_malloc(n int) unsafe.Pointer {
	return C.malloc(C.size_t(n))
}

func cgo_free(ptr unsafe.Pointer) {
	C.free(ptr)
}
