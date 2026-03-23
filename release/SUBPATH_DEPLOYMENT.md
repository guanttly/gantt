# 子路径部署配置指南

## 问题描述

当通过反向代理将应用部署到子路径（如 `/gantt`）时，前端静态资源加载失败，报错：
```
Failed to load module script: Expected a JavaScript module script 
but the server responded with a MIME type of "text/html"
```

## 根本原因

前端应用构建时使用的是根路径 `/`，生成的 HTML 中引用资源路径为：
- `/assets/index-xxx.js`
- `/assets/index-xxx.css`

但部署在 `/gantt` 路径下时，浏览器会请求：
- `http://domain.com/assets/index-xxx.js` ❌ (不存在)

而实际资源在：
- `http://domain.com/gantt/assets/index-xxx.js` ✅

## 解决方案

### 方案 1：使用环境变量配置 base 路径（推荐）

#### 1. 修改 `vite.config.ts`

```typescript
export default defineConfig(({ mode }) => {
  // 从环境变量读取 base 路径，默认为 '/'
  const base = process.env.VITE_BASE_PATH || '/'
  
  return {
    base, // 添加 base 配置
    resolve: {
      alias: {
        '@/': `${path.resolve(__dirname, 'src')}/`,
      },
    },
    // ... 其他配置
  }
})
```

#### 2. 创建 `.env.gantt` 文件

在 `frontend/web/` 目录下创建：

```env
# 子路径部署配置
VITE_BASE_PATH=/gantt/
```

#### 3. 修改 Dockerfile

```dockerfile
# 前端构建阶段 - 支持子路径部署
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# ... 安装依赖 ...

# 复制所有前端源代码和环境配置
COPY frontend/web/ ./

# 设置环境变量并构建（使用子路径）
ENV VITE_BASE_PATH=/gantt/
RUN pnpm run build

# 或者使用 .env.gantt
# RUN pnpm run build --mode gantt
```

#### 4. 修改 Nginx 配置

`release/docker/default.conf`:

```nginx
server {
    listen 80;
    server_name _;

    # 前端静态文件 - 子路径部署
    location /gantt/ {
        alias /usr/share/nginx/html/;
        try_files $uri $uri/ /gantt/index.html;
        index index.html;
        
        # 静态资源缓存
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }

    # API 代理（注意路径）
    location /gantt/api/management/ {
        rewrite ^/gantt/api/management/(.*) /$1 break;
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # ... 其他 API 代理配置
}
```

### 方案 2：直接修改 Vite 配置（简单但不灵活）

#### 修改 `frontend/web/vite.config.ts`

```typescript
export default defineConfig(() => {
  return {
    base: '/gantt/', // 固定为 /gantt/ 路径
    // ... 其他配置
  }
})
```

然后重新构建镜像。

### 方案 3：使用 Nginx rewrite（不推荐，复杂且容易出错）

通过 Nginx 重写规则处理，但需要处理很多边界情况，不推荐。

## 推荐实践

### 多环境构建支持

创建不同的环境配置文件：

```
frontend/web/
├── .env                    # 开发环境（默认）
├── .env.production        # 生产环境（根路径）
├── .env.gantt            # 子路径部署
└── .env.k8s              # K8s 部署
```

#### `.env.production` (根路径部署)
```env
VITE_BASE_PATH=/
VITE_API_BASE_URL=/api
```

#### `.env.gantt` (子路径部署)
```env
VITE_BASE_PATH=/gantt/
VITE_API_BASE_URL=/gantt/api
```

#### 构建命令

```bash
# 根路径部署
pnpm run build

# 子路径部署
pnpm run build --mode gantt
```

#### Dockerfile 支持多环境

```dockerfile
# 使用构建参数
ARG DEPLOY_MODE=production

# 复制所有环境配置
COPY frontend/web/ ./

# 根据部署模式构建
RUN if [ "$DEPLOY_MODE" = "gantt" ]; then \
      pnpm run build --mode gantt; \
    else \
      pnpm run build; \
    fi
```

构建时指定：
```bash
# 子路径部署
docker build --build-arg DEPLOY_MODE=gantt -t gantt-app:gantt .

# 根路径部署（默认）
docker build -t gantt-app:latest .
```

## 快速修复（当前镜像）

如果镜像已经构建完成，想快速修复：

### 方案：使用子域名而非子路径

不使用 `domain.com/gantt`，改用：
- `gantt.domain.com` 或
- 独立域名

这样无需修改前端构建配置，直接使用当前镜像即可。

## 验证部署

```bash
# 检查 index.html 中的资源路径
curl http://your-domain/gantt/ | grep -E 'src=|href='

# 应该看到：
# <script type="module" src="/gantt/assets/index-xxx.js">
# <link rel="stylesheet" href="/gantt/assets/index-xxx.css">
```

## 相关文件

- `frontend/web/vite.config.ts` - Vite 配置
- `release/docker/default.conf` - Nginx 配置
- `Dockerfile` - 构建配置
- `frontend/web/.env.*` - 环境变量配置

---

**推荐方案**: 使用方案 1（环境变量配置），灵活且可维护。
