package unsafe2

/*
	go_slice.go : []byte切片
 */


type goSlice struct {
	buf []byte
}

func newGoSlice(n int) Slice {
	return &goSlice{
		buf: make([]byte, n),
	}
}

func (s *goSlice) Buffer() []byte {
	return s.buf
}

func (s goSlice) reclaim() {
}
