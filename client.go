package o2o

import (
	"crypto/md5"
	"crypto/sha256"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ohko/omsg"
)

// Client 客户端
type Client struct {
	msg                   *omsg.Client
	serverPort, proxyPort string
	localServers          sync.Map // map[浏览器IP:Port + 本地服务IP:Port]本地服务连接
	localServersCount     int64    // 连接数
}

// Start 启动客户端
func (o *Client) Start(key, serverPort, proxyPort string, crc bool) error {
	lClient.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	lClient.SetPrefix("C")
	lClient.SetColor(true)
	o.serverPort, o.proxyPort = serverPort, proxyPort

	// setup AES
	if len(key) > 0 {
		aesEnable = true
		aesKey = sha256.Sum256([]byte(key))
		aesIV = md5.Sum([]byte(key))
		lClient.Log4Trace("AES crypt enabled")
	} else {
		lClient.Log4Trace("AES crypt disabled")
	}

	o.msg = omsg.NewClient(o, crc)
	return o.Reconnect()
}

// OmsgError ...
func (o *Client) OmsgError(err error) {
	if err != io.EOF {
		lClient.Log2Error(err)
	}
}

// OmsgClose ...
func (o *Client) OmsgClose() {
	lClient.Log0Debug("connect closed:", o.serverPort, o.proxyPort)

	// 清理本地数据
	o.localServers.Range(func(key, val interface{}) bool {
		val.(net.Conn).Close()
		o.localServers.Delete(key)
		atomic.AddInt64(&o.localServersCount, -1)
		return true
	})

	// 断线后重连
	o.Reconnect()
}

// OmsgData ...
func (o *Client) OmsgData(cmd, ext uint16, data []byte) {
	data = aesCrypt(data)
	// lClient.Log0Debug(fmt.Sprintf("0x%x-0x%x:\n%s", cmd, ext, hex.Dump(data)))

	switch cmd {
	case cmdTunnelSuccess: // 通道创建成功
		lClient.Log0Debug("tunnel success:", o.serverPort, string(data))
	case cmdTunnelFailed: // 通道创建失败，断开连接
		lClient.Log2Error("tunnel failed:", o.serverPort, string(data))
		o.msg.Close()
	case cmdUserClose: // 远端浏览器关闭
		if local, ok := o.localServers.Load(string(data)); ok {
			lClient.Log0Debug("close local connect:", string(data))
			local.(net.Conn).Close()
		}
	case cmdData: // 远端浏览器数据流
		o.browserDataStream(deData(data))
	}
}

// Send 原始数据加密后发送
func (o *Client) Send(cmd, ext uint16, originData []byte) error {
	return o.msg.Send(cmd, ext, aesCrypt(originData))
}

// Reconnect 重新连接服务器
func (o *Client) Reconnect() error {
	for {
		if err := o.msg.Connect(o.serverPort); err != nil {
			lClient.Log2Error(err)
			time.Sleep(time.Second)
			continue
		}

		// 连接成功，发送Tunnel请求
		o.Send(cmdTunnel, 0, []byte(o.proxyPort))
		break
	}
	lClient.Log0Debug("connect success:", o.serverPort, o.proxyPort)

	return nil
}

// 处理浏览器数据流
func (o *Client) browserDataStream(browserAddr, serveAddr string, browserData []byte) {
	var (
		err   error
		local net.Conn
	)

	defer func() {
		if err != nil {
			// 通知本地服务错误
			lClient.Log0Debug("local local error:", err)
			if err := o.Send(cmdLocaSrveClose, 0, enData(browserAddr, serveAddr, []byte(err.Error()))); err != nil {
				lClient.Log2Error(err)
			}
		}
	}()

	// 此浏览器的请求是否已有本地服务连接
	if _conn, ok := o.localServers.Load(browserAddr + serveAddr); ok {
		local = _conn.(net.Conn)
	}

	// 本地还没有与服务建立连接，创建一个新的服务连接
	if local == nil {
		// addr[0.0.0.0:2345:192.168.1.240:5000]
		tmp := strings.Split(serveAddr, ":")
		local, err = net.DialTimeout("tcp", strings.Join(tmp[2:], ":"), time.Second*3)
		if err != nil {
			return
		}
		lClient.Log0Debug("create local connect:", local.LocalAddr())
		o.localServers.Store(browserAddr+serveAddr, local)
		atomic.AddInt64(&o.localServersCount, 1)

		go func(local net.Conn, browserAddr, serveAddr string) {
			defer local.Close()
			lClient.Log0Debug("local start:", local.LocalAddr())
			buf := make([]byte, 1024)
			// 转发服务的数据
			for {
				n, err := local.Read(buf)
				if err != nil {
					break
				}
				// 数据转发到远端
				if err := o.Send(cmdData, 0, enData(browserAddr, serveAddr, buf[:n])); err != nil {
					// 发送失败，关闭本次通道
					local.Close()
					break
				}
			}

			// 通知本地服务错误
			if err := o.Send(cmdLocaSrveClose, 0, enData(browserAddr, serveAddr, []byte("local server close"))); err != nil {
				lClient.Log2Error(err)
			}

			lClient.Log0Debug("local end:", local.LocalAddr())
			o.localServers.Delete(browserAddr + serveAddr)
			atomic.AddInt64(&o.localServersCount, -1)
		}(local, browserAddr, serveAddr)
	}

	// 数据转发到本地服务
	if _, err := local.Write(browserData); err != nil {
		// 发送失败，关闭本次通道
		local.Close()
		return
	}
}
