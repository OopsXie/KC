#!/bin/bash

# 路径验证脚本
# 验证所有脚本中的相对路径是否正确配置

echo "=== MinFS Scripts 路径验证 ==="
echo

# 模拟脚本所在的实际路径结构
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "当前脚本目录: $SCRIPT_DIR"

# 测试相对路径
WORKPUBLISH_DIR="$(cd "$SCRIPT_DIR/../../minfs/workpublish" 2>/dev/null && pwd)"
if [ -n "$WORKPUBLISH_DIR" ]; then
    echo "✅ WORKPUBLISH_DIR 路径解析成功: $WORKPUBLISH_DIR"
else
    echo "❌ WORKPUBLISH_DIR 路径解析失败"
    echo "   期望路径: $SCRIPT_DIR/../../minfs/workpublish"
fi

echo
echo "=== 预期的目录结构 ==="
echo "data/"
echo "├── apps/"
echo "│   ├── web-jar/"
echo "│   │   └── scripts/        # 脚本位置"
echo "│   └── minfs/"
echo "│       └── workpublish/    # MinFS 工作目录"
echo "│           ├── metaServer/"
echo "│           └── dataServer/"
echo

echo "=== 路径配置总结 ==="
echo "- 脚本目录: data/apps/web-jar/scripts"
echo "- 工作目录: data/apps/minfs/workpublish"
echo "- 相对路径: ../../minfs/workpublish (从脚本到工作目录)"
echo

echo "=== 建议 ==="
echo "1. 确保在实际部署时创建正确的目录结构"
echo "2. 将脚本部署到 data/apps/web-jar/scripts 目录"
echo "3. 确保 data/apps/minfs/workpublish 目录存在且可访问"
echo "4. 运行脚本前先创建必要的子目录 (logs, pid, data, metadb 等)"
