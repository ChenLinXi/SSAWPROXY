package proxy

import (
	"testing"
	"log"
)

const address  = "xxxx"

func New(address string) *Server{
	log.Println("Creating server with address", address)
	server := &Server{
		address:address,
	}
	return server
}

/*
	测试结果：
	1.不支持redis-benchmark [PING/SET/GET] 单个命令，benchmark中断后代理会自动关闭
	2.过滤了部分不支持的命令：如 keys
	3.支持集群模式重连操作
	4.支持redis单机加密
 */
func Test(t *testing.T) {
	server := New(address)
	defer server.Listen()
}
