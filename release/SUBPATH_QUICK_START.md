# 子路径部署快速指南

## 构建子路径部署版本

### 方法 1：使用 Docker 构建参数

```bash
# 构建子路径部署版本（访问路径: /gantt）
docker build --build-arg DEPLOY_MODE=gantt -t gantt-app:gantt .

# 构建根路径部署版本（访问路径: /）
docker build -t gantt-app:latest .
```

### 方法 2：本地构建前端

```bash
cd frontend/web

# 子路径部署
pnpm run build --mode gantt

# 根路径部署
pnpm run build
```

## 运行容器

```bash
# 子路径版本
docker run -d --name gantt-app -p 80:80 gantt-app:gantt

# 根路径版本
docker run -d --name gantt-app -p 80:80 gantt-app:latest
```

## 外部 Nginx 反向代理配置

### 子路径部署 (/gantt)

```nginx
location /gantt {
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header REMOTE-HOST $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_pass http://127.0.0.1:9700/gantt;  # 注意末尾的 /gantt
}

# 或者使用 rewrite
location /gantt/ {
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_pass http://127.0.0.1:9700/;  # 注意末尾的 /
}
```

### 根路径部署 (/)

```nginx
location / {
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_pass http://127.0.0.1:9700;
}
```

## 验证部署

```bash
# 访问应用
curl http://your-domain/gantt/

# 检查静态资源路径
curl http://your-domain/gantt/ | grep -E 'src=|href='
# 应该看到 /gantt/assets/... 的路径
```

## 快速切换

已创建两个构建脚本：

### Windows
```cmd
REM 根路径部署
release\scripts\build-docker.bat

REM 子路径部署
docker build --build-arg DEPLOY_MODE=gantt -t gantt-app:gantt .
```

### Linux/Mac
```bash
# 根路径部署
./release/scripts/build-docker.sh

# 子路径部署
docker build --build-arg DEPLOY_MODE=gantt -t gantt-app:gantt .
```

## 环境配置文件

- `frontend/web/.env` - 开发环境（根路径）
- `frontend/web/.env.production` - 生产环境（根路径）
- `frontend/web/.env.gantt` - 子路径部署 (/gantt/)

## 常见问题

### Q: 为什么静态资源加载失败？
A: 确保使用了正确的 DEPLOY_MODE 构建镜像，或者检查 Nginx 代理配置是否正确。

### Q: 如何检查当前使用的 base 路径？
A: 访问首页，查看 HTML 源代码中 `<script>` 和 `<link>` 标签的路径。

### Q: API 请求404？
A: 检查外部 Nginx 的 API 代理配置，确保路径重写正确。

---

**推荐**: 根据部署需求选择合适的 DEPLOY_MODE
