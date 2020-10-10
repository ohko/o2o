package main

import (
	"flag"
	"log"
	"o2o"
	"runtime"
)

var (
	serverPort = flag.String("s", ":2399", "外网服务器服务端口，用于接收内网服务器连接")
	key        = flag.String("key", "20190303", "密钥，留空不启用AES加密")
	crc        = flag.Bool("crc", true, "是否启动crc校验数据")
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Flags() | log.Lshortfile)

	o := &o2o.Server{}
	if err := o.Start(*key, *serverPort, *crc); err != nil {
		log.Fatal(err)
	}

	o2o.WaitCtrlC()
}
