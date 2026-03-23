@echo off
REM ========================================
REM Docker 镜像一键构建脚本 (Windows 版本)
REM ========================================
setlocal enabledelayedexpansion

REM 设置字符编码为 UTF-8
chcp 65001 >nul

echo ========================================
echo Gantt Application Docker Build Script
echo ========================================
echo.

REM 获取脚本所在目录
set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%..\..\"
set "RELEASE_DIR=%PROJECT_ROOT%release"
set "BIN_DIR=%RELEASE_DIR%\bin"
set "TMP_DIR=%RELEASE_DIR%\tmp"

REM 配置变量
set "IMAGE_NAME=gantt-app"
set "IMAGE_TAG=latest"
set "FULL_IMAGE_NAME=%IMAGE_NAME%:%IMAGE_TAG%"

REM 生成时间戳
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set datetime=%%I
set "TIMESTAMP=%datetime:~0,4%%datetime:~4,2%%datetime:~6,2%_%datetime:~8,2%%datetime:~10,2%%datetime:~12,2%"
set "OUTPUT_TAR=%BIN_DIR%\%IMAGE_NAME%_%TIMESTAMP%.tar"

echo 项目根目录: %PROJECT_ROOT%
echo 发布目录: %RELEASE_DIR%
echo 镜像名称: %FULL_IMAGE_NAME%
echo 输出文件: %OUTPUT_TAR%
echo.

REM 步骤 1: 清理旧的构建产物
echo [1/5] 清理旧的构建产物...

REM 确保目录存在
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"
if not exist "%TMP_DIR%" mkdir "%TMP_DIR%"

if exist "%TMP_DIR%" (
    del /q "%TMP_DIR%\*.*" 2>nul
    echo 临时目录已清理
)
echo 目录结构已准备

REM 步骤 2: 检查必要文件
echo [2/5] 检查必要文件...
set "ERROR=0"

if not exist "%PROJECT_ROOT%Dockerfile" (
    echo 错误: 缺少 Dockerfile
    set "ERROR=1"
)
if not exist "%PROJECT_ROOT%go.mod" (
    echo 错误: 缺少 go.mod
    set "ERROR=1"
)
if not exist "%PROJECT_ROOT%frontend\web\package.json" (
    echo 错误: 缺少 frontend/web/package.json
    set "ERROR=1"
)
if not exist "%PROJECT_ROOT%config\common.yml" (
    echo 错误: 缺少 config/common.yml
    set "ERROR=1"
)

if "%ERROR%"=="1" (
    echo.
    echo 构建失败：缺少必要文件
    exit /b 1
)
echo 必要文件检查通过
echo.

REM 步骤 3: 构建 Docker 镜像
echo [3/5] 构建 Docker 镜像...
echo 这可能需要几分钟时间...
echo.
echo 提示: 如果遇到网络问题，Dockerfile 已配置使用国内镜像源
echo   - NPM: registry.npmmirror.com
echo   - Go: goproxy.cn
echo.
cd /d "%PROJECT_ROOT%"

docker build -t "%FULL_IMAGE_NAME%" ^
    --build-arg BUILDTIME="%date% %time%" ^
    --build-arg VERSION="%IMAGE_TAG%" ^
    -f Dockerfile ^
    .

if errorlevel 1 (
    echo.
    echo Docker 镜像构建失败！
    exit /b 1
)
echo Docker 镜像构建成功！
echo.

REM 步骤 4: 保存镜像到文件
echo [4/5] 保存镜像到文件...
docker save -o "%OUTPUT_TAR%" "%FULL_IMAGE_NAME%"

if errorlevel 1 (
    echo 保存镜像失败！
    exit /b 1
)

REM 获取文件大小
for %%A in ("%OUTPUT_TAR%") do set "FILE_SIZE=%%~zA"
set /a "FILE_SIZE_MB=FILE_SIZE/1024/1024"
echo 镜像已保存: %OUTPUT_TAR% (大小: %FILE_SIZE_MB% MB)
echo.

REM 步骤 5: 生成部署文档
echo [5/5] 生成部署文档...
set "DEPLOY_DOC=%BIN_DIR%\DEPLOY_%TIMESTAMP%.md"

(
echo # Gantt Application Docker 部署说明
echo.
echo **构建时间**: %date% %time%
echo **镜像名称**: %FULL_IMAGE_NAME%
echo **镜像文件**: %IMAGE_NAME%_%TIMESTAMP%.tar
echo.
echo ## 快速部署
echo.
echo ### 1. 加载镜像
echo.
echo ```bash
echo # 加载镜像到 Docker
echo docker load -i %IMAGE_NAME%_%TIMESTAMP%.tar
echo ```
echo.
echo ### 2. 运行容器
echo.
echo ```bash
echo # 运行容器（基础模式）
echo docker run -d \
echo   --name gantt-app \
echo   -p 80:80 \
echo   -p 8080:8080 \
echo   -p 8081:8081 \
echo   -p 8082:8082 \
echo   %FULL_IMAGE_NAME%
echo.
echo # 运行容器（挂载配置文件）
echo docker run -d \
echo   --name gantt-app \
echo   -p 80:80 \
echo   -p 8080:8080 \
echo   -p 8081:8081 \
echo   -p 8082:8082 \
echo   -v $^(pwd^)/config:/app/config \
echo   %FULL_IMAGE_NAME%
echo ```
echo.
echo ### 3. 验证部署
echo.
echo ```bash
echo # 查看容器状态
echo docker ps ^| grep gantt-app
echo.
echo # 查看容器日志
echo docker logs gantt-app
echo.
echo # 健康检查
echo curl http://localhost/health
echo ```
echo.
echo ## 服务端口说明
echo.
echo ^| 端口 ^| 服务 ^| 说明 ^|
echo ^|------^|------^|------^|
echo ^| 80 ^| Nginx ^| 前端页面和反向代理 ^|
echo ^| 8080 ^| Management Service ^| 管理服务 API ^|
echo ^| 8081 ^| Rostering MCP Server ^| MCP 服务器 ^(WebSocket^) ^|
echo ^| 8082 ^| Rostering Agent ^| 智能体服务 ^(WebSocket^) ^|
echo.
echo ## 访问地址
echo.
echo - **前端页面**: http://localhost/
echo - **健康检查**: http://localhost/health
echo - **管理服务 API**: http://localhost/api/management/
echo - **MCP WebSocket**: ws://localhost/api/mcp/
echo - **智能体 WebSocket**: ws://localhost/api/agent/
echo.
echo ## 常用命令
echo.
echo ```bash
echo # 停止容器
echo docker stop gantt-app
echo.
echo # 启动容器
echo docker start gantt-app
echo.
echo # 重启容器
echo docker restart gantt-app
echo.
echo # 删除容器
echo docker rm -f gantt-app
echo.
echo # 查看实时日志
echo docker logs -f gantt-app
echo.
echo # 进入容器 Shell
echo docker exec -it gantt-app sh
echo ```
echo.
echo ## 环境要求
echo.
echo - Docker 20.10+
echo - 可用内存: 至少 2GB
echo - 可用磁盘: 至少 5GB
echo.
) > "%DEPLOY_DOC%"

echo 部署文档已生成: %DEPLOY_DOC%
echo.

REM 显示摘要
echo ========================================
echo 构建完成！
echo ========================================
echo.
echo 镜像名称: %FULL_IMAGE_NAME%
echo 镜像文件: %OUTPUT_TAR% (%FILE_SIZE_MB% MB)
echo 部署文档: %DEPLOY_DOC%
echo.
echo 快速启动命令:
echo   docker run -d --name gantt-app -p 80:80 -p 8080:8080 -p 8081:8081 -p 8082:8082 %FULL_IMAGE_NAME%
echo.
echo 或者先加载镜像:
echo   docker load -i %OUTPUT_TAR%
echo   docker run -d --name gantt-app -p 80:80 -p 8080:8080 -p 8081:8081 -p 8082:8082 %FULL_IMAGE_NAME%
echo.

endlocal
