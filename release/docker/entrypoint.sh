#!/bin/bash

# Docker 容器启动脚本
# 该脚本负责启动所有服务：Management Service, MCP Server, Agent 和 Nginx

set -e

echo "=========================================="
echo "Starting Gantt Application Services"
echo "=========================================="

# 启动管理服务
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Management Service..."
cd /app/services/management-service
/app/bin/management-service > /var/log/management-service.log 2>&1 &
MANAGEMENT_PID=$!
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Management Service started with PID: $MANAGEMENT_PID"

# 等待管理服务启动
sleep 3

# 启动 Rostering MCP Server
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Rostering MCP Server..."
cd /app/mcp-servers/rostering-server
/app/bin/rostering-server > /var/log/rostering-server.log 2>&1 &
MCP_PID=$!
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Rostering MCP Server started with PID: $MCP_PID"

# 等待 MCP Server 启动
sleep 2

# 启动 Context MCP Server
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Context MCP Server..."
cd /app/mcp-servers/context-server
/app/bin/context-server > /var/log/context-server.log 2>&1 &
CONTEXT_PID=$!
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Context MCP Server started with PID: $CONTEXT_PID"

# 等待 Context Server 启动
sleep 2

# 启动智能体
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Rostering Agent..."
cd /app/agents/rostering
/app/bin/rostering-agent > /var/log/rostering-agent.log 2>&1 &
AGENT_PID=$!
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Rostering Agent started with PID: $AGENT_PID"

# 等待智能体启动
sleep 3

# 启动 Nginx
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting Nginx..."
nginx -g 'daemon off;' &
NGINX_PID=$!
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Nginx started with PID: $NGINX_PID"

echo "=========================================="
echo "All services started successfully!"
echo "=========================================="
echo "Management Service PID: $MANAGEMENT_PID"
echo "Rostering MCP Server PID: $MCP_PID"
echo "Context MCP Server PID: $CONTEXT_PID"
echo "Rostering Agent PID: $AGENT_PID"
echo "Nginx PID: $NGINX_PID"
echo "=========================================="

# 定义优雅关闭函数
shutdown() {
    echo ""
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Shutting down services..."
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Stopping Nginx..."
    kill -TERM $NGINX_PID 2>/dev/null || true
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Stopping Rostering Agent..."
    kill -TERM $AGENT_PID 2>/dev/null || true
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Stopping Rostering MCP Server..."
    kill -TERM $MCP_PID 2>/dev/null || true
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Stopping Context MCP Server..."
    kill -TERM $CONTEXT_PID 2>/dev/null || true
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Stopping Management Service..."
    kill -TERM $MANAGEMENT_PID 2>/dev/null || true
    
    # 等待进程退出
    wait $NGINX_PID 2>/dev/null || true
    wait $AGENT_PID 2>/dev/null || true
    wait $MCP_PID 2>/dev/null || true
    wait $CONTEXT_PID 2>/dev/null || true
    wait $MANAGEMENT_PID 2>/dev/null || true
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] All services stopped"
    exit 0
}

# 捕获退出信号
trap shutdown SIGTERM SIGINT SIGQUIT

# 持续监控服务状态
while true; do
    # 检查所有服务是否还在运行
    if ! kill -0 $MANAGEMENT_PID 2>/dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: Management Service died!"
        exit 1
    fi
    
    if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: MCP Server died!"
        exit 1
    fi
    
    if ! kill -0 $AGENT_PID 2>/dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: Agent died!"
        exit 1
    fi
    
    if ! kill -0 $NGINX_PID 2>/dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: Nginx died!"
        exit 1
    fi
    
    sleep 10
done
