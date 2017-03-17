package unsafe2

import (
	"SSAWPROXY/redisProxy/utils/sync2/atomic2"
	"unsafe"
	"reflect"
	"runtime"
)

/*
	cgo_slice.go : 内存切片
 */

type cgoSlice struct {
	ptr unsafe.Pointer
	buf []byte
}

var allocOffheapBytes atomic2.Int64

func OffheapBytes() int {
	return int(allocOffheapBytes.Get())
}

func newCGoSlice(n int, force bool) Slice {
	after := int(allocOffheapBytes.Add(int64(n)))
	if !force && after > MaxOffheapBytes() {
		allocOffheapBytes.Sub(int64(n))
		return nil
	}
	p := cgo_malloc(n)
	if p == nil {
		allocOffheapBytes.Sub(int64(n))
		return nil
	}
	s := &cgoSlice{
		ptr: p,
		buf: *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Data: uintptr(p), Len: n, Cap: n,
		})),
	}
	runtime.SetFinalizer(s, (*cgoSlice).reclaim)
	return s
}

func (s *cgoSlice) Buffer() []byte {
	return s.buf
}

func (s *cgoSlice) reclaim() {
	if s.ptr == nil {
		return
	}
	cgo_free(s.ptr)
	allocOffheapBytes.Sub(int64(len(s.buf)))
	s.ptr = nil
	s.buf = nil
	runtime.SetFinalizer(s, nil)
}



