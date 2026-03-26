@echo off
REM Docker 镜像测试脚本 (Windows)
REM 用于验证构建的镜像是否正常工作

setlocal enabledelayedexpansion

chcp 65001 >nul

set "IMAGE_NAME=gantt-app:latest"
set "CONTAINER_NAME=gantt-app-test"
set "TEST_PORT=8888"

echo ==========================================
echo Docker Image Test Script
echo ==========================================
echo.

REM 检查镜像是否存在
echo [1/5] 检查镜像...
docker images | findstr "gantt-app" >nul
if errorlevel 1 (
    echo 错误: 镜像 %IMAGE_NAME% 不存在
    echo 请先运行构建脚本: release\scripts\build-docker.bat
    exit /b 1
)
echo √ 镜像存在

REM 清理可能存在的测试容器
echo.
echo [2/5] 清理旧的测试容器...
docker rm -f %CONTAINER_NAME% 2>nul
echo √ 清理完成

REM 启动测试容器
echo.
echo [3/5] 启动测试容器...
docker run -d ^
  --name %CONTAINER_NAME% ^
  -p %TEST_PORT%:80 ^
  %IMAGE_NAME%

if errorlevel 1 (
    echo 启动失败
    exit /b 1
)
echo √ 容器已启动

REM 等待服务启动
echo.
echo [4/5] 等待服务启动 (30秒)...
timeout /t 30 /nobreak >nul

REM 测试服务
echo.
echo [5/5] 测试服务...

REM 测试健康检查端点
echo   - 健康检查端点: 
curl -sf http://localhost:%TEST_PORT%/health >nul
if errorlevel 1 (
    echo     × 失败
    docker logs %CONTAINER_NAME%
    docker rm -f %CONTAINER_NAME%
    exit /b 1
) else (
    echo     √ 成功
)

REM 测试前端页面
echo   - 前端页面: 
curl -sf http://localhost:%TEST_PORT%/ >nul
if errorlevel 1 (
    echo     × 失败
) else (
    echo     √ 成功
)

REM 检查各服务进程
echo   - Gantt Server: 
docker exec %CONTAINER_NAME% ps aux | findstr "gantt-server" >nul
if errorlevel 1 (
    echo     × 未运行
) else (
    echo     √ 运行中
)

echo   - Nginx: 
docker exec %CONTAINER_NAME% ps aux | findstr "nginx" >nul
if errorlevel 1 (
    echo     × 未运行
) else (
    echo     √ 运行中
)

REM 显示测试结果
echo.
echo ==========================================
echo 测试完成！
echo ==========================================
echo.
echo 容器信息:
docker ps --filter name=%CONTAINER_NAME%
echo.
echo 访问地址: http://localhost:%TEST_PORT%
echo.
echo 停止测试容器: docker stop %CONTAINER_NAME%
echo 删除测试容器: docker rm -f %CONTAINER_NAME%
echo.

endlocal
