package omsg

import (
	"io"
	"net"
	"time"
)

// Client ...
type Client struct {
	client  net.Conn                                      // 用户客户端
	OnData  func(cmd, ext uint16, data []byte, err error) // 收到命令行回调
	OnClose func()                                        // 连接断开回调
}

// NewClient 创建客户端
func NewClient() *Client {
	o := new(Client)
	return o
}

// Connect 连接到服务器
func (o *Client) Connect(address string) error {
	return o.ConnectTimeout(address, 0)
}

// ConnectTimeout 连接到服务器
func (o *Client) ConnectTimeout(address string, timeout time.Duration) error {
	var err error
	if o.client, err = net.DialTimeout("tcp", address, timeout); err != nil {
		return err
	}
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	for {
		cmd, ext, bs, err := recv(o.client)
		if err != nil && err == io.EOF {
			break
		}
		if o.OnData != nil {
			o.OnData(cmd, ext, bs, err)
		}
	}

	o.Close()
	if o.OnClose != nil {
		go o.OnClose()
	}
}

// Send 向服务器发送数据
func (o *Client) Send(cmd, ext uint16, data []byte) error {
	return send(o.client, cmd, ext, data)
}

// Close 关闭链接
func (o *Client) Close() {
	o.client.Close()
}
