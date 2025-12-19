# =========================================================================
# Stage 1: Builder - 编译 Go 应用
# =========================================================================
FROM golang:1.25-alpine AS builder

# 设置工作目录
WORKDIR /app

# 缓存依赖
COPY go.mod go.sum ./
RUN go mod download

# 拷贝所有源代码
COPY . .

# 编译应用，生成静态链接的二进制文件
RUN CGO_ENABLED=0 go build -o /vbep-cms -tags timetzdata .

# =========================================================================
# Stage 2: Runner - 构建最终的生产镜像
# =========================================================================
FROM alpine:latest

# 为应用文件设置工作目录
WORKDIR /app

# 从 'builder' 阶段拷贝编译好的二进制文件到当前目录 (/app)
COPY --from=builder /vbep-cms .

# 拷贝配置文件到当前目录 (/app)
COPY config.toml .

# 暴露应用监听的端口
EXPOSE 8080

# 容器启动时运行的命令 (在 /app 目录下执行 ./vbep-cms)
CMD ["./vbep-cms"]
