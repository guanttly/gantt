# Docker 一键部署配置完成清单

## 📋 已创建文件列表

### 核心配置文件

1. **Dockerfile** - 主路径
   - 多阶段构建配置
   - 包含前端构建、Go 服务构建和运行时镜像

2. **docker-compose.yml** - 主路径
   - Docker Compose 编排配置
   - 可选的 MySQL 和 Redis 配置模板

3. **.dockerignore** - 主路径
   - 优化构建上下文，排除不必要的文件

### Docker 配置文件

4. **release/docker/entrypoint.sh**
   - 容器启动脚本
   - 管理所有服务的启动和监控

5. **release/docker/nginx.conf**
   - Nginx 主配置文件
   - 基础的 HTTP 服务器配置

6. **release/docker/default.conf**
   - Nginx 站点配置
   - 包含前端静态文件服务和 API 反向代理

### 构建脚本

7. **release/scripts/build-docker.sh** (Linux/Mac)
   - 一键构建脚本
   - 自动化构建流程，生成镜像和文档

8. **release/scripts/build-docker.bat** (Windows)
   - Windows 版本的构建脚本
   - 功能与 shell 脚本一致

### 测试脚本

9. **release/scripts/test-docker.sh** (Linux/Mac)
   - 镜像测试脚本
   - 验证各服务是否正常运行

10. **release/scripts/test-docker.bat** (Windows)
    - Windows 版本的测试脚本

### 文档文件

11. **DOCKER.md** - 主路径
    - 快速使用指南
    - 常用命令汇总

12. **release/DEPLOYMENT_GUIDE.md**
    - 完整的部署指南
    - 包含多种部署方式和故障排查

13. **release/DOCKER_README.md**
    - Docker 详细说明文档
    - 包含架构、配置、维护等信息

### 辅助文件

14. **release/bin/.gitkeep**
    - 保持 bin 目录结构

15. **release/tmp/.gitkeep**
    - 保持 tmp 目录结构

16. **release/.gitignore_suggestion**
    - 建议添加到项目 .gitignore 的内容

## 📁 目录结构

```
gantt/app/
├── Dockerfile                          # Docker 镜像定义
├── docker-compose.yml                  # Docker Compose 配置
├── .dockerignore                       # Docker 忽略文件
├── DOCKER.md                          # 快速使用指南
│
├── release/
│   ├── bin/                           # 构建产物目录
│   │   └── .gitkeep
│   ├── tmp/                           # 临时文件目录
│   │   └── .gitkeep
│   ├── scripts/
│   │   ├── build-docker.sh            # Linux/Mac 构建脚本
│   │   ├── build-docker.bat           # Windows 构建脚本
│   │   ├── test-docker.sh             # Linux/Mac 测试脚本
│   │   └── test-docker.bat            # Windows 测试脚本
│   ├── docker/
│   │   ├── entrypoint.sh              # 容器启动脚本
│   │   ├── nginx.conf                 # Nginx 主配置
│   │   └── default.conf               # Nginx 站点配置
│   ├── DEPLOYMENT_GUIDE.md            # 部署指南
│   ├── DOCKER_README.md               # Docker 详细说明
│   └── .gitignore_suggestion          # Git 忽略建议
│
└── config/                            # 应用配置目录（已存在）
    ├── common.yml
    ├── management-service.yml
    ├── agents/
    │   └── rostering-agent.yml
    └── mcp-servers/
        └── rostering-server.yml
```

## 🚀 使用流程

### 1. 构建镜像

**Windows:**
```cmd
release\scripts\build-docker.bat
```

**Linux/Mac:**
```bash
chmod +x release/scripts/*.sh
./release/scripts/build-docker.sh
```

### 2. 测试镜像（可选）

**Windows:**
```cmd
release\scripts\test-docker.bat
```

**Linux/Mac:**
```bash
./release/scripts/test-docker.sh
```

### 3. 部署运行

**方式 A: 直接运行**
```bash
docker run -d --name gantt-app -p 80:80 -p 8080:8080 -p 8081:8081 -p 8082:8082 gantt-app:latest
```

**方式 B: Docker Compose**
```bash
docker-compose up -d
```

**方式 C: 加载镜像文件**
```bash
gunzip release/bin/gantt-app_*.tar.gz
docker load -i release/bin/gantt-app_*.tar
docker run -d --name gantt-app -p 80:80 gantt-app:latest
```

## 📦 构建产物

构建成功后，在 `release/bin/` 目录下会生成：

1. **gantt-app_YYYYMMDD_HHMMSS.tar.gz** - Docker 镜像压缩包
2. **DEPLOY_YYYYMMDD_HHMMSS.md** - 部署说明文档

## 🌐 访问地址

启动后可通过以下地址访问：

- **前端页面**: http://localhost
- **健康检查**: http://localhost/health
- **管理服务 API**: http://localhost/api/management/
- **MCP WebSocket**: ws://localhost/api/mcp/
- **智能体 WebSocket**: ws://localhost/api/agent/

## 🔧 服务架构

容器内包含以下服务：

1. **Nginx** (端口 80)
   - 前端静态文件服务
   - API 反向代理

2. **Management Service** (端口 8080)
   - 管理服务 REST API

3. **Rostering MCP Server** (端口 8081)
   - MCP 协议服务器 (WebSocket)

4. **Rostering Agent** (端口 8082)
   - 智能体服务 (WebSocket)

## 📝 重要说明

### 脚本权限（Linux/Mac）

首次使用需要赋予脚本执行权限：
```bash
chmod +x release/scripts/build-docker.sh
chmod +x release/scripts/test-docker.sh
chmod +x release/docker/entrypoint.sh
```

### 配置文件修改

如需修改服务配置：
1. 编辑 `config/` 目录下的相应配置文件
2. 重新构建镜像，或通过卷挂载覆盖配置

### Nginx 配置修改

如需修改 Nginx 配置：
1. 编辑 `release/docker/nginx.conf` 或 `release/docker/default.conf`
2. 重新构建镜像

### 端口冲突

如果端口被占用，可修改端口映射：
```bash
docker run -d --name gantt-app -p 8080:80 gantt-app:latest
```

## ✅ 验证清单

- [ ] 所有脚本文件已创建
- [ ] Docker 配置文件已就位
- [ ] 文档文件已生成
- [ ] 目录结构正确
- [ ] 可以成功构建镜像
- [ ] 容器可以正常启动
- [ ] 各服务可以正常访问

## 🆘 故障排查

### 构建失败

1. 检查 Docker 是否正常运行
2. 检查网络连接（需要下载依赖）
3. 检查磁盘空间是否充足
4. 查看构建日志定位具体错误

### 容器启动失败

1. 查看容器日志: `docker logs gantt-app`
2. 检查端口是否被占用
3. 检查配置文件是否正确
4. 进入容器调试: `docker exec -it gantt-app sh`

### 服务无法访问

1. 检查容器是否运行: `docker ps`
2. 检查端口映射: `docker port gantt-app`
3. 测试健康检查: `curl http://localhost/health`
4. 查看 Nginx 日志: `docker exec gantt-app tail -f /var/log/nginx/error.log`

## 📚 参考文档

- [快速使用指南](../DOCKER.md)
- [完整部署指南](DEPLOYMENT_GUIDE.md)
- [Docker 详细说明](DOCKER_README.md)

---

**创建日期**: 2025-11-20  
**版本**: 1.0.0  
**状态**: ✅ 完成
