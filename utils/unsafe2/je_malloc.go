package unsafe2

import (
	"unsafe"

	jemalloc "github.com/spinlock/jemalloc-go"
)

/*
	je_malloc.go : 分配指定大小的内存空间
 */

func cgo_malloc(n int) unsafe.Pointer {
	return jemalloc.Malloc(n)
}

func cgo_free(ptr unsafe.Pointer) {
	jemalloc.Free(ptr)
}
