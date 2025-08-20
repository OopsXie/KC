#!/bin/bash

# MinFS MetaServer 启动脚本 - 基于IP和端口
# 用法: ./start_metaServer.sh <host> <port>

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
PORT=${2:-"9090"}

# 路径定义 - 使用相对路径引用minfs目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" && pwd)"
METASERVER_DIR="$WORKPUBLISH_DIR/metaServer"
METASERVER_HOME=${METASERVER_HOME:-"$METASERVER_DIR"}
LOG_DIR=${LOG_DIR:-"$METASERVER_DIR/logs"}
PID_DIR=${PID_DIR:-"$METASERVER_DIR/pid"}
DATA_DIR=${DATA_DIR:-"$METASERVER_DIR/metadb"}

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

log_step "准备启动 MetaServer ${HOST}:${PORT}..."

# 检查进程是否已经运行
check_existing_process() {
    # 检查端口是否被占用
    if lsof -ti:$PORT > /dev/null 2>&1; then
        local existing_pid=$(lsof -ti:$PORT | head -1)
        log_warn "端口 $PORT 已被进程 $existing_pid 占用"
        
        # 检查是否是MetaServer进程
        local process_info=$(ps -p $existing_pid -o comm=,args= 2>/dev/null)
        if [[ "$process_info" == *"java"* ]] && [[ "$process_info" == *"metaServer"* ]]; then
            log_info "MetaServer ${HOST}:${PORT} 已经在运行中 (PID: $existing_pid)"
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
    log_step "检查MetaServer可执行文件..."
    
    # 检查二进制文件 (使用minfs目录下的)
    METASERVER_BIN="$METASERVER_DIR/metaServer"
    if [ -f "$METASERVER_BIN" ] && [ -x "$METASERVER_BIN" ]; then
        log_info "找到MetaServer二进制文件: $METASERVER_BIN"
        METASERVER_EXEC="$METASERVER_BIN"
        return 0
    fi
    
    # 检查JAR文件
    METASERVER_JAR="$METASERVER_HOME/metaServer.jar"
    if [ -f "$METASERVER_JAR" ]; then
        log_info "找到MetaServer JAR文件: $METASERVER_JAR"
        METASERVER_EXEC="java ${JAVA_OPTS:-'-Xmx1g -Xms512m'} -jar $METASERVER_JAR"
        return 0
    fi
    
    log_error "未找到MetaServer可执行文件"
    log_error "请确保以下文件之一存在:"
    log_error "  二进制文件: $METASERVER_BIN"
    log_error "  JAR文件: $METASERVER_JAR"
    log_error "  或者先运行 /root/minfs/workpublish/bin/build.sh 构建项目"
    exit 1
}

check_binaries

# 构建启动命令 - 使用与bin目录相同的参数格式
if [[ "$METASERVER_EXEC" == *"java"* ]]; then
    START_CMD="$METASERVER_EXEC --host=$HOST --port=$PORT --data-dir=$DATA_DIR"
else
    # 使用与bin目录相同的启动参数格式，支持端口指定
    server_id=""
    if [[ "$PORT" =~ ^909([0-9]+)$ ]]; then
        # 端口9090→ID1, 9091→ID2, 9092→ID3
        # 正则捕获的是端口最后一位(0,1,2)，需要加1得到ID(1,2,3)
        last_digit="${BASH_REMATCH[1]}"
        server_id="$((last_digit + 1))"
    else
        port_num=$((PORT - 9089))
        if [ $port_num -gt 0 ]; then
            server_id="$port_num"
        else
            server_id="1"
        fi
    fi
    
    node_id="metaServer-${PORT}"
    data_dir="./metadb${server_id}"
    
    # 添加端口参数，覆盖配置文件中的端口设置
    START_CMD="$METASERVER_EXEC -config=config.yaml -node-id=$node_id -data-dir=$data_dir -port=$PORT"
fi

log_info "启动命令: $START_CMD"
log_info "日志文件: $LOG_DIR"
log_info "数据目录: $DATA_DIR"

# 启动MetaServer
log_step "启动MetaServer进程..."
cd "$METASERVER_DIR"  # 切换到MetaServer目录，确保相对路径正确
nohup $START_CMD > "$LOG_DIR/metaServer_${HOST}_${PORT}.log" 2>&1 &
NEW_PID=$!

# 保存PID到文件 (但不生成PID文件，只用于临时记录)
log_info "MetaServer进程已启动，PID: $NEW_PID"

# 等待服务启动
wait_for_service() {
    local max_attempts=30
    local attempt=1
    
    log_step "等待MetaServer启动..."
    
    while [ $attempt -le $max_attempts ]; do
        # 检查进程是否还在运行
        if ! ps -p $NEW_PID > /dev/null 2>&1; then
            log_error "MetaServer启动失败，进程已退出"
            log_error "请检查日志文件: $LOG_DIR/metaServer_${HOST}_${PORT}.log"
            return 1
        fi
        
        # 检查端口是否开始监听
        if lsof -ti:$PORT > /dev/null 2>&1; then
            local pid=$(lsof -ti:$PORT | head -1)
            log_info "MetaServer ${HOST}:${PORT} 启动成功! (PID: $pid)"
            log_info "日志文件: $LOG_DIR/metaServer_${HOST}_${PORT}.log"
            return 0
        fi
        
        printf "."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo
    log_error "启动超时，MetaServer可能启动失败"
    log_error "请检查日志文件: $LOG_DIR/metaServer_${HOST}_${PORT}.log"
    return 1
}

if wait_for_service; then
    log_info "✅ MetaServer ${HOST}:${PORT} 启动完成!"
    exit 0
else
    log_error "❌ MetaServer ${HOST}:${PORT} 启动失败!"
    exit 1
fi