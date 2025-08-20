#!/bin/bash

# MinFS etcd 启动脚本
# 用法: ./start_etcd.sh [host] [client_port] [peer_port]

set -e

# 颜色输出定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 参数处理
HOST=${1:-"127.0.0.1"}
CLIENT_PORT=${2:-"2379"}
PEER_PORT=${3:-"2380"}

# 路径定义 - 使用相对路径引用minfs目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" && pwd)"
METASERVER_DIR="$WORKPUBLISH_DIR/metaServer"
ETCD_DATA_DIR="$METASERVER_DIR/etcd-data"
ETCD_LOG_DIR="$METASERVER_DIR/logs"
ETCD_PID_DIR="$METASERVER_DIR/pid"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# 确保目录存在
mkdir -p "$ETCD_LOG_DIR" "$ETCD_PID_DIR" "$ETCD_DATA_DIR"

log_step "准备启动 etcd 服务..."

# 检查etcd是否已安装
check_etcd_installation() {
    if ! command -v etcd &> /dev/null; then
        log_error "etcd 未安装，请先安装 etcd"
        log_error "安装方法:"
        log_error "  CentOS/RHEL: yum install etcd"
        log_error "  Ubuntu/Debian: apt-get install etcd"
        log_error "  或者从官网下载: https://github.com/etcd-io/etcd/releases"
        exit 1
    fi
    
    local etcd_version=$(etcd --version | head -1)
    log_info "找到 etcd: $etcd_version"
}

# 检查进程是否已经运行
check_existing_process() {
    # 检查PID文件
    local pid_file="$ETCD_PID_DIR/etcd.pid"
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            log_info "etcd 已经在运行中 (PID: $pid)"
            return 0
        else
            log_warn "发现过期的PID文件，正在清理..."
            rm -f "$pid_file"
        fi
    fi
    
    # 检查端口是否被占用
    if lsof -ti:$CLIENT_PORT > /dev/null 2>&1; then
        local existing_pid=$(lsof -ti:$CLIENT_PORT | head -1)
        log_warn "客户端端口 $CLIENT_PORT 已被进程 $existing_pid 占用"
        
        # 检查是否是etcd进程
        local process_info=$(ps -p $existing_pid -o comm=,args= 2>/dev/null)
        if [[ "$process_info" == *"etcd"* ]]; then
            log_info "etcd 已经在运行中 (PID: $existing_pid)"
            echo $existing_pid > "$pid_file"
            return 0
        else
            log_error "端口被其他进程占用: $process_info"
            return 1
        fi
    fi
    
    if lsof -ti:$PEER_PORT > /dev/null 2>&1; then
        local existing_pid=$(lsof -ti:$PEER_PORT | head -n 1)
        log_warn "对等端口 $PEER_PORT 已被进程 $existing_pid 占用"
        
        local process_info=$(ps -p $existing_pid -o comm=,args= 2>/dev/null)
        if [[ "$process_info" == *"etcd"* ]]; then
            log_info "etcd 已经在运行中 (PID: $existing_pid)"
            echo $existing_pid > "$ETCD_PID_DIR/etcd.pid"
            return 0
        else
            log_error "端口被其他进程占用: $process_info"
            return 1
        fi
    fi
    
    return 1
}

if check_existing_process; then
    exit 0
fi

# 检查etcd安装
check_etcd_installation

# 构建启动命令
START_CMD="etcd \
    --name=etcd-node \
    --data-dir=$ETCD_DATA_DIR \
    --listen-client-urls=http://0.0.0.0:$CLIENT_PORT \
    --advertise-client-urls=http://$HOST:$CLIENT_PORT \
    --listen-peer-urls=http://0.0.0.0:$PEER_PORT \
    --initial-advertise-peer-urls=http://$HOST:$PEER_PORT \
    --initial-cluster=etcd-node=http://$HOST:$PEER_PORT \
    --initial-cluster-state=new \
    --initial-cluster-token=minfs-cluster"

log_info "启动命令: $START_CMD"
log_info "数据目录: $ETCD_DATA_DIR"
log_info "日志文件: $ETCD_LOG_DIR/etcd.log"
log_info "客户端端口: $CLIENT_PORT"
log_info "对等端口: $PEER_PORT"

# 启动etcd
log_step "启动etcd进程..."
cd "$METASERVER_DIR"  # 切换到MetaServer目录，确保相对路径正确
nohup $START_CMD > "$ETCD_LOG_DIR/etcd.log" 2>&1 &
NEW_PID=$!

# 保存PID到文件
echo $NEW_PID > "$ETCD_PID_DIR/etcd.pid"
log_info "etcd进程已启动，PID: $NEW_PID"

# 等待服务启动
wait_for_service() {
    local max_attempts=30
    local attempt=1
    
    log_step "等待etcd启动..."
    
    while [ $attempt -le $max_attempts ]; do
        # 检查进程是否还在运行
        if ! ps -p $NEW_PID > /dev/null 2>&1; then
            log_error "etcd启动失败，进程已退出"
            log_error "请检查日志文件: $ETCD_LOG_DIR/etcd.log"
            return 1
        fi
        
        # 检查客户端端口是否开始监听
        if lsof -ti:$CLIENT_PORT > /dev/null 2>&1; then
            local pid=$(lsof -ti:$CLIENT_PORT | head -1)
            log_info "etcd 启动成功! (PID: $pid)"
            log_info "客户端端口: $CLIENT_PORT"
            log_info "对等端口: $PEER_PORT"
            log_info "日志文件: $ETCD_LOG_DIR/etcd.log"
            return 0
        fi
        
        printf "."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo
    log_error "启动超时，etcd可能启动失败"
    log_error "请检查日志文件: $ETCD_LOG_DIR/etcd.log"
    return 1
}

if wait_for_service; then
    log_info "✅ etcd 启动完成!"
    exit 0
else
    log_error "❌ etcd 启动失败!"
    exit 1
fi
