package proxy

import (
	"net"
	"bufio"
	"sync"
	"time"
	"SSAWPROXY/redisProxy/utils/errors"
)

type redisConn struct {
	conn		net.Conn
	server		*Server
	err 		error
	pending		int
	mu		sync.Mutex

	br		*bufio.Reader
	readTimeout	time.Duration

	bw		*bufio.Writer
	writeTimeout	time.Duration

	password	string
	LastWrite	time.Time
}

func NewConn(netConn net.Conn, readTimeout, writeTimeout time.Duration) *redisConn {
	return &redisConn{
		conn:         netConn,
		bw:           bufio.NewWriter(netConn),
		br:           bufio.NewReader(netConn),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

func (redisConn *redisConn) LocalAddr() string {
	return redisConn.conn.LocalAddr().String()
}

func (redisConn *redisConn) RemoteAddr() string {
	return redisConn.conn.RemoteAddr().String()
}

func (redisConn *redisConn) CloseReader() error {
	if t, ok := redisConn.conn.(*net.TCPConn); ok {
		return t.CloseRead()
	}
	return redisConn.Close()
}

/*
	set redis connection keep alive period
 */
func (redisConn *redisConn) SetKeepAlivePeriod(d time.Duration) error {
	if t, ok := redisConn.conn.(*net.TCPConn); ok {
		if err := t.SetKeepAlive(d != 0); err != nil {
			return err
		}
		if d != 0 {
			if err := t.SetKeepAlivePeriod(d); err != nil {
				return err
			}
		}
	}
	return nil
}

func (redisConn *redisConn) Conn() net.Conn{
	return redisConn.conn
}

func (redisConn *redisConn) Close() error {
	redisConn.mu.Lock()
	err := redisConn.err
	if redisConn.err == nil {
		redisConn.err = errors.New("redis: closed")
		err = redisConn.conn.Close()
	}
	redisConn.mu.Unlock()
	return err
}

func (redisConn *redisConn) Err() error {
	redisConn.mu.Lock()
	err := redisConn.err
	redisConn.mu.Unlock()
	return err
}

func (redisConn *redisConn) fatal(err error) error {
	redisConn.mu.Lock()
	if redisConn.err == nil {
		redisConn.err = err
		// Close connection to force errors on subsequent calls and to unblock
		// other reader or writer.
		redisConn.conn.Close()
	}
	redisConn.mu.Unlock()
	return err
}

func (redisConn *redisConn) SendBytes(b []byte) error {
	_, err := redisConn.conn.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func (redisConn *redisConn) peek() int {
	redisConn.br.Peek(1)
	length := redisConn.br.Buffered()
	return length
}

func (redisConn *redisConn) Receive() ([]byte, error) {
	msgLen := redisConn.peek()
	b := make([]byte, msgLen)
	n, err := redisConn.br.Read(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}

func (redisConn *redisConn) Flush() error {
	if redisConn.writeTimeout != 0 {
		redisConn.conn.SetWriteDeadline(time.Now().Add(redisConn.writeTimeout))
	}
	if err := redisConn.bw.Flush(); err != nil {
		return redisConn.fatal(err)
	}
	return nil
}

func IsTimeout(err error) bool {
	if err := errors.Cause(err); err != nil {
		e, ok := err.(*net.OpError)
		if ok{
			return e.Timeout()
		}
	}
	return false
}

func DialTimeout(addr string, timeout time.Duration, rbuf, wbuf int) (*redisConn, error) {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return NewConnection(c, rbuf, wbuf), nil
}

func NewConnection(sock net.Conn, rbuf, wbuf int) *redisConn {
	conn := &redisConn{conn: sock}
	return conn
}
