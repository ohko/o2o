package main

import (
	"flag"
	"log"
	"o2o"
	"runtime"
)

var (
	proxyPort  = flag.String("p", "0.0.0.0:2345:192.168.0.200:5000", "请求创建隧道：listenHost:listenPort:proxyHost:ProxyPort")
	serverPort = flag.String("s", ":2399", "外网服务器")
	key        = flag.String("key", "20190303", "密钥，留空不启用AES加密")
	crc        = flag.Bool("crc", true, "是否启动crc校验数据")
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Flags() | log.Lshortfile)

	o := &o2o.Client{}
	if err := o.Start(*key, *serverPort, *proxyPort, *crc); err != nil {
		log.Fatal(err)
	}

	o2o.WaitCtrlC()
}
