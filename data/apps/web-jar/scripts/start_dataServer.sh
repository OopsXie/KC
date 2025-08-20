#!/bin/bash

# MinFS DataServer 启动脚本 - 基于IP和端口
# 用法: ./start_dataServer.sh <host> <port>

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

# 路径定义 - 使用相对路径引用minfs目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" && pwd)"
DATASERVER_DIR="$WORKPUBLISH_DIR/dataServer"
DATASERVER_HOME=${DATASERVER_HOME:-"$DATASERVER_DIR"}
LOG_DIR=${LOG_DIR:-"$DATASERVER_DIR/logs"}
PID_DIR=${PID_DIR:-"$DATASERVER_DIR/pid"}
DATA_DIR=${DATA_DIR:-"$DATASERVER_DIR/data"}

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
mkdir -p "$LOG_DIR" "$PID_DIR" "$DATA_DIR"

# 文件路径 - 根据端口号生成服务器ID
server_id=""
if [[ "$PORT" =~ ^800([0-9]+)$ ]]; then
    server_id="${BASH_REMATCH[1]}"
else
    # 如果端口不符合8001,8002格式，尝试直接使用端口减去8000
    port_num=$((PORT - 8000))
    if [ $port_num -gt 0 ]; then
        server_id="$port_num"
    else
        server_id="1"  # 默认使用1
    fi
fi
PID_FILE="$PID_DIR/dataServer${server_id}.pid"
LOG_FILE="$LOG_DIR/dataServer${server_id}.log"

log_step "准备启动 DataServer ${HOST}:${PORT}..."

# 检查进程是否已经运行
check_existing_process() {
    # 检查PID文件
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            log_info "DataServer ${HOST}:${PORT} 已经在运行中 (PID: $pid)"
            return 0
        else
            log_warn "发现过期的PID文件，正在清理..."
            rm -f "$PID_FILE"
        fi
    fi
    
    # 检查端口是否被占用
    if lsof -ti:$PORT > /dev/null 2>&1; then
        local existing_pid=$(lsof -ti:$PORT | head -1)
        log_warn "端口 $PORT 已被进程 $existing_pid 占用"
        
        # 检查是否是DataServer进程
        local process_info=$(ps -p $existing_pid -o comm=,args= 2>/dev/null)
        if [[ "$process_info" == *"dataServer"* ]] || [[ "$process_info" == *"DataServer"* ]]; then
            log_info "DataServer ${HOST}:${PORT} 已经在运行中 (PID: $existing_pid)"
            echo $existing_pid > "$PID_FILE"
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

# 检查可执行文件 - 使用minfs目录下的二进制文件
check_binaries() {
    log_step "检查DataServer可执行文件..."
    
    # 检查二进制文件 (使用minfs目录下的)
    DATASERVER_BIN="$DATASERVER_DIR/dataServer"
    if [ -f "$DATASERVER_BIN" ] && [ -x "$DATASERVER_BIN" ]; then
        log_info "找到DataServer二进制文件: $DATASERVER_BIN"
        DATASERVER_EXEC="$DATASERVER_BIN"
        return 0
    fi
    
    # 检查JAR文件
    DATASERVER_JAR="$DATASERVER_HOME/dataServer.jar"
    if [ -f "$DATASERVER_JAR" ]; then
        log_info "找到DataServer JAR文件: $DATASERVER_JAR"
        DATASERVER_EXEC="java ${JAVA_OPTS:-'-Xmx1g -Xms512m'} -jar $DATASERVER_JAR"
        return 0
    fi
    
    log_error "未找到DataServer可执行文件"
    log_error "请确保以下文件之一存在:"
    log_error "  二进制文件: $DATASERVER_BIN"
    log_error "  JAR文件: $DATASERVER_JAR"
    log_error "  或者先运行 /root/minfs/workpublish/bin/build.sh 构建项目"
    exit 1
}

check_binaries

# 构建启动命令 - 使用与bin目录相同的参数格式
if [[ "$DATASERVER_EXEC" == *"java"* ]]; then
    START_CMD="$DATASERVER_EXEC --host=$HOST --port=$PORT --data-dir=$DATA_DIR"
else
    # 使用与bin目录相同的启动参数格式，支持端口指定
    START_CMD="$DATASERVER_EXEC -config=config.yaml -id=$server_id -port=$PORT"
fi

log_info "启动命令: $START_CMD"
log_info "日志文件: $LOG_FILE"
log_info "数据目录: $DATA_DIR"

# 启动DataServer
log_step "启动DataServer进程..."
cd "$DATASERVER_DIR"  # 切换到DataServer目录，确保相对路径正确
nohup $START_CMD > "$LOG_FILE" 2>&1 &
NEW_PID=$!

# 保存PID到文件 (但不生成PID文件，只用于临时记录)
log_info "DataServer进程已启动，PID: $NEW_PID"

# 等待服务启动
wait_for_service() {
    local max_attempts=30
    local attempt=1
    
    log_step "等待DataServer启动..."
    
    while [ $attempt -le $max_attempts ]; do
        # 检查进程是否还在运行
        if ! ps -p $NEW_PID > /dev/null 2>&1; then
            log_error "DataServer启动失败，进程已退出"
            log_error "请检查日志文件: $LOG_FILE"
            return 1
        fi
        
        # 检查端口是否开始监听 (使用多种方式)
        local port_listening=false
        
        # 方式1: 使用lsof
        if lsof -ti:$PORT > /dev/null 2>&1; then
            port_listening=true
        fi
        
        # 方式2: 使用netstat
        if netstat -tlnp 2>/dev/null | grep ":$PORT " > /dev/null 2>&1; then
            port_listening=true
        fi
        
        # 方式3: 使用ss
        if ss -tlnp 2>/dev/null | grep ":$PORT " > /dev/null 2>&1; then
            port_listening=true
        fi
        
        if [ "$port_listening" = true ]; then
            local pid=$(lsof -ti:$PORT 2>/dev/null | head -1)
            if [ -z "$pid" ]; then
                pid=$(netstat -tlnp 2>/dev/null | grep ":$PORT " | awk '{print $7}' | cut -d'/' -f1 | head -1)
            fi
            log_info "DataServer ${HOST}:${PORT} 启动成功! (PID: $pid)"
            log_info "日志文件: $LOG_FILE"
            return 0
        fi
        
        printf "."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo
    log_error "启动超时，DataServer可能启动失败"
    log_error "请检查日志文件: $LOG_FILE"
    log_error "进程状态: $(ps -p $NEW_PID -o pid,ppid,comm,args --no-headers 2>/dev/null)"
    return 1
}

if wait_for_service; then
    log_info "✅ DataServer ${HOST}:${PORT} 启动完成!"
    exit 0
else
    log_error "❌ DataServer ${HOST}:${PORT} 启动失败!"
    exit 1
fi