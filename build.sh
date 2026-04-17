#!/bin/bash

# 严格错误模式
set -e

# ==========================================
# 1. 变量及路径配置
# ==========================================
BUILD_DIR="./deploy"             # 最终生成的安装包目录
IMAGE_DIR="$BUILD_DIR/images"  # 镜像存放子目录
TARGET_PLATFORM="linux/arm64"  # 默认平台

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
            echo "  --deps           只打包依赖镜像"
            exit 0
            ;;
        *) echo "[!] 未知参数: $1"; exit 1 ;;
    esac
    shift
done

if [ "$BUILD_ALL" = true ]; then
    BUILD_FRONTEND=true; BUILD_BACKEND=true; BUILD_DEPS=true
fi

# ==========================================
# 3. 初始化目录结构
# ==========================================
echo "==========================================="
echo "[*] 开始执行离线安装包构建流程..."
echo "[*] 目标平台: $TARGET_PLATFORM"
echo "[*] 正在初始化构建目录: $BUILD_DIR"
# 如果没有 build，创建它；如果已经有，保留它（避免每次全量删掉已有的其他 tar）
mkdir -p "$IMAGE_DIR"
echo "==========================================="

# ==========================================
# 4. 环境准备 (Buildx)
# ==========================================
if [ "$BUILD_FRONTEND" = true ] || [ "$BUILD_BACKEND" = true ]; then
    BUILDER_NAME="mybuilder"
    if ! docker buildx inspect $BUILDER_NAME > /dev/null 2>&1; then
        docker buildx create --name $BUILDER_NAME --use
        docker buildx inspect --bootstrap
    else
        docker buildx use $BUILDER_NAME
    fi
fi

# ==========================================
# 5. 源码构建并导出到 build/images
# ==========================================
if [ "$BUILD_FRONTEND" = true ]; then
    echo "[*] 正在构建前端镜像 -> $IMAGE_DIR/frontend.tar"
    rm -f "$IMAGE_DIR/frontend.tar"
    docker buildx build --platform "$TARGET_PLATFORM" \
        -t "$FRONTEND_IMAGE" \
        --output type=docker,dest="$IMAGE_DIR/frontend.tar" \
        "$FRONTEND_DIR"
fi

if [ "$BUILD_BACKEND" = true ]; then
    echo "[*] 正在构建后端镜像 -> $IMAGE_DIR/backend.tar"
    rm -f "$IMAGE_DIR/backend.tar"
    docker buildx build --platform "$TARGET_PLATFORM" \
        -t "$BACKEND_IMAGE" \
        --output type=docker,dest="$IMAGE_DIR/backend.tar" \
        "$BACKEND_DIR"
fi

# ==========================================
# 6. 拉取依赖并导出到 build/images
# ==========================================
if [ "$BUILD_DEPS" = true ]; then
    echo "-------------------------------------------"
    echo "[*] 正在拉取依赖镜像 ($TARGET_PLATFORM)..."

    echo "[+] 处理数据库: postgres.tar"
    rm -f "$IMAGE_DIR/$POSTGRES_IMAGE.tar"
    docker pull --platform "$TARGET_PLATFORM" "$POSTGRES_IMAGE"
    docker save -o "$IMAGE_DIR/$POSTGRES_IMAGE.tar" "$POSTGRES_IMAGE"

    echo "[+] 处理缓存: redis.tar"
    rm -f "$IMAGE_DIR/$REDIS_IMAGE.tar"
    docker pull --platform "$TARGET_PLATFORM" "$REDIS_IMAGE"
    docker save -o "$IMAGE_DIR/$REDIS_IMAGE.tar" "$REDIS_IMAGE"
fi

# ==========================================
# 7. 拷贝部署脚本和配置文件到 build
# ==========================================
echo "-------------------------------------------"
echo "[*] 正在同步部署文件到 $BUILD_DIR ..."

# 检查源文件是否存在
for file in "deploy.sh" "compose.yaml"; do
    if [ -f "$file" ]; then
        cp "$file" "$BUILD_DIR/"
        echo "[OK] 已拷贝 $file"
    else
        echo "[!] 警告: 未找到 $file，请确保它在当前目录下！"
    fi
done

# 给出的 deploy.sh 增加执行权限
chmod +x "$BUILD_DIR/deploy.sh"

# ==========================================
# 8. 归档确认
# ==========================================
echo "-------------------------------------------"
echo "[完成] 安装包已准备就绪，存放在: $BUILD_DIR"
echo "[查看结果]:"
ls -R "$BUILD_DIR"
echo "-------------------------------------------"
echo "提示：您可以现在执行 'tar -czvf release.tar.gz $BUILD_DIR/' 进行打包发送。"