# Platform环境配置指南 (Linux)

## 1. 环境准备

### 1.1 系统依赖
确保已安装以下系统软件包：
- golang
- npm
- tar
- make
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

### 3.1 AI相关配置
修改`kdump-agent/new_agent_core/.env`文件：
```bash
DEEPSEEK_API_KEY = "your-deepseek-api-key"
TAVILY_API_KEY = "your-tavily-api-key"
```

### 3.2 路径配置
修改`kdump-agent/new_agent_core/KdumpAgent.py`中的以下路径变量：
```python
# 这些路径在完成Env脚本安装后可以确定
KERNEL_PATH = "/path/to/kernel/source"  # 内核源代码路径（通常位于Env/kernel_sources/）
VMLINUX_PATH = "/path/to/vmlinux"      # 内核vmlinux文件路径（编译后生成）
VMCORE_PATH = "/path/to/vmcore"        # vmcore文件路径（通常位于Env/vmcore_gen/）
KDUMP_GDBSERVER_PATH = "/path/to/kdump-gdbserver"  # 由build.sh安装到Env/extracted/
```

## 4. 使用说明

1. 激活conda环境：
```bash
conda activate kdump
```

2. 运行分析工具：
```bash
python kdump-agent/new_agent_core/KdumpAgent.py
```

## 5. 注意事项

1. 确保内核源代码版本与vmcore文件匹配
2. 编译环境需要至少4GB内存
3. 首次运行可能需要较长时间初始化
4. 如需使用虚拟环境，可在运行build.sh时指定：
```bash
./build.sh --install --venv /path/to/venv
```

## 6. 常见问题

### 6.1 缺少libkdumpfile
```bash
sudo ldconfig
```

### 6.2 GDB Python支持问题
重新安装带有Python支持的GDB：
```bash
sudo apt-get install gdb python3-dbg
```

### 6.3 权限问题
确保对vmcore文件有读取权限
