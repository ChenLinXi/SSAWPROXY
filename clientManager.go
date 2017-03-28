package proxy

type ClientManager struct {
	client *[]Client
	addr   []string
	remove func(address string)(error)
}