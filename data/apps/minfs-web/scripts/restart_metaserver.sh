#!/bin/bash

# MinFS MetaServer 重启脚本 - 基于IP和端口
# 用法: ./restart_metaServer.sh <host> <port>

set -e

# 颜色输出定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 参数处理
HOST=${1:-"localhost"}
PORT=${2:-"9090"}

# 路径定义 - 使用相对路径引用minfs目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" && pwd)"

echo -e "${BLUE}[INFO]${NC} 重启 MetaServer ${HOST}:${PORT}..."

# 停止服务
echo -e "${YELLOW}[STEP]${NC} 停止 MetaServer..."
if ./stop_metaServer.sh "$HOST" "$PORT"; then
    echo -e "${GREEN}[INFO]${NC} MetaServer 停止成功"
else
    echo -e "${YELLOW}[WARN]${NC} MetaServer 可能已经停止"
fi

# 等待一下确保完全停止
sleep 2

# 启动服务
echo -e "${YELLOW}[STEP]${NC} 启动 MetaServer..."
if ./start_metaServer.sh "$HOST" "$PORT"; then
    echo -e "${GREEN}[INFO]${NC} MetaServer 启动成功"
    echo -e "${GREEN}✅ MetaServer ${HOST}:${PORT} 重启完成!${NC}"
    exit 0
else
    echo -e "${RED}[ERROR]${NC} MetaServer 启动失败"
    echo -e "${RED}❌ MetaServer ${HOST}:${PORT} 重启失败!${NC}"
    exit 1
fi