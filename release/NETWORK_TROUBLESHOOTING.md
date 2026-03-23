# Docker 构建网络问题排查指南

## 常见网络问题

### 1. NPM 依赖下载超时

**症状：**
```
WARN GET https://registry.npmjs.org/@vue/runtime-dom/-/runtime-dom-3.5.13.tgz error (ETIMEDOUT)
```

**解决方案：**

Dockerfile 已经配置了国内镜像源（`registry.npmmirror.com`），如果还是超时，可以尝试：

#### 方法 A：使用其他镜像源

编辑 `Dockerfile`，将 npm 镜像源改为其他可用的源：

```dockerfile
# 腾讯云镜像
RUN npm config set registry https://mirrors.cloud.tencent.com/npm/
RUN pnpm config set registry https://mirrors.cloud.tencent.com/npm/

# 或使用华为云镜像
RUN npm config set registry https://repo.huaweicloud.com/repository/npm/
RUN pnpm config set registry https://repo.huaweicloud.com/repository/npm/
```

#### 方法 B：本地预先构建前端

如果网络实在不稳定，可以本地先构建前端，再复制到 Docker：

1. 在 `frontend/web/` 目录创建 `.npmrc`：
   ```
   registry=https://registry.npmmirror.com
   ```

2. 本地构建前端：
   ```bash
   cd frontend/web
   pnpm install
   pnpm run build
   ```

3. 修改 Dockerfile，直接复制 dist 目录：
   ```dockerfile
   # 从本地复制前端构建产物（跳过构建阶段）
   FROM alpine:latest AS frontend-builder
   WORKDIR /app/frontend
   COPY frontend/web/dist ./dist
   ```

### 2. Go 模块下载超时

**症状：**
```
go: downloading github.com/xxx timeout
```

**解决方案：**

Dockerfile 已配置使用 `goproxy.cn`，如果还是超时：

#### 方法 A：使用其他 Go 代理

编辑 `Dockerfile`：

```dockerfile
# 使用阿里云代理
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 或使用官方代理
ENV GOPROXY=https://proxy.golang.org,direct

# 或组合使用多个代理
ENV GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct
```

#### 方法 B：使用 vendor 目录

1. 本地下载所有依赖：
   ```bash
   go mod download
   go mod vendor
   ```

2. 修改 Dockerfile：
   ```dockerfile
   COPY vendor/ ./vendor/
   RUN go build -mod=vendor ...
   ```

### 3. Alpine 软件包下载慢

**症状：**
```
apk add 命令很慢或超时
```

**解决方案：**

在 Dockerfile 的 alpine 镜像阶段添加国内镜像源：

```dockerfile
FROM alpine:latest

# 使用阿里云 Alpine 镜像
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 或使用中科大镜像
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

RUN apk add --no-cache ...
```

### 4. Docker Hub 拉取基础镜像慢

**症状：**
```
Pulling from library/node ... very slow
```

**解决方案：**

#### 方法 A：配置 Docker 镜像加速器

创建或编辑 `C:\ProgramData\docker\config\daemon.json` (Windows) 或 `/etc/docker/daemon.json` (Linux)：

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.ccs.tencentyun.com"
  ]
}
```

重启 Docker：
```bash
# Windows: 重启 Docker Desktop
# Linux:
sudo systemctl restart docker
```

#### 方法 B：手动拉取镜像

在构建前先手动拉取基础镜像：

```bash
docker pull node:20-alpine
docker pull golang:1.23-alpine
docker pull alpine:latest
```

### 5. 完全离线构建

如果需要在完全离线环境构建：

1. **准备阶段（有网络环境）：**
   ```bash
   # 下载基础镜像
   docker pull node:20-alpine
   docker pull golang:1.23-alpine
   docker pull alpine:latest
   docker save -o base-images.tar node:20-alpine golang:1.23-alpine alpine:latest
   
   # 下载前端依赖
   cd frontend/web
   pnpm install
   
   # 下载 Go 依赖
   cd ../../
   go mod download
   go mod vendor
   ```

2. **构建阶段（离线环境）：**
   ```bash
   # 加载基础镜像
   docker load -i base-images.tar
   
   # 使用修改后的 Dockerfile 构建（使用 vendor）
   docker build -t gantt-app:latest .
   ```

## 网络诊断命令

### 测试 NPM 镜像源连通性

```bash
# 测试淘宝镜像
curl -I https://registry.npmmirror.com

# 测试腾讯云镜像
curl -I https://mirrors.cloud.tencent.com/npm/

# 测试下载速度
curl -o /dev/null https://registry.npmmirror.com/vue/latest
```

### 测试 Go 代理连通性

```bash
# 测试 goproxy.cn
curl -I https://goproxy.cn

# 测试下载模块
curl https://goproxy.cn/github.com/gin-gonic/gin/@v/list
```

### 查看 Docker 构建日志

```bash
# 详细构建日志
docker build --progress=plain -t gantt-app:latest .

# 不使用缓存重新构建
docker build --no-cache -t gantt-app:latest .
```

## 推荐配置（中国大陆）

### Dockerfile 最佳实践

```dockerfile
# 前端构建阶段
FROM node:20-alpine AS frontend-builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN npm config set registry https://registry.npmmirror.com
RUN pnpm config set registry https://registry.npmmirror.com

# Go 构建阶段
FROM golang:1.23-alpine AS go-builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 运行时阶段
FROM alpine:latest
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
```

✅ **当前 Dockerfile 已包含以上所有优化配置！**

## 故障排查检查清单

- [ ] Docker Desktop 是否正常运行
- [ ] 是否配置了 Docker 镜像加速器
- [ ] NPM 镜像源是否可访问（curl 测试）
- [ ] Go 代理是否可访问（curl 测试）
- [ ] 防火墙/代理设置是否正确
- [ ] 磁盘空间是否充足（至少 10GB）
- [ ] 网络是否稳定（ping 8.8.8.8）
- [ ] DNS 解析是否正常（nslookup registry.npmmirror.com）

---

**最后更新**: 2025-11-20
