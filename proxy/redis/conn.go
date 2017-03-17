package redis

import (
	"net"
	"time"

	"SSAWPROXY/redisProxy/utils/errors"
	"SSAWPROXY/redisProxy/utils/bufio2"
	"SSAWPROXY/redisProxy/utils/unsafe2"
)

/*
	tcp connect
 */

type Conn struct {
	Sock net.Conn	// net connection[tcp]

	*Decoder	// decode message from redis client
	*Encoder	// encode message from redis and send to redis client

	ReaderTimeout	time.Duration
	WriterTimeout	time.Duration

	LastWrite	time.Time
}

func DialTimeout(addr string, timeout time.Duration, rbuf, wbuf int) (*Conn, error) {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return NewConn(c, rbuf, wbuf), nil
}

func NewConn(sock net.Conn, rbuf, wbuf int) *Conn {
	conn := &Conn{Sock: sock}
	conn.Decoder = newConnDecoder(conn, rbuf)
	conn.Decoder = newConnEncoder(conn, wbuf)
	return conn
}

func (conn *Conn) LocalAddr() string{
	return conn.Sock.LocalAddr().String()
}

func (conn *Conn) RemoteAddr() string{
	return conn.Sock.RemoteAddr().String()
}

func (conn *Conn) Close() error {
	return conn.Sock.Close()
}

func (conn *Conn) CloseReader() error {
	if t, ok := conn.Sock.(*net.TCPConn); ok{
		return t.CloseRead()
	}
	return conn.Close()
}

func (conn *Conn) SetKeepAlivePeriod (d time.Duration) error {
	if t, ok := conn.Sock.(*net.TCPConn); ok {
		if err := t.SetKeepAlive(d != 0); err != nil{
			return errors.Trace(err)
		}
		if d != 0 {
			if err := t.SetKeepAlivePeriod(d); err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}

func (conn *Conn) FlushEncoder() *FlushEncoder {
	return &FlushEncoder{Conn: conn}
}

type connReader struct {
	*Conn
	unsafe2.Slice

	hasDeadline bool
}

func newConnDecoder(conn *Conn, bufsize int) *Decoder{
	r := &connReader{Conn: conn}
	r.Slice = unsafe2.MakeSlice(bufsize)
	return NewDecoderBuffer(bufio2.NewReaderBuffer(r, r.Buffer()))
}

func (reader *connReader) Read(b []byte) (int, error) {
	if timeout := reader.ReaderTimeout; timeout != 0 {
		if err := reader.Sock.SetDeadline(time.Now().Add(timeout)); err != nil {
			return 0, errors.Trace(err)
		}
		reader.hasDeadline = true
	} else if reader.hasDeadline {
		if err := reader.Sock.SetReadDeadline(time.Time{}); err != nil {
			return 0, errors.Trace(err)
		}
		reader.hasDeadline = false
	}
	n, err := reader.Sock.Read(b)
	if err != nil {
		err = errors.Trace(err)
	}
	return n, err
}

type connWriter struct {
	*Conn
	unsafe2.Slice

	hasDeadline bool
}

func newConnEncoder(conn *Conn, bufsize int) *Encoder{
	w := &connWriter{Conn: conn}
	w.Slice = unsafe2.MakeSlice(bufsize)
	return NewEncoderBuffer(bufio2.NewWriterBuffer(w, w.Buffer()))
}

func (writer *connWriter) Write(b []byte) (int, error) {
	if timeout := writer.WriterTimeout; timeout != 0 {
		if err := writer.Sock.SetWriteDeadline(time.Now().Add(timeout)); err != nil{
			return 0, errors.Trace(err)
		}
		writer.hasDeadline = true
	} else if writer.hasDeadline {
		if err := writer.Sock.SetWriteDeadline(time.Time{}); err != nil {
			return 0, errors.Trace(err)
		}
		writer.hasDeadline = false
	}
	n, err := writer.Sock.Write(b)
	if err != nil {
		err = errors.Trace(err)
	}
	writer.LastWrite = time.Now()
	return n, err
}

func IsTimeout(err error) bool {
	if err := errors.Cause(err); err != nil {
		e, ok := err.(*net.OpError)
		if ok {
			return e.Timeout()
		}
	}
	return false
}

type FlushEncoder struct {
	Conn *Conn

	MaxInterval time.Duration
	MaxBuffered int

	nbuffered int
}

func (p *FlushEncoder) NeedFlush() bool {
	if p.nbuffered != 0 {
		if p.MaxBuffered < p.nbuffered {
			return true
		}
		if p.MaxInterval < time.Since(p.Conn.LastWrite) {
			return true
		}
	}
	return false
}

func (p *FlushEncoder) Flush(force bool) error {
	if force || p.NeedFlush() {
		if err := p.Conn.Flush(); err != nil {
			return err
		}
		p.nbuffered = 0
	}
	return nil
}

func (p *FlushEncoder) Encode(resp *Resp) error {
	if err := p.Conn.Encode(resp, false); err != nil {
		return err
	} else {
		p.nbuffered++
		return nil
	}
}

func (p *FlushEncoder) EncodeMultiBulk(multi []*Resp) error {
	if err := p.Conn.encodeMultiBulk(multi, false); err != nil {
		return err
	} else {
		p.nbuffered++
		return nil
	}
}
