#!/bin/bash

# MinFS 脚本权限设置
# 用法: ./setup_permissions.sh

SCRIPT_DIR=$(dirname "$0")

echo "设置MinFS管理脚本权限..."

# 设置所有脚本为可执行
chmod +x "$SCRIPT_DIR"/*.sh

echo "权限设置完成:"
ls -la "$SCRIPT_DIR"/*.sh

echo ""
echo "脚本功能说明:"
echo ""
echo "📋 基础管理脚本:"
echo "  启动DataServer: ./start_dataServer.sh <host> <port>"
echo "  停止DataServer: ./stop_dataServer.sh <host> <port>"
echo "  重启DataServer: ./restart_dataServer.sh <host> <port>"
echo "  状态DataServer: ./status_dataServer.sh <host> <port>"
echo ""
echo "  启动MetaServer: ./start_metaServer.sh <host> <port>"
echo "  停止MetaServer: ./stop_metaServer.sh <host> <port>"
echo "  重启MetaServer: ./restart_metaServer.sh <host> <port>"
echo "  状态MetaServer: ./status_metaServer.sh <host> <port>"
echo ""
echo "🔧 高级管理脚本:"
echo "  强制停止服务: ./force_stop_service.sh <host> <port> [service_type]"
echo "  权限设置:      ./setup_permissions.sh"
echo ""
echo "启动逻辑说明:"
echo "  - 脚本会自动查找 /root/minfs/workpublish/ 目录下的二进制文件"
echo "  - 使用与bin目录相同的启动参数格式"
echo "  - 不生成PID文件，只查找系统生成的PID文件"
echo "  - 支持动态端口配置和自动服务类型检测"
echo ""
echo "端口映射:"
echo "  DataServer: 8001->dataServer1, 8002->dataServer2, 8003->dataServer3, 8004->dataServer4"
echo "  MetaServer: 9090->metaServer1, 9091->metaServer2, 9092->metaServer3"
echo ""
echo "使用示例:"
echo "  # 启动DataServer 1"
echo "  ./start_dataServer.sh localhost 8001"
echo ""
echo "  # 启动MetaServer 1"
echo "  ./start_metaServer.sh localhost 9090"
echo ""
echo "  # 强制停止服务（自动检测类型）"
echo "  ./force_stop_service.sh localhost 8001"
echo ""
echo "  # 强制停止指定类型的服务"
echo "  ./force_stop_service.sh localhost 9090 metaServer"
echo ""
echo "注意: 这些脚本现在基于真实的IP和端口来管理进程"
echo "后端会自动从集群信息中获取正确的IP和端口"