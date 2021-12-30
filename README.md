# Got

------

## 简介

got 是一个由 Go 语言实现的小工具，用于从远程主机查看、下载、上传文件。

* got 是 C/S 架构，C 端使用控制台指令对 S 端文件进行操作
* got 使用 gRPC 实现通讯

Got 的优势：

* 支持文件，文件夹的上传与下载。
* client 可从多个 server 操作文件。

* 轻量，无配置，无静态依赖。
* 易部署，server，client 端都运行一个二进制可执行文件即可。
* 支持跨平台文件操作，使用 Go 1.17 编译到多平台即可（包括树莓派）。

存在的问题：

* 默认使用 gRPC 的 insecure 认证，数据传输不加密。
* 未实现上传下载的 md5 校验。
* 不支持 client 高并发访问单 server。

项目结构：

```bash
got
├── cmd
│     ├── client
│     │     └── main.go
│     └── server
│       └── main.go
├── go.mod
├── go.sum
├── internal
│     ├── client.go
│     ├── message.pb.go
│     └── server.go
├── LICENSE
├── pkg
│     └── tool.go
├── protos
│     └── message.proto
└── README.md
```
-----

## 部署指南

运行 `got-server` 可执行文件，或在项目目录下执行一下指令即可。

```bash
# 需要 Go 1.17 运行
$ go run cmd/server/main.go
```

Got 服务器默认监听 9876 端口。如需更改端口可使用 `-p` 传入指定的端口，如下示例指定监听 8008 端口。

```bash
$ ./got-server -p 8008
```

或

```bash
$ go run cmd/server/main.go -p 8008
```

> 注意：运行服务器的目录将会作为 Got 客户端操作的起始目录。

-----

## 使用指南

got client 查看帮助信息：

```bash
# 也可以使用 go run cmd/client/main.go -h 来操作。
$ got -h
NAME:
   got - got is a simply tool for upload file to remote server

USAGE:
   got [global options] command [command options] [arguments...]

COMMANDS:
   list, l, ls, ll  list remote directory content
   change, c, cd    change remote directory content
   upload, u, up    upload file to remote directory
   download, d, down  download file from remote directory
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --addr value, -a value  Got server address
   --time, -t        show time cost (default: false)
   --help, -h        show help (default: false)
```

----

## 使用示例

示例环境：

* server 端：树莓派 zero
* client 端：windows 10

列出服务器（192.168.137.86，Got 服务监听 9876 端口）工作目录的文件信息：

```bash
$ got -a 192.168.137.86 ls
/home/pi/got_example:
drwxr-xr-x  test        4096
count: 1
```

切换目录：

```bash
$ got -a 192.168.137.86 cd test
/home/pi/got_example/test:
-rw-r--r--  file_download.txt   10
drwxr-xr-x  folder_download   4096
count: 2
```

上传文件或文件夹：

```bash
$ got -a 192.168.137.86 u file_test.txt
upload    finish    : [████████████████]
```

```bash
$ got -a 192.168.137.86 u folder_test
upload    finish    : [████████████████]
```

下载文件或文件夹：

```bash
$ got -a 192.168.137.86 d file_test2.txt
download  finish    : [████████████████]
```

```bash
$ got -a 192.168.137.86 d folder_test2
download  finish    : [████████████████]
```

切回上级目录

```bash
$ got -a 192.168.137.86 cd ..
/home/pi/got_example:
drwxr-xr-x  test        4096
count: 1
```
-----

## 提示与故障排查

提示：

* Got 在下载或上传文件过程中，如果遇到了同名文件会直接覆盖。
* Got 在下载或上传文件夹时，如果遇到了同名文件夹不会重建文件夹目录下的所有文件。例如 server 端 test 目录下存在 test_2 目录，但是 client 端的 test 目录下无 test_2 目录，将 client 的 test 上传到 server 并不会删除 test_2。
* Got 在传输文件夹时，先将文件夹目录下所有文件及文件夹遍历并打包为 .tar 临时文件，再将 .tar 文件进行传输，传输完成后再解包。因此，如果出现故障，可能在 client 或 server 的工作目录下会出现 .tar 文件。
* Got 文件传输的块大小为 4K。