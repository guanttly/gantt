# 多阶段构建 - 第一阶段：构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# 配置 Alpine 使用国内镜像源（如果需要安装系统包）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 配置 npm 使用国内镜像源（加速下载）
RUN npm config set registry https://registry.npmmirror.com

# 安装 pnpm 并配置镜像源
RUN npm install -g pnpm@10.14.0 && \
    pnpm config set registry https://registry.npmmirror.com

# 先复制依赖配置文件，利用 Docker 缓存
COPY frontend/web/package.json frontend/web/pnpm-lock.yaml ./

# 安装依赖
RUN pnpm install --frozen-lockfile

# 复制所有前端源代码
COPY frontend/web/ ./

# 设置子路径部署环境变量（部署在 /gantt 路径下）
ENV VITE_BASE_PATH=/gantt/
ENV VITE_APP_API_BASE_URL=/gantt/api/management-service/api/
ENV VITE_API_BASE_URL=/gantt/api/management-service/api/
ENV VITE_WS_URL=/gantt/ws

# 根据部署模式构建
RUN pnpm run build

# 多阶段构建 - 第二阶段：构建 Go 服务
FROM golang:1.24-alpine AS go-builder

# 配置 Alpine 使用国内镜像源（加速软件包下载）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev

# 配置 Go 模块代理（使用国内镜像加速）
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

WORKDIR /app

# 复制 go.mod 和 go.sum（所有模块）
COPY go.mod go.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY agents/rostering/go.mod agents/rostering/go.sum ./agents/rostering/
COPY sdk/rostering/go.mod sdk/rostering/go.sum ./sdk/rostering/
COPY sdk/context/go.mod sdk/context/go.sum ./sdk/context/
COPY mcp-servers/rostering/go.mod mcp-servers/rostering/go.sum ./mcp-servers/rostering/
COPY mcp-servers/context/go.mod mcp-servers/context/go.sum ./mcp-servers/context/
COPY services/management-service/go.mod services/management-service/go.sum ./services/management-service/

# 下载依赖
RUN go mod download

# 复制所有源代码
COPY . .

# 构建管理服务
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' \
    -o /app/bin/management-service ./cmd/services/management-service

# 构建 MCP Server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' \
    -o /app/bin/rostering-server ./cmd/mcp-servers/rostering-server

# 构建 Context MCP Server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' \
    -o /app/bin/context-server ./cmd/mcp-servers/context-server

# 构建智能体
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' \
    -o /app/bin/rostering-agent ./cmd/agents/rostering

# 多阶段构建 - 第三阶段：运行时镜像
FROM alpine:latest

# 配置 Alpine 使用国内镜像源（加速软件包下载）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要的运行时依赖和 nginx
RUN apk add --no-cache ca-certificates nginx bash tzdata && \
    mkdir -p /run/nginx && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=go-builder /app/bin/management-service /app/bin/
COPY --from=go-builder /app/bin/rostering-server /app/bin/
COPY --from=go-builder /app/bin/context-server /app/bin/
COPY --from=go-builder /app/bin/rostering-agent /app/bin/

# 从构建阶段复制前端构建产物
COPY --from=frontend-builder /build/dist /usr/share/nginx/html

# 为每个服务创建配置目录并复制配置文件
RUN mkdir -p /app/services/management-service/config && \
    mkdir -p /app/mcp-servers/rostering-server/config && \
    mkdir -p /app/mcp-servers/context-server/config && \
    mkdir -p /app/agents/rostering/config

# 复制生产配置文件到各服务目录（使用 release/config 避免与开发配置冲突）
COPY release/config/common.yml /app/services/management-service/config/
COPY release/config/management-service.yml /app/services/management-service/config/

COPY release/config/common.yml /app/mcp-servers/rostering-server/config/
COPY release/config/mcp-servers/rostering-server.yml /app/mcp-servers/rostering-server/config/

COPY release/config/common.yml /app/mcp-servers/context-server/config/
COPY release/config/mcp-servers/context-server.yml /app/mcp-servers/context-server/config/

COPY release/config/common.yml /app/agents/rostering/config/
COPY release/config/agents/rostering-agent.yml /app/agents/rostering/config/

# 复制 nginx 配置（根据部署模式选择不同的配置）
COPY release/docker/nginx.conf /etc/nginx/nginx.conf
COPY release/docker/default.conf /etc/nginx/http.d/

# 复制启动脚本
COPY release/docker/entrypoint.sh /app/
RUN chmod +x /app/entrypoint.sh

# 暴露端口
# 80 - Nginx (前端和反向代理)
# 8080 - Management Service
# 8081 - Rostering Server (MCP)
# 8083 - Context Server (MCP)
# 8082 - Rostering Agent
EXPOSE 80 8080 8081 8082 8083

# 设置启动命令
ENTRYPOINT ["/app/entrypoint.sh"]
