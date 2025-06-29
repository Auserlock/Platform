# Platform环境配置指南 (Linux)

## 1. 环境准备

### 1.1 系统依赖
确保已安装以下系统软件包：
- golang
- npm
- tar
- make
- docker
- gdb
- bear

在Ubuntu/Debian上安装：
```bash
sudo apt-get update
sudo apt-get install gdb tar make gcc clang
```
> 建议使用nvm管理并安装npm
> 由于编译较早版本的内核的复杂性，需要安装较早版本的gcc或clang，并修改源代码进行适配

## 2. 依赖安装

整个项目的目录如下：
```
├── backend
├── build-crash-image # 构建kdump的bzImage与initcpio
├── build-vmcore # 构建vmcore的工作目录
├── frontend # web UI
├── README.md
├── server
└── worker # worker
```

其中build-vmcore目录如下：

```
├── build # 编译内核工作文件夹
├── config.json # 可执行程序的配置文件
├── image # 存放启动镜像
├── kernel-benchmark
├── kernel-builder 可执行程序
├── log # qemu后台运行日志，（必需）
├── script # shell-scripts 用于辅助启动qemu等
├── toolchain # 存放gcc等工具链，需要在go的backend中手动配置
└── work # qemu启动的工作目录
```

## 3. 环境变量配置

### 3.1 路径配置
修改`backend/pkg/compile/kernel.go`中的以下路径变量：
```golang
var toolChains = map[string]string{
	"gcc-10.2.0": "toolchain/gcc/gcc-10.2.0",
	"gcc-10.1.0": "toolchain/gcc/gcc-10.1.0",
	"gcc-15.1.0": "toolchain/gcc/gcc-15.1.0",
	"gcc-9.1.0":  "toolchain/gcc/gcc-9.1.0",
}
```

### 3.2 config配置说明

- `worker/config/worker.json` 用于配置worker相关信息，可以修改ip address（如果是本地部署）
- `build-vmcore/config.json` 用于配置编译内核的信息，目前仅可以配置代理端口与虚拟机所需的memory大小

## 4. 使用说明

1. 编译服务器，worker与backend：
```bash
cd backend
go build -o kernel-builder cmd/main.go
mv kernel-builder ../build-vmcore
cd ../server
docker-compose up -d
go build -o server cmd/server.go
cd ../worker
go build cmd/worker.go
cd ../frontend
pnpm run dev # 用于测试
```

由于已经部署了网页前端，故运行worker即可

2. 运行工具：
```bash
./worker
```

3. 访问网页前端
> http://130.33.112.212 

## 5. 注意事项

1. 注意修改build-vmcore中的config,用于配置请求syzbot.com的资源
2. 编译内核时间较长，需等待
3. 需要手动编译gcc或clang放置在vmcore的toolchain下并修改源代码