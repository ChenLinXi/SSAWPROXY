package redis

/*
	resp.go: check/Create response
 */

import "fmt"

type RespType byte

const(
	TypeString    RespType = '+'
	TypeError     RespType = '-'
	TypeInt       RespType = ':'
	TypeBulkBytes RespType = '$'
	TypeArray     RespType = '*'
)

func (t RespType) String() string {
	switch t {
	case TypeString:
		return "<string>"
	case TypeError:
		return "<error>"
	case TypeInt:
		return "<int>"
	case TypeBulkBytes:
		return "<bulkbytes>"
	case TypeArray:
		return "<array>"
	default:
		return fmt.Sprintf("<unknown-0x%02x>", byte(t))
	}
}

type Resp struct {
	Type RespType

	Value []byte
	Array []*Resp
}

func (response *Resp) IsString() bool {
	return response.Type == TypeString
}

func (response *Resp) IsError() bool {
	return response.Type == TypeError
}

func (response *Resp) IsInt() bool {
	return response.Type == TypeInt
}

func (response *Resp) IsBulkBytes() bool {
	return response.Type == TypeBulkBytes
}

func (response *Resp) IsArray() bool {
	return response.Type == TypeArray
}

func NewString(value []byte) *Resp{
	response := &Resp{}
	response.Type = TypeString
	response.Value = value
	return response
}

func NewError(value []byte) *Resp {
	response := &Resp{}
	response.Type = TypeError
	response.Value = value
	return response
}

func NewErrorf(format string, args ...interface{}) *Resp {
	return NewError([]byte(fmt.Sprintf(format, args...)))
}

func NewInt(value []byte) *Resp {
	response := &Resp{}
	response.Type = TypeInt
	response.Value = value
	return response
}

func NewBulkBytes(value []byte) *Resp {
	response := &Resp{}
	response.Type = TypeBulkBytes
	response.Value = value
	return response
}

func NewArray(array []*Resp) *Resp {
	response := &Resp{}
	response.Type = TypeArray
	response.Array = array
	return response
}

