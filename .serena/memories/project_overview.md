# SaveAny-Bot 项目概览

## 项目目的
SaveAny-Bot 是一个用Go语言编写的Telegram机器人，可以将Telegram中的文件保存到多种存储端点。支持多种存储类型（本地存储、Alist、WebDAV、MinIO/S3、Telegram），多用户访问控制，基于规则的文件组织，以及流式下载。

## 技术栈
- **编程语言**: Go 1.23.5
- **CLI框架**: Cobra
- **Telegram客户端**: gotgproto (celestix/gotgproto)
- **数据库**: SQLite with GORM
- **配置**: TOML格式 (spf13/viper)
- **日志**: charmbracelet/log
- **国际化**: go-i18n
- **缓存**: ristretto
- **并发**: golang.org/x/sync
- **容器化**: Docker

## 核心组件
1. **命令层** (cmd/): 基于Cobra的CLI，包含主入口点和运行逻辑
2. **核心引擎** (core/): 任务队列管理和工作池执行
3. **客户端层** (client/): 
   - Bot客户端 (client/bot/) - Telegram机器人处理
   - 用户客户端 (client/user/) - 用户认证和管理
4. **存储抽象** (storage/): 可插拔的存储后端，统一接口
5. **配置** (config/): 基于TOML的配置，支持特定存储设置
6. **数据库** (database/): 基于SQLite的持久化，用于用户、聊天、规则和目录管理

## 关键设计模式
- **基于接口的存储**: 所有存储后端实现Storage接口，保证一致性处理
- **工作池**: 可配置数量的工作器并发处理下载任务
- **规则系统**: 通用规则匹配系统，用于自动文件组织
- **任务队列**: 基于优先级的任务执行，支持取消
- **中间件链**: Bot处理器的请求处理管道