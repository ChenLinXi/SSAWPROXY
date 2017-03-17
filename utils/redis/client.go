package redis

import (
	"container/list"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"SSAWPROXY/redisProxy/utils/errors"
	"SSAWPROXY/redisProxy/utils/math2"
	redigo "github.com/garyburd/redigo/redis"
)

/*
	client.go : 基于redigo.go连接的封装
 */

type Client struct {
	conn redigo.Conn	// redigo.Conn 连接指定redis
	Addr string		// redis Address[ip,port]
	Auth string		// redis password

	Database int		// redis database

	LastUse time.Time	// redis lastuse time
	Timeout time.Duration	// redis timeout
}

/*
	新建不需要密码的Client
	params: address, timeout
	return: *Client, error
 */
func NewClientNoAuth(addr string, timeout time.Duration) (*Client, error) {
	return NewClient(addr, "", timeout)
}

/*
	新建基于redigo.Conn的redis client
	params: address, auth(可选), timeout
	return: *Client, error
 */
func NewClient(addr string, auth string, timeout time.Duration) (*Client, error) {
	c, err := redigo.Dial("tcp", addr, []redigo.DialOption{
		redigo.DialConnectTimeout(math2.MinDuration(time.Second, timeout)),
		redigo.DialPassword(auth),
		redigo.DialReadTimeout(timeout), redigo.DialWriteTimeout(timeout),
	}...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Client{
		conn: c, Addr: addr, Auth: auth,
		LastUse: time.Now(), Timeout: timeout,
	}, nil
}

/*
	关闭redigo.Conn连接
	return: error
 */
func (c *Client) Close() error {
	return c.conn.Close()
}

/*
	执行redis命令
	params: cmd (string) , args ...interface{}
	return: r interface{}, error
 */
func (c *Client) Do(cmd string, args ...interface{}) (interface{}, error) {
	r, err := c.conn.Do(cmd, args...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	c.LastUse = time.Now()

	if err, ok := r.(redigo.Error); ok {
		return nil, errors.Trace(err)
	}
	return r, nil
}

/*
	清空redis数据
	params: cmd (string) , args ...interface{}
	return: error
 */
func (c *Client) Flush(cmd string, args ...interface{}) error {
	if err := c.conn.Send(cmd, args...); err != nil {
		return errors.Trace(err)
	}
	if err := c.conn.Flush(); err != nil {
		return errors.Trace(err)
	}
	c.LastUse = time.Now()
	return nil
}

/*
	执行命令后，接收从redis返回的数据
	return: r interface{}, error
 */
func (c *Client) Receive() (interface{}, error) {
	r, err := c.conn.Receive()
	if err != nil {
		return nil, errors.Trace(err)
	}
	c.LastUse = time.Now()

	if err, ok := r.(redigo.Error); ok {
		return nil, errors.Trace(err)
	}
	return r, nil
}

/*
	选择redis database
	params: int
	return: error
 */
func (c *Client) Select(database int) error {
	if c.Database == database {
		return nil
	}
	_, err := c.Do("SELECT", database)
	if err != nil {
		c.Close()
		return errors.Trace(err)
	}
	c.Database = database
	return nil
}

/*
	查询redis的info信息
	return: map[string]string，按照"\n"换行
 */
func (c *Client) Info() (map[string]string, error) {
	text, err := redigo.String(c.Do("INFO"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	info := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		if key := strings.TrimSpace(kv[0]); key != "" {
			info[key] = strings.TrimSpace(kv[1])
		}
	}
	return info, nil
}


/*
	查询keyspace
	return: map[int]string, 按照"\n"换行, error
 */
func (c *Client) InfoKeySpace() (map[int]string, error) {
	text, err := redigo.String(c.Do("INFO", "keyspace"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	info := make(map[int]string)
	for _, line := range strings.Split(text, "\n") {
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		if key := strings.TrimSpace(kv[0]); key != "" && strings.HasPrefix(key, "db") {
			n, err := strconv.Atoi(key[2:])
			if err != nil {
				return nil, errors.Trace(err)
			}
			info[n] = strings.TrimSpace(kv[1])
		}
	}
	return info, nil
}

/*
	查询redis config信息
	return: map[string]string，按照"\n"换行, error
 */
func (c *Client) InfoFull() (map[string]string, error) {
	if info, err := c.Info(); err != nil {
		return nil, errors.Trace(err)
	} else {
		host := info["master_host"]
		port := info["master_port"]
		if host != "" || port != "" {
			info["master_addr"] = net.JoinHostPort(host, port)
		}
		r, err := c.Do("CONFIG", "get", "maxmemory")
		if err != nil {
			return nil, errors.Trace(err)
		}
		p, err := redigo.Values(r, nil)
		if err != nil || len(p) != 2 {
			return nil, errors.Errorf("invalid response = %v", r)
		}
		v, err := redigo.Int(p[1], nil)
		if err != nil {
			return nil, errors.Errorf("invalid response = %v", r)
		}
		info["maxmemory"] = strconv.Itoa(v)
		return info, nil
	}
}

/*
	设置单机密码
	return: error
 */
func (c *Client) SetMaster(master string) error {
	var host, port string
	if master == "" || strings.ToUpper(master) == "NO:ONE" {
		host, port = "NO", "ONE"
	} else {
		_, err := c.Do("CONFIG", "set", "masterauth", c.Auth)
		if err != nil {
			return errors.Trace(err)
		}
		host, port, err = net.SplitHostPort(master)
		if err != nil {
			return errors.Trace(err)
		}
	}
	_, err := c.Do("SLAVEOF", host, port)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
	迁槽
	params: slot int, target string[目标redis地址]
	return: flag int, error
 */
func (c *Client) MigrateSlot(slot int, target string) (int, error) {
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return 0, errors.Trace(err)
	}
	mseconds := int(c.Timeout / time.Millisecond)
	if reply, err := c.Do("SLOTSMGRTTAGSLOT", host, port, mseconds, slot); err != nil {
		return 0, errors.Trace(err)
	} else {
		p, err := redigo.Ints(redigo.Values(reply, nil))
		if err != nil || len(p) != 2 {
			return 0, errors.Errorf("invalid response = %v", reply)
		}
		return p[1], nil
	}
}


/*
	异步迁槽参数配置
 */
type MigrateSlotAsyncOption struct {
	MaxBulks int
	MaxBytes int
	NumKeys  int
	Timeout  time.Duration
}

/*
	异步迁槽
	params: slot int, target string[目标redis地址], 异步迁槽配置参数
	return：flag int, error
 */
func (c *Client) MigrateSlotAsync(slot int, target string, option *MigrateSlotAsyncOption) (int, error) {
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return 0, errors.Trace(err)
	}
	if reply, err := c.Do("SLOTSMGRTTAGSLOT-ASYNC", host, port, int(option.Timeout/time.Millisecond),
		option.MaxBulks, option.MaxBytes, slot, option.NumKeys); err != nil {
		return 0, errors.Trace(err)
	} else {
		p, err := redigo.Ints(redigo.Values(reply, nil))
		if err != nil || len(p) != 2 {
			return 0, errors.Errorf("invalid response = %v", reply)
		}
		return p[1], nil
	}
}

/*
	查询槽信息
	return: map[slotFlag][slotID], error
 */
func (c *Client) SlotsInfo() (map[int]int, error) {
	if reply, err := c.Do("SLOTSINFO"); err != nil {
		return nil, errors.Trace(err)
	} else {
		infos, err := redigo.Values(reply, nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
		slots := make(map[int]int)
		for i, info := range infos {
			p, err := redigo.Ints(info, nil)
			if err != nil || len(p) != 2 {
				return nil, errors.Errorf("invalid response[%d] = %v", i, info)
			}
			slots[p[0]] = p[1]
		}
		return slots, nil
	}
}

/*
	查询client的 master, slave, sentinel信息
	return； info string, error
 */
func (c *Client) Role() (string, error) {
	if reply, err := c.Do("ROLE"); err != nil {
		return "", err
	} else {
		values, err := redigo.Values(reply, nil)
		if err != nil {
			return "", errors.Trace(err)
		}
		if len(values) == 0 {
			return "", errors.Errorf("invalid response = %v", reply)
		}
		role, err := redigo.String(values[0], nil)
		if err != nil {
			return "", errors.Errorf("invalid response[0] = %v", values[0])
		}
		return strings.ToUpper(role), nil
	}
}

var ErrClosedPool = errors.New("use of closed redis pool")

/*
	Pool(同一redis的多个clients组成的pool连接池)
	处理单个redis中的多个*redigo.Conn连接
	map[redis address]*list.List
 */
type Pool struct {
	mu sync.Mutex	//同步锁

	auth string	// 密码
	pool map[string]*list.List	// map[redis address]*Client.List

	timeout time.Duration	// 超时

	exit struct {	// clients
		   C chan struct{}
	     }

	closed bool	// 关闭标志
}

/*
	创建新的pool
	params: auth string, timeout time.Duration
	return: *Pool
 */
func NewPool(auth string, timeout time.Duration) *Pool {
	p := &Pool{
		auth: auth, timeout: timeout,
		pool: make(map[string]*list.List),
	}
	p.exit.C = make(chan struct{})

	if timeout != 0 {
		go func() {
			var ticker = time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-p.exit.C:
					return
				case <-ticker.C:
					p.Cleanup()
				}
			}
		}()
	}

	return p
}

/*
	检查pool
	return: flag bool
 */
func (p *Pool) isRecyclable(c *Client) bool {
	if c.conn.Err() != nil {
		return false
	}
	return p.timeout == 0 || time.Since(c.LastUse) < p.timeout
}

/*
	关闭pool
	return: error
 */
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	close(p.exit.C)

	for addr, list := range p.pool {	// 迭代从list头关闭client连接
		for i := list.Len(); i != 0; i-- {
			c := list.Remove(list.Front()).(*Client)
			c.Close()
		}
		delete(p.pool, addr)	// 删除pool
	}
	return nil
}

/*
	清空pool
	return : error
 */
func (p *Pool) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosedPool
	}

	for addr, list := range p.pool {
		for i := list.Len(); i != 0; i-- {
			c := list.Remove(list.Front()).(*Client)
			if p.isRecyclable(c) {
				list.PushBack(c)
			} else {
				c.Close()
			}
		}
		if list.Len() == 0 {
			delete(p.pool, addr)
		}
	}
	return nil
}

/*
	从pool中获得Client
	params: addr string[ip,port]
	return: *Client, error
 */
func (p *Pool) GetClient(addr string) (*Client, error) {
	c, err := p.getClientFromCache(addr)
	if err != nil || c != nil {
		return c, err
	}
	return NewClient(addr, p.auth, p.timeout)
}

/*
	从pool中获取List头连接
	params: addr string[ip,port]
	return: *Client, error
 */
func (p *Pool) getClientFromCache(addr string) (*Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, ErrClosedPool
	}
	if list := p.pool[addr]; list != nil {
		for i := list.Len(); i != 0; i-- {
			c := list.Remove(list.Front()).(*Client)
			if p.isRecyclable(c) {
				return c, nil
			} else {
				c.Close()
			}
		}
	}
	return nil, nil
}

/*
	put client to pool
	params: *Client
 */
func (p *Pool) PutClient(c *Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || !p.isRecyclable(c) {
		c.Close()
	} else {
		cache := p.pool[c.Addr]
		if cache == nil {
			cache = list.New()
			p.pool[c.Addr] = cache
		}
		cache.PushFront(c)
	}
}

/*
	从pool中查询client连接的redis信息
	params: client address
	return: redis info, error
 */
func (p *Pool) Info(addr string) (map[string]string, error) {
	c, err := p.GetClient(addr)	// 从pool中取出连接client
	if err != nil {
		return nil, err
	}
	defer p.PutClient(c)	// 执行c.Info()之后将连接client塞入pool
	return c.Info()
}

/*
	从pool中查询client连接的redis Full信息
	params: client address
	return: redis info, error
 */
func (p *Pool) InfoFull(addr string) (map[string]string, error) {
	c, err := p.GetClient(addr)
	if err != nil {
		return nil, err
	}
	defer p.PutClient(c)
	return c.InfoFull()
}

/*
	存储 redis 信息
 */
type InfoCache struct {
	mu sync.Mutex	// 同步锁

	Auth string	// 密码
	data map[string]map[string]string	// map[info]map[redis address]string

	Timeout time.Duration	// 超时
}

func (s *InfoCache) load(addr string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data != nil {
		return s.data[addr]
	}
	return nil
}

func (s *InfoCache) store(addr string, info map[string]string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]map[string]string)
	}
	if info != nil {
		s.data[addr] = info
	} else if s.data[addr] == nil {
		s.data[addr] = make(map[string]string)
	}
	return s.data[addr]
}

func (s *InfoCache) Get(addr string) map[string]string {
	info := s.load(addr)
	if info != nil {
		return info
	}
	info, _ = s.getSlow(addr)
	return s.store(addr, info)
}

func (s *InfoCache) GetRunId(addr string) string {
	return s.Get(addr)["run_id"]
}

func (s *InfoCache) getSlow(addr string) (map[string]string, error) {
	c, err := NewClient(addr, s.Auth, s.Timeout)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	return c.Info()
}
