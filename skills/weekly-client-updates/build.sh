#!/bin/bash

# 每周客户更新管理工具编译脚本

set -e

echo "🔨 编译每周客户更新管理工具..."

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Go未安装，请先安装Go"
    exit 1
fi

# 检查Go版本
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "📦 Go版本: $GO_VERSION"

# 清理旧编译文件
echo "🧹 清理旧文件..."
rm -f weekly-client-update
rm -rf dist/

# 下载依赖
echo "📥 下载依赖..."
go mod download

# 编译
echo "🔧 编译中..."
go build -o weekly-client-update .

# 检查是否编译成功
if [ -f "weekly-client-update" ]; then
    echo "✅ 编译成功！"
    echo "📄 可执行文件: ./weekly-client-update"
    
    # 显示使用方法
    echo ""
    echo "📖 使用方法:"
    echo "  ./weekly-client-update --help"
    
    # 创建dist目录并复制文件
    mkdir -p dist
    cp weekly-client-update dist/
    cp README.md dist/ 2>/dev/null || true
    
    echo ""
    echo "📁 可执行文件已复制到 dist/ 目录"
else
    echo "❌ 编译失败"
    exit 1
fi