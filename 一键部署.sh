#!/bin/bash

# ==============================================
# Gantt Application - 一键部署脚本 (Linux/Mac)
# ==============================================
# 此脚本会自动完成：构建 -> 测试 -> 启动
# ==============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo "╔════════════════════════════════════════════╗"
echo "║  Gantt Application 一键部署脚本           ║"
echo "╚════════════════════════════════════════════╝"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 步骤 1: 构建 Docker 镜像
echo -e "${BLUE}[步骤 1/3] 构建 Docker 镜像...${NC}"
echo ""

chmod +x release/scripts/build-docker.sh
./release/scripts/build-docker.sh

if [ $? -ne 0 ]; then
    echo ""
    echo -e "${RED}❌ 构建失败！${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ 构建成功！${NC}"
echo ""
sleep 2

# 步骤 2: 询问是否进行测试
echo -e "${BLUE}[步骤 2/3] 是否进行镜像测试？${NC}"
read -p "是否测试镜像? (y/n, 默认 n): " DO_TEST
if [[ "$DO_TEST" == "y" || "$DO_TEST" == "Y" ]]; then
    echo ""
    echo "开始测试..."
    chmod +x release/scripts/test-docker.sh
    ./release/scripts/test-docker.sh || {
        echo ""
        echo -e "${YELLOW}⚠ 测试失败，但可以继续部署${NC}"
    }
    echo ""
    sleep 2
else
    echo "跳过测试"
fi

# 步骤 3: 启动容器
echo ""
echo -e "${BLUE}[步骤 3/3] 启动容器...${NC}"
echo ""

# 检查是否已有运行的容器
if docker ps -a | grep -q "gantt-app"; then
    echo "检测到已存在的容器，是否删除？"
    read -p "删除旧容器? (y/n, 默认 y): " REMOVE_OLD
    if [[ "$REMOVE_OLD" != "n" && "$REMOVE_OLD" != "N" ]]; then
        echo "删除旧容器..."
        docker rm -f gantt-app 2>/dev/null || true
    fi
fi

echo "启动新容器..."
docker run -d \
  --name gantt-app \
  --restart unless-stopped \
  -p 80:80 \
  -p 8080:8080 \
  -p 8081:8081 \
  -p 8082:8082 \
  gantt-app:latest

if [ $? -ne 0 ]; then
    echo ""
    echo -e "${RED}❌ 容器启动失败！${NC}"
    exit 1
fi

# 等待服务启动
echo ""
echo "等待服务启动 (15秒)..."
sleep 15

# 健康检查
echo ""
echo "检查服务状态..."
if curl -sf http://localhost/health > /dev/null; then
    echo -e "${GREEN}✅ 服务已就绪！${NC}"
else
    echo -e "${YELLOW}⚠ 健康检查失败，服务可能还在启动中${NC}"
    echo "  请稍后访问: http://localhost"
fi

# 显示成功信息
echo ""
echo "╔════════════════════════════════════════════╗"
echo "║           🎉 部署成功！                    ║"
echo "╚════════════════════════════════════════════╝"
echo ""
echo -e "${GREEN}📱 访问地址:${NC}"
echo "   前端页面: http://localhost"
echo "   健康检查: http://localhost/health"
echo ""
echo -e "${GREEN}📊 容器状态:${NC}"
docker ps --filter name=gantt-app --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""
echo -e "${GREEN}📝 常用命令:${NC}"
echo "   查看日志: docker logs -f gantt-app"
echo "   停止服务: docker stop gantt-app"
echo "   重启服务: docker restart gantt-app"
echo "   删除容器: docker rm -f gantt-app"
echo ""
