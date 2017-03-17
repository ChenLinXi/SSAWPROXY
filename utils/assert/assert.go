package assert

import "SSAWPROXY/redisProxy/utils/log"

/*
	assert.go : 断言
 */

func Must(b bool) {
	if b {
		return
	}
	log.Panic("assertion failed")
}

func MustNoError(err error) {
	if err == nil {
		return
	}
	log.PanicError(err, "error happens, assertion failed")
}
