package proxy

import (
	"net"
	"log"
	"bufio"
	"bytes"
	"strings"
)

/*
	tcp server proxy lots of clients
 */
type Server struct {
	clients		[]Client
	address		string
}

/*
	handlerConnection(conn)
 */
func (server *Server) handleConnection(conn net.Conn){
	//引入命令过滤器
	var filter filter
	//引入工具类
	var utils utils
	// 命令过滤器初始化
	f := filter.filter()

	client := &Client{
		conn:conn,
		server:server,
		reader:bufio.NewReaderSize(conn, 1024),
		writer:bufio.NewWriterSize(conn, 1024),
		bufferSize:1024,
	}

	// 创建redis单机连接
	//redis://d1Iuw3qlDBntyx1w@10.100.150.31:6011
	rc, _ := net.Dial("tcp", "xxxx")
	rConn := &redisConn{
		conn: rc,
		server: server,
		br: bufio.NewReader(rc),
		bw: bufio.NewWriter(rc),
		password: "xxxx",
	}
	// 兼容单机模式加密
	rConn.SendBytes(utils.auth(rConn.password))
	rConn.Receive()

	go func(){
		for {
			// 从客户端接收数据
			message, err := client.readAll()
			if err != nil {
				log.Fatal(err)
			}
			if len(message) != 0 && string(message[0]) == "*" {
				charset := "\r\n"
				index := bytes.IndexAny(message[8:], charset)
				cmd := string(message[8:8 + index])
				_, ok := f[strings.ToUpper(cmd)]
				// 过滤不支持的命令
				if ok {
					// 返回错误信息
					res := []byte("-ERR the command not support\r\n")
					client.SendBytes(res)
				} else {
					// 向redis发送数据
					err = rConn.SendBytes(message)
					if err != nil {
						log.Fatal(err)
					}
					// 从redis接收数据
					res, _ := rConn.Receive()

					// 兼容集群模式
					// 创建新的redis连接
					if utils.cluster(res) != ""{
						rc, _ := net.Dial("tcp", strings.Fields(string(res))[2])
						redisConn := & redisConn{
							conn: rc,
							server: server,
							br: bufio.NewReader(rc),
							bw: bufio.NewWriter(rc),
						}
						err = redisConn.SendBytes(message)
						if err != nil {
							log.Fatal(err)
						}
						res, _ := redisConn.Receive()
						client.SendBytes(res)
						// 关闭redis连接
						rc.Close()
						redisConn.Close()
					}else {
						client.SendBytes(res)
					}
				}
			}
		}
	}()
}

/*
	listen tcp server
 */
func (server *Server) Listen() {

	listener, err := net.Listen("tcp", server.address)
	if err != nil{
		log.Fatal("Error starting TCP server")
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go server.handleConnection(conn)	// 调用处理方法
	}
}
