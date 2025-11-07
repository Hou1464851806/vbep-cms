#!/bin/bash
set -e # 任何命令失败时立即退出脚本
set -u # 使用未定义变量时报错

# --- 配置区 ---
APP_NAME="vbep-cms"
CONTAINER_NAME="${APP_NAME}-frontend"
DOCKER_COMPOSE_FILE="docker-compose.yml"
# --- END 配置区 ---

echo "--- 开始部署 ${APP_NAME} ---"

# --- 检查 Docker Compose 命令 ---
COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    # 存在旧版独立的 docker-compose 命令
    COMPOSE_CMD="docker-compose"
    echo "检测到使用 'docker-compose' (带连字符) 命令."
elif command -v docker &> /dev/null && docker compose version &> /dev/null; then
    # 存在新版集成到 docker CLI 的 docker compose 命令
    COMPOSE_CMD="docker compose"
    echo "检测到使用 'docker compose' (无连字符) 命令."
else
    echo "错误：未找到 'docker-compose' 或 'docker compose' 命令。"
    echo "请确保 Docker Compose 已安装并已添加到 PATH 中。"
    exit 1
fi
# --- Docker Compose 命令检查结束 ---

# 1. 停止并删除旧的容器 (如果存在)
echo "停止并删除旧的容器 ${CONTAINER_NAME}..."
# 使用检测到的 COMPOSE_CMD
${COMPOSE_CMD} -f "${DOCKER_COMPOSE_FILE}" stop frontend || true # 即使停止失败也继续
${COMPOSE_CMD} -f "${DOCKER_COMPOSE_FILE}" rm -f frontend || true # 即使删除失败也继续

# 2. 删除旧镜像 (可选，但推荐清理空间)
echo "删除旧的 Docker 镜像 ${APP_NAME}/frontend:latest (如果存在)..."
# 使用 '-f' 参数强制删除，'--no-prune' 阻止删除 dangling 镜像
docker rmi -f "${APP_NAME}/frontend:latest" > /dev/null 2>&1 || true

# 3. 构建新的 Docker 镜像
echo "构建新的 Docker 镜像 ${APP_NAME}/frontend:latest..."
# 使用检测到的 COMPOSE_CMD
${COMPOSE_CMD} -f "${DOCKER_COMPOSE_FILE}" build frontend

# 检查构建是否成功 (set -e 会自动处理，但显式检查更清晰)
if [ $? -ne 0 ]; then
  echo "--- Docker 镜像构建失败！请检查日志。---"
  exit 1
fi

# 4. 启动新的容器
echo "启动新的容器 ${CONTAINER_NAME}..."
# 使用检测到的 COMPOSE_CMD
${COMPOSE_CMD} -f "${DOCKER_COMPOSE_FILE}" up -d frontend

# 检查容器启动是否成功
if [ $? -ne 0 ]; then
  echo "--- 容器启动失败！请检查日志。---"
  exit 1
fi

# 5. 显示容器状态
echo "--- 部署完成！容器状态如下 ---"
docker ps -a | grep "${CONTAINER_NAME}" # 使用双引号防止变量中含有空格或其他特殊字符

echo "--- 你现在应该可以通过 http://localhost:8080 或你的服务器 IP 访问你的应用了 ---"

