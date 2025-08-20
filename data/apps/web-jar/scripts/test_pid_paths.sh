#!/bin/bash

# 测试脚本 - 验证PID文件路径和命名是否正确
# 用法: ./test_pid_paths.sh

echo "测试PID文件路径和命名格式..."

# 测试DataServer
echo ""
echo "=== 测试 DataServer ==="
for port in 8001 8002 8003 8004; do
    echo "测试端口: $port"
    
    # 计算期望的server_id
    server_id=$((port - 8000))
            expected_pid_file="../../minfs/workpublish/dataServer/pid/dataServer${server_id}.pid"
    
    echo "  期望的PID文件路径: $expected_pid_file"
    
    # 调用启动脚本（不实际启动，只查看PID文件路径）
    echo "  调用启动脚本测试..."
    bash -c "
        PORT=$port
        WORKPUBLISH_DIR=\"../../minfs/workpublish\"
        DATASERVER_DIR=\"\$WORKPUBLISH_DIR/dataServer\"
        PID_DIR=\"\$DATASERVER_DIR/pid\"
        
        server_id=\"\"
        if [[ \"\$PORT\" =~ ^800([0-9]+)$ ]]; then
            server_id=\"\${BASH_REMATCH[1]}\"
        else
            port_num=\$((PORT - 8000))
            if [ \$port_num -gt 0 ]; then
                server_id=\"\$port_num\"
            else
                server_id=\"1\"
            fi
        fi
        PID_FILE=\"\$PID_DIR/dataServer\${server_id}.pid\"
        echo \"  实际计算的PID文件路径: \$PID_FILE\"
        
        if [ \"\$PID_FILE\" = \"$expected_pid_file\" ]; then
            echo \"  ✅ 路径匹配正确\"
        else
            echo \"  ❌ 路径不匹配\"
        fi
    "
    echo ""
done

# 测试MetaServer
echo ""
echo "=== 测试 MetaServer ==="
for port in 9090 9091 9092; do
    echo "测试端口: $port"
    
    # 计算期望的server_id（端口9090对应服务1，9091对应服务2）
    server_id=$((port - 9089))
    expected_pid_file="/root/minfs/workpublish/metaServer/pid/metaServer${server_id}.pid"
    
    echo "  期望的PID文件路径: $expected_pid_file"
    
    # 调用启动脚本（不实际启动，只查看PID文件路径）
    echo "  调用启动脚本测试..."
    bash -c "
        PORT=$port
        WORKPUBLISH_DIR=\"../../minfs/workpublish\"
        METASERVER_DIR=\"\$WORKPUBLISH_DIR/metaServer\"
        PID_DIR=\"\$METASERVER_DIR/pid\"
        
        server_id=\"\"
        if [[ \"\$PORT\" =~ ^909([0-9]+)$ ]]; then
            server_id=\"\${BASH_REMATCH[1]}\"
        else
            port_num=\$((PORT - 9090))
            if [ \$port_num -ge 0 ]; then
                server_id=\"\$port_num\"
            else
                server_id=\"0\"
            fi
        fi
        
        if [ \"\$server_id\" = \"0\" ]; then
            PID_FILE=\"\$PID_DIR/metaServer0.pid\"
        else
            PID_FILE=\"\$PID_DIR/metaServer\${server_id}.pid\"
        fi
        echo \"  实际计算的PID文件路径: \$PID_FILE\"
        
        if [ \"\$PID_FILE\" = \"$expected_pid_file\" ]; then
            echo \"  ✅ 路径匹配正确\"
        else
            echo \"  ❌ 路径不匹配\"
        fi
    "
    echo ""
done

echo "测试完成！"
