#!/bin/bash

# ========================================
# Docker 镜像一键构建脚本
# ========================================
# 功能：
# 1. 清理旧的构建产物
# 2. 构建 Docker 镜像
# 3. 保存镜像到文件
# 4. 生成部署说明
# ========================================

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RELEASE_DIR="$PROJECT_ROOT/release"
BIN_DIR="$RELEASE_DIR/bin"
TMP_DIR="$RELEASE_DIR/tmp"

# 配置变量
IMAGE_NAME="gantt-app"
IMAGE_TAG="latest"
FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
OUTPUT_TAR="${BIN_DIR}/${IMAGE_NAME}_${TIMESTAMP}.tar"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Gantt Application Docker Build Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "项目根目录: $PROJECT_ROOT"
echo "发布目录: $RELEASE_DIR"
echo "镜像名称: $FULL_IMAGE_NAME"
echo "输出文件: $OUTPUT_TAR"
echo ""

# 步骤 1: 清理旧的构建产物
echo -e "${YELLOW}[1/5] 清理旧的构建产物...${NC}"
# 确保目录存在
mkdir -p "$BIN_DIR"
mkdir -p "$TMP_DIR"

if [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"/*
    echo "临时目录已清理"
fi
echo "目录结构已准备"

# 步骤 2: 检查必要文件
echo -e "${YELLOW}[2/5] 检查必要文件...${NC}"
REQUIRED_FILES=(
    "$PROJECT_ROOT/Dockerfile"
    "$PROJECT_ROOT/go.mod"
    "$PROJECT_ROOT/frontend/web/package.json"
  "$PROJECT_ROOT/config/config.yml"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}错误: 缺少必要文件 $file${NC}"
        exit 1
    fi
done
echo "必要文件检查通过"

# 步骤 3: 构建 Docker 镜像
echo -e "${YELLOW}[3/5] 构建 Docker 镜像...${NC}"
echo "这可能需要几分钟时间..."
echo ""
echo "提示: 如果遇到网络问题，Dockerfile 已配置使用国内镜像源"
echo "  - NPM: registry.npmmirror.com"
echo "  - Go: goproxy.cn"
echo ""
cd "$PROJECT_ROOT"

docker build -t "$FULL_IMAGE_NAME" \
    --build-arg BUILDTIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --build-arg VERSION="${IMAGE_TAG}" \
    -f Dockerfile \
    .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Docker 镜像构建成功！${NC}"
else
    echo -e "${RED}Docker 镜像构建失败！${NC}"
    exit 1
fi

# 步骤 4: 保存镜像到文件
echo -e "${YELLOW}[4/5] 保存镜像到文件...${NC}"
docker save -o "$OUTPUT_TAR" "$FULL_IMAGE_NAME"

if [ $? -eq 0 ]; then
    # 压缩镜像文件
    echo "压缩镜像文件..."
    gzip -f "$OUTPUT_TAR"
    OUTPUT_FILE="${OUTPUT_TAR}.gz"
    
    FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${GREEN}镜像已保存: $OUTPUT_FILE (大小: $FILE_SIZE)${NC}"
else
    echo -e "${RED}保存镜像失败！${NC}"
    exit 1
fi

# 步骤 5: 生成部署文档
echo -e "${YELLOW}[5/5] 生成部署文档...${NC}"
DEPLOY_DOC="${BIN_DIR}/DEPLOY_${TIMESTAMP}.md"

cat > "$DEPLOY_DOC" << EOF
# Gantt Application Docker 部署说明

**构建时间**: $(date '+%Y-%m-%d %H:%M:%S')
**镜像名称**: ${FULL_IMAGE_NAME}
**镜像文件**: ${IMAGE_NAME}_${TIMESTAMP}.tar.gz

## 快速部署

### 1. 加载镜像

\`\`\`bash
# 解压镜像文件
gunzip ${IMAGE_NAME}_${TIMESTAMP}.tar.gz

# 加载镜像到 Docker
docker load -i ${IMAGE_NAME}_${TIMESTAMP}.tar
\`\`\`

### 2. 运行容器

\`\`\`bash
# 运行容器（基础模式）
docker run -d \\
  --name gantt-app \\
  -p 80:80 \\
  -p 8080:8080 \\
  -p 8081:8081 \\
  -p 8082:8082 \\
  ${FULL_IMAGE_NAME}

# 运行容器（挂载配置文件）
docker run -d \\
  --name gantt-app \\
  -p 80:80 \\
  -p 8080:8080 \\
  -p 8081:8081 \\
  -p 8082:8082 \\
  -v \$(pwd)/config:/app/config \\
  ${FULL_IMAGE_NAME}
\`\`\`

### 3. 验证部署

\`\`\`bash
# 查看容器状态
docker ps | grep gantt-app

# 查看容器日志
docker logs gantt-app

# 健康检查
curl http://localhost/health
\`\`\`

## 服务端口说明

| 端口 | 服务 | 说明 |
|------|------|------|
| 80 | Nginx | 前端页面和反向代理 |
| 8080 | Management Service | 管理服务 API |
| 8081 | Rostering MCP Server | MCP 服务器 (WebSocket) |
| 8082 | Rostering Agent | 智能体服务 (WebSocket) |

## 访问地址

- **前端页面**: http://localhost/
- **健康检查**: http://localhost/health
- **管理服务 API**: http://localhost/api/management/
- **MCP WebSocket**: ws://localhost/api/mcp/
- **智能体 WebSocket**: ws://localhost/api/agent/

## 常用命令

\`\`\`bash
# 停止容器
docker stop gantt-app

# 启动容器
docker start gantt-app

# 重启容器
docker restart gantt-app

# 删除容器
docker rm -f gantt-app

# 查看实时日志
docker logs -f gantt-app

# 进入容器 Shell
docker exec -it gantt-app sh

# 查看容器内服务状态
docker exec gantt-app ps aux
\`\`\`

## Docker Compose 部署

如果使用 Docker Compose，可以创建 \`docker-compose.yml\`:

\`\`\`yaml
version: '3.8'

services:
  gantt-app:
    image: ${FULL_IMAGE_NAME}
    container_name: gantt-app
    ports:
      - "80:80"
      - "8080:8080"
      - "8081:8081"
      - "8082:8082"
    volumes:
      - ./config:/app/config
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
\`\`\`

然后运行:

\`\`\`bash
docker-compose up -d
\`\`\`

## 故障排查

### 查看各服务日志

\`\`\`bash
# 应用日志
docker exec gantt-app tail -f /var/log/gantt-server.log

# Nginx 日志
docker exec gantt-app tail -f /var/log/nginx/error.log
docker exec gantt-app tail -f /var/log/nginx/access.log
\`\`\`

### 检查服务进程

\`\`\`bash
docker exec gantt-app ps aux | grep -E "gantt-server|nginx"
\`\`\`

## 环境要求

- Docker 20.10+
- 可用内存: 至少 2GB
- 可用磁盘: 至少 5GB

## 注意事项

1. 确保配置文件 \`/app/config\` 中的数据库连接等信息正确
2. 如需持久化数据，建议挂载数据卷
3. 生产环境建议配置 HTTPS 和域名
4. 定期备份配置文件和数据

---

**构建信息**
- 构建时间: $(date '+%Y-%m-%d %H:%M:%S')
- 构建主机: $(hostname)
- Docker 版本: $(docker --version)
EOF

echo -e "${GREEN}部署文档已生成: $DEPLOY_DOC${NC}"

# 显示摘要
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}构建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "镜像名称: $FULL_IMAGE_NAME"
echo "镜像文件: $OUTPUT_FILE ($FILE_SIZE)"
echo "部署文档: $DEPLOY_DOC"
echo ""
echo -e "${YELLOW}快速启动命令:${NC}"
echo "  docker run -d --name gantt-app -p 80:80 -p 8080:8080 -p 8081:8081 -p 8082:8082 $FULL_IMAGE_NAME"
echo ""
echo -e "${YELLOW}或者先加载镜像:${NC}"
echo "  gunzip $OUTPUT_FILE"
echo "  docker load -i ${OUTPUT_TAR}"
echo "  docker run -d --name gantt-app -p 80:80 -p 8080:8080 -p 8081:8081 -p 8082:8082 $FULL_IMAGE_NAME"
echo ""
