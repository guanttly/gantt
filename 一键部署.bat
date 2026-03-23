@echo off
REM ==============================================
REM Gantt Application - 一键部署脚本 (Windows)
REM ==============================================
REM 此脚本会自动完成：构建 -> 测试 -> 启动
REM ==============================================

setlocal enabledelayedexpansion
chcp 65001 >nul

echo.
echo ╔════════════════════════════════════════════╗
echo ║  Gantt Application 一键部署脚本           ║
echo ╚════════════════════════════════════════════╝
echo.

set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

REM 步骤 1: 构建 Docker 镜像
echo [步骤 1/3] 构建 Docker 镜像...
echo.
call release\scripts\build-docker.bat
if errorlevel 1 (
    echo.
    echo ❌ 构建失败！
    pause
    exit /b 1
)

echo.
echo ✅ 构建成功！
echo.
timeout /t 3 /nobreak >nul

REM 步骤 2: 询问是否进行测试
echo [步骤 2/3] 是否进行镜像测试？
set /p "DO_TEST=是否测试镜像? (y/n, 默认 n): "
if /i "%DO_TEST%"=="y" (
    echo.
    echo 开始测试...
    call release\scripts\test-docker.bat
    if errorlevel 1 (
        echo.
        echo ⚠ 测试失败，但可以继续部署
    ) else (
        echo.
        echo ✅ 测试通过！
    )
    echo.
    timeout /t 3 /nobreak >nul
) else (
    echo 跳过测试
)

REM 步骤 3: 启动容器
echo.
echo [步骤 3/3] 启动容器...
echo.

REM 检查是否已有运行的容器
docker ps -a | findstr "gantt-app" >nul
if not errorlevel 1 (
    echo 检测到已存在的容器，是否删除？
    set /p "REMOVE_OLD=删除旧容器? (y/n, 默认 y): "
    if /i not "!REMOVE_OLD!"=="n" (
        echo 删除旧容器...
        docker rm -f gantt-app 2>nul
    )
)

echo 启动新容器...
docker run -d ^
  --name gantt-app ^
  --restart unless-stopped ^
  -p 80:80 ^
  -p 8080:8080 ^
  -p 8081:8081 ^
  -p 8082:8082 ^
  gantt-app:latest

if errorlevel 1 (
    echo.
    echo ❌ 容器启动失败！
    pause
    exit /b 1
)

REM 等待服务启动
echo.
echo 等待服务启动 (15秒)...
timeout /t 15 /nobreak >nul

REM 健康检查
echo.
echo 检查服务状态...
curl -sf http://localhost/health >nul
if errorlevel 1 (
    echo ⚠ 健康检查失败，服务可能还在启动中
    echo   请稍后访问: http://localhost
) else (
    echo ✅ 服务已就绪！
)

REM 显示成功信息
echo.
echo ╔════════════════════════════════════════════╗
echo ║           🎉 部署成功！                    ║
echo ╚════════════════════════════════════════════╝
echo.
echo 📱 访问地址:
echo    前端页面: http://localhost
echo    健康检查: http://localhost/health
echo.
echo 📊 容器状态:
docker ps --filter name=gantt-app --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo.
echo 📝 常用命令:
echo    查看日志: docker logs -f gantt-app
echo    停止服务: docker stop gantt-app
echo    重启服务: docker restart gantt-app
echo    删除容器: docker rm -f gantt-app
echo.

pause
