package redis

import (
	"net"
	"time"
	"testing"
	"SSAWPROXY/redisProxy/utils/assert"
)

func TestNewConn(t *testing.T) {
	l, err := net.Listen("tcp", "10.101.20.69:9999")
	assert.MustNoError(err)
	defer l.Close()

	const bufsize = 128 * 1024

	cc := make(chan *Conn, 1)
	go func() {
		defer close(cc)
		c, err := l.Accept()
		assert.MustNoError(err)
		cc <- NewConn(c, bufsize, bufsize)
	}()

	const timeout = time.Millisecond * 50

	conn1, err := DialTimeout(l.Addr().String(), timeout, bufsize, bufsize)
	assert.MustNoError(err)

	conn2, ok := <- cc
	assert.Must(ok)
	return conn1, conn2
}
