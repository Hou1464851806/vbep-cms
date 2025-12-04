# 使用的库
- Gin: 提供API服务
- GORM: 提供数据库访问
- Viper: 解析配置文件Toml
- Zap: 日志库
- JWT: golang-jwt
- 字段校验: validator Gin内置

# 用户接口
- 用户的注册：创建用户，重定向至登录（自动登录)
- 用户的登录：验证用户信息，发放Token，维护Session
## 需要实现：
- 连接状态：Token、Session
- 中间件：验证登录状态
- 操作层：service，提供模型的CRUD