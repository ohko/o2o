package omsg

import (
	"net"
	"sync"
	"time"
)

// Server 服务器
type Server struct {
	server        net.Listener          // 用于服务器
	OnNewClient   func(c net.Conn)      // 新客户端回调
	OnData        ServerCallback        // 数据回调
	OnClientClose func(c net.Conn)      // 客户端断开回调
	ClientList    map[net.Conn]*SClient // 客户端列表
	lock          sync.Mutex
}

// SClient 服务器客户端对象
type SClient struct {
	Conn net.Conn  // 客户端连接
	Time time.Time // 连入时间
}

// NewServer 创建
func NewServer() *Server {
	o := &Server{ClientList: make(map[net.Conn]*SClient)}
	return o
}

// StartServer 启动服务
func (o *Server) StartServer(laddr string) error {
	var err error
	if o.server, err = net.Listen("tcp", laddr); err != nil {
		return err
	}
	return o.hListener(o.server)
}

// 监听端口
func (o *Server) hListener(s net.Listener) error {
	for {
		conn, err := s.Accept()
		if err != nil {
			return err
		}
		go o.hServer(conn)
	}
}

// 接收数据
func (o *Server) hServer(conn net.Conn) {
	// 记录客户端
	o.lock.Lock()
	o.ClientList[conn] = &SClient{Conn: conn, Time: time.Now()}
	o.lock.Unlock()

	// 新客户端回调
	if o.OnNewClient != nil {
		go o.OnNewClient(conn)
	}

	for {
		bs, err := recv(conn)
		if err != nil {
			break
		}
		if o.OnData != nil {
			go o.OnData(conn, bs)
		}
	}

	// 断线
	if o.OnClientClose != nil {
		go o.OnClientClose(conn)
	}

	// 从客户端列表移除
	o.lock.Lock()
	delete(o.ClientList, conn)
	o.lock.Unlock()
}

// SendToAll 向所有客户端发送数据
func (o *Server) SendToAll(custom uint32, data []byte) {
	o.lock.Lock()
	defer o.lock.Unlock()
	for _, v := range o.ClientList {
		o.Send(v.Conn, data)
	}
}

// Send 向指定客户端发送数据
func (o *Server) Send(c net.Conn, data []byte) error {
	return send(c, data)
}

// Close 关闭服务器
func (o *Server) Close() {
	o.server.Close()
}
