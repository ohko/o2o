package omsg

import (
	"net"
	"time"
)

// Client ...
type Client struct {
	client  net.Conn       // 用户客户端
	sync    bool           // 同步请求
	OnData  ClientCallback // 收到命令行回调
	OnClose func()         // 连接断开回调
}

// NewClient 创建客户端
func NewClient(sync bool) *Client {
	o := &Client{sync: sync}
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
	if !o.sync {
		go o.hClient()
	}
	return nil
}

// 监听数据
func (o *Client) hClient() {
	for {
		bs, err := recv(o.client)
		if err != nil {
			break
		}
		if o.OnData != nil {
			go o.OnData(bs)
		}
	}

	o.Close()
	if o.OnClose != nil {
		go o.OnClose()
	}
}

// SendAsync 异步向服务器发送数据
func (o *Client) SendAsync(data []byte) error {
	return send(o.client, data)
}

// SendSync 同步向服务器发送数据
func (o *Client) SendSync(data []byte) ([]byte, error) {

	if err := send(o.client, data); err != nil {
		return nil, err
	}

	return recv(o.client)
}

// Close 关闭链接
func (o *Client) Close() {
	o.client.Close()
}
