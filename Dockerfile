FROM node:25-alpine as build-stage

WORKDIR /app

COPY package*.json ./

RUN npm install --omit=dev

COPY . .

# 执行构建命令
# 注意：Vite build 默认输出到 dist 目录
# 如果你的项目需要设置基础URL，请确保在 vite.config.js 中配置好 base
# 例如：base: '/你的应用路径/'，如果部署在子路径下
RUN npm run build

FROM nginx:alpine as production-stage

# 删除 Nginx 默认的配置文件
RUN rm /etc/nginx/conf.d/default.conf

# 复制自定义的 Nginx 配置
# 这个配置将 Nginx 指向到前端的 build 目录
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 从构建阶段复制构建好的静态文件到 Nginx 的服务目录
# 注意：/app/dist 是 Vite 构建的默认输出目录
COPY --from=build-stage /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]