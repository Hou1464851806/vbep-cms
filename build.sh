#!/bin/bash

set -e

# ==========================================
# 1. 变量及默认配置
# ==========================================
IMAGE_DIR="./images"
TARGET_PLATFORM="linux/arm64" # 默认平台

FRONTEND_IMAGE="vbep-cms/frontend:latest"
BACKEND_IMAGE="vbep-cms/backend:latest"
POSTGRES_IMAGE="postgres:15-alpine"
REDIS_IMAGE="redis:7-alpine"

FRONTEND_DIR="."
BACKEND_DIR="./server"

# 构建开关控制（默认全部构建）
BUILD_ALL=true
BUILD_FRONTEND=false
BUILD_BACKEND=false
BUILD_DEPS=false

# ==========================================
# 2. 解析命令参数
# ==========================================
# 示例: ./build.sh -p linux/amd64 --backend
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -p|--platform) TARGET_PLATFORM="$2"; shift ;;
        --frontend) BUILD_ALL=false; BUILD_FRONTEND=true ;;
        --backend)  BUILD_ALL=false; BUILD_BACKEND=true  ;;
        --deps)     BUILD_ALL=false; BUILD_DEPS=true     ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  -p, --platform   指定构建平台 (默认: linux/arm64)"
            echo "  --frontend       只构建前端镜像"
            echo "  --backend        只构建后端镜像"
            echo "  --deps           只拉取依赖镜像(postgres, redis)"
            echo "  (不添加具体模块参数时，默认构建全部镜像)"
            exit 0
            ;;
        *) echo "[!] 未知参数: $1"; exit 1 ;;
    esac
    shift
done

# 如果没有指定任何独立模块，则全局开启
if [ "$BUILD_ALL" = true ]; then
    BUILD_FRONTEND=true
    BUILD_BACKEND=true
    BUILD_DEPS=true
fi

echo "==========================================="
echo "[*] 开始执行离线安装包构建流程..."
echo "[*] 目标平台: $TARGET_PLATFORM"
echo "[*] 正在检查输出目录: $IMAGE_DIR"
if [ ! -d "$IMAGE_DIR" ]; then
    mkdir -p "$IMAGE_DIR"
fi
echo "==========================================="

# ==========================================
# 3. 环境准备 (初始化 buildx)
# ==========================================
# 只有在需要构建源码镜像时才需要 buildx
if [ "$BUILD_FRONTEND" = true ] || [ "$BUILD_BACKEND" = true ]; then
    BUILDER_NAME="mybuilder"
    if ! docker buildx inspect $BUILDER_NAME > /dev/null 2>&1; then
        echo "[*] 正在创建并启动 buildx 构建器..."
        docker buildx create --name $BUILDER_NAME --use
        docker buildx inspect --bootstrap
    else
        docker buildx use $BUILDER_NAME
    fi
fi

# ==========================================
# 4. 源码构建并导出
# ==========================================
if [ "$BUILD_FRONTEND" = true ]; then
    echo "-------------------------------------------"
    echo "[*] 正在构建前端镜像并导出: $FRONTEND_IMAGE"
    rm -f "$IMAGE_DIR/frontend.tar" # 只删自己的包，防止使用旧的
    docker buildx build --platform "$TARGET_PLATFORM" \
        -t "$FRONTEND_IMAGE" \
        --output type=docker,dest="$IMAGE_DIR/frontend.tar" \
        "$FRONTEND_DIR"
fi

if [ "$BUILD_BACKEND" = true ]; then
    echo "-------------------------------------------"
    echo "[*] 正在构建后端镜像并导出: $BACKEND_IMAGE"
    rm -f "$IMAGE_DIR/backend.tar" # 只删自己的包
    docker buildx build --platform "$TARGET_PLATFORM" \
        -t "$BACKEND_IMAGE" \
        --output type=docker,dest="$IMAGE_DIR/backend.tar" \
        "$BACKEND_DIR"
fi

# ==========================================
# 5. 拉取/转换依赖镜像
# ==========================================
if [ "$BUILD_DEPS" = true ]; then
    echo "-------------------------------------------"
    echo "[*] 正在拉取依赖镜像 ($TARGET_PLATFORM)..."

    echo "[+] 处理数据库: postgres.tar"
    rm -f "$IMAGE_DIR/postgres.tar"
    docker pull --platform "$TARGET_PLATFORM" "$POSTGRES_IMAGE"
    docker save -o "$IMAGE_DIR/postgres.tar" "$POSTGRES_IMAGE"

    echo "[+] 处理缓存: redis.tar"
    rm -f "$IMAGE_DIR/redis.tar"
    docker pull --platform "$TARGET_PLATFORM" "$REDIS_IMAGE"
    docker save -o "$IMAGE_DIR/redis.tar" "$REDIS_IMAGE"
fi

# ==========================================
# 6. 结果确认
# ==========================================
echo "-------------------------------------------"
echo "[OK] 针对平台 [$TARGET_PLATFORM] 的选定镜像打包完成！"
ls -lh "$IMAGE_DIR" | awk '{print $5, $9}' | grep "\.tar" || true

echo "-------------------------------------------"
echo "[提示] 如果在目标机器 load 后无法运行，请检查目标机器架构是否为 $TARGET_PLATFORM"