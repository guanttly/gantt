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
FROM golang:1.25-alpine AS go-builder

# 配置 Alpine 使用国内镜像源（加速软件包下载）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev

# 配置 Go 模块代理（使用国内镜像加速）
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

WORKDIR /app

# 复制根模块依赖文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制所有源代码
COPY . .

# 构建当前单体入口
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' \
    -o /app/bin/gantt-server ./cmd/server

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
COPY --from=go-builder /app/bin/gantt-server /app/bin/

# 从构建阶段复制前端构建产物
COPY --from=frontend-builder /build/dist /usr/share/nginx/html

# 创建应用配置目录并复制默认配置
RUN mkdir -p /app/config

COPY config/config.yml /app/config/

# 复制 nginx 配置（根据部署模式选择不同的配置）
COPY release/docker/nginx.conf /etc/nginx/nginx.conf
COPY release/docker/default.conf /etc/nginx/http.d/

# 复制启动脚本
COPY release/docker/entrypoint.sh /app/
RUN chmod +x /app/entrypoint.sh

# 暴露端口
# 80 - Nginx (前端和反向代理)
# 8080 - 单体后端服务
EXPOSE 80 8080

# 设置启动命令
ENTRYPOINT ["/app/entrypoint.sh"]
