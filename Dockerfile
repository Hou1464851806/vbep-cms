# 第一阶段：Vite构建前端文件（基于alpine）
FROM node:25-alpine AS builder

WORKDIR /app

# 下载开发时的依赖（锁定依赖）
COPY package*.json ./
RUN npm ci

COPY . .

RUN npm run build
# 第二阶段：构建前端服务运行环境（通过nginx托管）
FROM nginx:stable-alpine

# 使用新的nginx配置文件
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 将构建的产物放到nginx将要托管的网页目录
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80