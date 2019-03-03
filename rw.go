package o2o

import (
	"crypto/aes"
	"crypto/cipher"
	"net"
)

type rw struct {
	conn   net.Conn
	origin net.Conn // 远端(浏览器和Web站点)数据读写时不能加密
}

func (o *rw) Write(p []byte) (n int, err error) {
	if o.conn == o.origin {
		return o.conn.Write(p)
	}
	e, err := aesEncode(p)
	if err != nil {
		return 0, err
	}
	return o.conn.Write(e)
}
func (o *rw) Read(p []byte) (n int, err error) {
	if o.conn == o.origin {
		n, err = o.conn.Read(p)
		return
	}
	n, err = o.conn.Read(p)
	if err != nil {
		return n, err
	}

	e, err := aesEncode(p[:n])
	if err != nil {
		return n, err
	}
	copy(p, e)
	return
}

func aesEncode(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}
	buf := make([]byte, len(data))

	stream := cipher.NewCTR(block, aesIV[:])
	stream.XORKeyStream(buf, data)
	return buf, nil
}
