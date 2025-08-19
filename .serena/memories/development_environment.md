# SaveAny-Bot 开发环境设置

## 系统要求
- **Go版本**: 1.23.5 或更高
- **操作系统**: Linux (推荐), macOS, Windows
- **数据库**: SQLite (内置)
- **容器**: Docker (可选)

## 环境设置步骤

### 1. 克隆和初始化
```bash
git clone <repository-url>
cd SaveAny-Bot
go mod download
```

### 2. 配置文件设置
```bash
# 复制示例配置
cp config.example.toml config.toml

# 编辑配置文件，至少需要设置：
# - telegram.token (从 @BotFather 获取)
# - users (添加你的Telegram用户ID)
# - storages (配置至少一个存储端点)
```

### 3. 必要的外部依赖
- **Telegram Bot Token**: 从 [@BotFather](https://t.me/botfather) 获取
- **Telegram API凭证**: 从 [my.telegram.org](https://my.telegram.org/apps) 获取 (可选)

### 4. 开发工具
```bash
# 安装有用的Go工具
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## 项目结构理解
```
SaveAny-Bot/
├── cmd/                 # CLI命令定义
├── client/              # Telegram客户端
│   ├── bot/            # Bot处理器
│   ├── user/           # 用户客户端
│   └── middleware/     # 中间件
├── core/               # 核心业务逻辑
├── storage/            # 存储后端实现
├── config/             # 配置管理
├── database/           # 数据库模型
├── pkg/                # 可重用包
└── common/             # 通用工具
```

## 调试和开发技巧

### 日志配置
- 日志级别可在运行时调整
- 使用结构化日志，便于过滤和分析
- 支持中英文日志消息

### 测试策略
- 单元测试：专注于独立组件
- 集成测试：WebDAV客户端等外部依赖
- 并发测试：使用 `-race` 标志检测竞态条件

### 配置管理
- 开发环境使用本地配置文件
- 生产环境支持环境变量覆盖
- 配置验证在启动时进行

## 常见开发场景

### 添加新存储后端
1. 在 `storage/` 下创建新包
2. 实现 `Storage` 接口
3. 在 `storage/storage.go` 中注册构造函数
4. 添加相应的配置结构

### 添加新Bot命令
1. 在 `client/bot/handlers/` 中添加处理器
2. 在 `client/bot/handlers/register.go` 中注册
3. 更新命令列表 (在 `client/bot/bot.go`)

### 修改数据库模型
1. 更新 `database/model.go` 中的结构
2. 考虑数据迁移需求
3. 更新相关的数据库操作

## 性能监控
- 任务队列长度监控
- 下载速度和成功率
- 存储后端响应时间
- 内存和CPU使用情况