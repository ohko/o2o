package main

import (
	"flag"
	"log"
	"o2o"
)

var (
	proxyPort  = flag.String("p", "0.0.0.0:2345:192.168.0.200:5000", "请求创建隧道：internetPort:intranet:port")
	serverPort = flag.String("s", ":2399", "外网服务器")
	key        = flag.String("key", "20190303", "密钥，留空不启用AES加密")
)

func main() {
	flag.Parse()

	o := &o2o.Client{}
	if err := o.Start(*key, *serverPort, *proxyPort); err != nil {
		log.Fatal(err)
	}

	o2o.WaitCtrlC()
}
