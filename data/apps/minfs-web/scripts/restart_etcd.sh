#!/bin/bash

# MinFS etcd 重启脚本
# 用法: ./restart_etcd.sh [host] [client_port] [peer_port]

set -e

# 颜色输出定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 参数处理
HOST=${1:-"127.0.0.1"}
CLIENT_PORT=${2:-"2379"}
PEER_PORT=${3:-"2380"}

# 路径定义 - 使用相对路径引用minfs目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" && pwd)"

echo -e "${BLUE}[INFO]${NC} 重启 etcd 服务..."

# 停止服务
echo -e "${YELLOW}[STEP]${NC} 停止 etcd..."
if ./stop_etcd.sh "$CLIENT_PORT" "$PEER_PORT"; then
    echo -e "${GREEN}[INFO]${NC} etcd 停止成功"
else
    echo -e "${YELLOW}[WARN]${NC} etcd 可能已经停止"
fi

# 等待一下确保完全停止
sleep 2

# 启动服务
echo -e "${YELLOW}[STEP]${NC} 启动 etcd..."
if ./start_etcd.sh "$HOST" "$CLIENT_PORT" "$PEER_PORT"; then
    echo -e "${GREEN}[INFO]${NC} etcd 启动成功"
    echo -e "${GREEN}✅ etcd 重启完成!${NC}"
    exit 0
else
    echo -e "${RED}[ERROR]${NC} etcd 启动失败"
    echo -e "${RED}❌ etcd 重启失败!${NC}"
    exit 1
fi
