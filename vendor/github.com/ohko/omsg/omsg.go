package omsg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"sync"
)

type head struct {
	Sign uint16 // 2数据标志 HK
	CRC  uint16 // 2简单crc校验值
	Size uint32 // 4数据大小
}

// ServerCallback 服务器端数据回调函数
type ServerCallback func(c net.Conn, data []byte)

// ClientCallback 客户端数据回调函数
type ClientCallback func(data []byte)

var headSize = binary.Size(head{})
var sendLock sync.Mutex

const (
	signWord = 0x4B48 // HK
)

func send(conn net.Conn, data []byte) error {
	buffer := bytes.NewBuffer(nil)
	// defer func() { log.Println("send:", conn.RemoteAddr(), hex.Dump(buffer.Bytes())) }()

	// 标识位
	sign := make([]byte, 2)
	binary.LittleEndian.PutUint16(sign, signWord)
	if _, err := conn.Write(sign); err != nil {
		return err
	}
	buffer.Write(sign)

	// CRC
	icrc := make([]byte, 2)
	binary.LittleEndian.PutUint16(icrc, crc(data))
	if _, err := conn.Write(icrc); err != nil {
		return err
	}
	buffer.Write(icrc)

	// 数据长度
	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(len(data)))
	if _, err := conn.Write(size); err != nil {
		return err
	}
	buffer.Write(size)

	// 数据
	if _, err := conn.Write(data); err != nil {
		return err
	}
	buffer.Write(data)

	return nil
}

func recv(conn net.Conn) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	// defer func() { log.Println("recv:", conn.LocalAddr(), hex.Dump(buffer.Bytes())) }()

	// 读取2字节，判断标志
	sign, err := recvHelper(conn, 2)
	if err != nil {
		return nil, err
	}
	buffer.Write(sign)
	if binary.LittleEndian.Uint16(sign) != signWord {
		return nil, errors.New("sign error")
	}

	// 读取2字节，判断CRC
	icrc, err := recvHelper(conn, 2)
	if err != nil {
		return nil, err
	}
	buffer.Write(icrc)

	// 读取数据长度
	bsize, err := recvHelper(conn, 4)
	if err != nil {
		return nil, err
	}
	buffer.Write(bsize)
	size := binary.LittleEndian.Uint32(bsize)
	if size <= 0 {
		return nil, errors.New("size error")
	}

	// 3. 读取数据
	buf, err := recvHelper(conn, int(size))
	buffer.Write(buf)

	if binary.LittleEndian.Uint16(icrc) != crc(buf) {
		return nil, errors.New("sign error")
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
