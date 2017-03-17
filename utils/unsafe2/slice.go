package unsafe2

import "SSAWPROXY/redisProxy/utils/sync2/atomic2"

/*
	slice.go : 动态分配内存空间（切片）
 */

type Slice interface {
	Buffer() []byte
	reclaim()
}

var maxOffheapBytes atomic2.Int64

func MaxOffheapBytes() int {
	return int(maxOffheapBytes.Get())
}

func SetMaxOffheapBytes(n int) {
	maxOffheapBytes.Set(int64(n))
}

const MinOffheapSlice = 1024 * 16

/*
	创建不超过 "MinOffheapSlice" 大小的内存切片
 */
func MakeSlice(n int) Slice {
	if n >= MinOffheapSlice {
		if s := newCGoSlice(n, false); s != nil {
			return s
		}
	}
	return newGoSlice(n)
}

/*
	创建指定大小 堆内存切片
 */
func MakeOffheapSlice(n int) Slice {
	if n >= 0 {
		return newCGoSlice(n, true)
	}
	panic("make slice with negative size")
}

/*
	释放内存切片
 */
func FreeSlice(s Slice) {
	if s != nil {
		s.reclaim()
	}
}
