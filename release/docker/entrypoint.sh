#!/bin/bash

# Docker 容器启动脚本
# 该脚本负责启动单体后端服务和 Nginx

set -e

echo "=========================================="
echo "Starting Gantt Application Services"
echo "=========================================="

# 启动单体后端
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Gantt Server..."
cd /app
/app/bin/gantt-server > /var/log/gantt-server.log 2>&1 &
SERVER_PID=$!
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Gantt Server started with PID: $SERVER_PID"

# 等待后端启动
sleep 3

# 启动 Nginx
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Nginx..."
nginx -g 'daemon off;' &
NGINX_PID=$!
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Nginx started with PID: $NGINX_PID"

echo "=========================================="
echo "All services started successfully!"
echo "=========================================="
echo "Gantt Server PID: $SERVER_PID"
echo "Nginx PID: $NGINX_PID"
echo "=========================================="

# 定义优雅关闭函数
shutdown() {
    echo ""
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Shutting down services..."
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Stopping Nginx..."
    kill -TERM $NGINX_PID 2>/dev/null || true
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Stopping Gantt Server..."
    kill -TERM $SERVER_PID 2>/dev/null || true
    
    # 等待进程退出
    wait $NGINX_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] All services stopped"
    exit 0
}

# 捕获退出信号
trap shutdown SIGTERM SIGINT SIGQUIT

# 持续监控服务状态
while true; do
    # 检查所有服务是否还在运行
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: Gantt Server died!"
        exit 1
    fi
    
    if ! kill -0 $NGINX_PID 2>/dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: Nginx died!"
        exit 1
    fi
    
    sleep 10
done
