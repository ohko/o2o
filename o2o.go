package o2o

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// ...
const (
	CMDMSG     = "\x00" // 普通信息
	CMDCLIENT  = "\x01" // 1.客户端请求TCP隧道服务
	CMDREQUEST = "\x02" // 2.服务器发送浏览器请求服务
	CMDTENNEL  = "\x03" // 3.客户端建立TCP隧道连接
	signWord   = 0x4B48 // HK
)

// CatchCtrlC 捕捉Ctrl+C
func CatchCtrlC() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}

// Conn2IP 返回IP地址
func Conn2IP(conn net.Conn) string {
	return strings.Split(conn.RemoteAddr().String(), ":")[0]
}

// Send 发送数据
func Send(conn net.Conn, cmd, data string) error {
	dataBuf := make([]byte, len(cmd)+len(data))
	copy(dataBuf, cmd)
	copy(dataBuf[len(cmd):], data)

	// buffer := bytes.NewBuffer(nil)
	// defer func() { log.Println("send:", conn.RemoteAddr(), hex.Dump(buffer.Bytes())) }()

	// 标识位
	sign := make([]byte, 2)
	binary.LittleEndian.PutUint16(sign, signWord)
	if _, err := conn.Write(sign); err != nil {
		return err
	}
	// buffer.Write(sign)

	// CRC
	icrc := make([]byte, 2)
	binary.LittleEndian.PutUint16(icrc, crc(dataBuf))
	if _, err := conn.Write(icrc); err != nil {
		return err
	}
	// buffer.Write(icrc)

	// 数据长度
	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(len(dataBuf)))
	if _, err := conn.Write(size); err != nil {
		return err
	}
	// buffer.Write(size)

	// 数据
	if _, err := conn.Write(dataBuf); err != nil {
		return err
	}
	// buffer.Write(dataBuf)

	return nil
}

// Recv 接收数据
func Recv(conn net.Conn) ([]byte, error) {
	// buffer := bytes.NewBuffer(nil)
	// defer func() { log.Println("recv:", conn.LocalAddr(), hex.Dump(buffer.Bytes())) }()

	// 读取2字节，判断标志
	sign, err := recvHelper(conn, 2)
	if err != nil {
		return nil, err
	}
	// buffer.Write(sign)
	if binary.LittleEndian.Uint16(sign) != signWord {
		return nil, errors.New("sign error")
	}

	// 读取2字节，判断CRC
	icrc, err := recvHelper(conn, 2)
	if err != nil {
		return nil, err
	}
	// buffer.Write(icrc)

	// 读取数据长度
	bsize, err := recvHelper(conn, 4)
	if err != nil {
		return nil, err
	}
	// buffer.Write(bsize)
	size := binary.LittleEndian.Uint32(bsize)
	if size <= 0 {
		return nil, errors.New("size error")
	}

	// 3. 读取数据
	buf, err := recvHelper(conn, int(size))
	// buffer.Write(buf)

	if binary.LittleEndian.Uint16(icrc) != crc(buf) {
		return nil, errors.New("crc error")
	}
	return buf, nil
}

func recvHelper(conn net.Conn, size int) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	tmpBuf := make([]byte, size)
	for { // 从数据流读取足够量的数据
		n, err := conn.Read(tmpBuf)
		if err != nil {
			return nil, err
		}
		buf.Write(tmpBuf[:n])

		// 够了
		if buf.Len() == int(size) {
			break
		}

		// 继续读取差额数据量
		tmpBuf = make([]byte, int(size)-buf.Len())
	}
	return buf.Bytes(), nil
}
func crc(data []byte) uint16 {
	size := len(data)
	crc := 0xFFFF
	for i := 0; i < size; i++ {
		crc = (crc >> 8) ^ int(data[i])
		for j := 0; j < 8; j++ {
			flag := crc & 0x0001
			crc >>= 1
			if flag == 1 {
				crc ^= 0xA001
			}
		}
	}
	return uint16(crc)
}
