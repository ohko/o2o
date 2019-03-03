package o2o

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
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
	CMDSUCCESS = "\x02" // 2.服务器监听成功
	CMDREQUEST = "\x03" // 3.服务器发送浏览器请求服务
	CMDTENNEL  = "\x04" // 4.客户端建立TCP隧道连接
	signWord   = 0x4B48 // HK
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// WaitCtrlC 捕捉Ctrl+C
func WaitCtrlC() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}

// 返回IP地址
func conn2IP(conn net.Conn) string {
	return strings.Split(conn.RemoteAddr().String(), ":")[0]
}

// 发送数据
func send(conn net.Conn, cmd, data string) error {
	// 标志2+CRC2+长度4+len(CMD)+len(data)
	sum := 8 + len(cmd) + len(data)
	dataBuf := make([]byte, sum)
	// defer func() { log.Println("send:", conn.RemoteAddr(), "\n"+hex.Dump(dataBuf)) }()

	// CMD
	copy(dataBuf[8:], cmd)

	// data
	copy(dataBuf[8+len(cmd):], data)

	// 标识位
	binary.LittleEndian.PutUint16(dataBuf, signWord)

	// CRC
	binary.LittleEndian.PutUint16(dataBuf[2:], crc(dataBuf[8:]))

	// 数据长度
	binary.LittleEndian.PutUint32(dataBuf[4:], uint32(len(cmd)+len(data)))

	// 数据
	n, err := conn.Write(dataBuf)
	if err != nil {
		return err
	}
	if n != sum {
		return fmt.Errorf("%d!=%d", sum, n)
	}

	return nil
}

// 接收数据
func recv(conn net.Conn) ([]byte, error) {
	var header, buf []byte
	var err error
	// defer func() { log.Println("recv:", conn.LocalAddr(), "\n"+hex.Dump(header)+hex.Dump(buf)) }()

	// 预取8字节，标志2+CRC2+长度4
	header, err = recvHelper(conn, 8)
	if err != nil {
		return nil, err
	}

	// 读取2字节，判断标志
	if binary.LittleEndian.Uint16(header[:2]) != signWord {
		return nil, errors.New("sign error")
	}

	// 读取数据长度
	size := binary.LittleEndian.Uint32(header[4:])
	if size <= 0 {
		return nil, errors.New("size error")
	}

	// 3. 读取数据
	buf, err = recvHelper(conn, int(size))

	if binary.LittleEndian.Uint16(header[2:4]) != crc(buf) {
		return nil, errors.New("crc error")
	}
	return buf, nil
}

// 读取足够量的数据，返回数据副本
func recvHelper(conn net.Conn, size int) ([]byte, error) {
	buf := make([]byte, size)
	bufPos := 0
	tmpBuf := make([]byte, size)
	for {
		n, err := conn.Read(tmpBuf)
		if err != nil {
			return nil, err
		}

		// 一次就拿到预期值，直接返回数据
		if bufPos == 0 && n == size {
			return tmpBuf[:n], nil
		}

		// 移动拿到的数据
		copy(buf[bufPos:], tmpBuf[:n])
		bufPos += n

		// 如果够了
		if bufPos == size {
			break
		}

		// 继续读取差额数据量
		tmpBuf = make([]byte, size-bufPos)
	}
	return buf, nil
}

// 数据校验
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
