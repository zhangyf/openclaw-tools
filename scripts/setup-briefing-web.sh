#!/bin/bash
# 简报Web系统一键部署脚本

set -e

echo "📊 战争简报Web系统部署脚本"
echo "=============================="

WORKSPACE_DIR="/home/zhangyufeng/.openclaw/workspace"
SCRIPTS_DIR="$WORKSPACE_DIR/scripts"
BRIEFINGS_DIR="$WORKSPACE_DIR/briefings"

# 编译Go脚本
echo "🔧 编译Go脚本..."
cd "$SCRIPTS_DIR"

echo "1. 编译战争简报脚本..."
go build -o war-briefing-detailed war-briefing-detailed.go

echo "2. 编译Web服务器..."
go build -o briefing-web-server briefing-web-server.go

echo "3. 编译带链接版本..."
go build -o war-briefing-with-link war-briefing-with-link.go

echo "✅ 编译完成"

# 创建系统服务文件
echo "📋 创建系统服务配置..."
cat > /tmp/briefing-web.service << EOF
[Unit]
Description=战争简报Web服务器
After=network.target

[Service]
Type=simple
User=zhangyufeng
WorkingDirectory=$SCRIPTS_DIR
ExecStart=$SCRIPTS_DIR/briefing-web-server serve
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

echo "🔗 Web服务器信息:"
echo "   地址: http://localhost:8080"
echo "   目录: $BRIEFINGS_DIR"
echo "   最新简报: \$(ls -t $BRIEFINGS_DIR/war-briefing-detailed-*.md | head -1)"

# 测试生成带链接的简报
echo "🧪 测试生成带链接的简报..."
cd "$SCRIPTS_DIR"
./war-briefing-detailed 2>&1 | tail -20

echo ""
echo "🎯 使用说明:"
echo "1. 启动Web服务器: ./briefing-web-server serve"
echo "2. 生成简报: ./war-briefing-detailed"
echo "3. 带Web链接生成: ./war-briefing-with-link"
echo ""
echo "📅 Cron Job配置:"
echo "   当前使用: war-briefing-detailed"
echo "   可改为: war-briefing-with-link (需要Web服务器运行)"
echo ""
echo "✅ 部署完成"