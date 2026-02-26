#!/bin/bash

set -e

# ==========================================
# 1. 变量配置
# ==========================================
IMAGE_DIR="./images"

FRONTEND_IMAGE="vbep-cms/frontend:latest"
BACKEND_IMAGE="vbep-cms/backend:latest"

POSTGRES_IMAGE="postgres:15-alpine"
REDIS_IMAGE="redis:7-alpine"

FRONTEND_DIR="."
BACKEND_DIR="./server"

echo "==========================================="
echo "[*] 开始执行离线安装包构建流程..."
echo "==========================================="

# ==========================================
# 2. 环境准备
# ==========================================
if [ ! -d "$IMAGE_DIR" ]; then
    echo "[*] 创建镜像导出目录: $IMAGE_DIR"
    mkdir -p "$IMAGE_DIR"
else
    echo "[*] 清理旧的镜像包..."
    rm -f "$IMAGE_DIR"/*.tar
fi

# ==========================================
# 3. 源码构建
# ==========================================
echo "-------------------------------------------"
echo "[*] 正在构建前端镜像: $FRONTEND_IMAGE"
docker build -t "$FRONTEND_IMAGE" "$FRONTEND_DIR"

echo "-------------------------------------------"
echo "[*] 正在构建后端镜像: $BACKEND_IMAGE"
docker build -t "$BACKEND_IMAGE" "$BACKEND_DIR"

# ==========================================
# 4. 拉取依赖镜像
# ==========================================
echo "-------------------------------------------"
echo "[*] 正在拉取基础依赖镜像 (如果本地已有会直接跳过)..."
docker pull "$POSTGRES_IMAGE"
docker pull "$REDIS_IMAGE"

# ==========================================
# 5. 导出镜像至离线包
# ==========================================
echo "-------------------------------------------"
echo "[*] 开始导出镜像到 $IMAGE_DIR 目录 (这可能需要几分钟时间)..."

echo "[+] 导出前端: frontend.tar"
docker save -o "$IMAGE_DIR/frontend.tar" "$FRONTEND_IMAGE"

echo "[+] 导出后端: backend.tar"
docker save -o "$IMAGE_DIR/backend.tar" "$BACKEND_IMAGE"

echo "[+] 导出数据库: postgres.tar"
docker save -o "$IMAGE_DIR/postgres.tar" "$POSTGRES_IMAGE"

echo "[+] 导出缓存: redis.tar"
docker save -o "$IMAGE_DIR/redis.tar" "$REDIS_IMAGE"

# ==========================================
# 6. 结果确认
# ==========================================
echo "-------------------------------------------"
echo "[OK] 所有镜像打包完成！生成文件如下："
ls -lh "$IMAGE_DIR" | awk '{print $5, $9}' | grep "\.tar"

echo "-------------------------------------------"
echo "[OK] 构建流程结束。现在您可以将整个项目目录打包发送至服务器运行 deploy.sh 了。"