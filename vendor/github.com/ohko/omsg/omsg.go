package omsg

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
)

// DataError omsg error
type DataError struct {
	msg string
}

func (e *DataError) Error() string {
	return e.msg
}

type head struct {
	Sign uint16 // 2数据标志 HK
	CRC  uint16 // 2简单crc校验值
	Cmd  uint16 // 2指令代码
	Ext  uint16 // 2自定义扩展
	Size uint32 // 4数据大小
}

var headSize = binary.Size(head{})
var sendLock sync.Mutex

const (
	signWord = 0x4B48 // HK
)

func send(conn net.Conn, cmd, ext uint16, data []byte) error {
	// make buffer, sign+2 crc+2 size+4 len(data)
	buffer := make([]byte, 0xC+len(data))
	// defer func() { log.Println("send:\n" + hex.Dump(buffer)) }()

	// sign
	binary.LittleEndian.PutUint16(buffer, signWord)

	// CRC
	binary.LittleEndian.PutUint16(buffer[2:], crc(data))

	// Cmd
	binary.LittleEndian.PutUint16(buffer[4:], cmd)

	// Ext
	binary.LittleEndian.PutUint16(buffer[6:], ext)

	// data length
	binary.LittleEndian.PutUint32(buffer[8:], uint32(len(data)))

	// data
	copy(buffer[0xC:], data)

	// send
	if _, err := conn.Write(buffer); err != nil {
		return err
	}

	return nil
}

func recv(conn net.Conn) (uint16, uint16, []byte, error) {

	header := make([]byte, 0xC)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, 0, nil, err
	}
	// log.Println("recv header:\n" + hex.Dump(header))

	if signWord != binary.LittleEndian.Uint16(header) {
		return 0, 0, nil, &DataError{msg: "sign err"}
	}
	icrc := binary.LittleEndian.Uint16(header[2:])
	cmd := binary.LittleEndian.Uint16(header[4:])
	ext := binary.LittleEndian.Uint16(header[6:])
	size := binary.LittleEndian.Uint32(header[8:])
	buffer := make([]byte, int(size))
	if _, err := io.ReadFull(conn, buffer); err != nil {
		return 0, 0, nil, err
	}
	// log.Println("recv buffer:\n" + hex.Dump(buffer))

	if icrc != crc(buffer) {
		return 0, 0, nil, &DataError{msg: "crc err"}
	}

	return cmd, ext, buffer, nil
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
