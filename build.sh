#!/bin/bash

set -e

# ==========================================
# 1. 变量及平台配置
# ==========================================
IMAGE_DIR="./images"
# 默认构建平台（如果运行脚本时不传参数，则使用当前机器架构）
# 常用值: linux/amd64, linux/arm64, linux/amd64,linux/arm64
TARGET_PLATFORM=${1:-"linux/arm64"}

FRONTEND_IMAGE="vbep-cms/frontend:latest"
BACKEND_IMAGE="vbep-cms/backend:latest"

POSTGRES_IMAGE="postgres:15-alpine"
REDIS_IMAGE="redis:7-alpine"

FRONTEND_DIR="."
BACKEND_DIR="./server"

echo "==========================================="
echo "[*] 开始执行离线安装包构建流程..."
echo "[*] 目标平台: $TARGET_PLATFORM"
echo "==========================================="

# ==========================================
# 2. 环境准备 (初始化 buildx)
# ==========================================
if [ ! -d "$IMAGE_DIR" ]; then
    mkdir -p "$IMAGE_DIR"
else
    rm -f "$IMAGE_DIR"/*.tar
fi

# 检查/创建 buildx 构建器 (若要跨平台构建，必须使用 docker-container 驱动)
BUILDER_NAME="mybuilder"
if ! docker buildx inspect $BUILDER_NAME > /dev/null 2>&1; then
    echo "[*] 正在创建并启动 buildx 构建器..."
    docker buildx create --name $BUILDER_NAME --use
    docker buildx inspect --bootstrap
else
    docker buildx use $BUILDER_NAME
fi

# ==========================================
# 3. 源码构建并导出
# ==========================================
# 注意：buildx 构建跨平台镜像并存到本地 .tar 文件，
# 不能直接 build -t 然后 save，必须在 build 时通过 --output 指定导出
# 且目前 buildx 导出 .tar 文件时一次只能为一个 platform 构建。

echo "-------------------------------------------"
echo "[*] 正在构建前端镜像并导出: $FRONTEND_IMAGE"
docker buildx build --platform "$TARGET_PLATFORM" \
    -t "$FRONTEND_IMAGE" \
    --output type=docker,dest="$IMAGE_DIR/frontend.tar" \
    "$FRONTEND_DIR"

echo "-------------------------------------------"
echo "[*] 正在构建后端镜像并导出: $BACKEND_IMAGE"
docker buildx build --platform "$TARGET_PLATFORM" \
    -t "$BACKEND_IMAGE" \
    --output type=docker,dest="$IMAGE_DIR/backend.tar" \
    "$BACKEND_DIR"

# ==========================================
# 4. 拉取/转换依赖镜像
# ==========================================
# 依赖镜像（如 Postgres）需要拉取对应平台的版本并保存
echo "-------------------------------------------"
echo "[*] 正在拉取依赖镜像 ($TARGET_PLATFORM)..."

echo "[+] 处理数据库: postgres.tar"
docker pull --platform "$TARGET_PLATFORM" "$POSTGRES_IMAGE"
docker save -o "$IMAGE_DIR/postgres.tar" "$POSTGRES_IMAGE"

echo "[+] 处理缓存: redis.tar"
docker pull --platform "$TARGET_PLATFORM" "$REDIS_IMAGE"
docker save -o "$IMAGE_DIR/redis.tar" "$REDIS_IMAGE"

# ==========================================
# 5. 结果确认
# ==========================================
echo "-------------------------------------------"
echo "[OK] 所有镜像针对平台 [$TARGET_PLATFORM] 打包完成！"
ls -lh "$IMAGE_DIR" | awk '{print $5, $9}' | grep "\.tar"

echo "-------------------------------------------"
echo "[提示] 如果在目标机器 load 后无法运行，请检查目标机器架构是否为 $TARGET_PLATFORM"