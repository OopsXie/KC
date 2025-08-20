#!/bin/bash

# MinFS MetaServer 状态检查脚本 - 基于IP和端口
# 用法: ./status_metaServer.sh <host> <port>

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
PID_DIR=${PID_DIR:-"$METASERVER_DIR/pid"}

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

# PID文件路径 - 根据端口号生成服务器ID - 修复映射逻辑
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
PID_FILE="$PID_DIR/metaServer${server_id}.pid"

log_step "检查 MetaServer ${HOST}:${PORT} 状态..."

# 查找进程函数
find_process() {
    local pid=""
    
    # 1. 尝试从PID文件读取
    if [ -f "$PID_FILE" ]; then
        pid=$(cat "$PID_FILE")
        if [ -n "$pid" ] && ps -p $pid > /dev/null 2>&1; then
            echo "$pid"
            return
        else
            log_warn "PID文件中的进程已不存在，清理PID文件"
            rm -f "$PID_FILE"
        fi
    fi
    
    # 2. 尝试通过端口查找
    pid=$(lsof -ti:"$PORT" | head -n 1)
    if [ -n "$pid" ] && ps -p $pid > /dev/null 2>&1; then
        echo "$pid"
        return
    fi
    
    # 3. 尝试通过进程命令行查找
    pid=$(pgrep -f "java.*(M|m)eta(S|s)erver.*$PORT")
    if [ -n "$pid" ] && ps -p $pid > /dev/null 2>&1; then
        echo "$pid"
        return
    fi
    
    echo ""
}

# 显示进程状态
show_process_status() {
    local pid=$1
    
    if [ -z "$pid" ]; then
        echo -e "${RED}╔══════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║                    MetaServer 状态               ║${NC}"
        echo -e "${RED}╠══════════════════════════════════════════════════╣${NC}"
        echo -e "${RED}║  状态:   ${RED}停止${NC}                                    ║${NC}"
        echo -e "${RED}║  主机:   ${YELLOW}$HOST${NC}                                ║${NC}"
        echo -e "${RED}║  端口:   ${YELLOW}$PORT${NC}                                ║${NC}"
        echo -e "${RED}╚══════════════════════════════════════════════════╝${NC}"
        return 1
    fi
    
    # 获取进程详细信息
    local process_info=$(ps -p $pid -o pid,ppid,user,pcpu,pmem,vsz,rss,stat,time,command --no-headers 2>/dev/null)
    local user=$(echo "$process_info" | awk '{print $3}')
    local cpu=$(echo "$process_info" | awk '{print $4}')
    local mem=$(echo "$process_info" | awk '{print $5}')
    local vsz=$(echo "$process_info" | awk '{print $6}')
    local rss=$(echo "$process_info" | awk '{print $7}')
    local stat=$(echo "$process_info" | awk '{print $8}')
    local time=$(echo "$process_info" | awk '{print $9}')
    local cmd=$(echo "$process_info" | awk '{for(i=10;i<=NF;i++) printf $i" "; print ""}')
    
    # 获取端口信息
    local port_info=""
    if lsof -i:$PORT > /dev/null 2>&1; then
        port_info="监听中"
    else
        port_info="未监听"
    fi
    
    # 获取内存使用（MB）
    local mem_mb=$((vsz / 1024))
    local rss_mb=$((rss / 1024))
    
    # 显示状态信息
    echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    MetaServer 状态               ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║  状态:   ${GREEN}运行中${NC}                                  ║${NC}"
    echo -e "${GREEN}║  主机:   ${CYAN}$HOST${NC}                                ║${NC}"
    echo -e "${GREEN}║  端口:   ${CYAN}$PORT${NC}                                ║${NC}"
    echo -e "${GREEN}║  PID:    ${CYAN}$pid${NC}                                ║${NC}"
    echo -e "${GREEN}║  用户:   ${CYAN}$user${NC}                                ║${NC}"
    echo -e "${GREEN}║  端口状态: ${CYAN}$port_info${NC}                          ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║  系统资源使用情况:                               ║${NC}"
    echo -e "${GREEN}║  CPU:    ${CYAN}${cpu}%${NC}                               ║${NC}"
    echo -e "${GREEN}║  内存:   ${CYAN}${mem}%${NC}                               ║${NC}"
    echo -e "${GREEN}║  虚拟内存: ${CYAN}${mem_mb}MB${NC}                         ║${NC}"
    echo -e "${GREEN}║  物理内存: ${CYAN}${rss_mb}MB${NC}                         ║${NC}"
    echo -e "${GREEN}║  进程状态: ${CYAN}$stat${NC}                               ║${NC}"
    echo -e "${GREEN}║  运行时间: ${CYAN}$time${NC}                               ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║  启动命令:                                       ║${NC}"
    echo -e "${GREEN}║  ${CYAN}$cmd${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════╣${NC}"
    
    # 显示数据目录信息
    local data_dir="$METASERVER_DIR/metadb"
    if [ -d "$data_dir" ]; then
        local data_size=$(du -sh "$data_dir" 2>/dev/null | cut -f1)
        echo -e "${GREEN}║  数据大小:   ${GREEN}$data_size${NC}                    ║${NC}"
    fi
    
    echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
    
    # 更新PID文件
    if [ ! -f "$PID_FILE" ] || [ "$(cat "$PID_FILE")" != "$pid" ]; then
        echo $pid > "$PID_FILE"
        log_info "已更新PID文件: $PID_FILE"
    fi
    
    return 0
}

# 主逻辑
pid=$(find_process)

if show_process_status "$pid"; then
    log_info "✅ MetaServer ${HOST}:${PORT} 运行正常"
    exit 0
else
    log_warn "❌ MetaServer ${HOST}:${PORT} 未运行"
    exit 1
fi