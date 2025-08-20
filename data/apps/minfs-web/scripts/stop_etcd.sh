#!/bin/bash
# 用法: ./stop_etcd.sh [client_port] [peer_port]

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
CLIENT_PORT=${1:-"2379"}
PEER_PORT=${2:-"2380"}

# 路径定义 - 使用相对路径引用minfs目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" && pwd)"
METASERVER_DIR="$WORKPUBLISH_DIR/metaServer"
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

# 进程锁定机制
LOCK_DIR="/tmp/minfs"
LOCK_FILE="$LOCK_DIR/etcd_stop.lock"

acquire_lock() {
    mkdir -p "$LOCK_DIR"
    if ! (set -C; echo $$ > "$LOCK_FILE") 2>/dev/null; then
        local lock_pid=$(cat "$LOCK_FILE" 2>/dev/null)
        if [ -n "$lock_pid" ] && ps -p $lock_pid > /dev/null 2>&1; then
            log_error "另一个停止操作正在进行中 (PID: $lock_pid)"
            exit 1
        else
            log_warn "发现过期的锁文件，正在清理..."
            rm -f "$LOCK_FILE"
            if ! (set -C; echo $$ > "$LOCK_FILE") 2>/dev/null; then
                log_error "无法获取锁"
                exit 1
            fi
        fi
    fi
}

release_lock() {
    rm -f "$LOCK_FILE"
}

# 确保退出时释放锁
trap release_lock EXIT

PID_FILE="$ETCD_PID_DIR/etcd.pid"

log_step "准备停止 etcd 服务..."

# 获取锁
acquire_lock

# 查找进程函数
find_pid() {
    local pid=""
    
    # 1. 尝试从PID文件读取
    if [ -f "$PID_FILE" ]; then
        pid=$(cat "$PID_FILE" 2>/dev/null | tr -d '\n\r ')
        if [ -n "$pid" ] && [[ "$pid" =~ ^[0-9]+$ ]] && ps -p $pid > /dev/null 2>&1; then
            # 验证进程是否确实是etcd
            local cmd_line=$(ps -p $pid -o args= 2>/dev/null)
            if [[ "$cmd_line" == *"etcd"* ]]; then
                echo "$pid"
                return
            else
                log_warn "PID文件中的进程不是etcd，清理PID文件"
                rm -f "$PID_FILE"
            fi
        else
            log_warn "PID文件中的进程已不存在或无效，清理PID文件"
            rm -f "$PID_FILE"
        fi
    fi
    
    # 2. 尝试通过客户端端口查找
    if command -v lsof > /dev/null 2>&1; then
        pid=$(lsof -ti:"$CLIENT_PORT" 2>/dev/null | head -n 1)
        if [ -n "$pid" ] && [[ "$pid" =~ ^[0-9]+$ ]] && ps -p $pid > /dev/null 2>&1; then
            local cmd_line=$(ps -p $pid -o args= 2>/dev/null)
            if [[ "$cmd_line" == *"etcd"* ]]; then
                echo "$pid"
                return
            fi
        fi
    fi
    
    # 3. 尝试通过对等端口查找
    if command -v lsof > /dev/null 2>&1; then
        pid=$(lsof -ti:"$PEER_PORT" 2>/dev/null | head -n 1)
        if [ -n "$pid" ] && [[ "$pid" =~ ^[0-9]+$ ]] && ps -p $pid > /dev/null 2>&1; then
            local cmd_line=$(ps -p $pid -o args= 2>/dev/null)
            if [[ "$pid" =~ ^[0-9]+$ ]]; then
                echo "$pid"
                return
            fi
        fi
    fi
    
    # 4. 尝试通过进程命令行查找
    local pids=$(pgrep -f "etcd.*$CLIENT_PORT" 2>/dev/null)
    for pid in $pids; do
        if [ -n "$pid" ] && [[ "$pid" =~ ^[0-9]+$ ]] && ps -p $pid > /dev/null 2>&1; then
            local cmd_line=$(ps -p $pid -o args= 2>/dev/null)
            if [[ "$cmd_line" == *"etcd"* ]]; then
                echo "$pid"
                return
            fi
        fi
    done
    
    echo ""
}

# 停止服务函数
stop_service() {
    local pid=$1
    
    if [ -z "$pid" ]; then
        log_warn "未找到运行中的etcd进程"
        return 0
    fi
    
    log_info "正在停止etcd进程 (PID: $pid)..."
    
    # 直接使用SIGKILL强制停止
    if kill -9 "$pid" 2>/dev/null; then
        log_info "已发送SIGKILL信号到进程 $pid"
        sleep 1
        
        # 检查进程是否已停止
        if ! ps -p "$pid" > /dev/null 2>&1; then
            log_info "etcd进程已被强制终止"
            return 0
        else
            log_error "无法终止进程 $pid"
            return 1
        fi
    else
        log_error "无法发送SIGKILL信号到进程 $pid"
        return 1
    fi
}

# 主逻辑
log_step "查找etcd进程..."

FOUND_PID=$(find_pid)

if [ -n "$FOUND_PID" ]; then
    log_info "找到etcd进程: $FOUND_PID"
    
    if stop_service "$FOUND_PID"; then
        log_info "✅ etcd 停止成功!"
        
        
        exit 0
    else
        log_error "❌ etcd 停止失败!"
        exit 1
    fi
else
    log_warn "未找到运行中的etcd进程"
    log_info "✅ etcd 已经停止"
    exit 0
fi
