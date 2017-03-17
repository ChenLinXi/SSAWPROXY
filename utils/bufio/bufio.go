package bufio2

import (
	"io"
	"bytes"
	"bufio"
)

const DefaultBufferSize = 1024

 type Reader struct {
	 err error
	 buf []byte

	 rd	io.Reader
	 rpos	int	//read position -- 指向buffer
	 wpos	int	//write position -- 指向buffer

	 slice	sliceAlloc
 }

func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, DefaultBufferSize)
}

func NewReaderSize(rd io.Reader, size int) *Reader {
	if size <= 0 {
		size = DefaultBufferSize
	}
	return &Reader{rd: rd, buf: make([]byte, size)}
}

func NewReaderBuffer(rd io.Reader, buf []byte) *Reader{
	if len(buf) == 0{
		buf = make([]byte, DefaultBufferSize)
	}
	return &Reader{rd: rd, buf: buf}
}

// 读取buffer中新增的数据
func (reader *Reader) fill() error {
	if reader.err != nil {
		return reader.err
	}
	if reader.rpos > 0 {	// reader buffer数据迁移
		n := copy(reader.buf, reader.buf[reader.rpos:reader.wpos])
		reader.rpos = 0	// 重置rpos, wpos位置
		reader.wpos = n
	}
	n, err := reader.rd.Read(reader.buf[reader.wpos:])
	if err != nil{
		reader.err = err
	} else if n == 0 {
		reader.err = io.ErrNoProgress
	} else {
		reader.wpos += n
	}
	return reader.err
}

// 返回buffer中的数据大小
func (reader *Reader) buffered() int {
	return reader.wpos - reader.rpos
}

// 读取buffer中数据
func (reader *Reader) Read(p []byte) (int, error) {
	if reader.err != nil || len(p) == 0 {
		return 0, reader.err
	}
	if reader.buffered() == 0 {
		if len(p) >= len(reader.buf) {
			n, err := reader.rd.Read(p)
			if err != nil {
				reader.err = err
			}
			return n, reader.err
		}
		if reader.fill() != nil {
			return 0, reader.err
		}
	}
	n := copy(p, reader.buf[reader.rpos:reader.wpos])
	reader.rpos += n
	return n, nil
}

// 读取buffer byte数据
func (reader *Reader) ReadByte() (byte, error) {
	if reader.err != nil {
		return 0, reader.err
	}
	if reader.buffered() == 0 {
		if reader.fill() != nil {
			return 0, reader.err
		}
	}
	c := reader.buf[reader.rpos]
	reader.rpos += 1
	return c, nil
}

func (reader *Reader) PeekByte() (byte, error) {
	if reader.err != nil {
		return 0, reader.err
	}
	if reader.buffered() == 0 {
		if reader.fill() != nil {
			return 0, reader.err
		}
	}
	c := reader.buf[reader.rpos]
	return c, nil
}

// 接收buffer中的数据 换行标识符(delim) : "\n" or "\r"
func (reader *Reader) ReadSlice(delim byte) ([]byte, error) {
	if reader.err != nil {
		return nil, reader.err
	}
	for {
		var index = bytes.IndexByte(reader.buf[reader.rpos:reader.wpos], delim)
		if index >= 0 {
			limit := reader.rpos + index + 1
			slice := reader.buf[reader.rpos:limit]
			reader.rpos = limit
			return slice, nil
		}
		if reader.buffered() == len(reader.buf) {
			reader.rpos = reader.wpos	// 重置buffer
			return reader.buf, bufio.ErrBufferFull
		}
		if reader.fill() != nil {
			return nil, reader.err
		}
	}
}

func (reader *Reader) ReadBytes(delim byte) ([]byte, error) {
	var full [][]byte
	var last []byte
	var size int
	for last == nil {
		f, err := reader.ReadSlice(delim)
		if err != nil {
			if err != bufio.ErrBufferFull {
				return nil, reader.err
			}
			dup := reader.slice.Make(len(f))
			copy(dup, f)
			full = append(full, dup)
		} else {
			last = f
		}
		size += len(f)
	}
	var n int
	var buf = reader.slice.Make(size)
	for _, frag := range full {
		n += copy(buf[n:], frag)
	}
	copy(buf[n:], last)
	return buf, nil
}

// 读取buffer中全部内容
func (reader *Reader) ReadFull(n int) ([]byte, error) {
	if reader.err != nil || n == 0{
		return nil, reader.err
	}
	var buf = reader.slice.Make(n)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

type Writer struct {
	err error
	buf []byte

	wr	io.Writer
	wpos	int
}

func NewWriter(wr io.Writer) *Writer {
	return NewWriterSize(wr, DefaultBufferSize)
}

func NewWriterSize(wr io.Writer, size int) *Writer {
	if size <= 0 {
		size = DefaultBufferSize
	}
	return &Writer{wr: wr, buf: make([]byte, size)}
}

func NewWriterBuffer(wr io.Writer, buf []byte) *Writer {
	if len(buf) == 0 {
		buf = make([]byte, DefaultBufferSize)
	}
	return &Writer{wr: wr, buf: buf}
}

func (writer *Writer) Flush() error{
	return writer.flush()
}

func (writer *Writer) flush() error {
	if writer.err != nil {
		return writer.err
	}
	if writer.wpos == 0 {
		return nil
	}
	n, err := writer.wr.Write(writer.buf[:writer.wpos])
	if err != nil {
		writer.err = err
	} else if n < writer.wpos {
		writer.err = io.ErrShortWrite
	} else {
		writer.wpos = 0
	}
	return writer.err
}

func (writer *Writer) available() int {
	return len(writer.buf) - writer.wpos
}

func (writer *Writer) Write(p []byte) (nn int, err error) {
	for writer.err == nil && len(p) > writer.available() {
		var n int
		if writer.wpos == 0 {
			n, writer.err = writer.wr.Write(p)
		} else {
			n = copy(writer.buf[writer.wpos:], p)
			writer.wpos += n
			writer.flush()
		}
		nn, p = nn+n, p[n:]
	}
	if writer.err != nil || len(p) == 0 {
		return nn, writer.err
	}
	n := copy(writer.buf[writer.wpos:], p)
	writer.wpos += n
	return nn + n, nil
}

func (writer *Writer) WriteByte(c byte) error {
	if writer.err != nil {
		return writer.err
	}
	if writer.available() == 0 && writer.flush() != nil {
		return writer.err
	}
	writer.buf[writer.wpos] = c
	writer.wpos += 1
	return nil
}

func (writer *Writer) WriteString(s string) (nn int, err error) {
	for writer.err == nil && len(s) > writer.available() {
		n := copy(writer.buf[writer.wpos:], s)
		writer.wpos += n
		writer.flush()
		nn, s = nn+n, s[n:]
	}
	if writer.err != nil || len(s) == 0 {
		return nn, writer.err
	}
	n := copy(writer.buf[writer.wpos:], s)
	writer.wpos += n
	return nn + n, nil
}
