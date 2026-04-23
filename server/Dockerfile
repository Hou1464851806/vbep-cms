ARG EXEC=vbep-cms-backend
# 第一阶段：构建Go可执行文件（基于Debian 12）
FROM golang:1.25-bookworm AS builder

ARG EXEC

WORKDIR /app

# 下载开发时使用的依赖（锁定依赖版本）
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# -tags timetzdata会在tzdata缺失时使用内嵌的时区数据
RUN CGO_ENABLED=0 go build -o /${EXEC} -tags timetzdata .

# 第二阶段：构建后端服务运行环境
FROM debian:bookworm-slim

ARG EXEC

ENV RUN_EXEC=${EXEC}

ENV TZ=Asia/Shanghai

WORKDIR /app

# 安装 curl 用于健康检查
RUN apt-get update && apt-get install -y --no-install-recommends curl procps ca-certificates && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /app/secrets

# 复制第一阶段得到的可执行文件
COPY --from=builder /${EXEC} .

# 复制后端服务的配置文件
COPY config.production.toml ./config.toml
COPY html ./html

EXPOSE 8080

# 运行 ./vbep-cms 可执行文件
CMD ./${RUN_EXEC}
