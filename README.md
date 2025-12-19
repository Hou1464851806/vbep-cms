# vbep-cms

## 拉取项目

```
git clone https://github.com/Hou1464851806/vbep-cms.git
```

## 本地调试
### 安装依赖
```
npm install
```
### 开启本地调试服务器
```
npm run dev
```

## 部署
1. ~~安装docker (docker-compose)~~
2. ~~执行`deploy.sh`一键部署~~

### 前提
确保部署环境联网，安装了docker和docker-compose，并能访问dockerhub

### 设置环境变量
需要通过环境变量配置数据库的用户和密码
1. 使用.env文件
2. 直接在终端中设置DB_USER DB_PASSWORD
```text
#.env
DB_USER=MYUSER
DB_PASSWORD=MYPASSWORD
```

### 构建docker镜像
``docker compose build`` 构建前后端镜像

``docker compose build frontend`` 单独构建前端

``docker compose build backend `` 单独构建后端

### docker容器管理
``docker compose up -d`` 启动前后端服务，数据库及redis服务\
docker会自动拉取并建立postgres和redis容器，自动创建服务之间的网络，自动创建持久化的硬盘挂载

``docker compose stop`` 停止前后端服务

``docker compose down`` 停止并删除各服务

``docker compose down -v`` 停止并删除各服务，同时删除持久化数据

``docker compose logs <service>`` 查看某服务的日志

``docker ps`` 查看容器运行状态