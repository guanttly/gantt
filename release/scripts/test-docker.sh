#!/bin/bash

# Docker 镜像测试脚本
# 用于验证构建的镜像是否正常工作

set -e

IMAGE_NAME="gantt-app:latest"
CONTAINER_NAME="gantt-app-test"
TEST_PORT=8888

echo "=========================================="
echo "Docker Image Test Script"
echo "=========================================="
echo ""

# 检查镜像是否存在
echo "[1/5] 检查镜像..."
if ! docker images | grep -q "gantt-app"; then
    echo "错误: 镜像 $IMAGE_NAME 不存在"
    echo "请先运行构建脚本: ./release/scripts/build-docker.sh"
    exit 1
fi
echo "✓ 镜像存在"

# 清理可能存在的测试容器
echo ""
echo "[2/5] 清理旧的测试容器..."
docker rm -f $CONTAINER_NAME 2>/dev/null || true
echo "✓ 清理完成"

# 启动测试容器
echo ""
echo "[3/5] 启动测试容器..."
docker run -d \
  --name $CONTAINER_NAME \
  -p $TEST_PORT:80 \
  $IMAGE_NAME

echo "✓ 容器已启动"

# 等待服务启动
echo ""
echo "[4/5] 等待服务启动 (30秒)..."
sleep 30

# 测试服务
echo ""
echo "[5/5] 测试服务..."

# 测试健康检查端点
echo -n "  - 健康检查端点: "
if curl -sf http://localhost:$TEST_PORT/health > /dev/null; then
    echo "✓ 成功"
else
    echo "✗ 失败"
    docker logs $CONTAINER_NAME
    docker rm -f $CONTAINER_NAME
    exit 1
fi

# 测试前端页面
echo -n "  - 前端页面: "
if curl -sf http://localhost:$TEST_PORT/ > /dev/null; then
    echo "✓ 成功"
else
    echo "✗ 失败"
fi

# 检查各服务进程
echo -n "  - Management Service: "
if docker exec $CONTAINER_NAME ps aux | grep -q "[m]anagement-service"; then
    echo "✓ 运行中"
else
    echo "✗ 未运行"
fi

echo -n "  - Rostering Server: "
if docker exec $CONTAINER_NAME ps aux | grep -q "[r]ostering-server"; then
    echo "✓ 运行中"
else
    echo "✗ 未运行"
fi

echo -n "  - Rostering Agent: "
if docker exec $CONTAINER_NAME ps aux | grep -q "[r]ostering-agent"; then
    echo "✓ 运行中"
else
    echo "✗ 未运行"
fi

echo -n "  - Nginx: "
if docker exec $CONTAINER_NAME ps aux | grep -q "[n]ginx"; then
    echo "✓ 运行中"
else
    echo "✗ 未运行"
fi

# 显示测试结果
echo ""
echo "=========================================="
echo "测试完成！"
echo "=========================================="
echo ""
echo "容器信息:"
docker ps --filter name=$CONTAINER_NAME --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""
echo "访问地址: http://localhost:$TEST_PORT"
echo ""
echo "停止测试容器: docker stop $CONTAINER_NAME"
echo "删除测试容器: docker rm -f $CONTAINER_NAME"
echo ""
