# Docker 一键部署说明

## 快速开始

### Windows 用户

```cmd
# 构建 Docker 镜像
release\scripts\build-docker.bat

# 测试镜像（可选）
release\scripts\test-docker.bat

# 运行容器
docker run -d --name gantt-app -p 80:80 -p 8080:8080 -p 8081:8081 -p 8082:8082 gantt-app:latest
```

### Linux/Mac 用户

```bash
# 赋予执行权限
chmod +x release/scripts/*.sh

# 构建 Docker 镜像
./release/scripts/build-docker.sh

# 测试镜像（可选）
./release/scripts/test-docker.sh

# 运行容器
docker run -d --name gantt-app -p 80:80 -p 8080:8080 -p 8081:8081 -p 8082:8082 gantt-app:latest
```

### 使用 Docker Compose

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

## 访问应用

- **前端页面**: http://localhost
- **健康检查**: http://localhost/health

## 构建产物

构建成功后，产物位于：

- **镜像文件**: `release/bin/gantt-app_*.tar.gz`
- **部署文档**: `release/bin/DEPLOY_*.md`

## 详细文档

- [部署指南](release/DEPLOYMENT_GUIDE.md)
- [Docker 详细说明](release/DOCKER_README.md)

## 目录说明

```
release/
├── bin/                    # 构建产物（镜像文件）
├── tmp/                    # 临时文件
├── scripts/
│   ├── build-docker.sh     # Linux/Mac 构建脚本
│   ├── build-docker.bat    # Windows 构建脚本
│   ├── test-docker.sh      # Linux/Mac 测试脚本
│   └── test-docker.bat     # Windows 测试脚本
└── docker/
    ├── entrypoint.sh       # 容器启动脚本
    ├── nginx.conf          # Nginx 主配置
    └── default.conf        # Nginx 站点配置
```

## 服务端口

| 端口 | 服务 |
|------|------|
| 80 | Nginx (前端 + 反向代理) |
| 8080 | Management Service |
| 8081 | Rostering MCP Server |
| 8082 | Rostering Agent |
