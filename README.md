# sql-sniffer
MySQL、MongoDB和Redis流量嗅探工具

## 安装

### 方式1：直接安装（需要 libpcap）

>Ubuntu/Debian

sudo apt-get install libpcap-dev

>CentOS/RHEL

sudo yum install libpcap-devel

go install github.com/JacksonChan-X/sql-sniffer

### 方式2：使用docker

注意：为避免libpcap依赖，可以直接使用docker构建，根目录下已提供dockerfile

进入根目录后，执行

```
docker build -t sql-sniffer-builder .
docker run --rm -v $(pwd):/output sql-sniffer-builder cp /app/sql-sniffer /output/
```

## 使用
```
Usage:
sql-sniffer -i [interface] -mysql_port [port] -mongo_port [port] -redis_port [port]

Examples:
sql-sniffer -i eth0 -mysql_port 3306,3307 -mongo_port 27017

Flags:
  -d, --debug               启用调试模式
  -h, --help                help for sql-sniffer
  -i, --interfaces string   要监听的网卡，逗号分隔 (默认监听所有网卡)
      --mongo_port string   MongoDB端口，逗号分隔 (默认监听27017)
      --mysql_port string   MySQL端口，逗号分隔 (默认监听3306)
      --redis_port string   Redis端口，逗号分隔 (默认监听6379)
```