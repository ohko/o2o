package omsg

import (
	"net"
	"sync"
	"time"
)

// Server 服务器
type Server struct {
	si         ServerInterface
	crc        bool         // 是否启用crc校验
	Listener   net.Listener // 用于服务器
	ClientList sync.Map     // 客户端列表
}

// NewServer 创建
func NewServer(si ServerInterface, crc bool) *Server {
	return &Server{si: si, crc: crc}
}

// StartServer 启动服务
func (o *Server) StartServer(laddr string) error {
	var err error
	if o.Listener, err = net.Listen("tcp", laddr); err != nil {
		return err
	}

	// 监听端口
	for {
		conn, err := o.Listener.Accept()
		if err != nil {
			return err
		}
		go o.hServer(conn)
	}
}

// 接收数据
func (o *Server) hServer(conn net.Conn) {
	// 记录客户端联入时间
	o.ClientList.Store(conn, time.Now())

	// 新客户端回调
	o.si.OmsgNewClient(conn)

	// 断线
	defer func() {
		// 从客户端列表移除
		o.ClientList.Delete(conn)
		conn.Close()

		// 回调
		o.si.OmsgClientClose(conn)
	}()

	for {
		cmd, ext, bs, err := Recv(o.crc, conn)
		if err != nil {
			o.si.OmsgError(conn, err)
			break
		}
		o.si.OmsgData(conn, cmd, ext, bs)
	}
}

// Send 向客户端发送数据
func (o *Server) Send(conn net.Conn, cmd, ext uint16, data []byte) error {
	return Send(o.crc, conn, cmd, ext, data)
}

// SendToAll 向所有客户端发送数据
func (o *Server) SendToAll(cmd, ext uint16, data []byte) {
	o.ClientList.Range(func(key, value interface{}) bool {
		Send(o.crc, key.(net.Conn), cmd, ext, data)
		return true
	})
}

// Close 关闭服务器
func (o *Server) Close() {
	o.Listener.Close()
	o.ClientList.Range(func(key, value interface{}) bool {
		key.(net.Conn).Close()
		return true
	})
}
