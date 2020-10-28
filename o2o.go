package o2o

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"os"
	"os/signal"
	"syscall"

	"github.com/ohko/logger"
)

// ...
const (
	cmdHeart         = 0 // 0.心跳包
	cmdTunnel        = 1 // 1.客户端请求TCP隧道服务
	cmdTunnelSuccess = 2 // 2.服务器监听成功
	cmdTunnelFailed  = 3 // 3.服务器监听失败
	cmdData          = 4 // 4.数据流
	cmdUserClose     = 5 // 5.User关闭连接
	cmdLocaSrveClose = 6 // 6.LocalServer关闭或连接失败
	bufferSize       = 1024 * 1024
)

var (
	aesEnable bool
	aesKey    [32]byte
	aesIV     [16]byte
	lServer   = logger.NewLogger(nil)
	lClient   = logger.NewLogger(nil)
)

// WaitCtrlC 捕捉Ctrl+C
func WaitCtrlC() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}

func enData(browserAddr, serveAddr string, browserData []byte) []byte {
	sum := 2 + len(browserAddr) + 2 + len(serveAddr) + 4 + len(browserData)
	dataBuf := make([]byte, sum)
	// defer func() { log.Println("send:\n" + hex.Dump(dataBuf)) }()

	len1 := len(browserAddr)
	binary.LittleEndian.PutUint16(dataBuf[0:2], uint16(len1))
	copy(dataBuf[2:2+len1], []byte(browserAddr))

	len2 := len(serveAddr)
	binary.LittleEndian.PutUint16(dataBuf[2+len1:], uint16(len2))
	copy(dataBuf[2+len1+2:], []byte(serveAddr))

	len3 := len(browserData)
	binary.LittleEndian.PutUint16(dataBuf[2+len1+2+len2:], uint16(len3))
	copy(dataBuf[2+len1+2+len2+4:], browserData)

	return dataBuf
}

func deData(data []byte) (browserAddr string, serveAddr string, browserData []byte) {
	// log.Println("recv:\n" + hex.Dump(data))
	len1 := binary.LittleEndian.Uint16(data[0:2])
	len2 := binary.LittleEndian.Uint16(data[2+len1:])
	return string(data[2 : 2+len1]), string(data[2+len1+2 : 2+len1+2+len2]), data[2+len1+2+len2+4:]
}

func aesCrypt(data []byte) []byte {
	if !aesEnable {
		return data
	}
	block, _ := aes.NewCipher(aesKey[:])
	buf := make([]byte, len(data))

	stream := cipher.NewCTR(block, aesIV[:])
	stream.XORKeyStream(buf, data)
	return buf
}
