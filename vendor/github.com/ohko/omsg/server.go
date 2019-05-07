package omsg

import (
	"net"
	"sync"
	"time"
)

// Server 服务器
type Server struct {
	server        net.Listener                                      // 用于服务器
	OnNewClient   func(conn net.Conn)                               // 新客户端回调
	OnData        func(conn net.Conn, cmd, ext uint16, data []byte) // 数据回调
	OnError       func(conn net.Conn, err error)                    // 错误回调
	OnClientClose func(conn net.Conn)                               // 客户端断开回调
	clientList    sync.Map                                          // 客户端列表
}

// NewServer 创建
func NewServer() *Server {
	return new(Server)
}

// StartServer 启动服务
func (o *Server) StartServer(laddr string) error {
	var err error
	if o.server, err = net.Listen("tcp", laddr); err != nil {
		return err
	}

	// 监听端口
	for {
		conn, err := o.server.Accept()
		if err != nil {
			return err
		}
		go o.hServer(conn)
	}

	return nil
}

// 接收数据
func (o *Server) hServer(conn net.Conn) {
	// 记录客户端联入时间
	o.clientList.Store(conn, time.Now())

	// 新客户端回调
	if o.OnNewClient != nil {
		o.OnNewClient(conn)
	}

	// 断线
	defer func() {
		// 从客户端列表移除
		o.clientList.Delete(conn)
		conn.Close()

		// 回调
		if o.OnClientClose != nil {
			o.OnClientClose(conn)
		}
	}()

	for {
		cmd, ext, bs, err := recv(conn)
		if err != nil {
			if o.OnError != nil {
				o.OnError(conn, err)
			}
			break
		}
		if o.OnData != nil {
			o.OnData(conn, cmd, ext, bs)
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
