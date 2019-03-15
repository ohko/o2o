package omsg

import (
	"net"
	"sync"
	"time"
)

// Server 服务器
type Server struct {
	server        net.Listener                                                 // 用于服务器
	OnNewClient   func(conn net.Conn)                                          // 新客户端回调
	OnData        func(conn net.Conn, cmd, ext uint16, data []byte, err error) // 数据回调
	OnClientClose func(conn net.Conn)                                          // 客户端断开回调
	clientList    sync.Map                                                     // 客户端列表
}

// NewServer 创建
func NewServer() *Server {
	o := new(Server)
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
	// 记录客户端联入时间
	o.clientList.Store(conn, time.Now())

	// 新客户端回调
	if o.OnNewClient != nil {
		go o.OnNewClient(conn)
	}

	// 从客户端列表移除
	defer o.clientList.Delete(conn)

	// 断线
	if o.OnClientClose != nil {
		defer o.OnClientClose(conn)
	}

	for {
		cmd, ext, bs, err := recv(conn)
		switch err.(type) {
		case *DataError, nil:
			if o.OnData != nil {
				o.OnData(conn, cmd, ext, bs, err)
			}
		default:
			return
		}
	}
}

// SendToAll 向所有客户端发送数据
func (o *Server) SendToAll(cmd, ext uint16, data []byte) {
	o.clientList.Range(func(key, value interface{}) bool {
		o.Send(key.(net.Conn), cmd, ext, data)
		return true
	})
}

// Send 向指定客户端发送数据
func (o *Server) Send(conn net.Conn, cmd, ext uint16, data []byte) error {
	return send(conn, cmd, ext, data)
}

// Close 关闭服务器
func (o *Server) Close() {
	o.server.Close()
}
