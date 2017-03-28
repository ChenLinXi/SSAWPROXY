package proxy

import (
	"reflect"
	"bytes"
	"encoding/gob"
	"strconv"
	"encoding/binary"
	"fmt"
	"strings"
)

/*
	1.parse length of []byte
	2.parse int of []byte
	3.convert interface to slice
	4.convert interface to string
	5.parse send message to lots of package
	6.兼容单机auth
 */


type utils struct {}

/*
	parseLen get length(int) of []byte
 */
func (utils *utils) parseLen(p []byte) (int, error) {
	if len(p) == 0 {
		return -1, protocolError("malformed length")
	}

	if p[0] == '-' && len(p) == 2 && p[1] == '1' {
		// handle $-1 and $-1 null replies.
		return -1, nil
	}

	var n int
	for _, b := range p {
		n *= 10
		if b < '0' || b > '9' {
			return -1, protocolError("illegal bytes in length")
		}
		n += int(b - '0')
	}

	return n, nil
}

/*
	parseInt get length of []byte
 */
func (utils *utils) parseInt(p []byte) (interface{}, error) {
	if len(p) == 0 {
		return 0, protocolError("malformed integer")
	}

	var negate bool
	if p[0] == '-' {
		negate = true
		p = p[1:]
		if len(p) == 0 {
			return 0, protocolError("malformed integer")
		}
	}

	var n int64
	for _, b := range p {
		n *= 10
		if b < '0' || b > '9' {
			return 0, protocolError("illegal bytes in length")
		}
		n += int64(b - '0')
	}

	if negate {
		n = -n
	}
	return n, nil
}

/*
	将interface{}切割为[]interface{}数组
 */
func (utils *utils) convertInterfaceToSlice(arr interface{}) []interface{} {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice{
		panic("toslice arr not slice")
	}
	l := v.Len()
	ret := make([]interface{}, l)
	for i := 0; i < l; i++ {
		ret[i] = v.Index(i).Interface()
	}
	return ret
}

/*
	从interface中提取command命令
	如：set, get, pop之类
 */
func (utils *utils) convertInterfaceToString(key interface{}) (string, error){
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil{
		return "", err
	}
	return string(buf.Bytes()[4:]), nil
}

/*
	加工发送的数据
 */
func (utils *utils) parseBytes(message []byte) ([]byte, error){
	charset := []byte("\r\n")
	// 组装消息
	result := [][]byte{
		[]byte("*1"),
		[]byte("$" + strconv.Itoa(len(message))),
		message,
		[]byte(""),
	}
	res := bytes.Join(result, charset)
	return res, nil
}

/*
	加工整形数据 ：
 */
func (utils *utils) parseInt64(message int64) ([]byte, error) {
	charset := []byte("\r\n")
	// 组装消息
	result := [][]byte{
		[]byte("*1"),
		[]byte(":"),
		[]byte(strconv.Itoa(int(message))),
		[]byte(""),
	}
	res := bytes.Join(result, charset)
	return res, nil
}

/*
	加工错误信息 -
 */
//func (utils *utils) parseError(message error) ([]byte, error) {
//	charset := []byte("\r\n")
//	// 组装消息
//	result := [][]byte{
//		[]byte("*1"),
//		[]byte(":"),
//		[]byte(strconv.Itoa(int(message))),
//		[]byte(""),
//	}
//	res := bytes.Join(result, charset)
//	return res, nil
//}

/*
	加工状态信息
 */



/*
	convert int64 to []byte
 */
func (utils *utils)Int64ToBytes(i int64) []byte{
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	fmt.Println(buf)
	return buf
}

/*
	兼容集群模式
	判断从redis接收到的返回数据是否为 "-ERR MOVED slot IP"
	重新新连接与指定的ip相连
 */
func (utils *utils) cluster(b []byte) string {
	if b[0] == 45 && string(b[1:6]) == "MOVED" {
		return strings.Fields(string(b))[2]
	}else {
		return ""
	}
}

/*
	兼容单机加密
 */
func (utils *utils) auth(auth string) []byte {
	charset := []byte("\r\n")
	// 组装消息
	result := [][]byte{
		[]byte("*2"),
		[]byte("$4"),
		[]byte("AUTH"),
		[]byte("$" + strconv.Itoa(len(auth))),
		[]byte(auth),
		[]byte(""),
	}
	res := bytes.Join(result, charset)
	return res
}
