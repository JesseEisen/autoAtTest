## AT Auto Test

一个简易的 AT 基础命令测试工具，用于自动回归测试当前 AT 命令。测试方法如下:

- 编写 case 文件， 保存到 `design.md` 中
- 运行 test.exe 

## 编译运行

+ 安装 [Go](https://golangtc.com/download)
+ 安装 Go-serial, 在 git shell 中运行 go get github.com/jacobsa/go-serial/serial
+ 编译生成 exe 文件， go build test.go

## case 编写规则

```
#sleep=3
#port=COM11
#baudrate=115200
```

这三行作为对当前板卡基本信息的配置，其中 `sleep` 字段是作为每个命令之间运行的间隔时间; `port` 则为当前用于 AT 读写的端口号。

```
send AT+GMI
read [xxxx, xxx, OK]
```

case 的编写比较简单，`send` 表示发送 AT 命令， AT 命令可以是串联形式的。`read` 表示读取串口的返回值，列表形式（即两边使用`[]`）。如果返回值有多个，可以按照实际期望输出填写，并以逗号隔开即可。**需要保证 `send` `read` 成对出现。**


## 基本原理

测试工具是用 Go 语言编写，不过基本的思路很简单：

- 读取 case，将 case 的基本信息保存到一个 map 中
- 逐条执行相应的 case
- 待所有的 case 执行完毕后，进行结果的比对与输出。

所以在测试过程中会发现，运行的时间比较长，而最终输出结果时比较快，不过这个做法，在程序遇到问题时可能会丢失掉测试结果。后续根据实际情况做一些优化。
