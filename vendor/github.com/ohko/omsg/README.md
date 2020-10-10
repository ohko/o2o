[![github.com/ohko/omsg](https://goreportcard.com/badge/github.com/ohko/omsg)](https://goreportcard.com/report/github.com/ohko/omsg)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/e749fe7b9e414750b577556b6f0c9a2a)](https://www.codacy.com/app/ohko/omsg?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=ohko/omsg&amp;utm_campaign=Badge_Grade)

# omsg
通过TCP建立长连接通讯，解决拆包和粘包的问题。

## 数据结构
| 标志2字节  | 数据CRC校验值2字节 | 指令代码2字节 | 自定义扩展2字节 | 数据尺寸4字节         | 数据 |
|----------:|-----------------:|------------:|--------------:|--------------------:|-----|
| 0x48,0x4b | 0x81,0x91        | 0x01,0x00   | 0x02,0x00     | 0x18,0x00,0x00,0x00 | ... |
|        HK |    0x9181        |       0x1   |       0x2     |                0x18 | ... |

```
00000000  48 4b 81 91 01 00 02 00  14 00 00 00 31 32 33 34  |HK..........1234|
00000010  35 36 37 38 31 32 33 34  35 36 37 38 31 32 33 34  |5678123456781234|
```

## 使用
```shell
$ go get -u github.com/ohko/omsg
```

使用方法：[main_test.go](main_test.go)