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

### docker compose 约定

各服务在docker compose中的名称
- 前端: frontend
- 后端: backend

### 前提
确保部署环境联网，安装了docker和docker-compose，并能访问dockerhub

或者

不联网的情况下，提前准备好前端、后端、postgres和redis的镜像文件，使用``docker image load -i``导入镜像。

确保镜像名为
- 前端: vbep-cms/frontend:latest
- 后端: vbep-cms/backend:latest
- postgres: postgres:15-alpine
- redis: redis:7-alpine


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

#### 交叉编译
比如，构建镜像的系统为macos，目标平台为linux

则使用 ``docker compose build --platform linux/amd64`` 来进行交叉编译

### docker容器管理
``docker compose up -d`` 启动前后端服务，数据库及redis服务\
docker会自动拉取并建立postgres和redis容器，自动创建服务之间的网络，自动创建持久化的硬盘挂载

``docker compose stop`` 停止前后端服务

``docker compose down`` 停止并删除各服务

``docker compose down -v`` 停止并删除各服务，同时删除持久化数据

``docker compose logs <service>`` 查看某服务的日志

``docker ps`` 查看容器运行状态