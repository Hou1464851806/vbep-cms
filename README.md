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
1. 安装docker (docker-compose)
2. 执行`deploy.sh`一键部署
3. 首次部署时按提示输入默认 root 超级管理员邮箱
4. 部署完成后，从 `secrets/root_initial_password.txt` 获取 root 初始密码并立即修改
5. 网站暴露在8080端口上

## 部署相关
1. 基于docker容器部署服务
   1. 使用docker-compose管理多个服务
2. docker镜像的制作
   1. 后端服务底层镜像使用debian版本，不使用alpine版本（没有时区数据）
   2. 镜像最好直接提供打包后的文件
3. 自动化部署脚本
   1. 检测docker和docker-compose是否可用
   2. 自动导入镜像文件至docker
   3. 自动执行docker-compose部署

## 自动构建
1. 使用Github Action自动构建新版本的镜像文件
2. 使用Github Action打包部署所需所有文件
   1. deploy.sh: 部署脚本
   2. compose.yaml: docker compose配置
   3. secrets/: 数据库敏感配置
   4. images/: docker镜像

## 部署相关
1. 基于docker容器部署服务
   1. 使用docker-compose管理多个服务
2. docker镜像的制作
   1. 后端服务底层镜像使用debian版本，不使用alpine版本（没有时区数据）
   2. 镜像最好直接提供打包后的文件
3. 自动化部署脚本
   1. 检测docker和docker-compose是否可用
   2. 自动导入镜像文件至docker
   3. 自动执行docker-compose部署

## 自动构建
1. 使用Github Action自动构建新版本的镜像文件
2. 使用Github Action打包部署所需所有文件
   1. deploy.sh: 部署脚本
   2. compose.yaml: docker compose配置
   3. secrets/: 数据库敏感配置
   4. images/: docker镜像
