#!/bin/bash

# MinFS 强制停止服务脚本 - 基于IP和端口
# 用法: ./force_stop_service.sh <host> <port> [service_type]

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
HOST=${1:-"localhost"}
PORT=${2:-"8001"}
SERVICE_TYPE=${3:-"auto"}

# 路径定义 - 使用相对路径引用minfs目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" && pwd)"

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

# 自动检测服务类型
detect_service_type() {
    if [[ "$PORT" =~ ^800[0-9]+$ ]]; then
        echo "dataServer"
    elif [[ "$PORT" =~ ^909[0-9]+$ ]]; then
        echo "metaServer"
    else
        echo "unknown"
    fi
}

# 如果未指定服务类型，自动检测
if [ "$SERVICE_TYPE" = "auto" ]; then
    SERVICE_TYPE=$(detect_service_type)
fi

log_step "强制停止 $SERVICE_TYPE ${HOST}:${PORT}..."

# 查找进程函数
find_pid() {
    local pid=""
    
    # 1. 尝试通过端口查找
    if command -v lsof > /dev/null 2>&1; then
        pid=$(lsof -ti:"$PORT" 2>/dev/null | head -n 1)
        if [ -n "$pid" ] && [[ "$pid" =~ ^[0-9]+$ ]] && ps -p $pid > /dev/null 2>&1; then
            local cmd_line=$(ps -p $pid -o args= 2>/dev/null)
            if [[ "$cmd_line" == *"$SERVICE_TYPE"* ]]; then
                log_info "通过端口找到进程: $pid"
                echo "$pid"
                return
            fi
        fi
    fi
    
    # 2. 尝试使用netstat查找端口
    if command -v netstat > /dev/null 2>&1; then
        pid=$(netstat -tlnp 2>/dev/null | grep ":$PORT " | awk '{print $7}' | cut -d'/' -f1 | head -n 1)
        if [ -n "$pid" ] && [[ "$pid" =~ ^[0-9]+$ ]] && ps -p $pid > /dev/null 2>&1; then
            local cmd_line=$(ps -p $pid -o args= 2>/dev/null)
            if [[ "$cmd_line" == *"$SERVICE_TYPE"* ]]; then
                log_info "通过netstat找到进程: $pid"
                echo "$pid"
                return
            fi
        fi
    fi
    
    # 3. 尝试通过进程命令行查找
    local pids=$(pgrep -f ".*$SERVICE_TYPE.*$PORT" 2>/dev/null)
    for pid in $pids; do
        if [ -n "$pid" ] && ps -p $pid > /dev/null 2>&1; then
            local cmd_line=$(ps -p $pid -o args= 2>/dev/null)
            if [[ "$cmd_line" == *"$SERVICE_TYPE"* ]] && [[ "$cmd_line" == *"$PORT"* ]]; then
                log_info "通过命令行找到进程: $pid"
                echo "$pid"
                return
            fi
        fi
    done
    
    echo ""
}

# 强制停止进程
force_stop_process() {
    local pid=$1
    
    if [ -z "$pid" ]; then
        log_warn "未找到运行中的进程"
        return 0
    fi
    
    log_info "正在强制停止进程 (PID: $pid)..."
    
    # 显示进程信息
    local process_info=$(ps -p $pid -o pid,ppid,user,comm,args --no-headers 2>/dev/null)
    log_info "进程信息: $process_info"
    
    # 直接发送SIGKILL信号
    if kill -KILL "$pid" 2>/dev/null; then
        log_info "已发送SIGKILL信号到进程 $pid"
        sleep 1
        
        # 检查进程是否已停止
        if ! ps -p "$pid" > /dev/null 2>&1; then
            log_info "进程已被强制终止"
            return 0
        else
            log_warn "SIGKILL无法终止进程，尝试其他方法..."
            
            # 尝试使用pkill
            if command -v pkill > /dev/null 2>&1; then
                if pkill -9 -f ".*$PORT.*" 2>/dev/null; then
                    log_info "已通过pkill强制终止进程"
                    sleep 1
                    
                    if ! ps -p "$pid" > /dev/null 2>&1; then
                        log_info "进程已通过pkill终止"
                        return 0
                    fi
                fi
            fi
            
            # 最后尝试：直接删除进程文件（危险操作）
            log_error "无法通过信号终止进程，进程可能被锁定"
            log_error "请手动检查进程状态: ps -p $pid"
            return 1
        fi
    else
        log_error "无法发送SIGKILL信号到进程 $pid"
        return 1
    fi
}

# 清理PID文件
cleanup_pid_files() {
    local pid_dir=""
    
    if [ "$SERVICE_TYPE" = "dataServer" ]; then
        local server_id=""
        if [[ "$PORT" =~ ^800([0-9]+)$ ]]; then
            server_id="${BASH_REMATCH[1]}"
        else
            local port_num=$((PORT - 8000))
            if [ $port_num -gt 0 ]; then
                server_id="$port_num"
            else
                server_id="1"
            fi
        fi
        pid_dir="$WORKPUBLISH_DIR/dataServer/pid"
        local pid_file="$pid_dir/dataServer${server_id}.pid"
    elif [ "$SERVICE_TYPE" = "metaServer" ]; then
        local server_id=""
        if [[ "$PORT" =~ ^909([0-9]+)$ ]]; then
            server_id="$((${BASH_REMATCH[1]} + 1))"
        else
            local port_num=$((PORT - 9089))
            if [ $port_num -gt 0 ]; then
                server_id="$port_num"
            else
                server_id="1"
            fi
        fi
        pid_dir="$WORKPUBLISH_DIR/metaServer/pid"
        local pid_file="$pid_dir/metaServer${server_id}.pid"
    fi
    
    if [ -n "$pid_dir" ] && [ -d "$pid_dir" ]; then
        # 查找并清理所有相关的PID文件
        find "$pid_dir" -name "*$SERVICE_TYPE*.pid" -type f -exec rm -f {} \;
        log_info "已清理PID目录: $pid_dir"
    fi
}

# 主逻辑
log_step "查找进程..."

FOUND_PID=$(find_pid)

if [ -n "$FOUND_PID" ]; then
    log_info "找到进程: $FOUND_PID"
    
    if force_stop_process "$FOUND_PID"; then
        log_info "✅ $SERVICE_TYPE ${HOST}:${PORT} 强制停止成功!"
        
        # 清理PID文件
        cleanup_pid_files
        
        exit 0
    else
        log_error "❌ $SERVICE_TYPE ${HOST}:${PORT} 强制停止失败!"
        exit 1
    fi
else
    log_warn "未找到运行中的进程"
    log_info "✅ $SERVICE_TYPE ${HOST}:${PORT} 已经停止"
    
    # 清理可能存在的PID文件
    cleanup_pid_files
    
    exit 0
fi
