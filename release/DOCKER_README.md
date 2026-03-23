# Docker 部署相关文件说明

本目录包含 Docker 容器化部署所需的配置文件和脚本。

## 文件说明

### 核心文件

- **Dockerfile**: 多阶段构建的 Docker 镜像定义文件
- **docker-compose.yml**: Docker Compose 编排配置文件
- **release/docker/entrypoint.sh**: 容器启动脚本
- **release/docker/nginx.conf**: Nginx 主配置文件
- **release/docker/default.conf**: Nginx 站点配置文件

### 构建脚本

- **release/scripts/build-docker.sh**: Linux/Mac 下的一键构建脚本
- **release/scripts/build-docker.bat**: Windows 下的一键构建脚本

### 目录结构

```
release/
├── bin/                    # 构建产物存放目录
│   ├── gantt-app_*.tar     # Docker 镜像文件
│   └── DEPLOY_*.md         # 部署说明文档
├── tmp/                    # 中间产物存放目录
├── scripts/                # 构建脚本
│   ├── build-docker.sh     # Linux/Mac 构建脚本
│   └── build-docker.bat    # Windows 构建脚本
└── docker/                 # Docker 配置文件
    ├── entrypoint.sh       # 容器启动脚本
    ├── nginx.conf          # Nginx 主配置
    └── default.conf        # Nginx 站点配置
```

## 快速开始

### 方式一：使用构建脚本（推荐）

#### Linux/Mac

```bash
cd /path/to/project
chmod +x release/scripts/build-docker.sh
./release/scripts/build-docker.sh
```

#### Windows

```cmd
cd \path\to\project
release\scripts\build-docker.bat
```

构建完成后，镜像文件会保存在 `release/bin/` 目录下。

### 方式二：使用 Docker Compose（开发测试）

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 方式三：手动构建

```bash
# 构建镜像
docker build -t gantt-app:latest .

# 运行容器
docker run -d \
  --name gantt-app \
  -p 80:80 \
  -p 8080:8080 \
  -p 8081:8081 \
  -p 8082:8082 \
  gantt-app:latest
```

## 服务架构

容器内运行以下服务：

1. **Nginx** (端口 80)
   - 前端静态文件服务
   - 反向代理 API 请求到后端服务

2. **Management Service** (端口 8080)
   - 管理服务 API
   - 路由前缀: `/api/management/`

3. **Rostering MCP Server** (端口 8081)
   - MCP 协议服务器
   - WebSocket 路由: `/api/mcp/`

4. **Rostering Agent** (端口 8082)
   - 智能体服务
   - WebSocket 路由: `/api/agent/`

## 配置说明

### 环境变量

可以通过环境变量覆盖默认配置：

```bash
docker run -d \
  --name gantt-app \
  -e TZ=Asia/Shanghai \
  -e LOG_LEVEL=debug \
  -p 80:80 \
  gantt-app:latest
```

### 配置文件挂载

建议在生产环境挂载配置文件：

```bash
docker run -d \
  --name gantt-app \
  -v $(pwd)/config:/app/config:ro \
  -v $(pwd)/logs:/var/log:rw \
  -p 80:80 \
  gantt-app:latest
```

### 持久化数据

如果需要持久化数据，挂载数据目录：

```bash
docker run -d \
  --name gantt-app \
  -v gantt-data:/app/data \
  -p 80:80 \
  gantt-app:latest
```

## 健康检查

容器包含健康检查端点：

```bash
# 检查健康状态
curl http://localhost/health

# 查看 Docker 健康检查状态
docker inspect --format='{{.State.Health.Status}}' gantt-app
```

## 日志管理

### 查看日志

```bash
# 查看容器日志
docker logs gantt-app

# 实时跟踪日志
docker logs -f gantt-app

# 查看特定服务日志
docker exec gantt-app tail -f /var/log/management-service.log
docker exec gantt-app tail -f /var/log/rostering-server.log
docker exec gantt-app tail -f /var/log/rostering-agent.log
docker exec gantt-app tail -f /var/log/nginx/access.log
docker exec gantt-app tail -f /var/log/nginx/error.log
```

### 日志持久化

```bash
docker run -d \
  --name gantt-app \
  -v $(pwd)/logs:/var/log:rw \
  -p 80:80 \
  gantt-app:latest
```

## 网络配置

### 反向代理路由规则

- `/` → 前端静态文件
- `/api/management/` → Management Service (HTTP)
- `/api/mcp/` → Rostering MCP Server (WebSocket)
- `/api/agent/` → Rostering Agent (WebSocket)
- `/health` → 健康检查

### 自定义域名

使用 Nginx 反向代理时，修改 `release/docker/default.conf`:

```nginx
server_name your-domain.com;
```

## 生产部署建议

1. **HTTPS 配置**: 在 Nginx 中配置 SSL 证书
2. **资源限制**: 使用 `--memory` 和 `--cpus` 限制资源
3. **自动重启**: 使用 `--restart=always` 确保服务高可用
4. **日志轮转**: 配置日志轮转避免磁盘占满
5. **监控告警**: 集成监控系统（如 Prometheus）
6. **备份策略**: 定期备份配置和数据

## 故障排查

### 容器无法启动

```bash
# 查看容器日志
docker logs gantt-app

# 检查容器状态
docker inspect gantt-app
```

### 服务无法访问

```bash
# 检查端口映射
docker port gantt-app

# 检查网络连接
docker exec gantt-app netstat -tlnp

# 测试服务端点
docker exec gantt-app wget -O- http://localhost:8080/health
```

### 性能问题

```bash
# 查看资源使用
docker stats gantt-app

# 查看进程状态
docker exec gantt-app ps aux
```

## 更新和维护

### 更新镜像

```bash
# 停止旧容器
docker stop gantt-app
docker rm gantt-app

# 拉取/加载新镜像
docker load -i gantt-app_new.tar

# 启动新容器
docker run -d --name gantt-app -p 80:80 gantt-app:latest
```

### 滚动更新

使用 Docker Compose:

```bash
docker-compose pull
docker-compose up -d --no-deps --build gantt-app
```

## 安全建议

1. 不要在镜像中硬编码敏感信息
2. 使用 secrets 或环境变量传递密码
3. 定期更新基础镜像
4. 限制容器权限（避免使用 root）
5. 使用私有镜像仓库

## 技术支持

如有问题，请查看：

- 容器日志: `docker logs gantt-app`
- 部署文档: `release/bin/DEPLOY_*.md`
- 健康检查: `http://localhost/health`

---

**最后更新**: 2025-11-20
