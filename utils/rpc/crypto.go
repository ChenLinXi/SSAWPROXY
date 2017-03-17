package rpc

import (
	"net"
	"sort"
	"bytes"
	"fmt"
	"crypto/md5"
	"crypto/sha256"
)

/*
	crypto.go : 加密
 */

func NewToken(segs ...string) string {
	var list []string
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		addr := i.HardwareAddr.String()
		if addr != "" {
			list = append(list, addr)
		}
	}
	sort.Strings(list)

	t := &bytes.Buffer{}
	fmt.Fprintf(t, "Proxy-Token@%v", list)
	for _, s := range segs {
		fmt.Fprintf(t, "-{%s}", s)
	}
	b := md5.Sum(t.Bytes())
	return fmt.Sprintf("%x", b)
}

func NewXAuth(segs ...string) string {
	t := &bytes.Buffer{}
	fmt.Fprintf(t, "Proxy-XAuth")
	for _, s := range segs {
		fmt.Fprintf(t, "-[%s]", s)
	}
	b := sha256.Sum256(t.Bytes())
	return fmt.Sprintf("%x", b[:16])
}
