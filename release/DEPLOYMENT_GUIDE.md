# Gantt Application - Docker 一键部署

本文档说明如何使用 Docker 一键部署 Gantt 应用（包含单体后端、Nginx 和前端）。

## 📦 包含组件

- **单体后端** (Gantt Server) - 端口 8080
- **Nginx** - 端口 80（前端 + 反向代理）
- **前端应用** (Vue3)

## 🚀 快速开始

### 前置要求

- Docker 20.10+
- 可用内存：至少 2GB
- 可用磁盘：至少 5GB

### 一键构建

#### Windows 用户

```cmd
cd d:\works\gitlab\gantt\app
release\scripts\build-docker.bat
```

#### Linux/Mac 用户

```bash
cd /path/to/project
chmod +x release/scripts/build-docker.sh
./release/scripts/build-docker.sh
```

### 构建产物

构建完成后，会在以下目录生成文件：

```
release/
├── bin/
│   ├── gantt-app_YYYYMMDD_HHMMSS.tar.gz    # Docker 镜像压缩包
│   └── DEPLOY_YYYYMMDD_HHMMSS.md           # 部署说明文档
└── tmp/                                     # 临时文件（可清理）
```

## 📖 部署方式

### 方式 1：直接运行（开发测试）

构建后直接运行：

```bash
docker run -d \
  --name gantt-app \
  -p 80:80 \
  -p 8080:8080 \
  gantt-app:latest
```

访问：http://localhost

### 方式 2：加载镜像文件（生产部署）

将构建的镜像文件 `gantt-app_*.tar.gz` 传输到目标服务器：

```bash
# 解压
gunzip gantt-app_20251120_143000.tar.gz

# 加载镜像
docker load -i gantt-app_20251120_143000.tar

# 运行容器
docker run -d \
  --name gantt-app \
  -p 80:80 \
  -p 8080:8080 \
  gantt-app:latest
```

### 方式 3：Docker Compose（推荐）

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

## 🔧 配置

### 挂载配置文件

```bash
docker run -d \
  --name gantt-app \
  -p 80:80 \
  -v $(pwd)/config:/app/config:ro \
  gantt-app:latest
```

### 持久化日志

```bash
docker run -d \
  --name gantt-app \
  -p 80:80 \
  -v $(pwd)/logs:/var/log:rw \
  gantt-app:latest
```

## 🌐 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端页面 | http://localhost | Vue3 前端应用 |
| 健康检查 | http://localhost/health | 服务健康状态 |
| 管理服务 | http://localhost/api/management/ | REST API |
| MCP Server | ws://localhost/api/mcp/ | WebSocket |
| 智能体 | ws://localhost/api/agent/ | WebSocket |

## 📊 服务监控

### 查看容器状态

```bash
# 查看运行状态
docker ps | grep gantt-app

# 查看资源使用
docker stats gantt-app

# 健康检查
curl http://localhost/health
```

### 查看日志

```bash
# 容器总日志
docker logs gantt-app

# 实时跟踪
docker logs -f gantt-app

# 应用与 Nginx 日志
docker exec gantt-app tail -f /var/log/gantt-server.log
docker exec gantt-app tail -f /var/log/nginx/access.log
```

## 🔍 故障排查

### 容器无法启动

```bash
# 查看错误日志
docker logs gantt-app

# 检查容器详情
docker inspect gantt-app
```

### 服务无法访问

```bash
# 检查端口映射
docker port gantt-app

# 测试内部服务
docker exec gantt-app wget -O- http://localhost:8080/health
docker exec gantt-app ps aux
```

### 进入容器调试

```bash
# 进入容器 Shell
docker exec -it gantt-app sh

# 在容器内检查
ps aux                    # 查看进程
netstat -tlnp            # 查看端口
tail -f /var/log/*.log   # 查看日志
```

## 🛠️ 维护操作

### 重启服务

```bash
docker restart gantt-app
```

### 停止服务

```bash
docker stop gantt-app
```

### 删除容器

```bash
docker rm -f gantt-app
```

### 更新部署

```bash
# 停止旧容器
docker stop gantt-app && docker rm gantt-app

# 加载新镜像
docker load -i gantt-app_new.tar

# 启动新容器
docker run -d --name gantt-app -p 80:80 gantt-app:latest
```

## 📁 目录结构

```
.
├── Dockerfile                      # Docker 镜像定义
├── docker-compose.yml             # Docker Compose 配置
├── .dockerignore                  # Docker 构建忽略文件
├── release/
│   ├── bin/                       # 构建产物目录
│   ├── tmp/                       # 临时文件目录
│   ├── scripts/
│   │   ├── build-docker.sh        # Linux/Mac 构建脚本
│   │   └── build-docker.bat       # Windows 构建脚本
│   ├── docker/
│   │   ├── entrypoint.sh          # 容器启动脚本
│   │   ├── nginx.conf             # Nginx 主配置
│   │   └── default.conf           # Nginx 站点配置
│   └── DOCKER_README.md           # Docker 详细说明
└── config/                        # 应用配置文件
  ├── config.yml
  └── config.example.yml
```

## 🔒 安全建议

1. **不要暴露所有端口到公网**：生产环境只暴露 80/443
2. **使用环境变量管理敏感信息**：不要硬编码密码
3. **配置 HTTPS**：使用 Let's Encrypt 或其他证书
4. **限制资源使用**：使用 `--memory` 和 `--cpus` 参数
5. **定期更新镜像**：及时修复安全漏洞

## 🚀 生产部署示例

```bash
docker run -d \
  --name gantt-app \
  --restart=always \
  --memory=2g \
  --cpus=2 \
  -p 80:80 \
  -p 443:443 \
  -v /data/gantt/config:/app/config:ro \
  -v /data/gantt/logs:/var/log:rw \
  -v /data/gantt/ssl:/etc/nginx/ssl:ro \
  -e TZ=Asia/Shanghai \
  gantt-app:latest
```

## 📚 更多信息

详细的 Docker 部署说明请参考：
- [release/DOCKER_README.md](release/DOCKER_README.md)
- 构建后生成的 `release/bin/DEPLOY_*.md`

## ❓ 常见问题

**Q: 如何修改端口映射？**  
A: 使用 `-p 宿主机端口:容器端口` 参数，如 `-p 8080:80`

**Q: 如何查看各服务是否正常运行？**  
A: 执行 `docker exec gantt-app ps aux | grep -E "management|rostering|nginx"`

**Q: 构建失败怎么办？**  
A: 检查 Docker 是否正常运行，磁盘空间是否充足，网络是否能访问外部资源

**Q: 如何自定义 Nginx 配置？**  
A: 修改 `release/docker/default.conf` 后重新构建镜像

---

**构建日期**: 2025-11-20  
**维护者**: Gantt Team
