package omsg

import (
	"net"
	"sync"
	"time"
)

// Server 服务器
type Server struct {
	si            ServerInterface
	crc           bool         // 是否启用crc校验
	Listener      net.Listener // 用于服务器
	ClientList    sync.Map     // 客户端列表
	readDeadline  time.Duration
	writeDeadline time.Duration
}

// Listen 创建
func Listen(network, address string) (*Server, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	return &Server{Listener: listener}, nil
}

// Run 监听连接
func (o *Server) Run(si ServerInterface, crc bool) error {
	o.si = si
	o.crc = crc
	for {
		conn, err := o.Listener.Accept()
		if err != nil {
			return err
		}
		go o.accept(conn)
	}
}

// 接收数据
func (o *Server) accept(conn net.Conn) {
	defer conn.Close()

	// 新客户端回调
	if !o.si.OnAccept(conn) {
		return
	}

	// 记录客户端联入时间
	o.ClientList.Store(conn, time.Now())

	// 断线
	defer func() {
		// 从客户端列表移除
		o.ClientList.Delete(conn)

		// 回调
		o.si.OnClientClose(conn)
	}()

	for {
		if o.readDeadline > 0 {
			conn.SetReadDeadline(time.Now().Add(o.readDeadline))
		}
		cmd, ext, bs, err := Recv(o.crc, conn)
		if err != nil {
			o.si.OnRecvError(conn, err)
			break
		}
		if err := o.si.OnData(conn, cmd, ext, bs); err != nil {
			break
		}
	}
}

// Send 向客户端发送数据
func (o *Server) Send(conn net.Conn, cmd, ext uint16, data []byte) error {
	if o.writeDeadline > 0 {
		conn.SetWriteDeadline(time.Now().Add(o.writeDeadline))
	}
	return Send(o.crc, conn, cmd, ext, data)
}

// SendToAll 向所有客户端发送数据
func (o *Server) SendToAll(cmd, ext uint16, data []byte) {
	o.ClientList.Range(func(key, value interface{}) bool {
		if o.writeDeadline > 0 {
			key.(net.Conn).SetWriteDeadline(time.Now().Add(o.writeDeadline))
		}
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

// SetReadDeadline ...
func (o *Server) SetReadDeadline(deadline time.Duration) {
	o.readDeadline = deadline
}

// SetWriteDeadline ...
func (o *Server) SetWriteDeadline(deadline time.Duration) {
	o.writeDeadline = deadline
}

// SetDeadline ...
func (o *Server) SetDeadline(deadline time.Duration) {
	o.readDeadline = deadline
	o.writeDeadline = deadline
}
