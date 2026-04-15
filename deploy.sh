#!/bin/bash

# 严格错误模式
set -e
set -u

echo "[*] 开始部署..."

# ==========================================
# 0. 环境检测 (Docker & Compose)
# ==========================================
echo "[*] 检测系统环境..."
# 检查 Docker 是否安装且正在运行
if ! command -v docker >/dev/null 2>&1; then
    echo "[X] 错误: 未找到 docker 命令，请先安装 Docker！"
    exit 1
fi
if ! docker info >/dev/null 2>&1; then
    echo "[X] 错误: Docker 进程未启动，或者当前用户没有执行 Docker 的权限！"
    exit 1
fi
# 检查 Docker Compose，并自动兼容新旧版本命令
COMPOSE_CMD=""
if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
elif docker-compose version >/dev/null 2>&1 || docker-compose --version >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
else
    echo "[X] 错误: 未检测到 docker compose 插件或 docker-compose 命令！"
    exit 1
fi
echo "[OK] 检测到宿主机环境就绪，使用的编排命令为: [$COMPOSE_CMD]"

# ==========================================
# 1. 生成敏感配置
# ==========================================
echo "[*] 检查并初始化数据库凭证..."
mkdir -p secrets

# 检查密码文件是否存在
if [ ! -f "secrets/db_password.txt" ]; then
    echo "[+] 初次部署：正在生成强随机密码..."
    # 生成 24 位随机字符去换行，写入文件
    openssl rand -base64 24 | tr -d '\n' > secrets/db_password.txt

    echo -n "vbep-cms_user" > secrets/db_user.txt

#    # 防止非root用户读取（暂时不需要）
#    chmod 600 secrets/db_password.txt
#    chmod 600 secrets/db_user.txt
else
    echo "[OK] 凭证文件已存在，直接使用现有凭证。"
fi

# ==========================================
# 2. 生成/输入 SMTP 配置
# ==========================================
echo "[*] 检查 SMTP 凭证..."

# 检查 SMTP 密码文件是否存在
if [ ! -f "secrets/smtp_password.txt" ]; then
    echo "[+] 初次部署：需要配置 SMTP 服务。"

    # 初始化变量
    smtp_pass=""

    # 使用 while 循环，防止用户输入空值
    while [ -z "$smtp_pass" ]; do
        # -s 参数隐藏输入内容，-p 参数显示提示文本
        read -s -p "[-] 请输入 SMTP 客户端授权码 (输入时内容不可见，完成后按回车): " smtp_pass
        echo "" # 补充一个换行符，因为 read -s 吃掉了回车

        # 验证是否为空
        if [ -z "$smtp_pass" ]; then
            echo "[!] 错误：授权码不能为空，请重新输入！"
        fi
    done

    # 将输入的授权码写入文件（-n 防止写入多余的换行符）
    echo -n "$smtp_pass" > secrets/smtp_password.txt
    echo "[+] SMTP 授权码已保存到 secrets/smtp_password.txt"
else
    echo "[OK] SMTP 凭证文件已存在，直接使用现有凭证。"
fi

# ==========================================
# 2. 批量导入 Docker 镜像
# ==========================================
echo "[*] 准备加载本地 Docker 镜像..."
if [ -d "images" ]; then
    # 寻找 images 目录下所有的 .tar 或 .tar.gz 文件
    for img in images/*.tar*; do
        if [ -f "$img" ]; then
            echo "[+] 正在导入镜像文件: $img"
            docker load -i "$img"
        fi
    done
    echo "[OK] 本地镜像导入完成。"
else
    echo "[-] 警告: 未找到 images 目录，假设系统已经具备所需镜像。"
fi

# ==========================================
# 3. 编排并启动服务
# ==========================================
echo "[*] 启动 Docker Compose 服务..."
$COMPOSE_CMD up -d

# ==========================================
# 4. 后续清理和状态确认
# ==========================================
echo "[*] 清理长期无用的旧版遗留镜像释放空间..."
# 安全清理：只会删除那些没有名字且没有被任何当前存活容器使用的“悬空”镜像
docker image prune -f

echo "----------------------------------------------------"
echo "[*] 当前各服务运行状态："
$COMPOSE_CMD ps
HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
HOST_IP=${HOST_IP:-127.0.0.1}
FRONTEND_PORT=$($COMPOSE_CMD port frontend 80 2>/dev/null | cut -d ':' -f2)
echo "----------------------------------------------------"
echo "[OK] 部署/更新全部完成。"
if [ -n "$FRONTEND_PORT" ]; then
    echo ">>> 系统的访问入口为：http://${HOST_IP}:${FRONTEND_PORT}"
else
    echo ">>> 系统已启动！(提示：请在 compose.yaml 中确认前端对外暴露的端口进行访问)"
fi
echo "----------------------------------------------------"
