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
	CMDTUNNEL     = 1 // 1.客户端请求TCP隧道服务
	CMDSUCCESS    = 2 // 2.服务器监听成功
	CMDFAILED     = 3 // 3.服务器监听失败
	CMDDATA       = 4 // 4.数据流
	CMDCLOSE      = 5 // 5.浏览器关闭连接
	CMDLOCALCLOSE = 6 // 6.本地服务关闭或连接失败
	bufferSize    = 1024 * 1024
)

var (
	aesEnable bool
	aesKey    [32]byte
	aesIV     [16]byte
	ll        = logger.NewLogger(nil)
)

// WaitCtrlC 捕捉Ctrl+C
func WaitCtrlC() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}

func enData(client, addr string, data []byte) []byte {
	sum := 2 + len(client) + 2 + len(addr) + 4 + len(data)
	dataBuf := make([]byte, sum)
	// defer func() { log.Println("send:\n" + hex.Dump(dataBuf)) }()

	len1 := len(client)
	binary.LittleEndian.PutUint16(dataBuf[0:2], uint16(len1))
	copy(dataBuf[2:2+len1], []byte(client))

	len2 := len(addr)
	binary.LittleEndian.PutUint16(dataBuf[2+len1:], uint16(len2))
	copy(dataBuf[2+len1+2:], []byte(addr))

	len3 := len(data)
	binary.LittleEndian.PutUint16(dataBuf[2+len1+2+len2:], uint16(len3))
	copy(dataBuf[2+len1+2+len2+4:], data)

	return dataBuf
}

func deData(data []byte) (string, string, []byte) {
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
